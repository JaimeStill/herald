// Package repository provides database helper functions for transaction management
// and query execution.
package repository

import (
	"context"
	"database/sql"
)

// Querier is implemented by *sql.DB, *sql.Tx, and *sql.Conn.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Executor is implemented by *sql.DB, *sql.Tx, and *sql.Conn.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Scanner abstracts row scanning for use with query helpers.
type Scanner interface {
	Scan(dest ...any) error
}

// ScanFunc converts a Scanner into a typed value.
// Domain packages define their own scan functions for entity types.
type ScanFunc[T any] func(Scanner) (T, error)

// WithTx executes fn within a database transaction.
// It handles Begin, Commit, and Rollback automatically.
func WithTx[T any](ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) (T, error)) (T, error) {
	var zero T

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()

	result, err := fn(tx)
	if err != nil {
		return zero, err
	}

	if err := tx.Commit(); err != nil {
		return zero, err
	}

	return result, nil
}

// QueryOne executes a query expected to return a single row.
func QueryOne[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) (T, error) {
	var zero T
	row := q.QueryRowContext(ctx, query, args...)
	result, err := scan(row)
	if err != nil {
		return zero, err
	}
	return result, nil
}

// QueryMany executes a query expected to return multiple rows.
// Returns an empty slice if no rows are found.
func QueryMany[T any](ctx context.Context, q Querier, query string, args []any, scan ScanFunc[T]) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]T, 0)
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// ExecExpectOne executes a statement expected to affect exactly one row.
// Returns sql.ErrNoRows if no rows were affected.
func ExecExpectOne(ctx context.Context, e Executor, query string, args ...any) error {
	result, err := e.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
