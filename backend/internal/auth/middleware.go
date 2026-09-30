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
type staleKey struct{}

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
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), staleKey{}, true))
		case err != nil:
			slog.Error("load session", "err", err)
		default:
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxKey{}, a))
		}
		c.Next()
	}
}

// RequireLogin stops the request with 401 unless a session is loaded. The code
// says why: session_expired when the request carried a cookie that no longer
// works (expired, ended, or the account was banned), unauthorized when it
// carried none.
func RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := CurrentAccount(c); !ok {
			if stale, _ := c.Request.Context().Value(staleKey{}).(bool); stale {
				web.Error(c, model.ErrSessionExpired)
				return
			}
			web.Error(c, apperr.ErrUnauthorized)
			return
		}
		c.Next()
	}
}

// RequireAdmin stops the request with 403 unless the account is an admin
// (R-ADMIN-1). Use after RequireLogin.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !MustAccount(c).IsAdmin {
			web.Error(c, model.ErrAdminOnly)
			return
		}
		c.Next()
	}
}

// Viewer is who is looking: account ID (0 for visitors) and whether they are an admin.
func Viewer(c *gin.Context) (int64, bool) {
	a, _ := CurrentAccount(c)
	return a.ID, a.IsAdmin
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
