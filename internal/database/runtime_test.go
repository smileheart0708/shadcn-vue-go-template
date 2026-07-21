package database

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenSQLiteProvidesDomainRepositories(t *testing.T) {
	t.Parallel()

	runtime, err := Open(context.Background(), Config{
		Driver: DriverSQLite,
		DSN:    filepath.Join(t.TempDir(), "app.db"),
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	if runtime.Driver() != DriverSQLite {
		t.Fatalf("Driver() = %q, want %q", runtime.Driver(), DriverSQLite)
	}
	if runtime.Identity == nil || runtime.Sessions == nil || runtime.Setup == nil || runtime.AccountPolicies == nil {
		t.Fatal("expected SQLite runtime to provide every domain repository")
	}
}

func TestOpenRejectsDriverNotCompiledIntoBinary(t *testing.T) {
	t.Parallel()

	const dsn = "postgres://user:secret@example.test/app"
	_, err := Open(context.Background(), Config{
		Driver: "postgres",
		DSN:    dsn,
	})
	if err == nil {
		t.Fatal("expected unsupported driver error")
	}
	if strings.Contains(err.Error(), dsn) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsupported driver error leaks DSN: %v", err)
	}
}
