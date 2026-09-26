// Package httpx has the JSON response and error helpers shared by all handlers.
//
// Error body: {"error": {"code": "R-BOOK-5", "message": "..."}}.
// Codes reuse rule IDs from docs/domain.md, see docs/architecture.md#api-conventions.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"restaurants/internal/booking"
)

// Error is an API error with an explicit status and code.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func NewError(status int, code, message string) *Error {
	return &Error{status, code, message}
}

// Invalid is a 422 for input that fails validation without breaking a named rule.
func Invalid(format string, args ...any) *Error {
	return NewError(http.StatusUnprocessableEntity, "invalid_input", fmt.Sprintf(format, args...))
}

var (
	ErrUnauthorized = NewError(http.StatusUnauthorized, "unauthorized", "login required")
	ErrForbidden    = NewError(http.StatusForbidden, "forbidden", "not allowed")
	ErrNotFound     = NewError(http.StatusNotFound, "not_found", "not found")
)

// Decode reads a JSON body (max 1 MB) into v, rejecting unknown fields.
func Decode(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return NewError(http.StatusBadRequest, "bad_request", "invalid JSON body: "+err.Error())
	}
	return nil
}

// JSON writes v with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write json", "err", err)
	}
}

// WriteError maps err to a status and writes the error body.
// Unknown errors become 500 and are logged, never shown to the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := errors.AsType[*Error](err); ok {
		writeErr(w, e.Status, e.Code, e.Message)
		return
	}
	if v, ok := errors.AsType[*booking.Violation](err); ok {
		status := http.StatusUnprocessableEntity
		if v.Kind == booking.Conflict {
			status = http.StatusConflict
		}
		writeErr(w, status, v.Rule, v.Msg)
		return
	}
	slog.Error("internal error", "method", r.Method, "path", r.URL.Path, "err", err)
	writeErr(w, http.StatusInternalServerError, "internal", "internal server error")
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

// Handler is an http.HandlerFunc that returns an error, which is written with WriteError.
type Handler func(w http.ResponseWriter, r *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		WriteError(w, r, err)
	}
}
