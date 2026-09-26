// Package service holds the auth rules: passwords, sessions and how a Google
// login links to accounts (ADR-0002). SQL lives behind the Repository port.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"restaurants/internal/auth/model"
	"restaurants/internal/platform/apperr"
)

// Repository is the outbound port for account, identity and session storage.
// Lookups that find nothing return model.ErrNotFound.
type Repository interface {
	CreatePasswordAccount(ctx context.Context, email, displayName, passwordHash string) (int64, error) // ErrEmailTaken on duplicate
	PasswordHashByEmail(ctx context.Context, email string) (model.Account, string, error)
	PasswordHash(ctx context.Context, accountID int64) (string, error) // locks the row
	InsertPassword(ctx context.Context, accountID int64, hash string) error
	UpdatePassword(ctx context.Context, accountID int64, hash string) error

	// CreateSession stores a session; impersonator is the admin behind an
	// impersonation session, 0 for a normal login.
	CreateSession(ctx context.Context, tokenHash []byte, accountID, impersonator int64, expires time.Time) error
	// AccountBySession fills Account.Impersonator for impersonation sessions.
	// Those are dropped (ErrNotFound) once the impersonator is no longer an
	// admin or is banned.
	AccountBySession(ctx context.Context, tokenHash []byte, now time.Time) (model.Account, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error

	AccountByGoogleSubject(ctx context.Context, subject string) (int64, error)
	AccountByEmailForUpdate(ctx context.Context, email string) (id int64, verified bool, err error)
	DropPasswordAndSessions(ctx context.Context, accountID int64) error
	MarkEmailVerified(ctx context.Context, accountID int64) error
	CreateVerifiedAccount(ctx context.Context, email, displayName string) (int64, error)
	LinkGoogle(ctx context.Context, accountID int64, subject string) error // ErrGoogleConflict on duplicate
	Account(ctx context.Context, id int64) (model.Account, error)
	UpdateDisplayName(ctx context.Context, accountID int64, displayName string) error
}

// TxRunner runs fn in one database transaction (platform/database.DB).
type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service is the inbound port used by the HTTP handlers.
type Service interface {
	Signup(ctx context.Context, email, password, displayName string) (model.Account, error)
	Login(ctx context.Context, email, password string) (model.Account, error)
	SetPassword(ctx context.Context, accountID int64, current, next string) error
	UpdateProfile(ctx context.Context, accountID int64, displayName string) (model.Account, error)
	LoginWithGoogle(ctx context.Context, g model.GoogleIdentity) (model.Account, error)
	CreateSession(ctx context.Context, accountID int64) (model.Session, error)
	AccountForSession(ctx context.Context, token string) (model.Account, error)
	DeleteSession(ctx context.Context, token string) error
	// Impersonate ends the admin's session (token) and starts one as the
	// target user that remembers the admin (R-ADMIN-7).
	Impersonate(ctx context.Context, admin model.Account, token string, targetID int64) (model.Session, error)
	// StopImpersonating ends the impersonation session and logs the admin back in.
	StopImpersonating(ctx context.Context, token string) (model.Session, model.Account, error)
}

type service struct {
	repo Repository
	tx   TxRunner
	now  func() time.Time
}

func New(repo Repository, tx TxRunner) Service {
	return &service{repo: repo, tx: tx, now: time.Now}
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return "", apperr.InvalidInput("invalid email address")
	}
	return email, nil
}

func validatePassword(pw string) error {
	if len(pw) < model.MinPasswordLen || len(pw) > model.MaxPasswordLen {
		return apperr.InvalidInput("password must be %d to %d characters", model.MinPasswordLen, model.MaxPasswordLen)
	}
	return nil
}

func validateDisplayName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > 80 {
		return "", apperr.InvalidInput("display name must be 1 to 80 characters")
	}
	return name, nil
}

// UpdateProfile changes what others see (the display name). The email is
// the login identity and can't be changed.
func (s *service) UpdateProfile(ctx context.Context, accountID int64, displayName string) (model.Account, error) {
	name, err := validateDisplayName(displayName)
	if err != nil {
		return model.Account{}, err
	}
	if err := s.repo.UpdateDisplayName(ctx, accountID, name); err != nil {
		return model.Account{}, err
	}
	return s.repo.Account(ctx, accountID)
}

func (s *service) Signup(ctx context.Context, email, password, displayName string) (model.Account, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return model.Account{}, err
	}
	if err := validatePassword(password); err != nil {
		return model.Account{}, err
	}
	displayName, err = validateDisplayName(displayName)
	if err != nil {
		return model.Account{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.Account{}, err
	}
	id, err := s.repo.CreatePasswordAccount(ctx, email, displayName, string(hash))
	if err != nil {
		return model.Account{}, err
	}
	return model.Account{ID: id, Email: email, DisplayName: displayName, HasPassword: true}, nil
}

// dummyHash keeps Login's timing similar whether or not the email exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

func (s *service) Login(ctx context.Context, email, password string) (model.Account, error) {
	a, hash, err := s.repo.PasswordHashByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, model.ErrNotFound) {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return model.Account{}, model.ErrInvalidCredentials
	}
	if err != nil {
		return model.Account{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return model.Account{}, model.ErrInvalidCredentials
	}
	// Checked after the password, so the ban isn't revealed to someone guessing.
	if a.Banned {
		return model.Account{}, model.ErrBanned
	}
	return a, nil
}

