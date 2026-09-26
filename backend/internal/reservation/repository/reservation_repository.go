// Package repository implements the reservation service's Repository port with SQL.
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"restaurants/internal/booking"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/database"
	"restaurants/internal/reservation/model"
	restaurantrepo "restaurants/internal/restaurant/repository"
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository { return &Repository{db: db} }

func (r *Repository) RestaurantRules(ctx context.Context, restaurantID int64, lock, includeHidden bool) (booking.Restaurant, int64, error) {
	return restaurantrepo.LoadRules(ctx, r.db.Conn(ctx), restaurantID, lock, includeHidden)
}

func (r *Repository) RestaurantOf(ctx context.Context, reservationID int64) (int64, error) {
	var id int64
	err := r.db.Conn(ctx).QueryRow(ctx, `SELECT restaurant_id FROM reservations WHERE id = $1`, reservationID).Scan(&id)
	if database.IsNoRows(err) {
		return 0, apperr.ErrNotFound
	}
	return id, err
}

func (r *Repository) LockReservation(ctx context.Context, id int64) (model.Locked, error) {
	var l model.Locked
	err := r.db.Conn(ctx).QueryRow(ctx, `
		SELECT account_id, status, pax, starts_at, ends_at FROM reservations WHERE id = $1 FOR UPDATE`, id).
		Scan(&l.AccountID, &l.Status, &l.Pax, &l.Start, &l.End)
	if database.IsNoRows(err) {
		return l, apperr.ErrNotFound
	}
	return l, err
}

func (r *Repository) Overlapping(ctx context.Context, restaurantID int64, iv booking.Interval, excludeID int64) ([]booking.Reservation, error) {
	rows, _ := r.db.Conn(ctx).Query(ctx, `
		SELECT pax, starts_at, ends_at FROM reservations
		WHERE restaurant_id = $1 AND status = 'active'
			AND starts_at < $3 AND ends_at > $2 AND id <> $4`,
		restaurantID, iv.Start, iv.End, excludeID)
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (booking.Reservation, error) {
		var b booking.Reservation
		err := row.Scan(&b.Pax, &b.Start, &b.End)
		return b, err
	})
}

func (r *Repository) Insert(ctx context.Context, restaurantID, accountID int64, b booking.Reservation) (int64, error) {
	var id int64
	err := r.db.Conn(ctx).QueryRow(ctx, `
		INSERT INTO reservations (restaurant_id, account_id, pax, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		restaurantID, accountID, b.Pax, b.Start, b.End).Scan(&id)
	return id, err
}

func (r *Repository) Update(ctx context.Context, id int64, b booking.Reservation) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `
		UPDATE reservations SET pax = $2, starts_at = $3, ends_at = $4, updated_at = now()
		WHERE id = $1`, id, b.Pax, b.Start, b.End)
	return err
}

func (r *Repository) Cancel(ctx context.Context, id int64) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `UPDATE reservations SET status = 'cancelled', updated_at = now() WHERE id = $1`, id)
	return err
}

// listSelect returns reservations with their restaurant and customer.
const listSelect = `
	SELECT v.id, v.pax, v.starts_at, v.ends_at, v.status, r.cancel_cutoff_minutes,
		r.id, r.name, coalesce(ci.object_key, ''), a.id, a.display_name, a.email
	FROM reservations v
	JOIN restaurants r ON r.id = v.restaurant_id
	JOIN accounts a ON a.id = v.account_id
	LEFT JOIN restaurant_images ci ON ci.restaurant_id = r.id AND ci.is_cover`

func (r *Repository) list(ctx context.Context, sql string, args ...any) ([]model.Reservation, error) {
	rows, err := r.db.Conn(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Reservation, error) {
		var (
			res model.Reservation
			c   model.Customer
		)
		err := row.Scan(&res.ID, &res.Pax, &res.StartsAt, &res.EndsAt, &res.Status, &res.CutoffMinutes,
			&res.Restaurant.ID, &res.Restaurant.Name, &res.Restaurant.CoverKey, &c.ID, &c.DisplayName, &c.Email)
		res.Customer = &c
		return res, err
	})
}

func (r *Repository) ByID(ctx context.Context, accountID, id int64) (model.Reservation, error) {
	list, err := r.list(ctx, listSelect+` WHERE v.id = $1 AND v.account_id = $2`, id, accountID)
	if err != nil {
		return model.Reservation{}, err
	}
	if len(list) == 0 {
		return model.Reservation{}, apperr.ErrNotFound
	}
	return list[0], nil
}

// ByAccount lists current reservations first (soonest first), then past or
// cancelled ones (most recent first).
func (r *Repository) ByAccount(ctx context.Context, accountID int64, now time.Time, limit, offset int) ([]model.Reservation, int, error) {
	var total int
	if err := r.db.Conn(ctx).QueryRow(ctx, `SELECT count(*) FROM reservations WHERE account_id = $1`, accountID).Scan(&total); err != nil {
		return nil, 0, err
	}
	list, err := r.list(ctx, listSelect+`
		WHERE v.account_id = $1
		ORDER BY
			(v.ends_at <= $2 OR v.status = 'cancelled'),
			CASE WHEN v.ends_at > $2 AND v.status = 'active' THEN v.starts_at END,
			v.starts_at DESC, v.id DESC
		LIMIT $3 OFFSET $4`, accountID, now, limit, offset)
	return list, total, err
}

func (r *Repository) ByRestaurant(ctx context.Context, restaurantID int64, from, to time.Time) ([]model.Reservation, error) {
	return r.list(ctx, listSelect+`
		WHERE v.restaurant_id = $1 AND v.starts_at < $3 AND v.ends_at > $2
		ORDER BY v.starts_at, v.id`, restaurantID, from, to)
}
