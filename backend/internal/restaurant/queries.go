package restaurant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"restaurants/internal/httpx"
)

// summarySelect is shared by List and Get. Rating is rounded to one decimal
// here so every client shows the same number (R-REVIEW-5).
const summarySelect = `
	SELECT r.id, r.name, r.cuisine, r.location, r.seats,
		CASE WHEN r.review_count > 0 THEN round(r.rating_sum::numeric / r.review_count, 1)::float8 END,
		r.review_count, coalesce(ci.object_key, ''), a.id, a.display_name
	FROM restaurants r
	JOIN accounts a ON a.id = r.owner_id
	LEFT JOIN restaurant_images ci ON ci.restaurant_id = r.id AND ci.is_cover`

func (s *Service) scanSummary(row pgx.Row, extra ...any) (Summary, error) {
	var (
		sm  Summary
		key string
	)
	dest := append([]any{&sm.ID, &sm.Name, &sm.Cuisine, &sm.Location, &sm.Seats,
		&sm.Rating, &sm.ReviewCount, &key, &sm.Owner.ID, &sm.Owner.DisplayName}, extra...)
	if err := row.Scan(dest...); err != nil {
		return sm, err
	}
	sm.CoverURL = s.imageURL(key)
	return sm, nil
}

type ListQuery struct {
	Sort    string // "top_rated" (default) | "most_reviewed" | "newest"
	Q       string // name contains, case-insensitive
	Cuisine string // exact, case-insensitive
	OwnerID int64  // 0 = any
	Limit   int
	Offset  int
}

// List returns restaurants for browsing (R-REVIEW-6 sorts).
func (s *Service) List(ctx context.Context, q ListQuery) ([]Summary, error) {
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

	var order string
	switch q.Sort {
	case "", "top_rated":
		// Bayesian average (ADR-0004): (sum + m·C) / (count + m), where C is
		// the mean of all reviews. Few-review restaurants lean toward C.
		order = fmt.Sprintf(`(r.rating_sum + %d * g.c) / (r.review_count + %d) DESC, r.review_count DESC, r.name`,
			bayesianPrior, bayesianPrior)
	case "most_reviewed":
		order = "r.review_count DESC, r.name"
	case "newest":
		order = "r.created_at DESC, r.id DESC"
	default:
		return nil, httpx.Invalid("sort must be top_rated, most_reviewed or newest")
	}

	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	sql := summarySelect + `
	CROSS JOIN (SELECT coalesce(avg(rating), 0)::float8 AS c FROM reviews) g`
	if len(where) > 0 {
		sql += "\n\tWHERE " + strings.Join(where, " AND ")
	}
	sql += "\n\tORDER BY " + order + " LIMIT " + arg(limit) + " OFFSET " + arg(max(q.Offset, 0))

	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	out := []Summary{}
	defer rows.Close()
	for rows.Next() {
		sm, err := s.scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

// Get returns one restaurant. viewer is the logged-in account ID or 0.
func (s *Service) Get(ctx context.Context, viewer, id int64) (Detail, error) {
	var d Detail
	sm, err := s.scanSummary(s.db.QueryRow(ctx, summarySelect+`
		WHERE r.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, httpx.ErrNotFound
	}
	if err != nil {
		return d, err
	}
	d.Summary = sm
	err = s.db.QueryRow(ctx, `
		SELECT description, cancel_cutoff_minutes, max_reservation_minutes, timezone
		FROM restaurants WHERE id = $1`, id).
		Scan(&d.Description, &d.CancelCutoffMinutes, &d.MaxReservationMinutes, &d.Timezone)
	if err != nil {
		return d, err
	}

	shifts, err := loadHours(ctx, s.db, id)
	if err != nil {
		return d, err
	}
	d.Hours = make([]Hours, len(shifts))
	for i, sh := range shifts {
		d.Hours[i] = Hours{sh.weekday, sh.open.String(), sh.close.String()}
	}

	rows, _ := s.db.Query(ctx, `
		SELECT id, object_key, is_cover FROM restaurant_images
		WHERE restaurant_id = $1 ORDER BY is_cover DESC, position`, id)
	d.Images, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Image, error) {
		var (
			img Image
			key string
		)
		err := row.Scan(&img.ID, &key, &img.IsCover)
		img.URL = s.imageURL(key)
		return img, err
	})
	if err != nil {
		return d, err
	}

	d.IsOwner = viewer != 0 && viewer == d.Owner.ID
	if d.IsOwner {
		var n int
		err := s.db.QueryRow(ctx, `
			SELECT count(*) FROM reservations
			WHERE restaurant_id = $1 AND status = 'active' AND starts_at > $2`, id, s.now()).Scan(&n)
		if err != nil {
			return d, err
		}
		d.UpcomingReservations = &n
	}
	return d, nil
}