// SetPassword sets or replaces the password. An existing password must be
// confirmed with current; accounts without one (Google-only, or dropped on
// Google link, ADR-0002) can set it freely.
func (s *service) SetPassword(ctx context.Context, accountID int64, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		old, err := s.repo.PasswordHash(ctx, accountID)
		switch {
		case errors.Is(err, model.ErrNotFound):
			return s.repo.InsertPassword(ctx, accountID, string(hash))
		case err != nil:
			return err
		case bcrypt.CompareHashAndPassword([]byte(old), []byte(current)) != nil:
			return model.ErrInvalidCredentials
		}
		return s.repo.UpdatePassword(ctx, accountID, string(hash))
	})
}

// LoginWithGoogle finds, links or creates the account for a Google identity.
// Google takes priority: linking into an account whose email was never
// verified drops that account's password and sessions, because whoever set
// the password never proved they own the email (ADR-0002).
func (s *service) LoginWithGoogle(ctx context.Context, g model.GoogleIdentity) (model.Account, error) {
	if !g.EmailVerified {
		return model.Account{}, model.ErrGoogleEmailUnverified
	}
	email, err := normalizeEmail(g.Email)
	if err != nil {
		return model.Account{}, err
	}
	var id int64
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// 1. Known Google identity: just log in.
		id, err = s.repo.AccountByGoogleSubject(ctx, g.Subject)
		if err == nil || !errors.Is(err, model.ErrNotFound) {
			return err
		}
		// 2. Existing account with this email: link it. 3. Otherwise create one.
		var verified bool
		id, verified, err = s.repo.AccountByEmailForUpdate(ctx, email)
		switch {
		case err == nil && !verified:
			if err := s.repo.DropPasswordAndSessions(ctx, id); err != nil {
				return err
			}
			if err := s.repo.MarkEmailVerified(ctx, id); err != nil {
				return err
			}
		case errors.Is(err, model.ErrNotFound):
			if id, err = s.repo.CreateVerifiedAccount(ctx, email, googleDisplayName(g.Name, email)); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		return s.repo.LinkGoogle(ctx, id, g.Subject)
	})
	if err != nil {
		return model.Account{}, err
	}
	a, err := s.repo.Account(ctx, id)
	if err == nil && a.Banned {
		return model.Account{}, model.ErrBanned
	}
	return a, err
}

func googleDisplayName(name, email string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name
}

// CreateSession starts a session. The token goes in the cookie; only its
// SHA-256 is stored, so a leaked database can't be replayed as cookies.
func (s *service) CreateSession(ctx context.Context, accountID int64) (model.Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return model.Session{}, err
	}
	sess := model.Session{Token: base64.RawURLEncoding.EncodeToString(raw), Expires: s.now().Add(model.SessionTTL)}
	return sess, s.repo.CreateSession(ctx, hashToken(sess.Token), accountID, 0, sess.Expires)
}

func (s *service) Impersonate(ctx context.Context, admin model.Account, token string, targetID int64) (model.Session, error) {
	switch {
	case !admin.IsAdmin:
		return model.Session{}, model.ErrAdminOnly
	case admin.Impersonator != nil:
		// Already a stand-in: switch back first, so the chain is never longer than one.
		return model.Session{}, model.ErrAdminOnly
	case admin.ID == targetID:
		return model.Session{}, model.ErrImpersonateSelf
	}
	target, err := s.repo.Account(ctx, targetID)
	switch {
	case errors.Is(err, model.ErrNotFound):
		return model.Session{}, apperr.ErrNotFound
	case err != nil:
		return model.Session{}, err
	case target.IsAdmin:
		return model.Session{}, model.ErrImpersonateAdmin
	case target.Banned:
		return model.Session{}, model.ErrImpersonateBanned
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return model.Session{}, err
	}
	sess := model.Session{Token: base64.RawURLEncoding.EncodeToString(raw), Expires: s.now().Add(model.ImpersonationTTL)}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.DeleteSession(ctx, hashToken(token)); err != nil {
			return err
		}
		return s.repo.CreateSession(ctx, hashToken(sess.Token), target.ID, admin.ID, sess.Expires)
	})
	return sess, err
}

func (s *service) StopImpersonating(ctx context.Context, token string) (model.Session, model.Account, error) {
	a, err := s.AccountForSession(ctx, token)
	if err != nil {
		return model.Session{}, model.Account{}, err
	}
	if a.Impersonator == nil {
		return model.Session{}, model.Account{}, model.ErrNotImpersonating
	}
	var sess model.Session
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.DeleteSession(ctx, hashToken(token)); err != nil {
			return err
		}
		var err error
		sess, err = s.CreateSession(ctx, a.Impersonator.ID)
		return err
	})
	if err != nil {
		return model.Session{}, model.Account{}, err
	}
	admin, err := s.repo.Account(ctx, a.Impersonator.ID)
	return sess, admin, err
}

func (s *service) AccountForSession(ctx context.Context, token string) (model.Account, error) {
	a, err := s.repo.AccountBySession(ctx, hashToken(token), s.now())
	if errors.Is(err, model.ErrNotFound) {
		return model.Account{}, model.ErrNoSession
	}
	return a, err
}

func (s *service) DeleteSession(ctx context.Context, token string) error {
	return s.repo.DeleteSession(ctx, hashToken(token))
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
