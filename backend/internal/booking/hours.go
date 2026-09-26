package booking

import (
	"fmt"
	"time"
)

// Clock is a wall-clock time of day in minutes since local midnight (0–1439).
type Clock int

// ParseClock parses "HH:MM".
func ParseClock(s string) (Clock, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid clock %q, want HH:MM", s)
	}
	return Clock(h*60 + m), nil
}

func (c Clock) String() string { return fmt.Sprintf("%02d:%02d", int(c)/60, int(c)%60) }

// OnGrid reports whether c is on the 15-minute grid (R-HOURS-2).
func (c Clock) OnGrid() bool { return int(c)%int(Slot/time.Minute) == 0 }

// Shift is one opening window that starts on a given weekday (R-HOURS-3):
//   - Close > Open: same day, e.g. 11:00–22:00
//   - Close < Open: ends the next day, e.g. 18:00–02:00
//   - Close == Open: open 24h from Open
type Shift struct {
	Open, Close Clock
}

// WeeklyHours holds at most one shift per weekday, indexed by time.Weekday.
// A nil entry means closed that day (R-HOURS-1).
type WeeklyHours [7]*Shift

// OpenPeriods expands the weekly hours into concrete UTC intervals in loc,
// covering every shift that could touch [from, to), and merges shifts that
// touch or overlap into one continuous period (R-HOURS-4).
func (h WeeklyHours) OpenPeriods(loc *time.Location, from, to time.Time) []Interval {
	// Start one day early: yesterday's shift may run past midnight into `from`.
	fy, fm, fd := from.In(loc).Date()
	day := time.Date(fy, fm, fd-1, 0, 0, 0, 0, loc)
	ty, tm, td := to.In(loc).Date()
	last := time.Date(ty, tm, td, 0, 0, 0, 0, loc)

	var periods []Interval
	for ; !day.After(last); day = day.AddDate(0, 0, 1) {
		s := h[day.Weekday()]
		if s == nil {
			continue
		}
		y, m, d := day.Date()
		start := time.Date(y, m, d, int(s.Open)/60, int(s.Open)%60, 0, 0, loc)
		endDay := d
		if s.Close <= s.Open { // overnight or 24h
			endDay = d + 1
		}
		end := time.Date(y, m, endDay, int(s.Close)/60, int(s.Close)%60, 0, 0, loc)
		p := Interval{start.UTC(), end.UTC()}

		// Shifts come out in start order (one per day), so merging only
		// needs to look at the previous period.
		if n := len(periods); n > 0 && !p.Start.After(periods[n-1].End) {
			if p.End.After(periods[n-1].End) {
				periods[n-1].End = p.End
			}
			continue
		}
		periods = append(periods, p)
	}
	return periods
}

// Covers reports whether iv lies entirely inside one continuous open period (R-HOURS-4).
func (h WeeklyHours) Covers(loc *time.Location, iv Interval) bool {
	for _, p := range h.OpenPeriods(loc, iv.Start, iv.End) {
		if p.Contains(iv) {
			return true
		}
	}
	return false
}
