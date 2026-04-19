package database

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	ctx = normalizeContext(ctx)
	if db == nil {
		return fmt.Errorf("db: transaction requires a non-nil *sql.DB")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("db: rollback transaction after %v: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit transaction: %w", err)
	}

	return nil
}
