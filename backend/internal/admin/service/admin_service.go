// Package service holds the admin rules: who can be banned or deleted, and
// what that does (R-ADMIN-*). Bans are reversible: they set banned_at. Deletes
// are not: rows cascade, and photos go from storage after commit.
package service

import (
	"context"
	"log/slog"
	"slices"
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

	// LockUsers locks the listed accounts that exist and returns them.
	LockUsers(ctx context.Context, ids []int64) ([]model.User, error)
	// DeleteUsers deletes accounts and everything that cascades from them,
	// and returns how many went and the photo keys of their restaurants.
	DeleteUsers(ctx context.Context, ids []int64) (n int, imageKeys []string, err error)
	// DeleteRestaurants is DeleteUsers for restaurants (R-REST-5).
	DeleteRestaurants(ctx context.Context, ids []int64) (n int, imageKeys []string, err error)
}

// Images is the outbound port for photo storage.
type Images interface {
	Delete(ctx context.Context, keys ...string) error
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
	// DeleteUsers and DeleteRestaurants delete up to MaxDelete rows at once
	// (R-ADMIN-8, R-ADMIN-6) and return how many existed. Missing IDs are
	// skipped, so a retry is harmless.
	DeleteUsers(ctx context.Context, admin int64, ids []int64) (int, error)
	DeleteRestaurants(ctx context.Context, ids []int64) (int, error)
}

type service struct {
	repo   Repository
	tx     TxRunner
	images Images
	now    func() time.Time
}

func New(repo Repository, tx TxRunner, images Images) Service {
	return &service{repo: repo, tx: tx, images: images, now: time.Now}
}

func normalize(q model.ListQuery) (model.ListQuery, error) {
	switch q.Status {
	case "", "all":
		q.Status = ""
	case "active", "banned":
	default:
		return q, model.ErrBadStatus
	}
	if q.Limit <= 0 || q.Limit > model.MaxDelete {
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

// DeleteUsers is all or nothing: if any listed account is the acting admin
// or another admin, nothing is deleted (R-ADMIN-8).
func (s *service) DeleteUsers(ctx context.Context, admin int64, ids []int64) (int, error) {
	ids, err := checkIDs(ids)
	if err != nil {
		return 0, err
	}
	var (
		n    int
		keys []string
	)
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		users, err := s.repo.LockUsers(ctx, ids)
		if err != nil {
			return err
		}
		for _, u := range users {
			switch {
			case u.ID == admin:
				return model.ErrDeleteSelf
			case u.IsAdmin:
				return model.ErrDeleteAdmin
			}
		}
		n, keys, err = s.repo.DeleteUsers(ctx, ids)
		return err
	})
	if err != nil {
		return 0, err
	}
	s.deleteImages(keys)
	return n, nil
}

func (s *service) DeleteRestaurants(ctx context.Context, ids []int64) (int, error) {
	ids, err := checkIDs(ids)
	if err != nil {
		return 0, err
	}
	var (
		n    int
		keys []string
	)
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		n, keys, err = s.repo.DeleteRestaurants(ctx, ids)
		return err
	})
	if err != nil {
		return 0, err
	}
	s.deleteImages(keys)
	return n, nil
}

// checkIDs dedupes and bounds a bulk delete.
func checkIDs(ids []int64) ([]int64, error) {
	ids = slices.Compact(slices.Sorted(slices.Values(ids)))
	if len(ids) == 0 || len(ids) > model.MaxDelete || ids[0] <= 0 {
		return nil, model.ErrDeleteIDs
	}
	return ids, nil
}

// deleteImages runs after commit. A failure only leaks storage, so it is
// logged, not returned (like the restaurant service).
func (s *service) deleteImages(keys []string) {
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for chunk := range slices.Chunk(keys, 1000) { // S3 deletes at most 1000 keys per call
		if err := s.images.Delete(ctx, chunk...); err != nil {
			slog.Warn("delete image objects", "count", len(chunk), "err", err)
		}
	}
}

func checkReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > 500 {
		return "", model.ErrReason
	}
	return reason, nil
}
