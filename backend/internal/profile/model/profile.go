// Package model holds the public profile types (R-PROFILE-*).
package model

import "time"

// Profile is what anyone can see about an account (R-PRIV-1): never the email.
type Profile struct {
	ID              int64     `json:"id"`
	DisplayName     string    `json:"display_name"`
	CreatedAt       time.Time `json:"created_at"`
	RestaurantCount int       `json:"restaurant_count"`
	ReviewCount     int       `json:"review_count"`
	// Banned is only ever true for admins: everyone else gets a 404 (R-PROFILE-2).
	Banned bool `json:"banned,omitempty"`
}

type RestaurantRef struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	CoverURL string `json:"cover_url"`
	CoverKey string `json:"-"`
}

// Review is one of the profile's reviews, with the restaurant it's about.
type Review struct {
	ID         int64         `json:"id"`
	Rating     int           `json:"rating"`
	Body       string        `json:"body"`
	Verified   bool          `json:"verified"` // R-REVIEW-4
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Restaurant RestaurantRef `json:"restaurant"`
	// Hidden: customers can't see it (banned restaurant or owner). Only the
	// profile's owner and admins get hidden rows at all.
	Hidden bool `json:"hidden,omitempty"`
}
