// Package reservation implements booking, editing and cancelling
// reservations (R-BOOK-*, R-EDIT-*, R-CANCEL-*), plus availability and the
// owner's reservation table.
//
// Every write follows the same shape (ADR-0003):
//
//	BEGIN
//	  lock the restaurant row          -- serializes writes per restaurant
//	  load overlapping active reservations
//	  booking.Check*(...)              -- pure rules, unit-tested
//	  INSERT / UPDATE
//	COMMIT
//
// Two customers racing for the last seats therefore run one after the other;
// the second sees the first's booking and gets 409 R-BOOK-5.
package reservation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"restaurants/internal/booking"
	"restaurants/internal/httpx"
	"restaurants/internal/restaurant"
)

// Input is what a customer submits to book or edit.
type Input struct {
	Pax      int       `json:"pax"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

func (in Input) toBooking() booking.Reservation {
	return booking.Reservation{Interval: booking.Interval{Start: in.StartsAt.UTC(), End: in.EndsAt.UTC()}, Pax: in.Pax}
}

type RestaurantRef struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	CoverURL string `json:"cover_url"`
}

// Customer is shown to the restaurant's owner only (R-PRIV-2).
type Customer struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type Reservation struct {
	ID       int64     `json:"id"`
	Pax      int       `json:"pax"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	Status   string    `json:"status"` // "active" | "cancelled"
	// State is derived at read time: "upcoming", "in_progress", "completed" or "cancelled".
	State string `json:"state"`
	// ModifiableUntil is the last instant the customer may edit or cancel (R-EDIT-1, R-CANCEL-1).
	ModifiableUntil time.Time     `json:"modifiable_until"`
	CanModify       bool          `json:"can_modify"`
	Restaurant      RestaurantRef `json:"restaurant"`
	Customer        *Customer     `json:"customer,omitempty"`
}

type Service struct {
	db           *pgxpool.Pool
	imageBaseURL string
	now          func() time.Time
}

func NewService(db *pgxpool.Pool, imageBaseURL string) *Service {
	return &Service{db: db, imageBaseURL: strings.TrimSuffix(imageBaseURL, "/"), now: time.Now}
}

// Create books a reservation for me.
func (s *Service) Create(ctx context.Context, me, restaurantID int64, in Input) (int64, error) {
	req := in.toBooking()
	var id int64
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rules, _, err := restaurant.LockForBooking(ctx, tx, restaurantID)
		if err != nil {
			return err
		}
		others, err := overlapping(ctx, tx, restaurantID, req.Interval, 0)
		if err != nil {
			return err
		}
		if err := booking.CheckNew(rules, req, others, s.now()); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			INSERT INTO reservations (restaurant_id, account_id, pax, starts_at, ends_at)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			restaurantID, me, req.Pax, req.Start, req.End).Scan(&id)
	})
	return id, err
}

// Update changes my reservation's pax and/or time (R-EDIT-*).
func (s *Service) Update(ctx context.Context, me, id int64, in Input) error {
	updated := in.toBooking()
	return s.modify(ctx, me, id, func(tx pgx.Tx, rules booking.Restaurant, restaurantID int64, old booking.Reservation) error {
		// Exclude this reservation so its old seats aren't counted twice (R-EDIT-2).
		others, err := overlapping(ctx, tx, restaurantID, updated.Interval, id)
		if err != nil {
			return err
		}
		if err := booking.CheckEdit(rules, old, updated, others, s.now()); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE reservations SET pax = $2, starts_at = $3, ends_at = $4, updated_at = now()
			WHERE id = $1`, id, updated.Pax, updated.Start, updated.End)
		return err
	})
}

// Cancel cancels my reservation (R-CANCEL-*). The row is kept.
func (s *Service) Cancel(ctx context.Context, me, id int64) error {
	return s.modify(ctx, me, id, func(tx pgx.Tx, rules booking.Restaurant, _ int64, old booking.Reservation) error {
		if err := booking.CheckCancel(rules, old, s.now()); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE reservations SET status = 'cancelled', updated_at = now() WHERE id = $1`, id)
		return err
	})
}

// modify runs fn with the restaurant and reservation locked, after checking
// that me owns the reservation and it is still active.
func (s *Service) modify(ctx context.Context, me, id int64,
	fn func(tx pgx.Tx, rules booking.Restaurant, restaurantID int64, old booking.Reservation) error,
) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		// restaurant_id never changes, so it is safe to read before locking.
		// Lock order is always restaurant → reservation, like Create, to avoid deadlocks.
		var restaurantID int64
		err := tx.QueryRow(ctx, `SELECT restaurant_id FROM reservations WHERE id = $1`, id).Scan(&restaurantID)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		rules, _, err := restaurant.LockForBooking(ctx, tx, restaurantID)
		if err != nil {
			return err
		}

		var (
			owner  int64
			status string
			old    booking.Reservation
		)
		err = tx.QueryRow(ctx, `
			SELECT account_id, status, pax, starts_at, ends_at FROM reservations WHERE id = $1 FOR UPDATE`, id).
			Scan(&owner, &status, &old.Pax, &old.Start, &old.End)
		if err != nil {
			return err
		}
		if owner != me {
			return httpx.NewError(403, "forbidden", "this is not your reservation")
		}
		if status != "active" {
			return httpx.NewError(409, "not_active", "this reservation was cancelled")
		}
		return fn(tx, rules, restaurantID, old)
	})
}

