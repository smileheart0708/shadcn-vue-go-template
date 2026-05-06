package database

import (
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
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

var errMigrationNotApplied = errors.New("db: migration not applied")

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
		applied, migrationErr := getAppliedMigration(ctx, db, migration.Version)
		if migrationErr == nil {
			if reconcileErr := reconcileAppliedMigration(ctx, db, migration, applied); reconcileErr != nil {
				return reconcileErr
			}
			continue
		}
		if !errors.Is(migrationErr, errMigrationNotApplied) {
			return migrationErr
		}

		if applyErr := applySQLMigration(ctx, db, migration); applyErr != nil {
			return applyErr
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
		version, parseErr := migrationVersionFromFilename(name)
		if parseErr != nil {
			return nil, parseErr
		}

		if existingName, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("db: duplicate migration version %d in %s and %s", version, existingName, name)
		}

		content, readErr := embeddedMigrations.ReadFile(path.Join("migrations", name))
		if readErr != nil {
			return nil, fmt.Errorf("db: failed to read embedded migration %s: %w", name, readErr)
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
	if err != nil || hasChecksum {
		return err
	}

	if _, alterErr := db.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN checksum TEXT NULL`); alterErr != nil {
		return handleChecksumColumnAddError(ctx, db, alterErr)
	}
	return nil
}

func handleChecksumColumnAddError(ctx context.Context, db *sql.DB, err error) error {
	if !isSQLiteDuplicateColumnError(err) {
		return fmt.Errorf("db: failed to add checksum column to schema_migrations: %w", err)
	}

	hasChecksum, checkErr := schemaMigrationsHasColumn(ctx, db, "checksum")
	if checkErr != nil {
		return checkErr
	}
	if !hasChecksum {
		return fmt.Errorf("db: checksum column add reported duplicate but column is still missing: %w", err)
	}
	return nil
}

func schemaMigrationsHasColumn(ctx context.Context, db *sql.DB, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(schema_migrations)`)
	if err != nil {
		return false, fmt.Errorf("db: failed to inspect schema_migrations table: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.WarnContext(ctx, "failed to close schema metadata rows", "error", closeErr)
		}
	}()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if scanErr := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); scanErr != nil {
			return false, fmt.Errorf("db: failed to scan schema_migrations metadata: %w", scanErr)
		}
		if name == column {
			return true, nil
		}
	}
	if iterErr := rows.Err(); iterErr != nil {
		return false, fmt.Errorf("db: failed to iterate schema_migrations metadata: %w", iterErr)
	}

	return false, nil
}

func getAppliedMigration(ctx context.Context, db *sql.DB, version int64) (appliedMigration, error) {
	var applied appliedMigration
	err := db.QueryRowContext(ctx, `SELECT name, checksum, applied_at FROM schema_migrations WHERE version = ?`, version).Scan(&applied.Name, &applied.Checksum, &applied.AppliedAt)
	if err == nil {
		return applied, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return appliedMigration{}, errMigrationNotApplied
	}

	return appliedMigration{}, fmt.Errorf("db: failed to query schema_migrations for version %d: %w", version, err)
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

	if _, applyErr := tx.ExecContext(ctx, migration.SQL); applyErr != nil {
		return rollbackMigrationTx(tx, fmt.Errorf("db: failed to apply migration %d (%s): %w", migration.Version, migration.Name, applyErr))
	}

	if recordErr := recordAppliedMigration(ctx, tx, db, migration); recordErr != nil {
		return recordErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("db: failed to commit migration %d (%s): %w", migration.Version, migration.Name, commitErr)
	}

	return nil
}

func recordAppliedMigration(ctx context.Context, tx *sql.Tx, db *sql.DB, migration sqlMigration) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO schema_migrations(version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`,
		migration.Version,
		migration.Name,
		migration.Checksum,
		time.Now().Unix(),
	)
	if err == nil {
		return nil
	}

	recordErr := fmt.Errorf("db: failed to record migration %d (%s): %w", migration.Version, migration.Name, err)
	if !isSQLiteSchemaMigrationVersionConflict(err) {
		return rollbackMigrationTx(tx, recordErr)
	}

	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		return errors.Join(recordErr, fmt.Errorf("db: rollback migration transaction: %w", rollbackErr))
	}
	return reconcileConcurrentAppliedMigration(ctx, db, migration)
}

func reconcileConcurrentAppliedMigration(ctx context.Context, db *sql.DB, migration sqlMigration) error {
	applied, err := getAppliedMigration(ctx, db, migration.Version)
	if errors.Is(err, errMigrationNotApplied) {
		return fmt.Errorf("db: migration %d (%s) was concurrently recorded but cannot be reloaded", migration.Version, migration.Name)
	}
	if err != nil {
		return err
	}
	return reconcileAppliedMigration(ctx, db, migration, applied)
}

func rollbackMigrationTx(tx *sql.Tx, primaryErr error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		return errors.Join(primaryErr, fmt.Errorf("db: rollback migration transaction: %w", rollbackErr))
	}
	return primaryErr
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
