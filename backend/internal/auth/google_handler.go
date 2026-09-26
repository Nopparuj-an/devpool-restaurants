package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"restaurants/internal/auth/model"
	"restaurants/internal/platform/apperr"
	"restaurants/internal/platform/config"
	"restaurants/internal/platform/web"
)

// google runs the OIDC authorization-code flow with PKCE, state and nonce.
type google struct {
	cfg config.Google

	once     sync.Once
	provider *oidc.Provider
	initErr  error
}

const googleFlowCookie = "google_flow"

// init discovers Google's endpoints lazily, so the API starts offline.
func (g *google) init(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	g.once.Do(func() {
		g.provider, g.initErr = oidc.NewProvider(ctx, "https://accounts.google.com")
	})
	if g.initErr != nil {
		return nil, nil, fmt.Errorf("google discovery: %w", g.initErr)
	}
	return &oauth2.Config{
		ClientID:     g.cfg.ClientID,
		ClientSecret: g.cfg.ClientSecret,
		RedirectURL:  g.cfg.RedirectURL,
		Endpoint:     g.provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}, g.provider.Verifier(&oidc.Config{ClientID: g.cfg.ClientID}), nil
}

// GoogleStart redirects to Google. ?next=/path is where to land afterwards.
func (h *Handler) GoogleStart(c *gin.Context) error {
	if h.google == nil {
		return model.ErrGoogleDisabled
	}
	conf, _, err := h.google.init(c.Request.Context())
	if err != nil {
		return err
	}
	state, nonce, verifier := randomString(), randomString(), oauth2.GenerateVerifier()
	// The flow cookie binds the callback to this browser (CSRF on login).
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     googleFlowCookie,
		Value:    strings.Join([]string{state, nonce, verifier, safeNext(c.Query("next"))}, "|"),
		Path:     "/api/auth/google",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode, // sent on Google's top-level redirect back
	})
	c.Redirect(http.StatusFound, conf.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)))
	return nil
}

// GoogleCallback is where Google sends the browser back. The browser is on a
// full-page navigation, so errors go to the login page instead of JSON.
func (h *Handler) GoogleCallback(c *gin.Context) {
	if err := h.googleCallback(c); err != nil {
		if e, ok := errors.AsType[*apperr.Error](err); ok {
			c.Redirect(http.StatusFound, "/login?error="+e.Code)
			return
		}
		web.Error(c, err)
	}
}

func (h *Handler) googleCallback(c *gin.Context) error {
	if h.google == nil {
		return model.ErrGoogleDisabled
	}
	flow, err := c.Cookie(googleFlowCookie)
	parts := []string{}
	if err == nil {
		parts = strings.SplitN(flow, "|", 4)
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: googleFlowCookie, Path: "/api/auth/google", MaxAge: -1})
	if len(parts) != 4 || c.Query("state") != parts[0] {
		return apperr.New(apperr.BadRequest, "google_state", "login expired or was started in another browser; try again")
	}
	nonce, verifier, next := parts[1], parts[2], parts[3]
	if e := c.Query("error"); e != "" {
		return apperr.New(apperr.BadRequest, "google_denied", "Google login was cancelled: "+e)
	}

	ctx := c.Request.Context()
	conf, verify, err := h.google.init(ctx)
	if err != nil {
		return err
	}
	tok, err := conf.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		return apperr.New(apperr.BadRequest, "google_exchange", "could not complete Google login")
	}
	raw, _ := tok.Extra("id_token").(string)
	idToken, err := verify.Verify(ctx, raw) // signature, issuer, audience, expiry
	if err != nil || idToken.Nonce != nonce {
		return apperr.New(apperr.BadRequest, "google_token", "invalid Google ID token")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	a, err := h.svc.LoginWithGoogle(ctx, model.GoogleIdentity{
		Subject: idToken.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified, Name: claims.Name,
	})
	if err != nil {
		return err
	}
	if err := h.startSession(c, a.ID); err != nil {
		return err
	}
	c.Redirect(http.StatusFound, next)
	return nil
}

// safeNext only allows same-site relative paths (no open redirects).
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.Contains(next, `\`) || strings.Contains(next, "|") {
		return "/"
	}
	return next
}

func randomString() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
