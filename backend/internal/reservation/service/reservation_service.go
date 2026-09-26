// Package service holds the reservation flow. Every write has the same shape
// (ADR-0003):
//
//	WithinTx:
//	  lock the restaurant row          -- serializes writes per restaurant
//	  load overlapping active reservations
//	  booking.Check*(...)              -- pure rules, unit-tested
//	  insert / update
//
// Two customers racing for the last seats therefore run one after the other;
// the second sees the first's booking and gets 409 R-BOOK-5.
package service

import (
	"context"
	"strings"
	"time"

	"restaurants/internal/booking"
	"restaurants/internal/reservation/model"
)

// Repository is the outbound port for reservation data.
type Repository interface {
	// RestaurantRules returns a restaurant's booking rules and owner. With
	// lock, the restaurant row stays locked until the transaction ends.
	// Hidden restaurants are not found unless includeHidden (R-ADMIN-3, -4).
	RestaurantRules(ctx context.Context, restaurantID int64, lock, includeHidden bool) (booking.Restaurant, int64, error)
	RestaurantOf(ctx context.Context, reservationID int64) (int64, error) // apperr.ErrNotFound if missing
	LockReservation(ctx context.Context, id int64) (model.Locked, error)
	// Overlapping returns active reservations overlapping iv, except excludeID.
	Overlapping(ctx context.Context, restaurantID int64, iv booking.Interval, excludeID int64) ([]booking.Reservation, error)
	Insert(ctx context.Context, restaurantID, accountID int64, r booking.Reservation) (int64, error)
	Update(ctx context.Context, id int64, r booking.Reservation) error
	Cancel(ctx context.Context, id int64) error

	ByID(ctx context.Context, accountID, id int64) (model.Reservation, error) // apperr.ErrNotFound if missing
	ByAccount(ctx context.Context, accountID int64, now time.Time, limit, offset int) ([]model.Reservation, int, error)
	ByRestaurant(ctx context.Context, restaurantID int64, from, to time.Time) ([]model.Reservation, error)
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service is the inbound port used by the HTTP handlers.
type Service interface {
	Create(ctx context.Context, me, restaurantID int64, in model.Input) (int64, error)
	Update(ctx context.Context, me, id int64, in model.Input) error
	Cancel(ctx context.Context, me, id int64) error
	Get(ctx context.Context, me, id int64) (model.Reservation, error)
	// ListMine returns one page (limit ≤ 100, default 50) and the total.
	ListMine(ctx context.Context, me int64, limit, offset int) ([]model.Reservation, int, error)
	// ListForOwner is the owner's booking table; admins may see it too (R-ADMIN-6).
	ListForOwner(ctx context.Context, me int64, admin bool, restaurantID int64, from, to time.Time) ([]model.Reservation, error)
	Availability(ctx context.Context, restaurantID int64, from, to time.Time) (model.Availability, error)
}

type service struct {
	repo         Repository
	tx           TxRunner
	imageBaseURL string
	now          func() time.Time
}

func New(repo Repository, tx TxRunner, imageBaseURL string) Service {
	return &service{repo: repo, tx: tx, imageBaseURL: strings.TrimSuffix(imageBaseURL, "/"), now: time.Now}
}

func (s *service) Create(ctx context.Context, me, restaurantID int64, in model.Input) (int64, error) {
	req := in.Booking()
	var id int64
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		rules, _, err := s.repo.RestaurantRules(ctx, restaurantID, true, false)
		if err != nil {
			return err
		}
		others, err := s.repo.Overlapping(ctx, restaurantID, req.Interval, 0)
		if err != nil {
			return err
		}
		if err := booking.CheckNew(rules, req, others, s.now()); err != nil {
			return err
		}
		id, err = s.repo.Insert(ctx, restaurantID, me, req)
		return err
	})
	return id, err
}

func (s *service) Update(ctx context.Context, me, id int64, in model.Input) error {
	updated := in.Booking()
	return s.modify(ctx, me, id, false, func(ctx context.Context, rules booking.Restaurant, restaurantID int64, old booking.Reservation) error {
		// Exclude this reservation so its old seats aren't counted twice (R-EDIT-2).
		others, err := s.repo.Overlapping(ctx, restaurantID, updated.Interval, id)
		if err != nil {
			return err
		}
		if err := booking.CheckEdit(rules, old, updated, others, s.now()); err != nil {
			return err
		}
		return s.repo.Update(ctx, id, updated)
	})
}

func (s *service) Cancel(ctx context.Context, me, id int64) error {
	// Bookings at a restaurant that was hidden later can still be cancelled.
	return s.modify(ctx, me, id, true, func(ctx context.Context, rules booking.Restaurant, _ int64, old booking.Reservation) error {
		if err := booking.CheckCancel(rules, old, s.now()); err != nil {
			return err
		}
		return s.repo.Cancel(ctx, id)
	})
}

