package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigUsesJWTSecretEnvironmentVariable(t *testing.T) {
	dataDir := t.TempDir()
	secretPath := filepath.Join(dataDir, jwtSecretFileName)

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "env-secret")
	configureSQLiteDatabase(t, dataDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.JWTSecret != "env-secret" {
		t.Fatalf("expected JWT secret from environment, got %q", cfg.JWTSecret)
	}
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Fatalf("expected no persisted JWT secret file when env var is set, stat err=%v", err)
	}
}

func TestLoadConfigUsesPersistedJWTSecretFile(t *testing.T) {
	dataDir := t.TempDir()
	secretPath := filepath.Join(dataDir, jwtSecretFileName)

	if err := os.WriteFile(secretPath, []byte("persisted-secret\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "")
	configureSQLiteDatabase(t, dataDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.JWTSecret != "persisted-secret" {
		t.Fatalf("expected JWT secret from file, got %q", cfg.JWTSecret)
	}
}

func TestLoadConfigGeneratesAndPersistsJWTSecretWhenMissing(t *testing.T) {
	dataDir := t.TempDir()
	secretPath := filepath.Join(dataDir, jwtSecretFileName)

	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "")
	configureSQLiteDatabase(t, dataDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(cfg.JWTSecret) != jwtSecretEntropyBytes*2 {
		t.Fatalf("expected generated JWT secret length %d, got %d", jwtSecretEntropyBytes*2, len(cfg.JWTSecret))
	}

	storedSecret, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.TrimSpace(string(storedSecret)) != cfg.JWTSecret {
		t.Fatalf("expected persisted JWT secret to match generated secret")
	}

	nextCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("second LoadConfig() error = %v", err)
	}
	if nextCfg.JWTSecret != cfg.JWTSecret {
		t.Fatalf("expected generated JWT secret to persist across loads")
	}
}

func TestReadJWTSecretFileRejectsNonSecretFilename(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "other_secret")
	if err := os.WriteFile(filename, []byte("secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := readJWTSecretFile(filename); err == nil {
		t.Fatal("expected non-JWT secret filename to be rejected")
	}
}

func TestLoadConfigDefaultsSQLiteDatabaseDSN(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	t.Setenv("JWT_SECRET", "env-secret")
	unsetEnv(t, "DATABASE_DRIVER")
	unsetEnv(t, "DATABASE_DSN")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("Database.Driver = %q, want sqlite", cfg.Database.Driver)
	}
	if want := filepath.Join(dataDir, "data.db"); cfg.Database.DSN != want {
		t.Fatalf("Database.DSN = %q, want %q", cfg.Database.DSN, want)
	}
}

func TestLoadConfigRequiresDSNForExternalDriver(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("JWT_SECRET", "env-secret")
	t.Setenv("DATABASE_DRIVER", "postgres")
	t.Setenv("DATABASE_DSN", "")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected missing external DATABASE_DSN error")
	}
}

func configureSQLiteDatabase(t *testing.T, dataDir string) {
	t.Helper()
	t.Setenv("DATABASE_DRIVER", "sqlite")
	t.Setenv("DATABASE_DSN", filepath.Join(dataDir, "data.db"))
}

func unsetEnv(t *testing.T, name string) {
	t.Helper()
	previous, wasSet := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("Unsetenv(%q) error = %v", name, err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv(name, previous)
			return
		}
		_ = os.Unsetenv(name)
	})
}
