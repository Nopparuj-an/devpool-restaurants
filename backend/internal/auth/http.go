package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"restaurants/internal/httpx"
)

const cookieName = "session"

type ctxKey struct{}

// AccountFrom returns the logged-in account, if any.
func AccountFrom(ctx context.Context) (Account, bool) {
	a, ok := ctx.Value(ctxKey{}).(Account)
	return a, ok
}

// Require wraps a handler that needs a logged-in account (401 otherwise).
func Require(next func(w http.ResponseWriter, r *http.Request, me Account) error) httpx.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		me, ok := AccountFrom(r.Context())
		if !ok {
			return httpx.ErrUnauthorized
		}
		return next(w, r, me)
	}
}

type Handler struct {
	svc          *Service
	secureCookie bool
	google       *google // nil when Google login is not configured
}

func NewHandler(svc *Service, secureCookie bool, googleCfg GoogleConfig) *Handler {
	h := &Handler{svc: svc, secureCookie: secureCookie}
	if googleCfg.ClientID != "" {
		h.google = &google{cfg: googleCfg}
	}
	return h
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/auth/signup", httpx.Handler(h.signup))
	mux.Handle("POST /api/auth/login", httpx.Handler(h.login))
	mux.Handle("POST /api/auth/logout", httpx.Handler(h.logout))
	mux.Handle("GET /api/me", Require(h.me))
	mux.Handle("PUT /api/me/password", Require(h.setPassword))
	mux.Handle("GET /api/auth/providers", httpx.Handler(h.providers))
	mux.Handle("GET /api/auth/google/start", httpx.Handler(h.googleStart))
	mux.Handle("GET /api/auth/google/callback", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The browser lands here from Google, so show errors on the login page, not as JSON.
		if err := h.googleCallback(w, r); err != nil {
			if e, ok := errors.AsType[*httpx.Error](err); ok {
				http.Redirect(w, r, "/login?error="+e.Code, http.StatusFound)
				return
			}
			httpx.WriteError(w, r, err)
		}
	}))
}

// Middleware attaches the session's account to the request context.
// A stale or unknown cookie is cleared; the request continues anonymously.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		a, err := h.svc.AccountForSession(r.Context(), c.Value)
		switch {
		case errors.Is(err, ErrNoSession):
			h.clearCookie(w)
		case err != nil:
			slog.Error("load session", "err", err)
		default:
			r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, a))
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := httpx.Decode(w, r, &in); err != nil {
		return err
	}
	a, err := h.svc.Signup(r.Context(), in.Email, in.Password, in.DisplayName)
	if err != nil {
		return err
	}
	if err := h.startSession(w, r, a.ID); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, a)
	return nil
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(w, r, &in); err != nil {
		return err
	}
	a, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		return err
	}
	if err := h.startSession(w, r, a.ID); err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, a)
	return nil
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie(cookieName); err == nil {
		if err := h.svc.DeleteSession(r.Context(), c.Value); err != nil {
			return err
		}
	}
	h.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request, me Account) error {
	httpx.JSON(w, http.StatusOK, me)
	return nil
}

func (h *Handler) setPassword(w http.ResponseWriter, r *http.Request, me Account) error {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := httpx.Decode(w, r, &in); err != nil {
		return err
	}
	if err := h.svc.SetPassword(r.Context(), me.ID, in.CurrentPassword, in.NewPassword); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, accountID int64) error {
	token, expires, err := h.svc.CreateSession(r.Context(), accountID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode, // blocks cross-site POSTs from carrying the cookie (CSRF)
	})
	return nil
}

func (h *Handler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}
