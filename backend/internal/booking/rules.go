package booking

import (
	"fmt"
	"time"
)

// Kind says how a violation maps to HTTP: Invalid → 422, Conflict → 409.
type Kind int

const (
	Invalid  Kind = iota // the request itself is malformed or out of bounds
	Conflict             // valid request, but current state forbids it (full, too late)
)

// Violation is a broken business rule. Rule is the ID from docs/domain.md.
type Violation struct {
	Rule string
	Kind Kind
	Msg  string
}

func (v *Violation) Error() string { return v.Rule + ": " + v.Msg }

func invalid(rule, format string, args ...any) *Violation {
	return &Violation{rule, Invalid, fmt.Sprintf(format, args...)}
}

func conflict(rule, format string, args ...any) *Violation {
	return &Violation{rule, Conflict, fmt.Sprintf(format, args...)}
}

// CheckNew validates a new reservation (R-BOOK-1…5, 8).
// others must hold the restaurant's active reservations overlapping req.
func CheckNew(r Restaurant, req Reservation, others []Reservation, now time.Time) error {
	if err := checkShape(r, req, now); err != nil {
		return err
	}
	return checkCapacity(r, req, others)
}

// CheckEdit validates changing old into updated (R-EDIT-1…3).
// The caller has already checked that the reservation is active and belongs
// to the caller. others must NOT include the reservation being edited (R-EDIT-2).
func CheckEdit(r Restaurant, old, updated Reservation, others []Reservation, now time.Time) error {
	if err := checkCutoff(r, old, now, "R-EDIT-1", "edited"); err != nil {
		return err
	}
	if err := checkShape(r, updated, now); err != nil {
		return err
	}
	// R-EDIT-3: an edit that can only lower load skips the capacity check, so
	// customers can still shrink bookings after the owner lowered seats.
	if updated.Pax <= old.Pax && old.Contains(updated.Interval) {
		return nil
	}
	return checkCapacity(r, updated, others)
}

// CheckCancel validates cancelling res (R-CANCEL-1). The caller has already
// checked that the reservation is active and belongs to the caller.
func CheckCancel(r Restaurant, res Reservation, now time.Time) error {
	return checkCutoff(r, res, now, "R-CANCEL-1", "cancelled")
}

func checkShape(r Restaurant, res Reservation, now time.Time) error {
	switch {
	case res.Pax < 1:
		return invalid("R-BOOK-1", "party size must be at least 1")
	case res.Pax > r.Seats:
		return invalid("R-BOOK-1", "party of %d exceeds the restaurant's %d seats", res.Pax, r.Seats)
	case !OnGrid(res.Start) || !OnGrid(res.End):
		return invalid("R-BOOK-2", "times must be on the 15-minute grid")
	case !res.End.After(res.Start):
		return invalid("R-BOOK-2", "end must be after start")
	case res.End.Sub(res.Start) > r.MaxDuration:
		return invalid("R-BOOK-2", "reservation is longer than the restaurant's maximum of %s", r.MaxDuration)
	case !res.Start.After(now):
		return invalid("R-BOOK-3", "reservation must start in the future")
	case res.Start.After(now.Add(Horizon)):
		return invalid("R-BOOK-8", "reservations can be made at most 30 days ahead")
	case !r.Hours.Covers(r.Location, res.Interval):
		return invalid("R-BOOK-4", "reservation must be within opening hours")
	}
	return nil
}

func checkCapacity(r Restaurant, res Reservation, others []Reservation) error {
	if peak := PeakLoad(others, res.Interval); peak+res.Pax > r.Seats {
		return conflict("R-BOOK-5", "only %d seats left in that time range", max(0, r.Seats-peak))
	}
	return nil
}

func checkCutoff(r Restaurant, res Reservation, now time.Time, rule, verb string) error {
	if deadline := res.Start.Add(-r.CancelCutoff); now.After(deadline) {
		return conflict(rule, "reservation can only be %s until %s before it starts", verb, r.CancelCutoff)
	}
	return nil
}
