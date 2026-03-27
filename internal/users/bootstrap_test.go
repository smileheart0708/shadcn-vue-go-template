package users

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"main/internal/database"
)

func TestBootstrapManagerCreatesDefaultSuperAdminOnce(t *testing.T) {
	t.Parallel()

	store, dataDir, logs := newBootstrapTestStore(t)
	manager := NewBootstrapManager(store, dataDir, testLogger(logs))

	if err := manager.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure bootstrap admin: %v", err)
	}

	password := readPasswordFile(t, dataDir)
	superAdmin, err := store.GetFirstSuperAdmin(context.Background())
	if err != nil {
		t.Fatalf("failed to load bootstrap admin: %v", err)
	}

	if superAdmin.Username != "admin" {
		t.Fatalf("expected username admin, got %q", superAdmin.Username)
	}
	if superAdmin.Role != RoleSuperAdmin {
		t.Fatalf("expected role %d, got %d", RoleSuperAdmin, superAdmin.Role)
	}
	if !superAdmin.BootstrapPasswordActive {
		t.Fatal("expected bootstrap password flag to be active")
	}

	logs.Reset()
	if err := manager.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure bootstrap admin on restart: %v", err)
	}

	samePassword := readPasswordFile(t, dataDir)
	if samePassword != password {
		t.Fatalf("expected bootstrap password to persist across restarts, got %q then %q", password, samePassword)
	}

	count, err := store.CountSuperAdmins(context.Background())
	if err != nil {
		t.Fatalf("failed to count super admins: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one super admin, got %d", count)
	}
	if !strings.Contains(logs.String(), password) {
		t.Fatalf("expected bootstrap password to be logged, got %q", logs.String())
	}
}

func TestBootstrapManagerRotatesPasswordWhenFileMissing(t *testing.T) {
	t.Parallel()

	store, dataDir, logs := newBootstrapTestStore(t)
	manager := NewBootstrapManager(store, dataDir, testLogger(logs))

	if err := manager.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure bootstrap admin: %v", err)
	}

	originalPassword := readPasswordFile(t, dataDir)
	if err := os.Remove(BootstrapPasswordPath(dataDir)); err != nil {
		t.Fatalf("failed to remove bootstrap password file: %v", err)
	}

	logs.Reset()
	if err := manager.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to rotate bootstrap password: %v", err)
	}

	rotatedPassword := readPasswordFile(t, dataDir)
	if rotatedPassword == originalPassword {
		t.Fatalf("expected bootstrap password rotation after file deletion, got %q", rotatedPassword)
	}

	superAdmin, err := store.GetFirstSuperAdmin(context.Background())
	if err != nil {
		t.Fatalf("failed to load bootstrap admin: %v", err)
	}

	passwordMatches, err := VerifyPassword(rotatedPassword, superAdmin.PasswordHash)
	if err != nil {
		t.Fatalf("failed to verify rotated password: %v", err)
	}
	if !passwordMatches {
		t.Fatal("expected rotated password hash to match stored password")
	}
	if !strings.Contains(logs.String(), rotatedPassword) {
		t.Fatalf("expected rotated password to be logged, got %q", logs.String())
	}
}

func TestStoreEnforcesUniqueNullableEmail(t *testing.T) {
	t.Parallel()

	store, _, _ := newBootstrapTestStore(t)

	_, err := store.Create(context.Background(), CreateParams{
		Username:     "first",
		Email:        new("shared@example.com"),
		PasswordHash: mustHashPassword(t, "password123"),
		Role:         RoleUser,
	})
	if err != nil {
		t.Fatalf("failed to create first user: %v", err)
	}

	_, err = store.Create(context.Background(), CreateParams{
		Username:     "second",
		Email:        new("shared@example.com"),
		PasswordHash: mustHashPassword(t, "password123"),
		Role:         RoleUser,
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}

	_, err = store.Create(context.Background(), CreateParams{
		Username:     "third",
		PasswordHash: mustHashPassword(t, "password123"),
		Role:         RoleUser,
	})
	if err != nil {
		t.Fatalf("expected nil email to be allowed, got %v", err)
	}
}

func newBootstrapTestStore(t *testing.T) (*Store, string, *bytes.Buffer) {
	t.Helper()

	dataDir := t.TempDir()
	dbContainer, err := database.Open(context.Background(), database.Options{
		Path: filepath.Join(dataDir, "app.db"),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := dbContainer.Close(); err != nil {
			t.Fatalf("failed to close test database: %v", err)
		}
	})

	logs := &bytes.Buffer{}
	return NewStore(dbContainer.DB()), dataDir, logs
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	passwordHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return passwordHash
}

func readPasswordFile(t *testing.T, dataDir string) string {
	t.Helper()

	password, err := os.ReadFile(BootstrapPasswordPath(dataDir))
	if err != nil {
		t.Fatalf("failed to read bootstrap password: %v", err)
	}
	return strings.TrimSpace(string(password))
}

//go:fix inline
func stringPointer(value string) *string {
	return new(value)
}

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{}))
}
