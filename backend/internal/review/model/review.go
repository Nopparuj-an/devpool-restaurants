// Package model holds the review feature's types and errors.
package model

import (
	"time"

	"restaurants/internal/platform/apperr"
)

type Author struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	// Email is only shown to the restaurant's owner (R-PRIV-2).
	Email string `json:"email,omitempty"`
}

type Review struct {
	ID        int64     `json:"id"`
	Rating    int       `json:"rating"`
	Body      string    `json:"body"`
	Verified  bool      `json:"verified"` // author has a completed reservation here (R-REVIEW-4)
	Author    Author    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	OwnerID   int64     `json:"-"` // the restaurant's owner, for the privacy rule
}

// ListQuery pages a restaurant's reviews (R-REVIEW-8).
type ListQuery struct {
	Rating int    // 1 to 5 shows only that star rating; 0 shows all
	Sort   string // "newest" (default) or "oldest", by last update
	Limit  int
	Offset int
}

// Page is one page of reviews. Total counts the reviews that match Rating;
// RatingCounts has every star level ("1" to "5"), whatever the filter.
type Page struct {
	Reviews      []Review    `json:"reviews"`
	Total        int         `json:"total"`
	RatingCounts map[int]int `json:"rating_counts"`
}

type Input struct {
	Rating int    `json:"rating"`
	Body   string `json:"body"`
}

var (
	ErrRating    = apperr.New(apperr.Invalid, "R-REVIEW-1", "rating must be a whole number from 1 to 5")
	ErrBody      = apperr.New(apperr.Invalid, "R-REVIEW-1", "review text must be 1 to 2000 characters")
	ErrOwnReview = apperr.New(apperr.Forbidden, "R-REVIEW-3", "you can't review your own restaurant")
	ErrSort      = apperr.InvalidInput("sort must be newest or oldest")
	ErrFilter    = apperr.InvalidInput("rating filter must be 1 to 5")
)
