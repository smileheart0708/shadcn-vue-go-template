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

	tx, beginErr := db.BeginTx(ctx, nil)
	if beginErr != nil {
		return fmt.Errorf("db: begin transaction: %w", beginErr)
	}

	if fnErr := fn(tx); fnErr != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("db: rollback transaction after %v: %w", fnErr, rollbackErr)
		}
		return fnErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("db: commit transaction: %w", commitErr)
	}

	return nil
}
