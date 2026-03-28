package database

import (
	"cmp"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
)

// embeddedMigrations marks the template's consolidated baseline schema files.
//
// This repository is intentionally treated as a starter template, so it does
// not preserve historical incremental migrations yet. New databases are
// created directly from the embedded baseline SQL.
//
// Do not copy the current "rewrite baseline files freely" workflow into a
// production project that already has real environments or persistent user
// data. Production services should keep an append-only migration history:
//   - freeze the initial baseline as version 1
//   - add version 2, 3, 4... as forward-only migrations
//   - never rewrite a migration that may already have been applied
//
// If this template becomes the foundation of a real product, keep the embed
// loading mechanism but switch the workflow to append-only SQL files.
//
//go:embed migrations/*.sql
var embeddedMigrations embed.FS

type sqlMigration struct {
	Version int64
	Name    string
	SQL     string
}

// RunMigrations initializes the embedded template schema.
//
// This keeps SQL and Go separated by loading `.sql` files from the embedded
// migrations directory. The current repository still treats those files as a
// template baseline, not as immutable production history.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	ctx = normalizeContext(ctx)

	if db == nil {
		return fmt.Errorf("db: failed to run migrations: nil *sql.DB")
	}

	if err := ensureSchemaMigrationsTable(ctx, db); err != nil {
		return err
	}

	migrations, err := loadSQLMigrations()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		applied, err := hasAppliedSchema(ctx, db, migration.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := applySQLMigration(ctx, db, migration); err != nil {
			return err
		}
	}

	return nil
}

func ensureSchemaMigrationsTable(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at INTEGER NOT NULL
);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("db: failed to create schema_migrations table: %w", err)
	}
	return nil
}

func loadSQLMigrations() ([]sqlMigration, error) {
	entries, err := fs.ReadDir(embeddedMigrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("db: failed to read embedded migrations: %w", err)
	}

	migrations := make([]sqlMigration, 0, len(entries))
	seenVersions := make(map[int64]string, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		version, err := migrationVersionFromFilename(name)
		if err != nil {
			return nil, err
		}

		if existingName, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("db: duplicate migration version %d in %s and %s", version, existingName, name)
		}

		content, err := embeddedMigrations.ReadFile(path.Join("migrations", name))
		if err != nil {
			return nil, fmt.Errorf("db: failed to read embedded migration %s: %w", name, err)
		}

		sqlText := strings.TrimSpace(string(content))
		if sqlText == "" {
			return nil, fmt.Errorf("db: embedded migration %s is empty", name)
		}

		migrations = append(migrations, sqlMigration{
			Version: version,
			Name:    name,
			SQL:     sqlText,
		})
		seenVersions[version] = name
	}

	slices.SortFunc(migrations, func(a, b sqlMigration) int {
		return cmp.Compare(a.Version, b.Version)
	})

	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].Version >= migrations[i].Version {
			return nil, fmt.Errorf("db: migrations must be strictly increasing: %d then %d", migrations[i-1].Version, migrations[i].Version)
		}
	}

	return migrations, nil
}

func migrationVersionFromFilename(name string) (int64, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found {
		return 0, fmt.Errorf("db: invalid migration filename %q: expected <version>_<name>.sql", name)
	}
	if !strings.HasSuffix(name, ".sql") {
		return 0, fmt.Errorf("db: invalid migration filename %q: expected .sql suffix", name)
	}

	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("db: invalid migration filename %q: version must be a positive integer", name)
	}

	return version, nil
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

func applySQLMigration(ctx context.Context, db *sql.DB, migration sqlMigration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: failed to begin migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("db: failed to apply migration %d (%s): %w (rollback failed: %v)", migration.Version, migration.Name, err, rbErr)
		}
		return fmt.Errorf("db: failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO schema_migrations(version, name, applied_at) VALUES (?, ?, ?)`,
		migration.Version,
		migration.Name,
		time.Now().Unix(),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("db: failed to record migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: failed to commit migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	return nil
}
