package model

import (
	"errors"

	"restaurants/internal/platform/apperr"
)

var (
	ErrEmailTaken            = apperr.New(apperr.Conflict, "email_taken", "an account with this email already exists")
	ErrInvalidCredentials    = apperr.New(apperr.Unauthorized, "invalid_credentials", "wrong email or password")
	ErrGoogleEmailUnverified = apperr.New(apperr.Forbidden, "google_email_unverified", "your Google account's email is not verified")
	ErrGoogleConflict        = apperr.New(apperr.Conflict, "google_conflict", "this email is linked to a different Google account")
	ErrGoogleDisabled        = apperr.New(apperr.NotFound, "google_disabled", "Google login is not configured")

	// ErrNoSession means the cookie is missing, unknown or expired. It is not
	// shown to clients; the request simply continues logged out.
	ErrNoSession = errors.New("no valid session")
	// ErrNotFound is returned by the repository when a lookup finds nothing.
	ErrNotFound = errors.New("not found")
)
