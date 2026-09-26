package auth

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"

	"restaurants/internal/auth/model"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/web"
)

const cookieName = "session"

type ctxKey struct{}

// Session loads the account behind the session cookie, if any, for every
// request. A stale or unknown cookie is cleared and the request continues
// logged out.
func (h *Handler) Session() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil {
			c.Next()
			return
		}
		a, err := h.svc.AccountForSession(c.Request.Context(), token)
		switch {
		case errors.Is(err, model.ErrNoSession):
			h.clearCookie(c)
		case err != nil:
			slog.Error("load session", "err", err)
		default:
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxKey{}, a))
		}
		c.Next()
	}
}

// RequireLogin stops the request with 401 unless a session is loaded.
func RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := CurrentAccount(c); !ok {
			web.Error(c, apperr.ErrUnauthorized)
			return
		}
		c.Next()
	}
}

// CurrentAccount returns the logged-in account, if any.
func CurrentAccount(c *gin.Context) (model.Account, bool) {
	a, ok := c.Request.Context().Value(ctxKey{}).(model.Account)
	return a, ok
}

// MustAccount is CurrentAccount for routes behind RequireLogin.
func MustAccount(c *gin.Context) model.Account {
	a, _ := CurrentAccount(c)
	return a
}

// ViewerID is the logged-in account ID, or 0 for visitors.
func ViewerID(c *gin.Context) int64 {
	a, _ := CurrentAccount(c)
	return a.ID
}
