// Package model holds the auth feature's types and errors.
package model

import "time"

// Account is the logged-in user as returned by the API.
type Account struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	EmailVerified bool   `json:"email_verified"`
	HasPassword   bool   `json:"has_password"`
}

// GoogleIdentity is what we trust from a verified Google ID token.
type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

// Session is a new login: the cookie token and when it expires.
type Session struct {
	Token   string
	Expires time.Time
}

const (
	SessionTTL     = 30 * 24 * time.Hour
	MinPasswordLen = 8
	MaxPasswordLen = 72 // bcrypt ignores bytes past 72
)