// overlapping returns active reservations overlapping iv, except excludeID.
func overlapping(ctx context.Context, q restaurant.Querier, restaurantID int64, iv booking.Interval, excludeID int64) ([]booking.Reservation, error) {
	rows, _ := q.Query(ctx, `
		SELECT pax, starts_at, ends_at FROM reservations
		WHERE restaurant_id = $1 AND status = 'active'
			AND starts_at < $3 AND ends_at > $2 AND id <> $4`,
		restaurantID, iv.Start, iv.End, excludeID)
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (booking.Reservation, error) {
		var r booking.Reservation
		err := row.Scan(&r.Pax, &r.Start, &r.End)
		return r, err
	})
}

// listSelect returns reservations with their restaurant and customer.
const listSelect = `
	SELECT v.id, v.pax, v.starts_at, v.ends_at, v.status, r.cancel_cutoff_minutes,
		r.id, r.name, coalesce(ci.object_key, ''), a.id, a.display_name, a.email
	FROM reservations v
	JOIN restaurants r ON r.id = v.restaurant_id
	JOIN accounts a ON a.id = v.account_id
	LEFT JOIN restaurant_images ci ON ci.restaurant_id = r.id AND ci.is_cover`

func (s *Service) collect(rows pgx.Rows, withCustomer bool) ([]Reservation, error) {
	now := s.now()
	out := []Reservation{}
	defer rows.Close()
	for rows.Next() {
		var (
			r      Reservation
			cutoff int
			key    string
			c      Customer
		)
		err := rows.Scan(&r.ID, &r.Pax, &r.StartsAt, &r.EndsAt, &r.Status, &cutoff,
			&r.Restaurant.ID, &r.Restaurant.Name, &key, &c.ID, &c.DisplayName, &c.Email)
		if err != nil {
			return nil, err
		}
		if key != "" {
			r.Restaurant.CoverURL = s.imageBaseURL + "/" + key
		}
		if withCustomer {
			r.Customer = &c
		}
		r.ModifiableUntil = r.StartsAt.Add(-time.Duration(cutoff) * time.Minute)
		switch {
		case r.Status == "cancelled":
			r.State = "cancelled"
		case r.StartsAt.After(now):
			r.State = "upcoming"
			r.CanModify = !now.After(r.ModifiableUntil)
		case r.EndsAt.After(now):
			r.State = "in_progress"
		default:
			r.State = "completed"
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Get returns one of my reservations.
func (s *Service) Get(ctx context.Context, me, id int64) (Reservation, error) {
	rows, err := s.db.Query(ctx, listSelect+` WHERE v.id = $1 AND v.account_id = $2`, id, me)
	if err != nil {
		return Reservation{}, err
	}
	list, err := s.collect(rows, false)
	if err != nil {
		return Reservation{}, err
	}
	if len(list) == 0 {
		return Reservation{}, httpx.ErrNotFound
	}
	return list[0], nil
}

// ListMine returns all my reservations, soonest upcoming first, then past.
func (s *Service) ListMine(ctx context.Context, me int64) ([]Reservation, error) {
	rows, err := s.db.Query(ctx, listSelect+`
		WHERE v.account_id = $1
		ORDER BY
			(v.ends_at <= $2 OR v.status = 'cancelled'),                          -- current first
			CASE WHEN v.ends_at > $2 AND v.status = 'active' THEN v.starts_at END, -- soonest current first
			v.starts_at DESC                                                      -- most recent past first`, me, s.now())
	if err != nil {
		return nil, err
	}
	return s.collect(rows, false)
}

// ListForOwner returns a restaurant's reservations overlapping [from, to),
// with customer details, for its owner only (R-PRIV-2).
func (s *Service) ListForOwner(ctx context.Context, me, restaurantID int64, from, to time.Time) ([]Reservation, error) {
	_, owner, err := restaurant.LoadRules(ctx, s.db, restaurantID)
	if err != nil {
		return nil, err
	}
	if owner != me {
		return nil, httpx.NewError(403, "forbidden", "only the owner can see this restaurant's reservations")
	}
	rows, err := s.db.Query(ctx, listSelect+`
		WHERE v.restaurant_id = $1 AND v.starts_at < $3 AND v.ends_at > $2
		ORDER BY v.starts_at, v.id`, restaurantID, from, to)
	if err != nil {
		return nil, err
	}
	return s.collect(rows, true)
}

type Slot struct {
	Start     time.Time `json:"start"`
	SeatsLeft int       `json:"seats_left"`
	Limited   bool      `json:"limited"`
}

type Availability struct {
	Seats            int    `json:"seats"`
	LimitedThreshold int    `json:"limited_threshold"`
	Limited          bool   `json:"limited"` // any slot limited (R-SEATS-1)
	Slots            []Slot `json:"slots"`   // 15-minute slots inside opening hours
}

// maxAvailabilitySpan bounds the availability query (7 days = 672 slots).
const maxAvailabilitySpan = 7 * 24 * time.Hour

// Availability returns seats left per 15-minute open slot in [from, to).
func (s *Service) Availability(ctx context.Context, restaurantID int64, from, to time.Time) (Availability, error) {
	if !to.After(from) || to.Sub(from) > maxAvailabilitySpan {
		return Availability{}, httpx.Invalid("range must be positive and at most 7 days")
	}
	rules, _, err := restaurant.LoadRules(ctx, s.db, restaurantID)
	if err != nil {
		return Availability{}, err
	}
	window := booking.Interval{Start: from.UTC(), End: to.UTC()}
	existing, err := overlapping(ctx, s.db, restaurantID, window, 0)
	if err != nil {
		return Availability{}, err
	}

	a := Availability{Seats: rules.Seats, LimitedThreshold: booking.LimitedThreshold(rules.Seats), Slots: []Slot{}}
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
			a.Slots = append(a.Slots, Slot{Start: t, SeatsLeft: max(0, left), Limited: limited})
		}
	}
	return a, nil
}
