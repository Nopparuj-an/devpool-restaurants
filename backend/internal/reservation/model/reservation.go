// Package model holds the reservation feature's types and errors.
package model

import (
	"time"

	"restaurants/internal/booking"
	"restaurants/internal/platform/apperr"
)

// Input is what a customer submits to book or change a table.
type Input struct {
	Pax      int       `json:"pax"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

func (in Input) Booking() booking.Reservation {
	return booking.Reservation{Interval: booking.Interval{Start: in.StartsAt.UTC(), End: in.EndsAt.UTC()}, Pax: in.Pax}
}

type RestaurantRef struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	CoverURL string `json:"cover_url"`
	CoverKey string `json:"-"`
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
	// ModifiableUntil is the last instant the customer may change or cancel (R-EDIT-1, R-CANCEL-1).
	ModifiableUntil time.Time     `json:"modifiable_until"`
	CanModify       bool          `json:"can_modify"`
	Restaurant      RestaurantRef `json:"restaurant"`
	Customer        *Customer     `json:"customer,omitempty"`
	CutoffMinutes   int           `json:"-"`
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

// Locked is a reservation read under a row lock before a change.
type Locked struct {
	AccountID int64
	Status    string
	booking.Reservation
}

const (
	StatusActive    = "active"
	StatusCancelled = "cancelled"
	// MaxAvailabilitySpan bounds availability queries (7 days = 672 slots).
	MaxAvailabilitySpan = 7 * 24 * time.Hour
)

var (
	ErrNotYours     = apperr.New(apperr.Forbidden, "forbidden", "this is not your reservation")
	ErrNotActive    = apperr.New(apperr.Conflict, "not_active", "this reservation was cancelled")
	ErrOwnerOnly    = apperr.New(apperr.Forbidden, "forbidden", "only the owner can see this restaurant's reservations")
	ErrBadRange     = apperr.InvalidInput("range must be positive and at most 7 days")
	ErrMissingTimes = apperr.InvalidInput("starts_at and ends_at are required (RFC 3339)")
)
