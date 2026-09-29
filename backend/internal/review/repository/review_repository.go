// Package repository implements the review service's Repository port with SQL.
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/database"
	restaurantrepo "restaurants/internal/restaurant/repository"
	"restaurants/internal/review/model"
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) LockRestaurant(ctx context.Context, restaurantID int64) (int64, error) {
	var owner int64
	err := r.db.Conn(ctx).QueryRow(ctx, `
		SELECT r.owner_id FROM restaurants r JOIN accounts a ON a.id = r.owner_id
		WHERE r.id = $1 AND `+restaurantrepo.Visible+` FOR UPDATE OF r`, restaurantID).Scan(&owner)
	if database.IsNoRows(err) {
		return 0, apperr.ErrNotFound // missing or hidden (R-ADMIN-3, -4)
	}
	return owner, err
}

func (r *Repository) Restaurant(ctx context.Context, restaurantID int64) (int64, bool, error) {
	var (
		owner  int64
		hidden bool
	)
	err := r.db.Conn(ctx).QueryRow(ctx, `
		SELECT r.owner_id, NOT (`+restaurantrepo.Visible+`)
		FROM restaurants r JOIN accounts a ON a.id = r.owner_id WHERE r.id = $1`, restaurantID).Scan(&owner, &hidden)
	if database.IsNoRows(err) {
		return 0, false, apperr.ErrNotFound
	}
	return owner, hidden, err
}

func (r *Repository) Rating(ctx context.Context, restaurantID, accountID int64) (int, error) {
	var rating int
	err := r.db.Conn(ctx).QueryRow(ctx,
		`SELECT rating FROM reviews WHERE restaurant_id = $1 AND account_id = $2`, restaurantID, accountID).Scan(&rating)
	if database.IsNoRows(err) {
		return 0, apperr.ErrNotFound
	}
	return rating, err
}

func (r *Repository) Insert(ctx context.Context, restaurantID, accountID int64, rating int, body string) error {
	_, err := r.db.Conn(ctx).Exec(ctx,
		`INSERT INTO reviews (restaurant_id, account_id, rating, body) VALUES ($1, $2, $3, $4)`,
		restaurantID, accountID, rating, body)
	return err
}

func (r *Repository) Update(ctx context.Context, restaurantID, accountID int64, rating int, body string) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `
		UPDATE reviews SET rating = $3, body = $4, updated_at = now()
		WHERE restaurant_id = $1 AND account_id = $2`, restaurantID, accountID, rating, body)
	return err
}

func (r *Repository) Delete(ctx context.Context, restaurantID, accountID int64) error {
	tag, err := r.db.Conn(ctx).Exec(ctx,
		`DELETE FROM reviews WHERE restaurant_id = $1 AND account_id = $2`, restaurantID, accountID)
	if err == nil && tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return err
}

// listSelect includes "verified": the author has a completed reservation here.
const listSelect = `
	SELECT rv.id, rv.rating, rv.body, rv.created_at, rv.updated_at,
		a.id, a.display_name, a.email, r.owner_id,
		EXISTS (
			SELECT 1 FROM reservations v
			WHERE v.restaurant_id = rv.restaurant_id AND v.account_id = rv.account_id
				AND v.status = 'active' AND v.ends_at <= $2
		)
	FROM reviews rv
	JOIN accounts a ON a.id = rv.account_id AND a.banned_at IS NULL -- banned reviewers are hidden (R-ADMIN-3)
	JOIN restaurants r ON r.id = rv.restaurant_id`

func (r *Repository) collect(ctx context.Context, sql string, args ...any) ([]model.Review, error) {
	rows, err := r.db.Conn(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Review, error) {
		var rv model.Review
		err := row.Scan(&rv.ID, &rv.Rating, &rv.Body, &rv.CreatedAt, &rv.UpdatedAt,
			&rv.Author.ID, &rv.Author.DisplayName, &rv.Author.Email, &rv.OwnerID, &rv.Verified)
		return rv, err
	})
}

func (r *Repository) List(ctx context.Context, restaurantID int64, now time.Time, limit, offset int) ([]model.Review, int, error) {
	q := r.db.Conn(ctx)
	var total int
	if err := q.QueryRow(ctx, `SELECT review_count FROM restaurants WHERE id = $1`, restaurantID).Scan(&total); err != nil {
		return nil, 0, err
	}
	list, err := r.collect(ctx, listSelect+`
		WHERE rv.restaurant_id = $1
		ORDER BY rv.updated_at DESC, rv.id DESC
		LIMIT $3 OFFSET $4`, restaurantID, now, limit, offset)
	return list, total, err
}

func (r *Repository) Mine(ctx context.Context, restaurantID, accountID int64, now time.Time) (model.Review, error) {
	list, err := r.collect(ctx, listSelect+` WHERE rv.restaurant_id = $1 AND rv.account_id = $3`, restaurantID, now, accountID)
	if err != nil {
		return model.Review{}, err
	}
	if len(list) == 0 {
		return model.Review{}, apperr.ErrNotFound
	}
	return list[0], nil
}
