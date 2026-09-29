// Package model holds the admin feature's types and errors.
package model

import (
	"time"

	"restaurants/internal/platform/apperr"
)

// User is one row of the admin user list.
type User struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"display_name"`
	IsAdmin      bool       `json:"is_admin"`
	BannedAt     *time.Time `json:"banned_at"`
	BanReason    string     `json:"ban_reason"`
	CreatedAt    time.Time  `json:"created_at"`
	Restaurants  int        `json:"restaurants"`  // owned
	Reviews      int        `json:"reviews"`      // written
	Reservations int        `json:"reservations"` // made
}

// UserDetail is a user plus the restaurants they own.
type UserDetail struct {
	User
	OwnedRestaurants []Restaurant `json:"owned_restaurants"`
}

type Owner struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Banned      bool   `json:"banned"`
}

// Restaurant is one row of the admin restaurant list.
type Restaurant struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Cuisine     string     `json:"cuisine"`
	Location    string     `json:"location"`
	Rating      *float64   `json:"rating"`
	ReviewCount int        `json:"review_count"`
	BannedAt    *time.Time `json:"banned_at"`
	BanReason   string     `json:"ban_reason"`
	CreatedAt   time.Time  `json:"created_at"`
	Owner       Owner      `json:"owner"`
}

// ListQuery filters both admin lists.
type ListQuery struct {
	Q      string // users: email or name; restaurants: name, cuisine or owner email
	Status string // "" = all, "active", "banned"
	Limit  int
	Offset int
}

// UserInput is what an admin can change on a profile (R-ADMIN-6). The email
// is the login identity and stays as it is.
type UserInput struct {
	DisplayName string `json:"display_name"`
}

// DeleteInput names the users or restaurants to delete in one request.
type DeleteInput struct {
	IDs []int64 `json:"ids"`
}

// MaxDelete caps one bulk delete; the admin pages send bigger selections in chunks.
const MaxDelete = 500

type BanInput struct {
	Reason string `json:"reason"` // optional, shown to the owner and other admins
}

var (
	ErrBanSelf     = apperr.New(apperr.Conflict, "R-ADMIN-5", "you can't ban yourself")
	ErrBanAdmin    = apperr.New(apperr.Conflict, "R-ADMIN-5", "admins can't be banned; remove their admin rights in the database first")
	ErrDeleteSelf  = apperr.New(apperr.Conflict, "R-ADMIN-8", "you can't delete your own account")
	ErrDeleteAdmin = apperr.New(apperr.Conflict, "R-ADMIN-8", "admins can't be deleted; remove their admin rights in the database first")
	ErrDeleteIDs   = apperr.InvalidInput("ids must list 1 to 500 ids")
	ErrBadStatus   = apperr.InvalidInput("status must be active or banned")
	ErrReason      = apperr.InvalidInput("reason must be at most 500 characters")
	ErrName        = apperr.InvalidInput("display name must be 1 to 80 characters")
)
