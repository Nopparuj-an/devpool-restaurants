// Package repository implements the restaurant service's Repository port with
// SQL, plus LoadRules, which the reservation feature uses to lock a
// restaurant and read its booking rules.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/database"
	"restaurants/internal/restaurant/model"
)

// bayesianPrior is m in ADR-0004: how many "average" reviews every
// restaurant is assumed to start with when ranking by rating.
const bayesianPrior = 5

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) LockOwner(ctx context.Context, id int64) (int64, error) {
	var owner int64
	err := r.db.Conn(ctx).QueryRow(ctx, `SELECT owner_id FROM restaurants WHERE id = $1 FOR UPDATE`, id).Scan(&owner)
	if database.IsNoRows(err) {
		return 0, apperr.ErrNotFound
	}
	return owner, err
}

func (r *Repository) Insert(ctx context.Context, ownerID int64, v model.Valid) (int64, error) {
	var id int64
	err := r.db.Conn(ctx).QueryRow(ctx, `
		INSERT INTO restaurants (owner_id, name, description, cuisine, location, seats,
			cancel_cutoff_minutes, max_reservation_minutes, timezone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		ownerID, v.Name, v.Description, v.Cuisine, v.Location, v.Seats, v.CutoffMinutes, v.MaxMinutes, v.Timezone).Scan(&id)
	return id, err
}

func (r *Repository) UpdateDetails(ctx context.Context, id int64, v model.Valid) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `
		UPDATE restaurants SET name = $2, description = $3, cuisine = $4, location = $5,
			seats = $6, cancel_cutoff_minutes = $7, max_reservation_minutes = $8, updated_at = now()
		WHERE id = $1`,
		id, v.Name, v.Description, v.Cuisine, v.Location, v.Seats, v.CutoffMinutes, v.MaxMinutes)
	return err
}

func (r *Repository) ReplaceHours(ctx context.Context, id int64, shifts []model.Shift) error {
	q := r.db.Conn(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM restaurant_hours WHERE restaurant_id = $1`, id); err != nil {
		return err
	}
	for _, sh := range shifts {
		if _, err := q.Exec(ctx, `
			INSERT INTO restaurant_hours (restaurant_id, weekday, open_time, close_time)
			VALUES ($1, $2, $3::time, $4::time)`, id, sh.Weekday, sh.Open.String(), sh.Close.String()); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `DELETE FROM restaurants WHERE id = $1`, id)
	return err
}

func (r *Repository) ImageKeys(ctx context.Context, id int64) ([]string, error) {
	rows, _ := r.db.Conn(ctx).Query(ctx, `SELECT object_key FROM restaurant_images WHERE restaurant_id = $1`, id)
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (r *Repository) ImageStats(ctx context.Context, id int64) (model.ImageStats, error) {
	var s model.ImageStats
	err := r.db.Conn(ctx).QueryRow(ctx, `
		SELECT count(*), coalesce(max(position) + 1, 0), coalesce(bool_or(is_cover), false)
		FROM restaurant_images WHERE restaurant_id = $1`, id).Scan(&s.Count, &s.NextPosition, &s.HasCover)
	return s, err
}

func (r *Repository) InsertImage(ctx context.Context, id int64, key string, isCover bool, position int) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `
		INSERT INTO restaurant_images (restaurant_id, object_key, is_cover, position)
		VALUES ($1, $2, $3, $4)`, id, key, isCover, position)
	return err
}

func (r *Repository) DeleteImage(ctx context.Context, id, imageID int64) (string, bool, error) {
	var (
		key      string
		wasCover bool
	)
	err := r.db.Conn(ctx).QueryRow(ctx, `
		DELETE FROM restaurant_images WHERE id = $1 AND restaurant_id = $2
		RETURNING object_key, is_cover`, imageID, id).Scan(&key, &wasCover)
	if database.IsNoRows(err) {
		return "", false, apperr.ErrNotFound
	}
	return key, wasCover, err
}

func (r *Repository) PromoteFirstImage(ctx context.Context, id int64) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `
		UPDATE restaurant_images SET is_cover = true
		WHERE id = (SELECT id FROM restaurant_images WHERE restaurant_id = $1 ORDER BY position LIMIT 1)`, id)
	return err
}

