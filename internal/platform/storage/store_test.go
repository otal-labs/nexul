package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
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
		if err := testutil.SeedGeneralProject(db); err != nil {
			templateErr = err
			return
		}
		// A truncating checkpoint folds the WAL into the main file so the copy below is complete on its own.
		_, templateErr = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	})
	require.NoError(t, templateErr, "build migrated template db")
	return templatePath
}

var (
	snapshotsDir = filepath.Join(os.TempDir(), fmt.Sprintf("nexul-storage-snapshots-%d", os.Getpid()))

	// migrationSnapshots holds the database as it stands before each embedded migration (index i) and after the last
	// (index len), built in one pass so a migration test copies a file instead of replaying its predecessors.
	migrationSnapshots = sync.OnceValues(func() ([]string, error) {
		if err := os.MkdirAll(snapshotsDir, 0o700); err != nil {
			return nil, err
		}
		versions, err := embeddedVersions()
		if err != nil {
			return nil, err
		}
		db, err := OpenDB(filepath.Join(snapshotsDir, "chain.db"))
		if err != nil {
			return nil, err
		}
		defer func() { _ = db.Close() }()
		if err := ensureMigrationsTable(db); err != nil {
			return nil, err
		}
		snapshot := func(i int) (string, error) {
			// A truncating checkpoint folds the WAL into the main file so the copy is complete on its own.
			if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
				return "", err
			}
			data, err := os.ReadFile(filepath.Join(snapshotsDir, "chain.db"))
			if err != nil {
				return "", err
			}
			path := filepath.Join(snapshotsDir, fmt.Sprintf("%d.db", i))
			return path, os.WriteFile(path, data, 0o600)
		}
		paths := make([]string, 0, len(versions)+1)
		for i, v := range versions {
			path, err := snapshot(i)
			if err != nil {
				return nil, err
			}
			paths = append(paths, path)
			script, err := migrationFS.ReadFile("migrations/" + v + ".sql")
			if err != nil {
				return nil, err
			}
			if err := applyMigration(db, v, string(script)); err != nil {
				return nil, err
			}
		}
		path, err := snapshot(len(versions))
		if err != nil {
			return nil, err
		}
		return append(paths, path), nil
	})
)

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
	s := New(db, testEncKey)
	require.NoError(t, s.Close())
}

func TestClassifyWriteErr_PassesThroughNonConstraint(t *testing.T) {
	err := classifyWriteErr(context.Canceled)
	require.ErrorIs(t, err, context.Canceled)
}

func TestMain(m *testing.M) {
	code := m.Run()
	if templatePath != "" {
		if err := os.Remove(templatePath); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "remove template db:", err) // best-effort diagnostic; exit code carries the real result
		}
	}
	_ = os.RemoveAll(snapshotsDir)
	os.Exit(code)
}
