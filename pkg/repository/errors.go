package repository

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const pgDuplicateKeyCode = "23505"

// MapError translates database errors to domain errors.
// It maps sql.ErrNoRows to notFoundErr and PostgreSQL unique violation (23505)
// to duplicateErr. Other errors are returned unchanged.
func MapError(err error, notFoundErr, duplicateErr error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return notFoundErr
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgDuplicateKeyCode {
		return duplicateErr
	}

	return err
}
