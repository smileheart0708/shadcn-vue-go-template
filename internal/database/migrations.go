package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Migration 单个迁移定义。每个迁移应具有唯一递增的 Version。
type Migration struct {
	Version int64
	Up      func(ctx context.Context, tx *sql.Tx) error
}

// migrations 模板项目默认不包含业务表迁移。
// 使用模板时按需在此追加迁移，并确保 Version 递增且唯一。
var migrations = []Migration{}

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
