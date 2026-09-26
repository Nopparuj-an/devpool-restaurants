package booking

import (
	"errors"
	"testing"
	"time"
)

var bkk = mustLoad("Asia/Bangkok")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// at returns 2026-10-12 (a Monday) plus dayOffset days at hh:mm Bangkok time, in UTC.
func at(dayOffset int, hhmm string) time.Time {
	c, err := ParseClock(hhmm)
	if err != nil {
		panic(err)
	}
	return time.Date(2026, 10, 12+dayOffset, int(c)/60, int(c)%60, 0, 0, bkk).UTC()
}

func iv(from, to string) Interval { return Interval{at(0, from), at(0, to)} }

func res(pax int, from, to string) Reservation { return Reservation{iv(from, to), pax} }

func everyDay(open, close string) WeeklyHours {
	o, _ := ParseClock(open)
	c, _ := ParseClock(close)
	var h WeeklyHours
	for d := range h {
		h[d] = &Shift{o, c}
	}
	return h
}

func TestPeakLoad(t *testing.T) {
	tests := []struct {
		name   string
		rs     []Reservation
		window Interval
		want   int
	}{
		{"empty", nil, iv("12:00", "13:00"), 0},
		{
			// Brief example 1: 7 booked 12:00–13:00, peak in 12:30–13:00 is 7.
			"single overlapping", []Reservation{res(7, "12:00", "13:00")}, iv("12:30", "13:00"), 7,
		},
		{
			// Brief example 2: A and B don't overlap each other, so the peak is 7, not 14.
			"sequential do not stack",
			[]Reservation{res(7, "12:00", "12:30"), res(7, "12:30", "13:00")},
			iv("12:00", "13:00"), 7,
		},
		{
			"true overlap stacks",
			[]Reservation{res(4, "12:00", "13:00"), res(3, "12:30", "13:30")},
			iv("12:00", "14:00"), 7,
		},
		{
			"back-to-back with window edge is not load",
			[]Reservation{res(9, "11:00", "12:00"), res(9, "13:00", "14:00")},
			iv("12:00", "13:00"), 0,
		},
		{
			"load outside window ignored",
			[]Reservation{res(2, "11:00", "12:30"), res(5, "12:15", "12:45"), res(6, "12:45", "14:00")},
			iv("12:00", "12:30"), 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PeakLoad(tt.rs, tt.window); got != tt.want {
				t.Errorf("PeakLoad() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCovers(t *testing.T) {
	regular := everyDay("11:00", "22:00")
	overnight := everyDay("18:00", "02:00")
	allDay := everyDay("00:00", "00:00")
	var mondayOnlyNight WeeklyHours
	mondayOnlyNight[time.Monday] = &Shift{18 * 60, 2 * 60}

	tests := []struct {
		name  string
		hours WeeklyHours
		iv    Interval
		want  bool
	}{
		{"inside regular", regular, iv("12:00", "14:00"), true},
		{"exactly the shift", regular, iv("11:00", "22:00"), true},
		{"starts before open", regular, iv("10:45", "12:00"), false},
		{"ends after close", regular, iv("21:00", "22:15"), false},
		{"overnight crosses midnight", overnight, Interval{at(0, "23:00"), at(1, "01:00")}, true},
		{"overnight after midnight uses previous day's shift", overnight, Interval{at(1, "00:30"), at(1, "01:30")}, true},
		{"overnight past close", overnight, Interval{at(1, "01:00"), at(1, "02:15")}, false},
		{"overnight gap", overnight, Interval{at(1, "03:00"), at(1, "04:00")}, false},
		{"previous day closed", mondayOnlyNight, Interval{at(0, "01:00"), at(0, "01:30")}, false}, // Sunday closed
		{"24h spans two shifts via merge", allDay, Interval{at(0, "23:00"), at(1, "01:00")}, true},
		{"closed weekday", WeeklyHours{}, iv("12:00", "13:00"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hours.Covers(bkk, tt.iv); got != tt.want {
				t.Errorf("Covers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func restaurant(seats int) Restaurant {
	return Restaurant{
		Seats:        seats,
		MaxDuration:  4 * time.Hour,
		CancelCutoff: 30 * time.Minute,
		Hours:        everyDay("10:00", "23:00"),
		Location:     bkk,
	}
}

func ruleOf(err error) string {
	if v, ok := errors.AsType[*Violation](err); ok {
		return v.Rule
	}
	if err != nil {
		return "unexpected: " + err.Error()
	}
	return ""
}

func TestCheckNew(t *testing.T) {
	now := at(-1, "09:00")
	r := restaurant(10)
	tests := []struct {
		name   string
		req    Reservation
		others []Reservation
		now    time.Time
		want   string // "" = allowed
	}{
		{"ok", res(4, "12:00", "13:00"), nil, now, ""},
		{"zero pax", res(0, "12:00", "13:00"), nil, now, "R-BOOK-1"},
		{"more than seats", res(11, "12:00", "13:00"), nil, now, "R-BOOK-1"},
		{"off grid", res(2, "12:10", "13:00"), nil, now, "R-BOOK-2"},
		{"end before start", res(2, "13:00", "12:00"), nil, now, "R-BOOK-2"},
		{"too long", res(2, "12:00", "16:15"), nil, now, "R-BOOK-2"},
		{"in the past", res(2, "12:00", "13:00"), nil, at(0, "12:00"), "R-BOOK-3"},
		{"beyond horizon", res(2, "12:00", "13:00"), nil, at(-31, "09:00"), "R-BOOK-8"},
		{"outside hours", res(2, "09:00", "10:30"), nil, now, "R-BOOK-4"},
		{
			// Brief: 7 booked, asking 5 over the same half hour → 12 > 10.
			"brief: over capacity", res(5, "12:00", "13:00"),
			[]Reservation{res(7, "12:00", "13:00")}, now, "R-BOOK-5",
		},
		{
			// Brief: A=7 then B=7 back to back; C=3 over the whole hour fits.
			"brief: A+B+C=17 fits", res(3, "12:00", "13:00"),
			[]Reservation{res(7, "12:00", "12:30"), res(7, "12:30", "13:00")}, now, "",
		},
		{
			"fills exactly", res(4, "12:00", "13:00"),
			[]Reservation{res(6, "11:00", "14:00")}, now, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ruleOf(CheckNew(r, tt.req, tt.others, tt.now)); got != tt.want {
				t.Errorf("CheckNew() rule = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckEdit(t *testing.T) {
	now := at(-1, "09:00")
	a := res(7, "12:00", "13:00")
	b := res(3, "12:00", "13:00") // together A+B fill 10 seats exactly
	tests := []struct {
		name    string
		r       Restaurant
		updated Reservation
		others  []Reservation
		now     time.Time
		want    string
	}{
		// Brief examples: A's own old seats must not be counted twice.
		{"brief: A 7→8 fails", restaurant(10), res(8, "12:00", "13:00"), []Reservation{b}, now, "R-BOOK-5"},
		{"brief: A 7→5 ok", restaurant(10), res(5, "12:00", "13:00"), []Reservation{b}, now, ""},
		{"move to a full slot", restaurant(10), res(7, "13:00", "14:00"), []Reservation{res(4, "13:00", "14:00")}, now, "R-BOOK-5"},
		{"past cutoff", restaurant(10), res(5, "12:00", "13:00"), []Reservation{b}, at(0, "11:45"), "R-EDIT-1"},
		{"at cutoff is still ok", restaurant(10), res(5, "12:00", "13:00"), []Reservation{b}, at(0, "11:30"), ""},
		{
			// Owner lowered seats to 8 after A and B were booked: 7+3 > 8.
			// A shrinking to 6 still leaves 9 > 8, but it can only lower load (R-EDIT-3).
			"shrink allowed when over capacity", restaurant(8), res(6, "12:15", "13:00"), []Reservation{b}, now, "",
		},
		{
			"growing time is not a shrink", restaurant(8), res(6, "12:00", "13:15"), []Reservation{b}, now, "R-BOOK-5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ruleOf(CheckEdit(tt.r, a, tt.updated, tt.others, tt.now)); got != tt.want {
				t.Errorf("CheckEdit() rule = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckCancel(t *testing.T) {
	r := restaurant(10)
	booking := res(2, "12:00", "13:00")
	// Brief: 12:00 booking with the default 30-minute cutoff can be cancelled until 11:30.
	if err := CheckCancel(r, booking, at(0, "11:30")); err != nil {
		t.Errorf("cancel at 11:30: %v", err)
	}
	if got := ruleOf(CheckCancel(r, booking, at(0, "11:31"))); got != "R-CANCEL-1" {
		t.Errorf("cancel at 11:31 rule = %q, want R-CANCEL-1", got)
	}
}

func TestLimitedThreshold(t *testing.T) {
	for seats, want := range map[int]int{1: 2, 10: 2, 12: 2, 15: 3, 30: 6} {
		if got := LimitedThreshold(seats); got != want {
			t.Errorf("LimitedThreshold(%d) = %d, want %d", seats, got, want)
		}
	}
}
