// Package booking holds the pure reservation rules from docs/domain.md.
//
// It does no I/O: callers load the restaurant and its overlapping reservations
// (inside the locking transaction, ADR-0003), call a Check* function, then persist.
// Rule IDs in comments and Violation.Rule match docs/domain.md.
package booking

import "time"

const (
	// Slot is the booking grid step (ADR-0001).
	Slot = 15 * time.Minute
	// Horizon is how far ahead a reservation may start (R-BOOK-8).
	Horizon = 30 * 24 * time.Hour
)

// Interval is a half-open time range [Start, End) (R-TIME-3).
type Interval struct {
	Start, End time.Time
}

// Overlaps reports whether i and o share at least one instant.
// Back-to-back intervals (i.End == o.Start) do not overlap.
func (i Interval) Overlaps(o Interval) bool {
	return i.Start.Before(o.End) && o.Start.Before(i.End)
}

// Contains reports whether o lies entirely inside i.
func (i Interval) Contains(o Interval) bool {
	return !o.Start.Before(i.Start) && !o.End.After(i.End)
}

// Reservation is the part of a reservation the rules care about.
type Reservation struct {
	Interval
	Pax int
}

// Restaurant is the part of a restaurant the rules care about.
type Restaurant struct {
	Seats        int
	MaxDuration  time.Duration // R-REST-6
	CancelCutoff time.Duration // R-REST-3
	Hours        WeeklyHours
	Location     *time.Location // R-TIME-5
}

// OnGrid reports whether t falls on a 15-minute boundary. Every real UTC offset
// is a multiple of 15 minutes, so checking in UTC is the same as checking in
// any local time (ADR-0007).
func OnGrid(t time.Time) bool {
	return t.Nanosecond() == 0 && t.Unix()%int64(Slot/time.Second) == 0
}
