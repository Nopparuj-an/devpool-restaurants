package model

import (
	"errors"

	"restaurants/internal/platform/apperr"
)

var (
	ErrEmailTaken            = apperr.New(apperr.Conflict, "email_taken", "an account with this email already exists")
	ErrInvalidCredentials    = apperr.New(apperr.Unauthorized, "invalid_credentials", "wrong email or password")
	ErrSessionExpired        = apperr.New(apperr.Unauthorized, "session_expired", "your session has expired; log in again")
	ErrGoogleEmailUnverified = apperr.New(apperr.Forbidden, "google_email_unverified", "your Google account's email is not verified")
	ErrGoogleConflict        = apperr.New(apperr.Conflict, "google_conflict", "this email is linked to a different Google account")
	ErrGoogleDisabled        = apperr.New(apperr.NotFound, "google_disabled", "Google login is not configured")
	ErrBanned                = apperr.New(apperr.Forbidden, "account_banned", "this account has been suspended")
	ErrAdminOnly             = apperr.New(apperr.Forbidden, "admin_only", "only admins can do this")
	ErrImpersonateSelf       = apperr.New(apperr.Conflict, "R-ADMIN-7", "you're already logged in as yourself")
	ErrImpersonateAdmin      = apperr.New(apperr.Conflict, "R-ADMIN-7", "admins can't be impersonated")
	ErrImpersonateBanned     = apperr.New(apperr.Conflict, "R-ADMIN-7", "banned users can't be impersonated; unban them first")
	ErrNotImpersonating      = apperr.New(apperr.Conflict, "not_impersonating", "this session isn't impersonating anyone")
	ErrWhileImpersonating    = apperr.New(apperr.Forbidden, "R-ADMIN-7", "you can't change the password while impersonating")

	// ErrNoSession means the cookie is missing, unknown or expired. It is not
	// shown to clients; the request simply continues logged out.
	ErrNoSession = errors.New("no valid session")
	// ErrNotFound is returned by the repository when a lookup finds nothing.
	ErrNotFound = errors.New("not found")
)
