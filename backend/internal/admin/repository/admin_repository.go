// Package repository implements the admin service's Repository port with SQL.
package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"restaurants/internal/admin/model"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/database"
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository { return &Repository{db: db} }

const userSelect = `
	SELECT a.id, a.email, a.display_name, a.is_admin, a.banned_at, a.ban_reason, a.created_at,
		(SELECT count(*) FROM restaurants r WHERE r.owner_id = a.id),
		(SELECT count(*) FROM reviews rv WHERE rv.account_id = a.id),
		(SELECT count(*) FROM reservations v WHERE v.account_id = a.id)
	FROM accounts a`

func userDest(u *model.User) []any {
	return []any{&u.ID, &u.Email, &u.DisplayName, &u.IsAdmin, &u.BannedAt, &u.BanReason, &u.CreatedAt,
		&u.Restaurants, &u.Reviews, &u.Reservations}
}

// filters builds the WHERE clause shared by a list and its count.
type filters struct {
	where []string
	args  []any
}

func (f *filters) add(cond string, v any) {
	f.args = append(f.args, v)
	f.where = append(f.where, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(f.args))))
}

func (f *filters) sql() string {
	if len(f.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(f.where, " AND ")
}

func likeArg(q string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
}

func statusCond(col, status string) string {
	if status == "banned" {
		return col + " IS NOT NULL"
	}
	return col + " IS NULL"
}

func (r *Repository) Users(ctx context.Context, q model.ListQuery) ([]model.User, int, error) {
	var f filters
	if q.Q != "" {
		f.add("(a.email ILIKE ? OR a.display_name ILIKE ?)", likeArg(q.Q))
	}
	if q.Status != "" {
		f.where = append(f.where, statusCond("a.banned_at", q.Status))
	}
	conn := r.db.Conn(ctx)
	var total int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM accounts a"+f.sql(), f.args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := append(f.args, q.Limit, q.Offset)
	rows, _ := conn.Query(ctx, userSelect+f.sql()+
		fmt.Sprintf(" ORDER BY a.created_at DESC, a.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.User, error) {
		var u model.User
		err := row.Scan(userDest(&u)...)
		return u, err
	})
	return list, total, err
}

func (r *Repository) User(ctx context.Context, id int64) (model.User, error) {
	var u model.User
	err := r.db.Conn(ctx).QueryRow(ctx, userSelect+" WHERE a.id = $1", id).Scan(userDest(&u)...)
	if database.IsNoRows(err) {
		return u, apperr.ErrNotFound
	}
	return u, err
}

func (r *Repository) LockUser(ctx context.Context, id int64) (model.User, error) {
	var u model.User
	err := r.db.Conn(ctx).QueryRow(ctx,
		`SELECT id, is_admin, banned_at FROM accounts WHERE id = $1 FOR UPDATE`, id).Scan(&u.ID, &u.IsAdmin, &u.BannedAt)
	if database.IsNoRows(err) {
		return u, apperr.ErrNotFound
	}
	return u, err
}

func (r *Repository) SetUserBan(ctx context.Context, id int64, at *time.Time, reason string) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `UPDATE accounts SET banned_at = $2, ban_reason = $3 WHERE id = $1`, id, at, reason)
	return err
}

func (r *Repository) SetDisplayName(ctx context.Context, id int64, name string) error {
	tag, err := r.db.Conn(ctx).Exec(ctx, `UPDATE accounts SET display_name = $2 WHERE id = $1`, id, name)
	if err == nil && tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return err
}

func (r *Repository) DeleteSessions(ctx context.Context, accountID int64) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `DELETE FROM sessions WHERE account_id = $1`, accountID)
	return err
}

func (r *Repository) LockUsers(ctx context.Context, ids []int64) ([]model.User, error) {
	rows, _ := r.db.Conn(ctx).Query(ctx,
		`SELECT id, is_admin FROM accounts WHERE id = ANY($1) ORDER BY id FOR UPDATE`, ids)
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.User, error) {
		var u model.User
		err := row.Scan(&u.ID, &u.IsAdmin)
		return u, err
	})
}

