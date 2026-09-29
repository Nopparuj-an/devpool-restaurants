// Package service holds the review rules: one review per account and
// restaurant, and no reviewing your own place. The restaurant's rating totals
// are kept by database triggers on every review change (ADR-0015).
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
	// LockRestaurant locks the restaurant row (serializing reviews of it) and
	// returns its owner; apperr.ErrNotFound if missing or hidden.
	LockRestaurant(ctx context.Context, restaurantID int64) (int64, error)
	// Restaurant returns the owner and whether customers can't see it (R-ADMIN-3, -4).
	Restaurant(ctx context.Context, restaurantID int64) (owner int64, hidden bool, err error)
	Rating(ctx context.Context, restaurantID, accountID int64) (int, error) // apperr.ErrNotFound if none
	Insert(ctx context.Context, restaurantID, accountID int64, rating int, body string) error
	Update(ctx context.Context, restaurantID, accountID int64, rating int, body string) error
	Delete(ctx context.Context, restaurantID, accountID int64) error // apperr.ErrNotFound if none
	List(ctx context.Context, restaurantID int64, now time.Time, q model.ListQuery) (model.Page, error)
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
	// List returns one page of reviews, optionally one star rating only,
	// newest or oldest first, with the per-star counts (R-REVIEW-8); viewer
	// is the logged-in account or 0. Hidden restaurants' reviews are only
	// shown to the owner and admins.
	List(ctx context.Context, viewer int64, admin bool, restaurantID int64, q model.ListQuery) (model.Page, error)
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
		_, err = s.repo.Rating(ctx, restaurantID, me)
		switch {
		case errors.Is(err, apperr.ErrNotFound):
			created = true
			return s.repo.Insert(ctx, restaurantID, me, in.Rating, body)
		case err != nil:
			return err
		}
		return s.repo.Update(ctx, restaurantID, me, in.Rating, body)
	})
	return created, err
}

// Delete removes my review (R-REVIEW-7); the trigger takes it out of the totals.
func (s *service) Delete(ctx context.Context, me, restaurantID int64) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.LockRestaurant(ctx, restaurantID); err != nil {
			return err
		}
		return s.repo.Delete(ctx, restaurantID, me)
	})
}

func (s *service) List(ctx context.Context, viewer int64, admin bool, restaurantID int64, q model.ListQuery) (model.Page, error) {
	switch q.Sort {
	case "":
		q.Sort = "newest"
	case "newest", "oldest":
	default:
		return model.Page{}, model.ErrSort
	}
	if q.Rating < 0 || q.Rating > 5 {
		return model.Page{}, model.ErrFilter
	}
	owner, hidden, err := s.repo.Restaurant(ctx, restaurantID)
	if err != nil {
		return model.Page{}, err
	}
	if hidden && viewer != owner && !admin {
		return model.Page{}, apperr.ErrNotFound
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}
	q.Offset = max(q.Offset, 0)
	page, err := s.repo.List(ctx, restaurantID, s.now(), q)
	for i := range page.Reviews {
		hideEmail(&page.Reviews[i], viewer)
	}
	return page, err
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
