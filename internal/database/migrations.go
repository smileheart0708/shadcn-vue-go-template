package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Migration 单个迁移定义。每个迁移应具有唯一递增的 Version。
type Migration struct {
	Version int64
	Up      func(ctx context.Context, tx *sql.Tx) error
}

var migrations = []Migration{
	{
		Version: 1,
		Up: func(ctx context.Context, tx *sql.Tx) error {
			const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL COLLATE NOCASE UNIQUE,
    email TEXT NULL COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    avatar_path TEXT NULL,
    role INTEGER NOT NULL CHECK (role IN (0, 1, 2)),
    bootstrap_password_active INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);`
			if _, err := tx.ExecContext(ctx, createUsersTable); err != nil {
				return fmt.Errorf("db: failed to create users table: %w", err)
			}

			const createUsersEmailIndex = `
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
ON users(email)
WHERE email IS NOT NULL AND email <> '';`
			if _, err := tx.ExecContext(ctx, createUsersEmailIndex); err != nil {
				return fmt.Errorf("db: failed to create users email index: %w", err)
			}

			return nil
		},
	},
	{
		Version: 2,
		Up: func(ctx context.Context, tx *sql.Tx) error {
			const addAuthVersion = `
ALTER TABLE users ADD COLUMN auth_version INTEGER NOT NULL DEFAULT 1;`
			if _, err := tx.ExecContext(ctx, addAuthVersion); err != nil && !isDuplicateColumnError(err) {
				return fmt.Errorf("db: failed to add users.auth_version: %w", err)
			}

			const addPasswordChangedAt = `
ALTER TABLE users ADD COLUMN password_changed_at INTEGER NULL;`
			if _, err := tx.ExecContext(ctx, addPasswordChangedAt); err != nil && !isDuplicateColumnError(err) {
				return fmt.Errorf("db: failed to add users.password_changed_at: %w", err)
			}

			const createRefreshSessionsTable = `
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
);`
			if _, err := tx.ExecContext(ctx, createRefreshSessionsTable); err != nil {
				return fmt.Errorf("db: failed to create auth_refresh_sessions table: %w", err)
			}

			const createRefreshUserIndex = `
CREATE INDEX IF NOT EXISTS auth_refresh_sessions_user_idx
ON auth_refresh_sessions(user_id);`
			if _, err := tx.ExecContext(ctx, createRefreshUserIndex); err != nil {
				return fmt.Errorf("db: failed to create auth_refresh_sessions user index: %w", err)
			}

			const createRefreshRevokedIndex = `
CREATE INDEX IF NOT EXISTS auth_refresh_sessions_revoked_idx
ON auth_refresh_sessions(revoked_at);`
			if _, err := tx.ExecContext(ctx, createRefreshRevokedIndex); err != nil {
				return fmt.Errorf("db: failed to create auth_refresh_sessions revoked index: %w", err)
			}

			const createAuthLoginLogsTable = `
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
);`
			if _, err := tx.ExecContext(ctx, createAuthLoginLogsTable); err != nil {
				return fmt.Errorf("db: failed to create auth_login_logs table: %w", err)
			}

			const createAuthLoginLogsCreatedAtIndex = `
CREATE INDEX IF NOT EXISTS auth_login_logs_created_at_idx
ON auth_login_logs(created_at DESC);`
			if _, err := tx.ExecContext(ctx, createAuthLoginLogsCreatedAtIndex); err != nil {
				return fmt.Errorf("db: failed to create auth_login_logs created_at index: %w", err)
			}

			const createAuthLoginLogsUserIndex = `
CREATE INDEX IF NOT EXISTS auth_login_logs_user_idx
ON auth_login_logs(user_id, created_at DESC);`
			if _, err := tx.ExecContext(ctx, createAuthLoginLogsUserIndex); err != nil {
				return fmt.Errorf("db: failed to create auth_login_logs user index: %w", err)
			}

			return nil
		},
	},
}

// RunMigrations 执行所有未应用的迁移。
// 模板默认 migrations 为空，将直接返回 nil，不会创建任何表。
func RunMigrations(ctx context.Context, db *sql.DB) error {
	ctx = normalizeContext(ctx)

	if db == nil {
		return fmt.Errorf("db: failed to run migrations: nil *sql.DB")
	}
	if len(migrations) == 0 {
		return nil
	}

	if err := validateMigrations(); err != nil {
		return err
	}

	if err := ensureSchemaMigrationsTable(ctx, db); err != nil {
		return err
	}

	applied, err := loadAppliedVersions(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return err
		}
	}

	return nil
}

func validateMigrations() error {
	seen := make(map[int64]struct{}, len(migrations))
	var prev int64
	for i, m := range migrations {
		if m.Version <= 0 {
			return fmt.Errorf("db: invalid migration version %d", m.Version)
		}
		if _, ok := seen[m.Version]; ok {
			return fmt.Errorf("db: duplicate migration version %d", m.Version)
		}
		seen[m.Version] = struct{}{}
		if i > 0 && m.Version <= prev {
			return fmt.Errorf("db: migrations must be strictly increasing: %d then %d", prev, m.Version)
		}
		if m.Up == nil {
			return fmt.Errorf("db: nil migration Up for version %d", m.Version)
		}
		prev = m.Version
	}
	return nil
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

func loadAppliedVersions(ctx context.Context, db *sql.DB) (map[int64]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("db: failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("db: failed to scan schema_migrations: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: failed to iterate schema_migrations: %w", err)
	}

	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: failed to begin migration %d: %w", m.Version, err)
	}

	if err := m.Up(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("db: failed to apply migration %d: %w (rollback failed: %v)", m.Version, err, rbErr)
		}
		return fmt.Errorf("db: failed to apply migration %d: %w", m.Version, err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`,
		m.Version,
		time.Now().Unix(),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("db: failed to record migration %d: %w", m.Version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: failed to commit migration %d: %w", m.Version, err)
	}

	return nil
}

func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	return strings.Contains(message, "duplicate column name: auth_version") || strings.Contains(message, "duplicate column name: password_changed_at")
}
