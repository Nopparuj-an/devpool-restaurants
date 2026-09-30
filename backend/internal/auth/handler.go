// Package auth is the accounts, login and session feature (ADR-0002).
//
//	routes.go            URL → handler
//	handler.go           password signup/login/logout, me, change password, impersonation
//	google_handler.go    Sign in with Google (OIDC)
//	middleware.go        session cookie → current account, RequireLogin
//	module.go            wiring
//	model/               Account, errors
//	service/             rules (Service) + storage port (Repository)
//	repository/          SQL
package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"restaurants/internal/auth/model"
	"restaurants/internal/auth/service"
	"restaurants/internal/platform/config"
	"restaurants/internal/platform/web"
)

type Handler struct {
	svc          service.Service
	secureCookie bool
	google       *google // nil when Google login is not configured
}

func NewHandler(svc service.Service, secureCookie bool, googleCfg config.Google) *Handler {
	h := &Handler{svc: svc, secureCookie: secureCookie}
	if googleCfg.ClientID != "" {
		h.google = &google{cfg: googleCfg}
	}
	return h
}

func (h *Handler) Signup(c *gin.Context) error {
	var in struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := web.Decode(c, &in); err != nil {
		return err
	}
	a, err := h.svc.Signup(c.Request.Context(), in.Email, in.Password, in.DisplayName)
	if err != nil {
		return err
	}
	if err := h.startSession(c, a.ID); err != nil {
		return err
	}
	c.JSON(http.StatusCreated, a)
	return nil
}

func (h *Handler) Login(c *gin.Context) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := web.Decode(c, &in); err != nil {
		return err
	}
	a, err := h.svc.Login(c.Request.Context(), in.Email, in.Password)
	if err != nil {
		return err
	}
	if err := h.startSession(c, a.ID); err != nil {
		return err
	}
	c.JSON(http.StatusOK, a)
	return nil
}

func (h *Handler) Logout(c *gin.Context) error {
	if token, err := c.Cookie(cookieName); err == nil {
		if err := h.svc.DeleteSession(c.Request.Context(), token); err != nil {
			return err
		}
	}
	h.clearCookie(c)
	c.Status(http.StatusNoContent)
	return nil
}

func (h *Handler) Me(c *gin.Context) error {
	c.JSON(http.StatusOK, MustAccount(c))
	return nil
}

func (h *Handler) UpdateProfile(c *gin.Context) error {
	var in struct {
		DisplayName string `json:"display_name"`
	}
	if err := web.Decode(c, &in); err != nil {
		return err
	}
	a, err := h.svc.UpdateProfile(c.Request.Context(), MustAccount(c).ID, in.DisplayName)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, a)
	return nil
}

func (h *Handler) SetPassword(c *gin.Context) error {
	var in struct {
		NewPassword string `json:"new_password"`
	}
	if err := web.Decode(c, &in); err != nil {
		return err
	}
	me := MustAccount(c)
	if me.Impersonator != nil {
		return model.ErrWhileImpersonating
	}
	if err := h.svc.SetPassword(c.Request.Context(), me.ID, in.NewPassword); err != nil {
		return err
	}
	c.Status(http.StatusNoContent)
	return nil
}

// Impersonate swaps the admin's session for one as the user (R-ADMIN-7).
func (h *Handler) Impersonate(c *gin.Context) error {
	id, err := web.PathID(c, "id")
	if err != nil {
		return err
	}
	admin := MustAccount(c)
	token, _ := c.Cookie(cookieName)
	sess, err := h.svc.Impersonate(c.Request.Context(), admin, token, id)
	if err != nil {
		return err
	}
	slog.Info("impersonation started", "admin", admin.ID, "user", id)
	h.setCookie(c, sess)
	c.Status(http.StatusNoContent)
	return nil
}

// StopImpersonating logs the admin back in as themselves.
func (h *Handler) StopImpersonating(c *gin.Context) error {
	token, _ := c.Cookie(cookieName)
	sess, admin, err := h.svc.StopImpersonating(c.Request.Context(), token)
	if err != nil {
		return err
	}
	slog.Info("impersonation stopped", "admin", admin.ID, "user", MustAccount(c).ID)
	h.setCookie(c, sess)
	c.JSON(http.StatusOK, admin)
	return nil
}

func (h *Handler) Providers(c *gin.Context) error {
	c.JSON(http.StatusOK, gin.H{"password": true, "google": h.google != nil})
	return nil
}

func (h *Handler) startSession(c *gin.Context, accountID int64) error {
	sess, err := h.svc.CreateSession(c.Request.Context(), accountID)
	if err != nil {
		return err
	}
	h.setCookie(c, sess)
	return nil
}

func (h *Handler) setCookie(c *gin.Context, sess model.Session) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.Expires,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode, // blocks cross-site POSTs from carrying the cookie (CSRF)
	})
}

func (h *Handler) clearCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}
