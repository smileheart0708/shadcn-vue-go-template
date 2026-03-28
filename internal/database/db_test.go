package database

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenConfiguresSQLitePragmasAndSingleConnectionPool(t *testing.T) {
	t.Parallel()

	const busyTimeout = 7 * time.Second

	container, err := Open(context.Background(), Options{
		Path:        filepath.Join(t.TempDir(), "app.db"),
		BusyTimeout: busyTimeout,
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	})

	db := container.DB()

	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("failed to query foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign_keys pragma to be enabled, got %d", foreignKeys)
	}

	var actualBusyTimeout int64
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&actualBusyTimeout); err != nil {
		t.Fatalf("failed to query busy_timeout pragma: %v", err)
	}
	if actualBusyTimeout < busyTimeout.Milliseconds() {
		t.Fatalf("expected busy_timeout >= %dms, got %dms", busyTimeout.Milliseconds(), actualBusyTimeout)
	}

	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("failed to query journal_mode pragma: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("expected journal_mode wal, got %q", journalMode)
	}

	stats := db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("expected MaxOpenConnections=1, got %d", stats.MaxOpenConnections)
	}
}

func TestOpenRejectsMultipleSQLiteConnections(t *testing.T) {
	t.Parallel()

	_, err := Open(context.Background(), Options{
		Path:         filepath.Join(t.TempDir(), "app.db"),
		MaxOpenConns: 2,
	})
	if err == nil {
		t.Fatal("expected sqlite multi-connection configuration to be rejected")
	}
}

func TestOpenConfiguresSQLitePragmasForReopenedConnections(t *testing.T) {
	t.Parallel()

	container, err := Open(context.Background(), Options{
		Path: filepath.Join(t.TempDir(), "app.db"),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	})

	db := container.DB()
	db.SetMaxIdleConns(0)

	for i := range 3 {
		var foreignKeys int
		if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
			t.Fatalf("iteration %d: failed to query foreign_keys pragma: %v", i, err)
		}
		if foreignKeys != 1 {
			t.Fatalf("iteration %d: expected foreign_keys pragma to be enabled, got %d", i, foreignKeys)
		}

		var journalMode string
		if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
			t.Fatalf("iteration %d: failed to query journal_mode pragma: %v", i, err)
		}
		if !strings.EqualFold(journalMode, "wal") {
			t.Fatalf("iteration %d: expected journal_mode wal, got %q", i, journalMode)
		}
	}
}
