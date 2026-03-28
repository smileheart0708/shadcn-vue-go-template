package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

const (
	baselineMigrationVersion    = 1
	baselineMigrationName       = "0001_baseline.sql"
	authLogsActiveUserIndexName = "auth_refresh_sessions_active_user_idx"
)

func TestRunMigrationsAppliesCurrentSchema(t *testing.T) {
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

	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("expected schema application to be idempotent: %v", err)
	}

	assertAppliedMigration(t, db, baselineMigrationVersion, baselineMigrationName)
	assertForeignKeyCount(t, db, "auth_login_logs", 0)
	assertIndexExists(t, db, authLogsActiveUserIndexName)
}

func TestRunMigrationsRejectsModifiedAppliedMigration(t *testing.T) {
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
	if _, err := db.Exec(`UPDATE schema_migrations SET checksum = 'tampered' WHERE version = ?`, baselineMigrationVersion); err != nil {
		t.Fatalf("failed to tamper with migration checksum: %v", err)
	}

	err = RunMigrations(context.Background(), db)
	if err == nil {
		t.Fatal("expected modified applied migration to be rejected")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
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

func assertAppliedMigration(t *testing.T, db *sql.DB, version int64, name string) {
	t.Helper()

	var (
		gotName  string
		checksum sql.NullString
	)
	if err := db.QueryRow(`SELECT name, checksum FROM schema_migrations WHERE version = ?`, version).Scan(&gotName, &checksum); err != nil {
		t.Fatalf("failed to query schema_migrations version %d: %v", version, err)
	}
	if gotName != name {
		t.Fatalf("expected migration %d to be recorded as %q, got %q", version, name, gotName)
	}
	if !checksum.Valid || checksum.String == "" {
		t.Fatalf("expected migration %d (%s) to have a checksum", version, name)
	}
}

func assertForeignKeyCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()

	rows, err := db.Query(`PRAGMA foreign_key_list(` + table + `)`)
	if err != nil {
		t.Fatalf("failed to inspect foreign keys for %s: %v", table, err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate foreign keys for %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("expected %d foreign keys for %s, got %d", want, table, count)
	}
}

func assertIndexExists(t *testing.T, db *sql.DB, indexName string) {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, indexName).Scan(&count); err != nil {
		t.Fatalf("failed to query sqlite indexes: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected index %q to exist, got count=%d", indexName, count)
	}
}
