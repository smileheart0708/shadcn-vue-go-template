package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func TestWithTxJoinsFunctionAndRollbackErrors(t *testing.T) {
	t.Parallel()

	fnErr := errors.New("function failed")
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	})

	err = WithTx(context.Background(), db, func(tx *sql.Tx) error {
		if _, execErr := tx.ExecContext(context.Background(), "THIS IS NOT SQL"); execErr == nil {
			t.Fatal("expected invalid statement to fail")
		}
		if commitErr := tx.Commit(); commitErr != nil {
			t.Fatalf("forced commit failed before rollback test: %v", commitErr)
		}
		return fnErr
	})
	if err == nil {
		t.Fatal("expected WithTx error")
	}
	if !errors.Is(err, fnErr) {
		t.Fatalf("expected function error in joined chain, got %v", err)
	}
	if !errors.Is(err, sql.ErrTxDone) {
		t.Fatalf("expected rollback error in joined chain, got %v", err)
	}
}
