// Package repository implements the auth service's Repository port with SQL.
package repository

import (
	"context"
	"strconv"
	"time"

	"restaurants/internal/auth/model"
	"restaurants/internal/platform/database"
)

type Repository struct {
	db *database.DB
}

func New(db *database.DB) *Repository { return &Repository{db: db} }

// accountColumns selects a model.Account; queries alias accounts as `a`.
const accountColumns = `a.id, a.email, a.display_name, a.email_verified,
	EXISTS (SELECT 1 FROM auth_identities i WHERE i.account_id = a.id AND i.provider = 'password'),
	a.is_admin, a.banned_at IS NOT NULL`

func scanAccount(row interface{ Scan(...any) error }) (model.Account, error) {
	var a model.Account
	err := row.Scan(&a.ID, &a.Email, &a.DisplayName, &a.EmailVerified, &a.HasPassword, &a.IsAdmin, &a.Banned)
	if database.IsNoRows(err) {
		return a, model.ErrNotFound
	}
	return a, err
}

func (r *Repository) CreatePasswordAccount(ctx context.Context, email, displayName, hash string) (int64, error) {
	var id int64
	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		q := r.db.Conn(ctx)
		if err := q.QueryRow(ctx,
			`INSERT INTO accounts (email, display_name) VALUES ($1, $2) RETURNING id`, email, displayName).Scan(&id); err != nil {
			return err
		}
		return r.InsertPassword(ctx, id, hash)
	})
	if database.IsUniqueViolation(err) {
		return 0, model.ErrEmailTaken
	}
	return id, err
}

func (r *Repository) PasswordHashByEmail(ctx context.Context, email string) (model.Account, string, error) {
	var (
		a    model.Account
		hash string
	)
	err := r.db.Conn(ctx).QueryRow(ctx, `
		SELECT a.id, a.email, a.display_name, a.email_verified, a.is_admin, a.banned_at IS NOT NULL, i.password_hash
		FROM accounts a JOIN auth_identities i ON i.account_id = a.id AND i.provider = 'password'
		WHERE lower(a.email) = $1`, email).
		Scan(&a.ID, &a.Email, &a.DisplayName, &a.EmailVerified, &a.IsAdmin, &a.Banned, &hash)
	if database.IsNoRows(err) {
		return a, "", model.ErrNotFound
	}
	a.HasPassword = true
	return a, hash, err
}

func (r *Repository) PasswordHash(ctx context.Context, accountID int64) (string, error) {
	var hash string
	err := r.db.Conn(ctx).QueryRow(ctx,
		`SELECT password_hash FROM auth_identities WHERE account_id = $1 AND provider = 'password' FOR UPDATE`,
		accountID).Scan(&hash)
	if database.IsNoRows(err) {
		return "", model.ErrNotFound
	}
	return hash, err
}

func (r *Repository) InsertPassword(ctx context.Context, accountID int64, hash string) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `
		INSERT INTO auth_identities (account_id, provider, provider_subject, password_hash)
		VALUES ($1, 'password', $2, $3)`, accountID, strconv.FormatInt(accountID, 10), hash)
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, accountID int64, hash string) error {
	_, err := r.db.Conn(ctx).Exec(ctx,
		`UPDATE auth_identities SET password_hash = $2 WHERE account_id = $1 AND provider = 'password'`, accountID, hash)
	return err
}

func (r *Repository) CreateSession(ctx context.Context, tokenHash []byte, accountID int64, expires time.Time) error {
	_, err := r.db.Conn(ctx).Exec(ctx,
		`INSERT INTO sessions (token_hash, account_id, expires_at) VALUES ($1, $2, $3)`, tokenHash, accountID, expires)
	return err
}

func (r *Repository) AccountBySession(ctx context.Context, tokenHash []byte, now time.Time) (model.Account, error) {
	return scanAccount(r.db.Conn(ctx).QueryRow(ctx, `
		SELECT `+accountColumns+`
		FROM sessions s JOIN accounts a ON a.id = s.account_id
		WHERE s.token_hash = $1 AND s.expires_at > $2 AND a.banned_at IS NULL`, tokenHash, now))
}

func (r *Repository) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *Repository) AccountByGoogleSubject(ctx context.Context, subject string) (int64, error) {
	var id int64
	err := r.db.Conn(ctx).QueryRow(ctx,
		`SELECT account_id FROM auth_identities WHERE provider = 'google' AND provider_subject = $1`, subject).Scan(&id)
	if database.IsNoRows(err) {
		return 0, model.ErrNotFound
	}
	return id, err
}

func (r *Repository) AccountByEmailForUpdate(ctx context.Context, email string) (int64, bool, error) {
	var (
		id       int64
		verified bool
	)
	err := r.db.Conn(ctx).QueryRow(ctx,
		`SELECT id, email_verified FROM accounts WHERE lower(email) = $1 FOR UPDATE`, email).Scan(&id, &verified)
	if database.IsNoRows(err) {
		return 0, false, model.ErrNotFound
	}
	return id, verified, err
}

func (r *Repository) DropPasswordAndSessions(ctx context.Context, accountID int64) error {
	q := r.db.Conn(ctx)
	if _, err := q.Exec(ctx, `DELETE FROM auth_identities WHERE account_id = $1 AND provider = 'password'`, accountID); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `DELETE FROM sessions WHERE account_id = $1`, accountID)
	return err
}

func (r *Repository) MarkEmailVerified(ctx context.Context, accountID int64) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `UPDATE accounts SET email_verified = true WHERE id = $1`, accountID)
	return err
}

func (r *Repository) CreateVerifiedAccount(ctx context.Context, email, displayName string) (int64, error) {
	var id int64
	err := r.db.Conn(ctx).QueryRow(ctx,
		`INSERT INTO accounts (email, display_name, email_verified) VALUES ($1, $2, true) RETURNING id`,
		email, displayName).Scan(&id)
	return id, err
}

func (r *Repository) LinkGoogle(ctx context.Context, accountID int64, subject string) error {
	_, err := r.db.Conn(ctx).Exec(ctx,
		`INSERT INTO auth_identities (account_id, provider, provider_subject) VALUES ($1, 'google', $2)`, accountID, subject)
	if database.IsUniqueViolation(err) {
		// The account is already linked to a different Google account.
		return model.ErrGoogleConflict
	}
	return err
}

func (r *Repository) UpdateDisplayName(ctx context.Context, accountID int64, displayName string) error {
	_, err := r.db.Conn(ctx).Exec(ctx, `UPDATE accounts SET display_name = $2 WHERE id = $1`, accountID, displayName)
	return err
}

func (r *Repository) Account(ctx context.Context, id int64) (model.Account, error) {
	return scanAccount(r.db.Conn(ctx).QueryRow(ctx, `SELECT `+accountColumns+` FROM accounts a WHERE a.id = $1`, id))
}