func (r *Repository) DeleteUsers(ctx context.Context, ids []int64) (int, []string, error) {
	return r.deleteWithImages(ctx, "owner_id", `DELETE FROM accounts WHERE id = ANY($1)`, ids)
}

func (r *Repository) DeleteRestaurants(ctx context.Context, ids []int64) (int, []string, error) {
	return r.deleteWithImages(ctx, "id", `DELETE FROM restaurants WHERE id = ANY($1)`, ids)
}

// deleteWithImages locks the restaurants that will go (so no photo is added
// meanwhile), reads their photo keys, then runs del. Must run in a transaction.
func (r *Repository) deleteWithImages(ctx context.Context, col, del string, ids []int64) (int, []string, error) {
	q := r.db.Conn(ctx)
	if _, err := q.Exec(ctx, `SELECT 1 FROM restaurants WHERE `+col+` = ANY($1) ORDER BY id FOR UPDATE`, ids); err != nil {
		return 0, nil, err
	}
	rows, _ := q.Query(ctx, `
		SELECT i.object_key FROM restaurant_images i JOIN restaurants r ON r.id = i.restaurant_id
		WHERE r.`+col+` = ANY($1)`, ids)
	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, nil, err
	}
	tag, err := q.Exec(ctx, del, ids)
	if err != nil {
		return 0, nil, err
	}
	return int(tag.RowsAffected()), keys, nil
}

const restaurantSelect = `
	SELECT r.id, r.name, r.cuisine, r.location,
		CASE WHEN r.review_count > 0 THEN round(r.rating_sum::numeric / r.review_count, 1)::float8 END,
		r.review_count, r.banned_at, r.ban_reason, r.created_at,
		a.id, a.display_name, a.email, a.banned_at IS NOT NULL
	FROM restaurants r JOIN accounts a ON a.id = r.owner_id`

func scanRestaurants(rows pgx.Rows) ([]model.Restaurant, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Restaurant, error) {
		var x model.Restaurant
		err := row.Scan(&x.ID, &x.Name, &x.Cuisine, &x.Location, &x.Rating, &x.ReviewCount,
			&x.BannedAt, &x.BanReason, &x.CreatedAt, &x.Owner.ID, &x.Owner.DisplayName, &x.Owner.Email, &x.Owner.Banned)
		return x, err
	})
}

func (r *Repository) OwnedRestaurants(ctx context.Context, ownerID int64) ([]model.Restaurant, error) {
	rows, _ := r.db.Conn(ctx).Query(ctx, restaurantSelect+` WHERE r.owner_id = $1 ORDER BY r.created_at DESC, r.id DESC LIMIT 100`, ownerID)
	return scanRestaurants(rows)
}

func (r *Repository) Restaurants(ctx context.Context, q model.ListQuery) ([]model.Restaurant, int, error) {
	var f filters
	if q.Q != "" {
		f.add("(r.name ILIKE ? OR r.cuisine ILIKE ? OR a.email ILIKE ?)", likeArg(q.Q))
	}
	if q.Status != "" {
		f.where = append(f.where, statusCond("r.banned_at", q.Status))
	}
	conn := r.db.Conn(ctx)
	var total int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM restaurants r JOIN accounts a ON a.id = r.owner_id"+f.sql(), f.args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := append(f.args, q.Limit, q.Offset)
	rows, _ := conn.Query(ctx, restaurantSelect+f.sql()+
		fmt.Sprintf(" ORDER BY r.created_at DESC, r.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	list, err := scanRestaurants(rows)
	return list, total, err
}

func (r *Repository) SetRestaurantBan(ctx context.Context, id int64, at *time.Time, reason string) error {
	tag, err := r.db.Conn(ctx).Exec(ctx, `UPDATE restaurants SET banned_at = $2, ban_reason = $3 WHERE id = $1`, id, at, reason)
	if err == nil && tag.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return err
}
