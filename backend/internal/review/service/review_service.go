// Package service holds the review rules: one review per account and
// restaurant, no reviewing your own place, and keeping the restaurant's
// rating totals in step in the same transaction (ADR-0004).
package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"restaurants/internal/platform/apperr"
	"restaurants/internal/review/model"
)

// Repository is the outbound port for review data.
type Repository interface {
	// LockRestaurant locks the restaurant row (serializing its rating
	// updates) and returns its owner; apperr.ErrNotFound if missing.
	LockRestaurant(ctx context.Context, restaurantID int64) (int64, error)
	RestaurantExists(ctx context.Context, restaurantID int64) (bool, error)
	Rating(ctx context.Context, restaurantID, accountID int64) (int, error) // apperr.ErrNotFound if none
	Insert(ctx context.Context, restaurantID, accountID int64, rating int, body string) error
	Update(ctx context.Context, restaurantID, accountID int64, rating int, body string) error
	Delete(ctx context.Context, restaurantID, accountID int64) (int, error) // returns the old rating
	AdjustAggregate(ctx context.Context, restaurantID int64, sumDelta, countDelta int) error
	List(ctx context.Context, restaurantID int64, now time.Time, limit, offset int) ([]model.Review, error)
	Mine(ctx context.Context, restaurantID, accountID int64, now time.Time) (model.Review, error) // ErrNotFound if none
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service is the inbound port used by the HTTP handlers.
type Service interface {
	// Upsert creates or replaces my review (R-REVIEW-2) and reports whether it was created.
	Upsert(ctx context.Context, me, restaurantID int64, in model.Input) (bool, error)
	Delete(ctx context.Context, me, restaurantID int64) error
	// List returns reviews, most recently updated first; viewer is the logged-in account or 0.
	List(ctx context.Context, viewer, restaurantID int64, limit, offset int) ([]model.Review, error)
	Mine(ctx context.Context, me, restaurantID int64) (model.Review, error)
}

type service struct {
	repo Repository
	tx   TxRunner
	now  func() time.Time
}

func New(repo Repository, tx TxRunner) Service {
	return &service{repo: repo, tx: tx, now: time.Now}
}

func (s *service) Upsert(ctx context.Context, me, restaurantID int64, in model.Input) (bool, error) {
	body := strings.TrimSpace(in.Body)
	if in.Rating < 1 || in.Rating > 5 {
		return false, model.ErrRating
	}
	if n := utf8.RuneCountInString(body); n < 1 || n > 2000 {
		return false, model.ErrBody
	}
	var created bool
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		owner, err := s.repo.LockRestaurant(ctx, restaurantID)
		if err != nil {
			return err
		}
		if owner == me {
			return model.ErrOwnReview
		}
		old, err := s.repo.Rating(ctx, restaurantID, me)
		switch {
		case errors.Is(err, apperr.ErrNotFound):
			created = true
			if err := s.repo.Insert(ctx, restaurantID, me, in.Rating, body); err != nil {
				return err
			}
			return s.repo.AdjustAggregate(ctx, restaurantID, in.Rating, 1)
		case err != nil:
			return err
		}
		if err := s.repo.Update(ctx, restaurantID, me, in.Rating, body); err != nil {
			return err
		}
		return s.repo.AdjustAggregate(ctx, restaurantID, in.Rating-old, 0)
	})
	return created, err
}

// Delete removes my review (R-REVIEW-7) and takes it out of the totals.
func (s *service) Delete(ctx context.Context, me, restaurantID int64) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.LockRestaurant(ctx, restaurantID); err != nil {
			return err
		}
		old, err := s.repo.Delete(ctx, restaurantID, me)
		if err != nil {
			return err
		}
		return s.repo.AdjustAggregate(ctx, restaurantID, -old, -1)
	})
}

func (s *service) List(ctx context.Context, viewer, restaurantID int64, limit, offset int) ([]model.Review, error) {
	exists, err := s.repo.RestaurantExists(ctx, restaurantID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperr.ErrNotFound
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, err := s.repo.List(ctx, restaurantID, s.now(), limit, max(offset, 0))
	for i := range list {
		hideEmail(&list[i], viewer)
	}
	return list, err
}

func (s *service) Mine(ctx context.Context, me, restaurantID int64) (model.Review, error) {
	rv, err := s.repo.Mine(ctx, restaurantID, me, s.now())
	hideEmail(&rv, me)
	return rv, err
}

// hideEmail keeps reviewer emails for the restaurant's owner only (R-PRIV-1, R-PRIV-2).
func hideEmail(rv *model.Review, viewer int64) {
	if viewer == 0 || viewer != rv.OwnerID {
		rv.Author.Email = ""
	}
}
