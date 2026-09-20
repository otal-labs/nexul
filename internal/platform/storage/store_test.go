package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	templateOnce sync.Once
	templatePath string
	templateErr  error
)

// migratedTemplate migrates one database per test binary; replaying every
// migration per test costs ~2s under -race, which pushed this package past
// Go's 10-minute test timeout in CI.
func migratedTemplate(t *testing.T) string {
	t.Helper()
	templateOnce.Do(func() {
		f, err := os.CreateTemp("", "nexul-storage-template-*.db")
		if err != nil {
			templateErr = err
			return
		}
		templatePath = f.Name()
		if err := f.Close(); err != nil {
			templateErr = err
			return
		}
		db, err := OpenDB(templatePath)
		if err != nil {
			templateErr = err
			return
		}
		defer func() {
			if cerr := db.Close(); cerr != nil {
				templateErr = errors.Join(templateErr, cerr)
			}
		}()
		if err := Migrate(db); err != nil {
			templateErr = err
			return
		}
		// A truncating checkpoint folds the WAL into the main file so the copy below is complete on its own.
		_, templateErr = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	})
	require.NoError(t, templateErr, "build migrated template db")
	return templatePath
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	data, err := os.ReadFile(migratedTemplate(t))
	require.NoError(t, err, "read template db")
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, data, 0o600), "copy template db")
	db, err := OpenDB(path)
	require.NoError(t, err, "OpenDB")
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

// testEncKey is a fixed 32-byte AES key for tests.
var testEncKey = []byte("0123456789abcdef0123456789abcdef")

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return New(newTestDB(t), testEncKey)
}

func TestStore_Close(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	s := New(db, testEncKey)
	require.NoError(t, s.Close())
}

func TestClassifyWriteErr_PassesThroughNonConstraint(t *testing.T) {
	err := classifyWriteErr(context.Canceled)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOpenDB_FileDB_Migrate(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, Migrate(db))

	want := []string{
		"docs", "tickets", "deploys", "topology", "runners",
		"outbox", "processed_events", "dead_letters", "code_reviews",
		"projects", "project_repos",
		"schema_migrations", "docs_fts", "tickets_fts",
	}
	for _, table := range want {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table', 'view') AND name = ?`, table).Scan(&n)
		require.NoError(t, err, "check %s", table)
		assert.Equal(t, 1, n, "table %s exists", table)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	require.NoError(t, Migrate(db))
	require.NoError(t, Migrate(db))

	files, err := fs.Glob(migrationFS, "migrations/*.sql")
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n))
	assert.Equal(t, len(files), n, "each embedded migration recorded exactly once")
}

func TestMigrate_CreatesTheFTSTriggers(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, Migrate(db))

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'docs_fts_ai' AND type = 'trigger'`).Scan(&n))
	assert.Equal(t, 1, n, "FTS trigger exists after ordered migration")
}

func TestMain(m *testing.M) {
	code := m.Run()
	if templatePath != "" {
		if err := os.Remove(templatePath); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "remove template db:", err) // best-effort diagnostic; exit code carries the real result
		}
	}
	os.Exit(code)
}
