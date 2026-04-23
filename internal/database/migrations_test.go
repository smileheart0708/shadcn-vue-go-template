package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const (
	baselineMigrationVersion = 1
	baselineMigrationName    = "0001_baseline.sql"
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
		"avatar_path",
		"status",
		"security_version",
		"disabled_at",
		"created_at",
		"updated_at",
		"role",
	})
	assertTableColumns(t, db, "credentials", []string{
		"user_id",
		"password_hash",
		"password_changed_at",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, db, "auth_sessions", []string{
		"id",
		"user_id",
		"refresh_token_hash",
		"created_at",
		"last_used_at",
		"last_rotated_at",
		"expires_at",
		"idle_expires_at",
		"revoked_at",
		"revoke_reason",
		"ip",
		"user_agent",
	})
	assertTableColumns(t, db, "audit_logs", []string{
		"id",
		"actor_user_id",
		"subject_user_id",
		"auth_session_id",
		"event_type",
		"outcome",
		"reason",
		"ip",
		"user_agent",
		"metadata_json",
		"occurred_at",
	})
	assertTableColumns(t, db, "install_state", []string{
		"id",
		"setup_state",
		"owner_user_id",
		"setup_completed_at",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, db, "account_policies", []string{
		"id",
		"public_registration_enabled",
		"self_service_account_deletion_enabled",
	})
	assertTableMissing(t, db, "user_roles")
	assertTableMissing(t, db, "roles")
	assertTableMissing(t, db, "permissions")
	assertTableMissing(t, db, "role_permissions")
	assertTableMissing(t, db, "system_settings")

	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("expected schema application to be idempotent: %v", err)
	}

	assertAppliedMigration(t, db, baselineMigrationVersion, baselineMigrationName)
	assertAppliedMigrationCount(t, db, 1)
	assertIndexExists(t, db, "users_owner_unique_idx")
	assertDefaultInstallState(t, db)
	assertDefaultAccountPolicies(t, db)
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

func TestRunMigrationsBackfillsChecksumForLegacySchemaTable(t *testing.T) {
	t.Parallel()

	db, cleanup := openRawSQLiteDB(t)
	defer cleanup()

	if _, err := db.Exec(`
CREATE TABLE schema_migrations (
	version INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at INTEGER NOT NULL
);`); err != nil {
		t.Fatalf("failed to create legacy schema_migrations table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations(version, name, applied_at) VALUES (?, ?, ?)`, baselineMigrationVersion, baselineMigrationName, 1_700_000_000); err != nil {
		t.Fatalf("failed to seed legacy schema_migrations row: %v", err)
	}
	if _, err := db.Exec(string(mustReadEmbeddedMigration(t, baselineMigrationName))); err != nil {
		t.Fatalf("failed to apply baseline schema manually: %v", err)
	}

	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("expected legacy schema_migrations table to be upgraded: %v", err)
	}

	assertAppliedMigration(t, db, baselineMigrationVersion, baselineMigrationName)
	assertAppliedMigrationCount(t, db, 1)
	assertIndexExists(t, db, "users_owner_unique_idx")
	assertDefaultAccountPolicies(t, db)
}

func TestRunMigrationsAllowsConcurrentStartup(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "app.db")
	db1, cleanup1 := openRawSQLiteDBAtPath(t, path)
	defer cleanup1()

	db2, cleanup2 := openRawSQLiteDBAtPath(t, path)
	defer cleanup2()

	start := make(chan struct{})
	errs := make(chan error, 2)

	var wg sync.WaitGroup
	run := func(db *sql.DB) {
		defer wg.Done()
		<-start
		errs <- RunMigrations(context.Background(), db)
	}

	wg.Add(2)
	go run(db1)
	go run(db2)
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("expected concurrent migration startup to succeed, got %v", err)
		}
	}

	assertAppliedMigration(t, db1, baselineMigrationVersion, baselineMigrationName)
	assertAppliedMigrationCount(t, db1, 1)
}

func mustReadEmbeddedMigration(t *testing.T, name string) []byte {
	t.Helper()

	content, err := embeddedMigrations.ReadFile("migrations/" + name)
	if err != nil {
		t.Fatalf("failed to read embedded migration %s: %v", name, err)
	}
	return content
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

func assertAppliedMigrationCount(t *testing.T, db *sql.DB, want int) {
	t.Helper()

	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&got); err != nil {
		t.Fatalf("failed to count schema_migrations rows: %v", err)
	}
	if got != want {
		t.Fatalf("expected %d applied migrations, got %d", want, got)
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

func assertDefaultInstallState(t *testing.T, db *sql.DB) {
	t.Helper()

	var setupState string
	if err := db.QueryRow(`SELECT setup_state FROM install_state WHERE id = 1`).Scan(&setupState); err != nil {
		t.Fatalf("failed to load install state: %v", err)
	}
	if setupState != "pending" {
		t.Fatalf("expected default setup_state pending, got %q", setupState)
	}
}

func assertDefaultAccountPolicies(t *testing.T, db *sql.DB) {
	t.Helper()

	var (
		publicRegistrationEnabled         int
		selfServiceAccountDeletionEnabled int
	)
	if err := db.QueryRow(
		`SELECT public_registration_enabled, self_service_account_deletion_enabled
		FROM account_policies
		WHERE id = 1`,
	).Scan(
		&publicRegistrationEnabled,
		&selfServiceAccountDeletionEnabled,
	); err != nil {
		t.Fatalf("failed to load account policies: %v", err)
	}

	if publicRegistrationEnabled != 0 || selfServiceAccountDeletionEnabled != 0 {
		t.Fatalf(
			"expected default account policies to be disabled, got public_registration_enabled=%d self_service_account_deletion_enabled=%d",
			publicRegistrationEnabled,
			selfServiceAccountDeletionEnabled,
		)
	}
}

func assertTableMissing(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		t.Fatalf("failed to inspect table %s: %v", table, err)
	}
	if count != 0 {
		t.Fatalf("expected table %s to be removed, got count=%d", table, count)
	}
}

func openRawSQLiteDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	return openRawSQLiteDBAtPath(t, filepath.Join(t.TempDir(), "app.db"))
}

func openRawSQLiteDBAtPath(t *testing.T, path string) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", sqliteDSN(Options{Path: path}))
	if err != nil {
		t.Fatalf("failed to open raw sqlite database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Fatalf("failed to ping raw sqlite database: %v", err)
	}

	return db, func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close raw sqlite database: %v", err)
		}
	}
}