func (r *Repository) SetCover(ctx context.Context, id, imageID int64) error {
	q := r.db.Conn(ctx)
	var exists bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM restaurant_images WHERE id = $1 AND restaurant_id = $2)`, imageID, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return apperr.ErrNotFound
	}
	// Two statements: the one-cover unique index is checked row by row.
	if _, err := q.Exec(ctx, `UPDATE restaurant_images SET is_cover = false WHERE restaurant_id = $1 AND is_cover`, id); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `UPDATE restaurant_images SET is_cover = true WHERE id = $1`, imageID)
	return err
}

// summarySelect is shared by List and Get. The rating is rounded to one
// decimal here so every client shows the same number (R-REVIEW-5).
const summarySelect = `
	SELECT r.id, r.name, r.cuisine, r.location, r.seats,
		CASE WHEN r.review_count > 0 THEN round(r.rating_sum::numeric / r.review_count, 1)::float8 END,
		r.review_count, coalesce(ci.object_key, ''), a.id, a.display_name
	FROM restaurants r
	JOIN accounts a ON a.id = r.owner_id
	LEFT JOIN restaurant_images ci ON ci.restaurant_id = r.id AND ci.is_cover`

func scanSummary(row pgx.Row) (model.Summary, error) {
	var sm model.Summary
	err := row.Scan(&sm.ID, &sm.Name, &sm.Cuisine, &sm.Location, &sm.Seats,
		&sm.Rating, &sm.ReviewCount, &sm.CoverKey, &sm.Owner.ID, &sm.Owner.DisplayName)
	return sm, err
}

// List sorts by the Bayesian average for top_rated (ADR-0004):
// (sum + m·C) / (count + m), where C is the mean of all reviews.
func (r *Repository) List(ctx context.Context, q model.ListQuery) ([]model.Summary, error) {
	var (
		where []string
		args  []any
	)
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if q.Q != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q.Q)
		where = append(where, "r.name ILIKE '%' || "+arg(escaped)+" || '%'")
	}
	if q.Cuisine != "" {
		where = append(where, "lower(r.cuisine) = lower("+arg(q.Cuisine)+")")
	}
	if q.OwnerID != 0 {
		where = append(where, "r.owner_id = "+arg(q.OwnerID))
	}
	order := map[string]string{
		"":              fmt.Sprintf("(r.rating_sum + %[1]d * g.c) / (r.review_count + %[1]d) DESC, r.review_count DESC, r.name", bayesianPrior),
		"most_reviewed": "r.review_count DESC, r.name",
		"newest":        "r.created_at DESC, r.id DESC",
	}
	order["top_rated"] = order[""]

	sql := summarySelect + `
	CROSS JOIN (SELECT coalesce(avg(rating), 0)::float8 AS c FROM reviews) g`
	if len(where) > 0 {
		sql += "\n\tWHERE " + strings.Join(where, " AND ")
	}
	sql += "\n\tORDER BY " + order[q.Sort] + " LIMIT " + arg(q.Limit) + " OFFSET " + arg(q.Offset)

	rows, err := r.db.Conn(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Summary{}
	for rows.Next() {
		sm, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (model.Detail, error) {
	q := r.db.Conn(ctx)
	var d model.Detail
	sm, err := scanSummary(q.QueryRow(ctx, summarySelect+` WHERE r.id = $1`, id))
	if database.IsNoRows(err) {
		return d, apperr.ErrNotFound
	}
	if err != nil {
		return d, err
	}
	d.Summary = sm
	if err := q.QueryRow(ctx, `
		SELECT description, cancel_cutoff_minutes, max_reservation_minutes, timezone
		FROM restaurants WHERE id = $1`, id).
		Scan(&d.Description, &d.CancelCutoffMinutes, &d.MaxReservationMinutes, &d.Timezone); err != nil {
		return d, err
	}

	shifts, err := loadShifts(ctx, q, id)
	if err != nil {
		return d, err
	}
	d.Hours = make([]model.Hours, len(shifts))
	for i, sh := range shifts {
		d.Hours[i] = model.Hours{Weekday: sh.Weekday, Open: sh.Open.String(), Close: sh.Close.String()}
	}

	rows, _ := q.Query(ctx, `
		SELECT id, object_key, is_cover FROM restaurant_images
		WHERE restaurant_id = $1 ORDER BY is_cover DESC, position`, id)
	d.Images, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Image, error) {
		var img model.Image
		err := row.Scan(&img.ID, &img.Key, &img.IsCover)
		return img, err
	})
	return d, err
}

func (r *Repository) UpcomingReservations(ctx context.Context, id int64, now time.Time) (int, error) {
	var n int
	err := r.db.Conn(ctx).QueryRow(ctx, `
		SELECT count(*) FROM reservations
		WHERE restaurant_id = $1 AND status = 'active' AND starts_at > $2`, id, now).Scan(&n)
	return n, err
}
