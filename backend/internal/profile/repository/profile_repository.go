// Package repository implements the profile service's Repository port with SQL.
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/database"
	"restaurants/internal/profile/model"
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository { return &Repository{db: db} }

// visibleReview is the SQL condition for a review customers can see: its
// author (u), its restaurant (r) and that restaurant's owner (o) are not
// banned (R-ADMIN-3, -4). $2 = include hidden rows anyway.
const visibleReview = `($2 OR (u.banned_at IS NULL AND r.banned_at IS NULL AND o.banned_at IS NULL))`

func (r *Repository) Profile(ctx context.Context, id int64, includeHidden bool) (model.Profile, error) {
	var p model.Profile
	err := r.db.Conn(ctx).QueryRow(ctx, `
		SELECT u.id, u.display_name, u.created_at, u.banned_at IS NOT NULL,
			(SELECT count(*) FROM restaurants r
				WHERE r.owner_id = u.id AND ($2 OR (r.banned_at IS NULL AND u.banned_at IS NULL))),
			(SELECT count(*) FROM reviews rv
				JOIN restaurants r ON r.id = rv.restaurant_id
				JOIN accounts o ON o.id = r.owner_id
				WHERE rv.account_id = u.id AND `+visibleReview+`)
		FROM accounts u WHERE u.id = $1`, id, includeHidden).
		Scan(&p.ID, &p.DisplayName, &p.CreatedAt, &p.Banned, &p.RestaurantCount, &p.ReviewCount)
	if database.IsNoRows(err) {
		return p, apperr.ErrNotFound
	}
	return p, err
}

func (r *Repository) Reviews(ctx context.Context, id int64, includeHidden bool, now time.Time, limit, offset int) ([]model.Review, int, error) {
	rows, err := r.db.Conn(ctx).Query(ctx, `
		SELECT count(*) OVER (), rv.id, rv.rating, rv.body, rv.created_at, rv.updated_at,
			r.id, r.name, coalesce(img.object_key, ''),
			NOT (r.banned_at IS NULL AND o.banned_at IS NULL),
			EXISTS (
				SELECT 1 FROM reservations v
				WHERE v.restaurant_id = rv.restaurant_id AND v.account_id = rv.account_id
					AND v.status = 'active' AND v.ends_at <= $3
			)
		FROM reviews rv
		JOIN accounts u ON u.id = rv.account_id
		JOIN restaurants r ON r.id = rv.restaurant_id
		JOIN accounts o ON o.id = r.owner_id
		LEFT JOIN restaurant_images img ON img.restaurant_id = r.id AND img.is_cover
		WHERE rv.account_id = $1 AND `+visibleReview+`
		ORDER BY rv.updated_at DESC, rv.id DESC
		LIMIT $4 OFFSET $5`, id, includeHidden, now, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Review, error) {
		var rv model.Review
		err := row.Scan(&total, &rv.ID, &rv.Rating, &rv.Body, &rv.CreatedAt, &rv.UpdatedAt,
			&rv.Restaurant.ID, &rv.Restaurant.Name, &rv.Restaurant.CoverKey, &rv.Hidden, &rv.Verified)
		return rv, err
	})
	if err != nil {
		return nil, 0, err
	}
	if len(list) == 0 && offset > 0 {
		// Past the last page the window has no rows to report the total on.
		p, err := r.Profile(ctx, id, includeHidden)
		return list, p.ReviewCount, err
	}
	return list, total, nil
}
