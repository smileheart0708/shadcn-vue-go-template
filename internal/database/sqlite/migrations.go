package sqlite

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
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

type sqlMigration struct {
	Version  int64
	Name     string
	SQL      string
	Checksum string
}

type appliedMigration struct {
	Name     string
	Checksum string
}

var errMigrationNotApplied = errors.New("database: SQLite migration not applied")

func migrate(ctx context.Context, db *sql.DB) error {
	ctx = normalizeContext(ctx)
	if db == nil {
		return errors.New("database: migrate SQLite with nil database")
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
		if err == nil {
			if err := reconcileAppliedMigration(migration, applied); err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, errMigrationNotApplied) {
			return err
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
    checksum TEXT NOT NULL,
    applied_at INTEGER NOT NULL
);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("database: create SQLite schema_migrations table: %w", err)
	}
	return nil
}

func loadSQLMigrations() ([]sqlMigration, error) {
	entries, err := fs.ReadDir(embeddedMigrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("database: read SQLite migrations: %w", err)
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
		if existing, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("database: duplicate SQLite migration version %d in %s and %s", version, existing, name)
		}
		content, err := embeddedMigrations.ReadFile(path.Join("migrations", name))
		if err != nil {
			return nil, fmt.Errorf("database: read SQLite migration %s: %w", name, err)
		}
		if strings.TrimSpace(string(content)) == "" {
			return nil, fmt.Errorf("database: SQLite migration %s is empty", name)
		}
		checksum := sha256.Sum256(content)
		migrations = append(migrations, sqlMigration{
			Version:  version,
			Name:     name,
			SQL:      string(content),
			Checksum: hex.EncodeToString(checksum[:]),
		})
		seenVersions[version] = name
	}

	slices.SortFunc(migrations, func(a, b sqlMigration) int {
		return cmp.Compare(a.Version, b.Version)
	})
	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].Version >= migrations[i].Version {
			return nil, fmt.Errorf("database: SQLite migrations must be strictly increasing: %d then %d", migrations[i-1].Version, migrations[i].Version)
		}
	}
	return migrations, nil
}

func migrationVersionFromFilename(name string) (int64, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found || !strings.HasSuffix(name, ".sql") {
		return 0, fmt.Errorf("database: invalid SQLite migration filename %q: expected <version>_<name>.sql", name)
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("database: invalid SQLite migration filename %q: version must be a positive integer", name)
	}
	return version, nil
}

func getAppliedMigration(ctx context.Context, db *sql.DB, version int64) (appliedMigration, error) {
	var applied appliedMigration
	err := db.QueryRowContext(ctx, `SELECT name, checksum FROM schema_migrations WHERE version = ?`, version).Scan(&applied.Name, &applied.Checksum)
	if err == nil {
		return applied, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return appliedMigration{}, errMigrationNotApplied
	}
	return appliedMigration{}, fmt.Errorf("database: query SQLite migration %d: %w", version, err)
}

func reconcileAppliedMigration(migration sqlMigration, applied appliedMigration) error {
	if applied.Name != migration.Name {
		return fmt.Errorf("database: SQLite migration %d was recorded as %q but is now %q; published migrations must be immutable", migration.Version, applied.Name, migration.Name)
	}
	if applied.Checksum != migration.Checksum {
		return fmt.Errorf("database: SQLite migration %d (%s) checksum mismatch; published migrations must be immutable", migration.Version, migration.Name)
	}
	return nil
}

func applySQLMigration(ctx context.Context, db *sql.DB, migration sqlMigration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database: begin SQLite migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return rollbackMigrationTx(tx, fmt.Errorf("database: apply SQLite migration %d (%s): %w", migration.Version, migration.Name, err))
	}
	shouldCommit, err := recordAppliedMigration(ctx, tx, db, migration)
	if err != nil {
		return err
	}
	if !shouldCommit {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("database: commit SQLite migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	return nil
}

func recordAppliedMigration(ctx context.Context, tx *sql.Tx, db *sql.DB, migration sqlMigration) (bool, error) {
	_, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`, migration.Version, migration.Name, migration.Checksum, time.Now().Unix())
	if err == nil {
		return true, nil
	}

	recordErr := fmt.Errorf("database: record SQLite migration %d (%s): %w", migration.Version, migration.Name, err)
	if !isSQLiteMigrationVersionConflict(err) {
		return false, rollbackMigrationTx(tx, recordErr)
	}
	// A concurrent opener committed the same migration. This transaction was
	// rolled back, so the caller must not attempt a second commit.
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		return false, errors.Join(recordErr, fmt.Errorf("database: rollback SQLite migration transaction: %w", rollbackErr))
	}
	applied, reloadErr := getAppliedMigration(ctx, db, migration.Version)
	if errors.Is(reloadErr, errMigrationNotApplied) {
		return false, fmt.Errorf("database: SQLite migration %d (%s) was concurrently recorded but cannot be reloaded", migration.Version, migration.Name)
	}
	if reloadErr != nil {
		return false, reloadErr
	}
	return false, reconcileAppliedMigration(migration, applied)
}

func rollbackMigrationTx(tx *sql.Tx, primaryErr error) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		return errors.Join(primaryErr, fmt.Errorf("database: rollback SQLite migration transaction: %w", rollbackErr))
	}
	return primaryErr
}

func isSQLiteMigrationVersionConflict(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint failed: schema_migrations.version")
}
