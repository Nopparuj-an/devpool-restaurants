// Package web holds the Gin helpers every feature handler uses: one error
// format, JSON decoding, and path parameters.
//
// Error body: {"error": {"code": "R-BOOK-5", "message": "..."}}
// (docs/architecture.md#api-conventions).
package web

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"restaurants/internal/booking"
	"restaurants/internal/platform/apperr"
)

var statusOf = map[apperr.Kind]int{
	apperr.Invalid:      http.StatusUnprocessableEntity,
	apperr.BadRequest:   http.StatusBadRequest,
	apperr.Unauthorized: http.StatusUnauthorized,
	apperr.Forbidden:    http.StatusForbidden,
	apperr.NotFound:     http.StatusNotFound,
	apperr.Conflict:     http.StatusConflict,
	apperr.TooLarge:     http.StatusRequestEntityTooLarge,
	apperr.Unavailable:  http.StatusServiceUnavailable,
}

// Handle adapts a handler that returns an error: errors are written in the
// standard format, so handlers only write the success response.
func Handle(fn func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := fn(c); err != nil {
			Error(c, err)
		}
	}
}

// Error writes err as JSON and aborts. Unknown errors become a logged 500
// whose details never reach the client.
func Error(c *gin.Context, err error) {
	if e, ok := errors.AsType[*apperr.Error](err); ok {
		abort(c, statusOf[e.Kind], e.Code, e.Message)
		return
	}
	if v, ok := errors.AsType[*booking.Violation](err); ok {
		status := http.StatusUnprocessableEntity
		if v.Kind == booking.Conflict {
			status = http.StatusConflict
		}
		abort(c, status, v.Rule, v.Msg)
		return
	}
	slog.Error("internal error", "method", c.Request.Method, "path", c.Request.URL.Path, "err", err)
	abort(c, http.StatusInternalServerError, "internal", "internal server error")
}

func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

// Decode reads a JSON body (max 1 MB) into v, rejecting unknown fields.
func Decode(c *gin.Context, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return apperr.New(apperr.BadRequest, "bad_request", "invalid JSON body: "+err.Error())
	}
	return nil
}

// PathID parses a positive integer path parameter; anything else is a 404.
func PathID(c *gin.Context, name string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.ErrNotFound
	}
	return id, nil
}
