// Package service holds the admin rules: who can be banned, and what a ban
// does (R-ADMIN-*). Bans are reversible: they set banned_at, never delete.
package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"restaurants/internal/admin/model"
)

// Repository is the outbound port for admin data.
type Repository interface {
	Users(ctx context.Context, q model.ListQuery) ([]model.User, int, error)
	User(ctx context.Context, id int64) (model.User, error) // apperr.ErrNotFound if missing
	OwnedRestaurants(ctx context.Context, ownerID int64) ([]model.Restaurant, error)
	Restaurants(ctx context.Context, q model.ListQuery) ([]model.Restaurant, int, error)

	// LockUser locks the account row and returns it; apperr.ErrNotFound if missing.
	LockUser(ctx context.Context, id int64) (model.User, error)
	SetUserBan(ctx context.Context, id int64, at *time.Time, reason string) error
	SetDisplayName(ctx context.Context, id int64, name string) error // apperr.ErrNotFound if missing
	DeleteSessions(ctx context.Context, accountID int64) error
	SetRestaurantBan(ctx context.Context, id int64, at *time.Time, reason string) error // ErrNotFound if missing
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service is the inbound port used by the HTTP handlers. `admin` is the
// acting admin's account ID.
type Service interface {
	ListUsers(ctx context.Context, q model.ListQuery) ([]model.User, int, error)
	GetUser(ctx context.Context, id int64) (model.UserDetail, error)
	// UpdateUser edits another user's profile (R-ADMIN-6).
	UpdateUser(ctx context.Context, id int64, in model.UserInput) (model.User, error)
	BanUser(ctx context.Context, admin, id int64, reason string) (model.User, error)
	UnbanUser(ctx context.Context, admin, id int64) (model.User, error)
	ListRestaurants(ctx context.Context, q model.ListQuery) ([]model.Restaurant, int, error)
	BanRestaurant(ctx context.Context, id int64, reason string) error
	UnbanRestaurant(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
	tx   TxRunner
	now  func() time.Time
}

func New(repo Repository, tx TxRunner) Service {
	return &service{repo: repo, tx: tx, now: time.Now}
}

func normalize(q model.ListQuery) (model.ListQuery, error) {
	switch q.Status {
	case "", "all":
		q.Status = ""
	case "active", "banned":
	default:
		return q, model.ErrBadStatus
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}
	q.Offset = max(q.Offset, 0)
	q.Q = strings.TrimSpace(q.Q)
	return q, nil
}

func (s *service) ListUsers(ctx context.Context, q model.ListQuery) ([]model.User, int, error) {
	q, err := normalize(q)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.Users(ctx, q)
}

func (s *service) GetUser(ctx context.Context, id int64) (model.UserDetail, error) {
	u, err := s.repo.User(ctx, id)
	if err != nil {
		return model.UserDetail{}, err
	}
	owned, err := s.repo.OwnedRestaurants(ctx, id)
	return model.UserDetail{User: u, OwnedRestaurants: owned}, err
}

func (s *service) UpdateUser(ctx context.Context, id int64, in model.UserInput) (model.User, error) {
	name := strings.TrimSpace(in.DisplayName)
	if n := utf8.RuneCountInString(name); n < 1 || n > 80 {
		return model.User{}, model.ErrName
	}
	if err := s.repo.SetDisplayName(ctx, id, name); err != nil {
		return model.User{}, err
	}
	return s.repo.User(ctx, id)
}

// BanUser suspends an account (R-ADMIN-3): it can't log in, its sessions end
// now, its restaurants and reviews are hidden, and the ratings it affected
// are recomputed without its reviews (by a database trigger, ADR-0015).
func (s *service) BanUser(ctx context.Context, admin, id int64, reason string) (model.User, error) {
	reason, err := checkReason(reason)
	if err != nil {
		return model.User{}, err
	}
	if id == admin {
		return model.User{}, model.ErrBanSelf
	}
	now := s.now()
	return s.setUserBan(ctx, id, func(ctx context.Context, u model.User) error {
		if u.IsAdmin {
			return model.ErrBanAdmin
		}
		if err := s.repo.SetUserBan(ctx, id, &now, reason); err != nil {
			return err
		}
		return s.repo.DeleteSessions(ctx, id)
	})
}

// UnbanUser reverses BanUser; ratings are recomputed with the reviews back in.
func (s *service) UnbanUser(ctx context.Context, admin, id int64) (model.User, error) {
	return s.setUserBan(ctx, id, func(ctx context.Context, _ model.User) error {
		return s.repo.SetUserBan(ctx, id, nil, "")
	})
}

// setUserBan locks the account and applies change in one transaction; the
// ban trigger recomputes the ratings the user's reviews count in. change receives
// the transaction's ctx and must use it (not an outer ctx), or its writes run
// outside the transaction and wait forever on the row lock.
func (s *service) setUserBan(ctx context.Context, id int64, change func(ctx context.Context, u model.User) error) (model.User, error) {
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		u, err := s.repo.LockUser(ctx, id)
		if err != nil {
			return err
		}
		return change(ctx, u)
	})
	if err != nil {
		return model.User{}, err
	}
	return s.repo.User(ctx, id)
}

func (s *service) ListRestaurants(ctx context.Context, q model.ListQuery) ([]model.Restaurant, int, error) {
	q, err := normalize(q)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.Restaurants(ctx, q)
}

// BanRestaurant hides a restaurant from customers (R-ADMIN-4). Its bookings
// and reviews are kept, so unbanning restores it as it was.
func (s *service) BanRestaurant(ctx context.Context, id int64, reason string) error {
	reason, err := checkReason(reason)
	if err != nil {
		return err
	}
	now := s.now()
	return s.repo.SetRestaurantBan(ctx, id, &now, reason)
}

func (s *service) UnbanRestaurant(ctx context.Context, id int64) error {
	return s.repo.SetRestaurantBan(ctx, id, nil, "")
}

func checkReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > 500 {
		return "", model.ErrReason
	}
	return reason, nil
}
