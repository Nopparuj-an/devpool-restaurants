package service

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"restaurants/internal/booking"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/restaurant/model"
)

// validate checks an Input against R-REST-* and R-HOURS-* and applies defaults.
func validate(in model.Input, creating bool) (model.Valid, error) {
	v := model.Valid{
		Name:          strings.TrimSpace(in.Name),
		Description:   strings.TrimSpace(in.Description),
		Cuisine:       strings.TrimSpace(in.Cuisine),
		Location:      strings.TrimSpace(in.Location),
		Seats:         in.Seats,
		CutoffMinutes: 30,
		MaxMinutes:    240,
		Timezone:      in.Timezone,
	}
	for _, f := range []struct {
		name, value string
		max         int
	}{
		{"name", v.Name, 100},
		{"description", v.Description, 2000},
		{"cuisine", v.Cuisine, 50},
		{"location", v.Location, 200},
	} {
		if n := utf8.RuneCountInString(f.value); n < 1 || n > f.max {
			return v, apperr.InvalidInput("%s must be 1 to %d characters", f.name, f.max)
		}
	}
	if in.Seats < 1 || in.Seats > 1000 {
		return v, apperr.InvalidInput("seats must be 1 to 1000")
	}
	if in.CancelCutoffMinutes != nil {
		v.CutoffMinutes = *in.CancelCutoffMinutes
	}
	if v.CutoffMinutes < 30 || v.CutoffMinutes%15 != 0 || v.CutoffMinutes > 7*24*60 {
		return v, model.ErrCutoff
	}
	if in.MaxReservationMinutes != nil {
		v.MaxMinutes = *in.MaxReservationMinutes
	}
	if v.MaxMinutes < 15 || v.MaxMinutes%15 != 0 || v.MaxMinutes > 24*60 {
		return v, model.ErrMaxDuration
	}
	if creating {
		// LoadLocation accepts "" and "Local", which mean nothing to other viewers.
		if _, err := time.LoadLocation(in.Timezone); err != nil || in.Timezone == "" || in.Timezone == "Local" {
			return v, apperr.InvalidInput("timezone must be a valid IANA name like Asia/Bangkok")
		}
	}

	if len(in.Hours) == 0 {
		return v, model.ErrHoursRequired
	}
	seen := map[int]bool{}
	for _, h := range in.Hours {
		if h.Weekday < 0 || h.Weekday > 6 || seen[h.Weekday] {
			return v, model.ErrShiftPerDay
		}
		seen[h.Weekday] = true
		open, err1 := booking.ParseClock(h.Open)
		closing, err2 := booking.ParseClock(h.Close)
		if err1 != nil || err2 != nil || !open.OnGrid() || !closing.OnGrid() {
			return v, model.ErrShiftGrid
		}
		v.Shifts = append(v.Shifts, model.Shift{Weekday: h.Weekday, Open: open, Close: closing})
	}
	slices.SortFunc(v.Shifts, func(a, b model.Shift) int { return a.Weekday - b.Weekday })
	return v, nil
}
