package storage

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate applies every pending migration. Test helpers use this no-backup path directly; production boots
// through MigrateWithBackup instead.
func Migrate(db *sql.DB) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	pending, err := Pending(db)
	if err != nil {
		return err
	}
	return applyPending(db, pending)
}

// MigrateWithBackup refuses to run against a schema newer than this binary (see checkNotNewer), then — only when
// a migration is actually pending — snapshots db into dir before applying anything, keeping the newest 5
// snapshots.
func MigrateWithBackup(db *sql.DB, dir, version string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	if err := checkNotNewer(db, dir, version); err != nil {
		return err
	}
	pending, err := Pending(db)
	if err != nil {
		return err
	}
	if len(pending) > 0 {
		if _, err := Backup(db, dir, version); err != nil {
			return fmt.Errorf("backup before migrate: %w", err)
		}
		if err := PruneBackups(dir, 5); err != nil {
			return fmt.Errorf("prune backups: %w", err)
		}
	}
	return applyPending(db, pending)
}

// Pending returns the embedded migration versions not yet recorded in schema_migrations, in apply order.
func Pending(db *sql.DB) ([]string, error) {
	if err := ensureMigrationsTable(db); err != nil {
		return nil, err
	}
	versions, err := embeddedVersions()
	if err != nil {
		return nil, err
	}
	var pending []string
	for _, version := range versions {
		applied, err := migrationApplied(db, version)
		if err != nil {
			return nil, err
		}
		if !applied {
			pending = append(pending, version)
		}
	}
	return pending, nil
}

func applyPending(db *sql.DB, pending []string) error {
	for _, version := range pending {
		script, err := migrationFS.ReadFile("migrations/" + version + ".sql")
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		if err := applyMigration(db, version, string(script)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
	}
	return nil
}

// checkNotNewer refuses to proceed when schema_migrations names a version this binary's embedded set does not
// know: an older binary against a newer database, guarded rather than let loose on a schema it can't understand.
func checkNotNewer(db *sql.DB, backupDir, binaryVersion string) (err error) {
	versions, err := embeddedVersions()
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(versions))
	for _, v := range versions {
		known[v] = true
	}

	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("read schema_migrations: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return fmt.Errorf("scan schema_migrations: %w", err)
		}
		if known[v] {
			continue
		}
		return newerSchemaError(v, backupDir, binaryVersion)
	}
	return rows.Err()
}

func newerSchemaError(schemaVersion, backupDir, binaryVersion string) error {
	restore := "the newest snapshot in " + backupDir
	if path := newestBackupPath(backupDir); path != "" {
		restore = path
	}
	return fmt.Errorf("database schema %s is newer than this binary (%s); restore %s or upgrade the binary", schemaVersion, binaryVersion, restore)
}

func ensureMigrationsTable(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version     TEXT PRIMARY KEY,
		applied_at  INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func embeddedVersions() ([]string, error) {
	files, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)
	versions := make([]string, len(files))
	for i, file := range files {
		versions[i] = strings.TrimSuffix(strings.TrimPrefix(file, "migrations/"), ".sql")
	}
	return versions, nil
}

func migrationApplied(db *sql.DB, version string) (bool, error) {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&n); err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return n > 0, nil
}

func applyMigration(db *sql.DB, version, script string) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	defer func() {
		// Rollback after a successful Commit is a harmless no-op that returns sql.ErrTxDone.
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			err = errors.Join(err, rerr)
		}
	}()

	if _, err := tx.Exec(script); err != nil {
		return fmt.Errorf("exec migration %s: %w", version, err)
	}
	// hand-written: schema_migrations is created here, outside the migrations sqlc reads, so it has no generated query
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, version, time.Now().Unix()); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}
