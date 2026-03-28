package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const currentSchemaVersion int64 = 1

var schemaStatements = []string{
	`
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL COLLATE NOCASE UNIQUE,
    email TEXT NULL COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    avatar_path TEXT NULL,
    role INTEGER NOT NULL CHECK (role IN (0, 1, 2)),
    bootstrap_password_active INTEGER NOT NULL DEFAULT 0,
    auth_version INTEGER NOT NULL DEFAULT 1,
    password_changed_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);`,
	`
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
ON users(email)
WHERE email IS NOT NULL AND email <> '';`,
	`
CREATE TABLE IF NOT EXISTS auth_refresh_sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL,
    issued_at INTEGER NOT NULL,
    last_used_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    idle_expires_at INTEGER NOT NULL,
    revoked_at INTEGER NULL,
    revoke_reason TEXT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);`,
	`
CREATE INDEX IF NOT EXISTS auth_refresh_sessions_user_idx
ON auth_refresh_sessions(user_id);`,
	`
CREATE INDEX IF NOT EXISTS auth_refresh_sessions_revoked_idx
ON auth_refresh_sessions(revoked_at);`,
	`
CREATE TABLE IF NOT EXISTS auth_login_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NULL,
    session_id TEXT NULL,
    identifier TEXT NULL,
    ip TEXT NULL,
    user_agent TEXT NULL,
    event_type TEXT NOT NULL,
    success INTEGER NOT NULL,
    failure_reason TEXT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (session_id) REFERENCES auth_refresh_sessions(id) ON DELETE SET NULL
);`,
	`
CREATE INDEX IF NOT EXISTS auth_login_logs_created_at_idx
ON auth_login_logs(created_at DESC);`,
	`
CREATE INDEX IF NOT EXISTS auth_login_logs_user_idx
ON auth_login_logs(user_id, created_at DESC);`,
}

// RunMigrations 初始化当前 schema 基线。
// 该项目不保留历史增量迁移，新的数据库直接从 currentSchemaVersion 开始。
func RunMigrations(ctx context.Context, db *sql.DB) error {
	ctx = normalizeContext(ctx)

	if db == nil {
		return fmt.Errorf("db: failed to run migrations: nil *sql.DB")
	}

	if err := ensureSchemaMigrationsTable(ctx, db); err != nil {
		return err
	}

	applied, err := hasAppliedSchema(ctx, db, currentSchemaVersion)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	return applyCurrentSchema(ctx, db)
}

func ensureSchemaMigrationsTable(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("db: failed to create schema_migrations table: %w", err)
	}
	return nil
}

func hasAppliedSchema(ctx context.Context, db *sql.DB, version int64) (bool, error) {
	var appliedAt int64
	err := db.QueryRowContext(ctx, `SELECT applied_at FROM schema_migrations WHERE version = ?`, version).Scan(&appliedAt)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}

	return false, fmt.Errorf("db: failed to query schema_migrations for version %d: %w", version, err)
}

func applyCurrentSchema(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: failed to begin schema setup: %w", err)
	}

	for _, statement := range schemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("db: failed to apply schema statement: %w (rollback failed: %v)", err, rbErr)
			}
			return fmt.Errorf("db: failed to apply schema statement: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`,
		currentSchemaVersion,
		time.Now().Unix(),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("db: failed to record schema version %d: %w", currentSchemaVersion, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: failed to commit schema version %d: %w", currentSchemaVersion, err)
	}

	return nil
}