// modify runs fn with the restaurant and reservation locked, after checking
// that me owns the reservation and it is still active. Lock order is always
// restaurant → reservation, like Create, so two writers can't deadlock.
func (s *service) modify(ctx context.Context, me, id int64, includeHidden bool,
	fn func(ctx context.Context, rules booking.Restaurant, restaurantID int64, old booking.Reservation) error,
) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// restaurant_id never changes, so it is safe to read before locking.
		restaurantID, err := s.repo.RestaurantOf(ctx, id)
		if err != nil {
			return err
		}
		rules, _, err := s.repo.RestaurantRules(ctx, restaurantID, true, includeHidden)
		if err != nil {
			return err
		}
		locked, err := s.repo.LockReservation(ctx, id)
		if err != nil {
			return err
		}
		if locked.AccountID != me {
			return model.ErrNotYours
		}
		if locked.Status != model.StatusActive {
			return model.ErrNotActive
		}
		return fn(ctx, rules, restaurantID, locked.Reservation)
	})
}

func (s *service) Get(ctx context.Context, me, id int64) (model.Reservation, error) {
	r, err := s.repo.ByID(ctx, me, id)
	if err != nil {
		return r, err
	}
	s.derive(&r, s.now(), false)
	return r, nil
}

func (s *service) ListMine(ctx context.Context, me int64, limit, offset int) ([]model.Reservation, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	now := s.now()
	list, total, err := s.repo.ByAccount(ctx, me, now, limit, max(offset, 0))
	for i := range list {
		s.derive(&list[i], now, false)
	}
	return list, total, err
}

func (s *service) ListForOwner(ctx context.Context, me int64, admin bool, restaurantID int64, from, to time.Time) ([]model.Reservation, error) {
	_, owner, err := s.repo.RestaurantRules(ctx, restaurantID, false, true)
	if err != nil {
		return nil, err
	}
	if owner != me && !admin {
		return nil, model.ErrOwnerOnly
	}
	now := s.now()
	list, err := s.repo.ByRestaurant(ctx, restaurantID, from, to)
	for i := range list {
		s.derive(&list[i], now, true)
	}
	return list, err
}

// derive fills the read-time fields: state, cutoff, cover URL, and hides the
// customer unless the viewer is the restaurant owner (R-PRIV-2).
func (s *service) derive(r *model.Reservation, now time.Time, forOwner bool) {
	if r.Restaurant.CoverKey != "" {
		r.Restaurant.CoverURL = s.imageBaseURL + "/" + r.Restaurant.CoverKey
	}
	if !forOwner {
		r.Customer = nil
	}
	r.ModifiableUntil = r.StartsAt.Add(-time.Duration(r.CutoffMinutes) * time.Minute)
	switch {
	case r.Status == model.StatusCancelled:
		r.State = "cancelled"
	case r.StartsAt.After(now):
		r.State = "upcoming"
		r.CanModify = !now.After(r.ModifiableUntil)
	case r.EndsAt.After(now):
		r.State = "in_progress"
	default:
		r.State = "completed"
	}
}

// Availability returns seats left per 15-minute open slot in [from, to).
func (s *service) Availability(ctx context.Context, restaurantID int64, from, to time.Time) (model.Availability, error) {
	if !to.After(from) || to.Sub(from) > model.MaxAvailabilitySpan {
		return model.Availability{}, model.ErrBadRange
	}
	rules, _, err := s.repo.RestaurantRules(ctx, restaurantID, false, false)
	if err != nil {
		return model.Availability{}, err
	}
	window := booking.Interval{Start: from.UTC(), End: to.UTC()}
	existing, err := s.repo.Overlapping(ctx, restaurantID, window, 0)
	if err != nil {
		return model.Availability{}, err
	}

	a := model.Availability{Seats: rules.Seats, LimitedThreshold: booking.LimitedThreshold(rules.Seats), Slots: []model.Slot{}}
	for _, p := range rules.Hours.OpenPeriods(rules.Location, window.Start, window.End) {
		start := p.Start.Truncate(booking.Slot)
		if start.Before(p.Start) {
			start = start.Add(booking.Slot)
		}
		for t := start; !t.Add(booking.Slot).After(p.End); t = t.Add(booking.Slot) {
			slot := booking.Interval{Start: t, End: t.Add(booking.Slot)}
			if !window.Contains(slot) {
				continue
			}
			left := rules.Seats - booking.PeakLoad(existing, slot)
			limited := left <= a.LimitedThreshold
			a.Limited = a.Limited || limited
			a.Slots = append(a.Slots, model.Slot{Start: t, SeatsLeft: max(0, left), Limited: limited})
		}
	}
	return a, nil
}
