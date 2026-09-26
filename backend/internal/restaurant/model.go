// Package restaurant implements restaurant CRUD, opening hours and images
// (R-REST-*, R-HOURS-*).
package restaurant

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"restaurants/internal/booking"
	"restaurants/internal/httpx"
)

// Input is the editable part of a restaurant (create and update).
type Input struct {
	Name                  string  `json:"name"`
	Description           string  `json:"description"`
	Cuisine               string  `json:"cuisine"`
	Location              string  `json:"location"`
	Seats                 int     `json:"seats"`
	CancelCutoffMinutes   *int    `json:"cancel_cutoff_minutes"`   // default 30 (R-REST-3)
	MaxReservationMinutes *int    `json:"max_reservation_minutes"` // default 240 (R-REST-6)
	Hours                 []Hours `json:"hours"`
	// Timezone is the owner's browser IANA zone. Only read on create (R-TIME-5).
	Timezone string `json:"timezone,omitempty"`
}

// Hours is one weekday's shift (R-HOURS-*). Weekday 0 = Sunday.
type Hours struct {
	Weekday int    `json:"weekday"`
	Open    string `json:"open"`  // "HH:MM"
	Close   string `json:"close"` // "HH:MM"; < open = next day, == open = 24h
}

type Owner struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
}

// Summary is a restaurant in a list.
type Summary struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Cuisine     string   `json:"cuisine"`
	Location    string   `json:"location"`
	Seats       int      `json:"seats"`
	Rating      *float64 `json:"rating"` // average, one decimal; null with no reviews (R-REVIEW-5)
	ReviewCount int      `json:"review_count"`
	CoverURL    string   `json:"cover_url"`
	Owner       Owner    `json:"owner"`
}

type Image struct {
	ID      int64  `json:"id"`
	URL     string `json:"url"`
	IsCover bool   `json:"is_cover"`
}

// Detail is a single restaurant.
type Detail struct {
	Summary
	Description           string  `json:"description"`
	CancelCutoffMinutes   int     `json:"cancel_cutoff_minutes"`
	MaxReservationMinutes int     `json:"max_reservation_minutes"`
	Timezone              string  `json:"timezone"`
	Hours                 []Hours `json:"hours"`
	Images                []Image `json:"images"`
	IsOwner               bool    `json:"is_owner"`
	// UpcomingReservations is only set for the owner, e.g. to warn before delete (R-REST-5).
	UpcomingReservations *int `json:"upcoming_reservations,omitempty"`
}

// validated is Input after validation, with defaults applied.
type validated struct {
	Input
	cutoff, maxDuration int
	shifts              []shift
}

type shift struct {
	weekday     int
	open, close booking.Clock
}

func validate(in Input, creating bool) (validated, error) {
	v := validated{Input: in, cutoff: 30, maxDuration: 240}
	v.Name = strings.TrimSpace(in.Name)
	v.Description = strings.TrimSpace(in.Description)
	v.Cuisine = strings.TrimSpace(in.Cuisine)
	v.Location = strings.TrimSpace(in.Location)

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
			return v, httpx.Invalid("%s must be 1–%d characters", f.name, f.max)
		}
	}
	if in.Seats < 1 || in.Seats > 1000 {
		return v, httpx.Invalid("seats must be 1–1000")
	}
	if in.CancelCutoffMinutes != nil {
		v.cutoff = *in.CancelCutoffMinutes
	}
	if v.cutoff < 30 || v.cutoff%15 != 0 || v.cutoff > 7*24*60 {
		return v, httpx.NewError(422, "R-REST-3", "cancel cutoff must be at least 30 minutes, in 15-minute steps, at most 7 days")
	}
	if in.MaxReservationMinutes != nil {
		v.maxDuration = *in.MaxReservationMinutes
	}
	if v.maxDuration < 15 || v.maxDuration%15 != 0 || v.maxDuration > 24*60 {
		return v, httpx.NewError(422, "R-REST-6", "max reservation length must be 15 minutes to 24 hours, in 15-minute steps")
	}
	if creating {
		if _, err := loadLocation(in.Timezone); err != nil {
			return v, httpx.Invalid("timezone must be a valid IANA name like Asia/Bangkok")
		}
	}

	if len(in.Hours) == 0 {
		return v, httpx.NewError(422, "R-REST-1", "opening hours are required")
	}
	seen := map[int]bool{}
	for _, h := range in.Hours {
		if h.Weekday < 0 || h.Weekday > 6 || seen[h.Weekday] {
			return v, httpx.NewError(422, "R-HOURS-1", "hours need at most one shift per weekday 0–6")
		}
		seen[h.Weekday] = true
		open, err1 := booking.ParseClock(h.Open)
		closing, err2 := booking.ParseClock(h.Close)
		if err1 != nil || err2 != nil || !open.OnGrid() || !closing.OnGrid() {
			return v, httpx.NewError(422, "R-HOURS-2", "opening times must be HH:MM on the 15-minute grid")
		}
		v.shifts = append(v.shifts, shift{h.Weekday, open, closing})
	}
	slices.SortFunc(v.shifts, func(a, b shift) int { return a.weekday - b.weekday })
	return v, nil
}

// loadLocation rejects "" and "Local", which LoadLocation would accept.
func loadLocation(name string) (*time.Location, error) {
	if name == "" || name == "Local" {
		return nil, httpx.Invalid("missing timezone")
	}
	return time.LoadLocation(name)
}
