// Package model holds the restaurant feature's types and errors.
package model

import "restaurants/internal/booking"

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

// Hours is one weekday's shift as the API sends it (R-HOURS-*). Weekday 0 = Sunday.
type Hours struct {
	Weekday int    `json:"weekday"`
	Open    string `json:"open"`  // "HH:MM"
	Close   string `json:"close"` // "HH:MM"; < open = next day, == open = 24h
}

// Shift is one weekday's validated opening window.
type Shift struct {
	Weekday     int
	Open, Close booking.Clock
}

// Valid is Input after validation, with defaults applied.
type Valid struct {
	Name, Description, Cuisine, Location string
	Seats                                int
	CutoffMinutes, MaxMinutes            int
	Timezone                             string
	Shifts                               []Shift
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
	CoverKey    string   `json:"-"` // object key; the service turns it into CoverURL
	Owner       Owner    `json:"owner"`
	// Banned is set when an admin hid the restaurant (R-ADMIN-4). Only the
	// owner and admins ever see hidden restaurants.
	Banned      bool `json:"banned,omitempty"`
	OwnerBanned bool `json:"owner_banned,omitempty"`
}

// Hidden reports whether customers can't see the restaurant (R-ADMIN-3, -4).
func (s Summary) Hidden() bool { return s.Banned || s.OwnerBanned }

type Image struct {
	ID      int64  `json:"id"`
	URL     string `json:"url"`
	Key     string `json:"-"`
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
	BanReason             string  `json:"ban_reason,omitempty"` // owner and admins only
	// UpcomingReservations is only set for the owner, e.g. to warn before delete (R-REST-5).
	UpcomingReservations *int `json:"upcoming_reservations,omitempty"`
}

type ListQuery struct {
	Sort    string // "top_rated" (default) | "most_reviewed" | "newest"
	Q       string // name or cuisine contains, case-insensitive
	Cuisine string // exact, case-insensitive
	OwnerID int64  // 0 = any
	// IncludeHidden also lists banned restaurants (the owner's own list).
	IncludeHidden bool
	Limit         int
	Offset        int
}

// Upload is a validated image file from a multipart request.
type Upload struct {
	Data        []byte
	ContentType string
	Ext         string
}

// ImageStats is what adding images needs to know about the existing ones.
type ImageStats struct {
	Count, NextPosition int
	HasCover            bool
}

const MaxImages = 10
