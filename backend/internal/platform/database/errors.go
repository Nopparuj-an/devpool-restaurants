package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func errorsAs(err error) (*pgconn.PgError, bool) {
	return errors.AsType[*pgconn.PgError](err)
}

// IsNoRows reports that a QueryRow found nothing.
func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
