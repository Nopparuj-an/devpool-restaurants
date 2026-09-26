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
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"restaurants/internal/httpx"
)

// GoogleIdentity is what we trust from a verified Google ID token.
type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

var ErrGoogleEmailUnverified = httpx.NewError(403, "google_email_unverified", "your Google account's email is not verified")

// LoginWithGoogle finds, links or creates the account for a Google identity
// (ADR-0002). Google takes priority: when it links into an existing account
// whose email was never verified, that account's password and sessions are
// dropped, because whoever set that password never proved they own the email.
func (s *Service) LoginWithGoogle(ctx context.Context, g GoogleIdentity) (Account, error) {
	if !g.EmailVerified {
		return Account{}, ErrGoogleEmailUnverified
	}
	email, err := normalizeEmail(g.Email)
	if err != nil {
		return Account{}, err
	}

	var id int64
	err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		// 1. Known Google identity: just log in.
		err := tx.QueryRow(ctx, `
			SELECT account_id FROM auth_identities WHERE provider = 'google' AND provider_subject = $1`,
			g.Subject).Scan(&id)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		// 2. Existing account with this email: link it.
		var verified bool
		err = tx.QueryRow(ctx, `SELECT id, email_verified FROM accounts WHERE lower(email) = $1 FOR UPDATE`, email).
			Scan(&id, &verified)
		switch {
		case err == nil:
			if !verified {
				if _, err := tx.Exec(ctx, `DELETE FROM auth_identities WHERE account_id = $1 AND provider = 'password'`, id); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE account_id = $1`, id); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `UPDATE accounts SET email_verified = true WHERE id = $1`, id); err != nil {
					return err
				}
			}
		case errors.Is(err, pgx.ErrNoRows):
			// 3. New account.
			name := strings.TrimSpace(g.Name)
			if name == "" {
				name, _, _ = strings.Cut(email, "@")
			}
			if len(name) > 80 {
				name = name[:80]
			}
			if err := tx.QueryRow(ctx, `
				INSERT INTO accounts (email, display_name, email_verified) VALUES ($1, $2, true) RETURNING id`,
				email, name).Scan(&id); err != nil {
				return err
			}
		default:
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO auth_identities (account_id, provider, provider_subject) VALUES ($1, 'google', $2)`,
			id, g.Subject)
		if isUniqueViolation(err) {
			// The account is already linked to a different Google account.
			return httpx.NewError(409, "google_conflict", "this email is linked to a different Google account")
		}
		return err
	})
	if err != nil {
		return Account{}, err
	}
	return scanAccount(s.db.QueryRow(ctx, `SELECT `+accountColumns+` FROM accounts a WHERE a.id = $1`, id))
}

// GoogleConfig enables "Sign in with Google" when ClientID is set.
type GoogleConfig struct {
	ClientID, ClientSecret string
	// RedirectURL must be registered in Google Cloud Console, e.g.
	// http://localhost:3000/api/auth/google/callback (through the Next.js rewrite).
	RedirectURL string
}

// google runs the OIDC authorization-code flow with PKCE, state and nonce.
type google struct {
	cfg GoogleConfig

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

func (h *Handler) providers(w http.ResponseWriter, r *http.Request) error {
	httpx.JSON(w, http.StatusOK, map[string]bool{"password": true, "google": h.google != nil})
	return nil
}

// googleStart redirects to Google. ?next=/path is where to land afterwards.
func (h *Handler) googleStart(w http.ResponseWriter, r *http.Request) error {
	if h.google == nil {
		return httpx.NewError(404, "google_disabled", "Google login is not configured")
	}
	conf, _, err := h.google.init(r.Context())
	if err != nil {
		return err
	}
	state, nonce, verifier := randomString(), randomString(), oauth2.GenerateVerifier()
	// The flow cookie binds the callback to this browser (CSRF on login).
	http.SetCookie(w, &http.Cookie{
		Name:     googleFlowCookie,
		Value:    strings.Join([]string{state, nonce, verifier, safeNext(r.URL.Query().Get("next"))}, "|"),
		Path:     "/api/auth/google",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode, // sent on Google's top-level redirect back
	})
	http.Redirect(w, r, conf.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusFound)
	return nil
}

func (h *Handler) googleCallback(w http.ResponseWriter, r *http.Request) error {
	if h.google == nil {
		return httpx.NewError(404, "google_disabled", "Google login is not configured")
	}
	c, err := r.Cookie(googleFlowCookie)
	parts := []string{}
	if err == nil {
		parts = strings.SplitN(c.Value, "|", 4)
	}
	http.SetCookie(w, &http.Cookie{Name: googleFlowCookie, Path: "/api/auth/google", MaxAge: -1})
	if len(parts) != 4 || r.URL.Query().Get("state") != parts[0] {
		return httpx.NewError(400, "google_state", "login expired or was started in another browser; try again")
	}
	nonce, verifier, next := parts[1], parts[2], parts[3]
	if e := r.URL.Query().Get("error"); e != "" {
		return httpx.NewError(400, "google_denied", "Google login was cancelled: "+e)
	}

	conf, verify, err := h.google.init(r.Context())
	if err != nil {
		return err
	}
	tok, err := conf.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		return httpx.NewError(400, "google_exchange", "could not complete Google login")
	}
	raw, _ := tok.Extra("id_token").(string)
	idToken, err := verify.Verify(r.Context(), raw) // signature, issuer, audience, expiry
	if err != nil || idToken.Nonce != nonce {
		return httpx.NewError(400, "google_token", "invalid Google ID token")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return err
	}

	a, err := h.svc.LoginWithGoogle(r.Context(), GoogleIdentity{
		Subject: idToken.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified, Name: claims.Name,
	})
	if err != nil {
		return err
	}
	if err := h.startSession(w, r, a.ID); err != nil {
		return err
	}
	http.Redirect(w, r, next, http.StatusFound)
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
