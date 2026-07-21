package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/identity"
	"main/internal/setup"
)

var (
	_ identity.Repository        = (*Store)(nil)
	_ auth.SessionRepository     = (*Store)(nil)
	_ setup.Repository           = (*Store)(nil)
	_ accountpolicies.Repository = (*Store)(nil)
)

type dbTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) requireDB() (*sql.DB, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("database: nil SQLite store")
	}
	return s.db, nil
}

func withTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	ctx = normalizeContext(ctx)
	if db == nil {
		return errors.New("database: transaction requires a non-nil SQLite database")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database: begin SQLite transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("database: rollback SQLite transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("database: commit SQLite transaction: %w", err)
	}
	return nil
}
