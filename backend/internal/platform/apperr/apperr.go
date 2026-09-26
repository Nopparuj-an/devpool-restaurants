// Package apperr defines errors that services return without knowing about
// HTTP. The web layer maps each Kind to a status code (platform/web).
package apperr

import "fmt"

type Kind int

const (
	Invalid      Kind = iota // input breaks a rule → 422
	BadRequest               // malformed request → 400
	Unauthorized             // not logged in → 401
	Forbidden                // logged in, not allowed → 403
	NotFound                 // → 404
	Conflict                 // valid, but the current state forbids it → 409
	TooLarge                 // → 413
	Unavailable              // a dependency is down → 503
)

// Error is a failure the client should see. Code is stable (often a rule ID
// from docs/domain.md, e.g. "R-REST-1"); Message is shown to users.
type Error struct {
	Kind    Kind
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// InvalidInput is a 422 for input that fails validation without breaking a named rule.
func InvalidInput(format string, args ...any) *Error {
	return New(Invalid, "invalid_input", fmt.Sprintf(format, args...))
}

var (
	ErrUnauthorized = New(Unauthorized, "unauthorized", "login required")
	ErrForbidden    = New(Forbidden, "forbidden", "not allowed")
	ErrNotFound     = New(NotFound, "not_found", "not found")
)
