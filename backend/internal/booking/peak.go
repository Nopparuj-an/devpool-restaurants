package booking

import (
	"cmp"
	"slices"
	"time"
)

// PeakLoad returns the highest total pax held by reservations at any single
// instant inside window (R-BOOK-5).
//
// Summing every overlapping reservation is wrong: two reservations that overlap
// the window but not each other never occupy seats at the same time. Instead we
// sweep through start/end events in time order and track the running total.
func PeakLoad(reservations []Reservation, window Interval) int {
	type event struct {
		at    time.Time
		delta int
	}
	events := make([]event, 0, 2*len(reservations))
	for _, r := range reservations {
		if !r.Overlaps(window) {
			continue
		}
		// Clip to the window so load outside it can't count.
		start, end := r.Start, r.End
		if start.Before(window.Start) {
			start = window.Start
		}
		if end.After(window.End) {
			end = window.End
		}
		events = append(events, event{start, r.Pax}, event{end, -r.Pax})
	}

	slices.SortFunc(events, func(a, b event) int {
		if c := a.at.Compare(b.at); c != 0 {
			return c
		}
		// Same instant: apply ends (negative) before starts. Intervals are
		// half-open, so a reservation ending at 12:30 frees its seats for one
		// starting at 12:30.
		return cmp.Compare(a.delta, b.delta)
	})

	load, peak := 0, 0
	for _, e := range events {
		load += e.delta
		peak = max(peak, load)
	}
	return peak
}
