package repository_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/JaimeStill/herald/pkg/repository"
)

var (
	errNotFound  = errors.New("not found")
	errDuplicate = errors.New("duplicate")
)

func TestMapErrorNil(t *testing.T) {
	got := repository.MapError(nil, errNotFound, errDuplicate)
	if got != nil {
		t.Errorf("MapError(nil) = %v, want nil", got)
	}
}

func TestMapErrorNotFound(t *testing.T) {
	got := repository.MapError(sql.ErrNoRows, errNotFound, errDuplicate)
	if !errors.Is(got, errNotFound) {
		t.Errorf("MapError(ErrNoRows) = %v, want %v", got, errNotFound)
	}
}

func TestMapErrorDuplicate(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505"}
	got := repository.MapError(pgErr, errNotFound, errDuplicate)
	if !errors.Is(got, errDuplicate) {
		t.Errorf("MapError(PgError 23505) = %v, want %v", got, errDuplicate)
	}
}

func TestMapErrorPassthrough(t *testing.T) {
	original := errors.New("some other error")
	got := repository.MapError(original, errNotFound, errDuplicate)
	if got != original {
		t.Errorf("MapError(other) = %v, want %v", got, original)
	}
}

func TestMapErrorPgNonDuplicate(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23503"}
	got := repository.MapError(pgErr, errNotFound, errDuplicate)
	if got != pgErr {
		t.Errorf("MapError(PgError 23503) should pass through, got %v", got)
	}
}
