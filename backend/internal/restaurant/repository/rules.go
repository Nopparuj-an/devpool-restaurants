package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"restaurants/internal/booking"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/database"
	"restaurants/internal/restaurant/model"
)

// LoadRules reads what the booking rules need about a restaurant, plus its
// owner. With lock, the restaurant row stays locked until the transaction
// ends: that is the per-restaurant booking lock (ADR-0003, R-BOOK-6).
// Hidden restaurants (R-ADMIN-3, -4) are "not found" unless includeHidden.
// The reservation feature calls this from its own repository.
func LoadRules(ctx context.Context, q database.Querier, id int64, lock, includeHidden bool) (booking.Restaurant, int64, error) {
	var (
		r                   booking.Restaurant
		owner               int64
		cutoff, maxDuration int
		tz                  string
		hidden              bool
	)
	sql := `SELECT r.owner_id, r.seats, r.cancel_cutoff_minutes, r.max_reservation_minutes, r.timezone,
			NOT (r.banned_at IS NULL AND a.banned_at IS NULL)
		FROM restaurants r JOIN accounts a ON a.id = r.owner_id WHERE r.id = $1`
	if lock {
		sql += ` FOR UPDATE OF r`
	}
	err := q.QueryRow(ctx, sql, id).Scan(&owner, &r.Seats, &cutoff, &maxDuration, &tz, &hidden)
	if err == nil && hidden && !includeHidden {
		return r, 0, apperr.ErrNotFound
	}
	if database.IsNoRows(err) {
		return r, 0, apperr.ErrNotFound
	}
	if err != nil {
		return r, 0, err
	}
	r.CancelCutoff = time.Duration(cutoff) * time.Minute
	r.MaxDuration = time.Duration(maxDuration) * time.Minute
	if r.Location, err = time.LoadLocation(tz); err != nil {
		return r, 0, fmt.Errorf("restaurant %d timezone %q: %w", id, tz, err)
	}
	shifts, err := loadShifts(ctx, q, id)
	if err != nil {
		return r, 0, err
	}
	for _, sh := range shifts {
		r.Hours[sh.Weekday] = &booking.Shift{Open: sh.Open, Close: sh.Close}
	}
	return r, owner, nil
}

func loadShifts(ctx context.Context, q database.Querier, id int64) ([]model.Shift, error) {
	rows, _ := q.Query(ctx, `
		SELECT weekday, (extract(epoch FROM open_time) / 60)::int, (extract(epoch FROM close_time) / 60)::int
		FROM restaurant_hours WHERE restaurant_id = $1 ORDER BY weekday`, id)
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Shift, error) {
		var sh model.Shift
		err := row.Scan(&sh.Weekday, &sh.Open, &sh.Close)
		return sh, err
	})
}
