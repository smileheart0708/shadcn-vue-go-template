package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

const (
	baselineMigrationVersion = 1
	baselineMigrationName    = "0001_baseline.sql"
)

func TestRunMigrationsAppliesCurrentBaselineSchema(t *testing.T) {
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

	assertTableColumns(t, db, "users", []string{
		"id",
		"username",
		"email",
		"password_hash",
		"avatar_path",
		"role",
		"bootstrap_password_active",
		"auth_version",
		"password_changed_at",
		"created_at",
		"updated_at",
		"is_banned",
		"banned_at",
	})
	assertTableColumns(t, db, "auth_refresh_sessions", []string{
		"id",
		"user_id",
		"token_hash",
		"issued_at",
		"last_used_at",
		"expires_at",
		"idle_expires_at",
		"revoked_at",
		"revoke_reason",
		"ip",
		"user_agent",
	})
	assertTableColumns(t, db, "auth_login_logs", []string{
		"id",
		"user_id",
		"session_id",
		"identifier",
		"ip",
		"user_agent",
		"event_type",
		"success",
		"failure_reason",
		"created_at",
	})
	assertTableColumns(t, db, "system_settings", []string{
		"id",
		"registration_mode",
		"updated_at",
	})

	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ? AND name = ?`, baselineMigrationVersion, baselineMigrationName).Scan(&migrationCount); err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("expected one schema_migrations row for version %d, got %d", baselineMigrationVersion, migrationCount)
	}

	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("expected baseline schema application to be idempotent: %v", err)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ? AND name = ?`, baselineMigrationVersion, baselineMigrationName).Scan(&migrationCount); err != nil {
		t.Fatalf("failed to query schema_migrations after rerun: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("expected schema_migrations to remain deduplicated after rerun, got %d rows", migrationCount)
	}
}

func assertTableColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()

	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("failed to inspect table %s: %v", table, err)
	}
	defer rows.Close()

	got := make([]string, 0, len(want))
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			t.Fatalf("failed to scan table %s metadata: %v", table, err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate table %s metadata: %v", table, err)
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d columns in %s, got %d (%v)", len(want), table, len(got), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected column %d in %s to be %q, got %q", i, table, want[i], got[i])
		}
	}
}
