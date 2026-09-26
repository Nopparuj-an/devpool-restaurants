// Package auth implements accounts, password login, Google login, and
// server-side sessions (ADR-0002).
//
// Sessions: the browser holds an opaque random token in an HttpOnly cookie;
// the database stores only its SHA-256 hash, so a leaked DB dump can't be
// replayed as cookies. Logout deletes the row.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"restaurants/internal/httpx"
)

const (
	SessionTTL     = 30 * 24 * time.Hour
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt ignores bytes past 72
)

var (
	ErrEmailTaken         = httpx.NewError(409, "email_taken", "an account with this email already exists")
	ErrInvalidCredentials = httpx.NewError(401, "invalid_credentials", "wrong email or password")
	ErrNoSession          = errors.New("no valid session")
)

// Account is the logged-in user as returned by the API.
type Account struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	EmailVerified bool   `json:"email_verified"`
	HasPassword   bool   `json:"has_password"`
}

type Service struct {
	db  *pgxpool.Pool
	now func() time.Time
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db, now: time.Now}
}

// accountColumns selects an Account; callers join `accounts a`.
const accountColumns = `a.id, a.email, a.display_name, a.email_verified,
	EXISTS (SELECT 1 FROM auth_identities i WHERE i.account_id = a.id AND i.provider = 'password')`

func scanAccount(row pgx.Row) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.Email, &a.DisplayName, &a.EmailVerified, &a.HasPassword)
	return a, err
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return "", httpx.Invalid("invalid email address")
	}
	return email, nil
}

func validatePassword(pw string) error {
	if len(pw) < minPasswordLen || len(pw) > maxPasswordLen {
		return httpx.Invalid("password must be %d to %d characters", minPasswordLen, maxPasswordLen)
	}
	return nil
}

// Signup creates an account with a password identity.
func (s *Service) Signup(ctx context.Context, email, password, displayName string) (Account, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return Account{}, err
	}
	if err := validatePassword(password); err != nil {
		return Account{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(displayName) > 80 {
		return Account{}, httpx.Invalid("display name must be 1 to 80 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Account{}, err
	}

	var a Account
	err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO accounts (email, display_name) VALUES ($1, $2) RETURNING id`,
			email, displayName).Scan(&a.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO auth_identities (account_id, provider, provider_subject, password_hash)
			 VALUES ($1, 'password', $2, $3)`, a.ID, strconv.FormatInt(a.ID, 10), string(hash))
		return err
	})
	if isUniqueViolation(err) {
		return Account{}, ErrEmailTaken
	}
	if err != nil {
		return Account{}, err
	}
	return Account{ID: a.ID, Email: email, DisplayName: displayName, HasPassword: true}, nil
}

// dummyHash keeps Login's timing similar whether or not the email exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

// Login checks an email/password pair.
func (s *Service) Login(ctx context.Context, email, password string) (Account, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var (
		a    Account
		hash string
	)
	err := s.db.QueryRow(ctx, `
		SELECT a.id, a.email, a.display_name, a.email_verified, i.password_hash
		FROM accounts a JOIN auth_identities i ON i.account_id = a.id AND i.provider = 'password'
		WHERE lower(a.email) = $1`, email).
		Scan(&a.ID, &a.Email, &a.DisplayName, &a.EmailVerified, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return Account{}, ErrInvalidCredentials
	}
	if err != nil {
		return Account{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return Account{}, ErrInvalidCredentials
	}
	a.HasPassword = true
	return a, nil
}

// SetPassword sets or replaces the account's password. If the account already
// has one, current must match it; accounts without one (Google-only, or whose
// password was dropped on Google link, ADR-0002) can set it freely.
func (s *Service) SetPassword(ctx context.Context, accountID int64, current, next string) error {
	if err := validatePassword(next); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		var old string
		err := tx.QueryRow(ctx,
			`SELECT password_hash FROM auth_identities WHERE account_id = $1 AND provider = 'password' FOR UPDATE`,
			accountID).Scan(&old)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			_, err = tx.Exec(ctx,
				`INSERT INTO auth_identities (account_id, provider, provider_subject, password_hash)
				 VALUES ($1, 'password', $2, $3)`, accountID, strconv.FormatInt(accountID, 10), string(hash))
			return err
		case err != nil:
			return err
		case bcrypt.CompareHashAndPassword([]byte(old), []byte(current)) != nil:
			return ErrInvalidCredentials
		}
		_, err = tx.Exec(ctx,
			`UPDATE auth_identities SET password_hash = $2 WHERE account_id = $1 AND provider = 'password'`,
			accountID, string(hash))
		return err
	})
}

// CreateSession starts a session and returns the cookie token.
func (s *Service) CreateSession(ctx context.Context, accountID int64) (token string, expires time.Time, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expires = s.now().Add(SessionTTL)
	_, err = s.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, account_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(token), accountID, expires)
	return token, expires, err
}

// AccountForSession returns the account owning a live session token.
func (s *Service) AccountForSession(ctx context.Context, token string) (Account, error) {
	a, err := scanAccount(s.db.QueryRow(ctx, `
		SELECT `+accountColumns+`
		FROM sessions s JOIN accounts a ON a.id = s.account_id
		WHERE s.token_hash = $1 AND s.expires_at > $2`, hashToken(token), s.now()))
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNoSession
	}
	return a, err
}

// DeleteSession ends a session (logout). Unknown tokens are ignored.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(token))
	return err
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == "23505"
}
