package database

import (
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
)

// embeddedMigrations contains the template's baseline schema files.
//
// The template itself ships a single defined baseline schema. Downstream
// projects that evolve from this template can append newer versioned files and
// keep the same runner, but this repository intentionally stays on one
// baseline file.
//
//go:embed migrations/*.sql
var embeddedMigrations embed.FS

type sqlMigration struct {
	Version  int64
	Name     string
	SQL      string
	Checksum string
}

type appliedMigration struct {
	Name      string
	Checksum  sql.NullString
	AppliedAt int64
}

// RunMigrations applies embedded SQL migrations in version order.
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
		applied, err := getAppliedMigration(ctx, db, migration.Version)
		if err != nil {
			return err
		}
		if applied != nil {
			if err := reconcileAppliedMigration(ctx, db, migration, *applied); err != nil {
				return err
			}
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
    checksum TEXT NULL,
    applied_at INTEGER NOT NULL
);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("db: failed to create schema_migrations table: %w", err)
	}
	if err := ensureSchemaMigrationsChecksumColumn(ctx, db); err != nil {
		return err
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

		checksumBytes := sha256.Sum256(content)
		migrations = append(migrations, sqlMigration{
			Version:  version,
			Name:     name,
			SQL:      string(content),
			Checksum: hex.EncodeToString(checksumBytes[:]),
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

func ensureSchemaMigrationsChecksumColumn(ctx context.Context, db *sql.DB) error {
	hasChecksum, err := schemaMigrationsHasColumn(ctx, db, "checksum")
	if err != nil {
		return err
	}
	if hasChecksum {
		return nil
	}

	if _, err := db.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN checksum TEXT NULL`); err != nil {
		if isSQLiteDuplicateColumnError(err) {
			hasChecksum, checkErr := schemaMigrationsHasColumn(ctx, db, "checksum")
			if checkErr != nil {
				return checkErr
			}
			if hasChecksum {
				return nil
			}
		}
		return fmt.Errorf("db: failed to add checksum column to schema_migrations: %w", err)
	}
	return nil
}

func schemaMigrationsHasColumn(ctx context.Context, db *sql.DB, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(schema_migrations)`)
	if err != nil {
		return false, fmt.Errorf("db: failed to inspect schema_migrations table: %w", err)
	}
	defer rows.Close()

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
			return false, fmt.Errorf("db: failed to scan schema_migrations metadata: %w", err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("db: failed to iterate schema_migrations metadata: %w", err)
	}

	return false, nil
}

func getAppliedMigration(ctx context.Context, db *sql.DB, version int64) (*appliedMigration, error) {
	var applied appliedMigration
	err := db.QueryRowContext(ctx, `SELECT name, checksum, applied_at FROM schema_migrations WHERE version = ?`, version).Scan(&applied.Name, &applied.Checksum, &applied.AppliedAt)
	if err == nil {
		return &applied, nil
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return nil, fmt.Errorf("db: failed to query schema_migrations for version %d: %w", version, err)
}

func reconcileAppliedMigration(ctx context.Context, db *sql.DB, migration sqlMigration, applied appliedMigration) error {
	if applied.Name != migration.Name {
		return fmt.Errorf(
			"db: migration %d was previously recorded as %q but the current file is %q; published migrations must be immutable",
			migration.Version,
			applied.Name,
			migration.Name,
		)
	}

	if strings.TrimSpace(applied.Checksum.String) == "" {
		if _, err := db.ExecContext(
			ctx,
			`UPDATE schema_migrations SET checksum = ? WHERE version = ? AND (checksum IS NULL OR checksum = '')`,
			migration.Checksum,
			migration.Version,
		); err != nil {
			return fmt.Errorf("db: failed to backfill checksum for migration %d (%s): %w", migration.Version, migration.Name, err)
		}
		return nil
	}

	if applied.Checksum.String != migration.Checksum {
		return fmt.Errorf(
			"db: migration %d (%s) checksum mismatch; published migrations must be immutable",
			migration.Version,
			migration.Name,
		)
	}

	return nil
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
		`INSERT INTO schema_migrations(version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`,
		migration.Version,
		migration.Name,
		migration.Checksum,
		time.Now().Unix(),
	); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("db: failed to record migration %d (%s): %w (rollback failed: %v)", migration.Version, migration.Name, err, rbErr)
		}
		if isSQLiteSchemaMigrationVersionConflict(err) {
			applied, getErr := getAppliedMigration(ctx, db, migration.Version)
			if getErr != nil {
				return getErr
			}
			if applied == nil {
				return fmt.Errorf("db: migration %d (%s) was concurrently recorded but cannot be reloaded", migration.Version, migration.Name)
			}
			return reconcileAppliedMigration(ctx, db, migration, *applied)
		}
		return fmt.Errorf("db: failed to record migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: failed to commit migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	return nil
}

func isSQLiteDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column name")
}

func isSQLiteSchemaMigrationVersionConflict(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed: schema_migrations.version")
}
