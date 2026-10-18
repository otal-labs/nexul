// Package migrated hands tests a real SQLite database that is already migrated, copied from one template built per
// test binary, because replaying every migration per test costs seconds under -race.
package migrated

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
)

const key = "0123456789abcdef0123456789abcdef"

var (
	plain  = sync.OnceValues(func() ([]byte, error) { return build(false) })
	seeded = sync.OnceValues(func() ([]byte, error) { return build(true) })
)

func build(seedProject bool) ([]byte, error) {
	dir, err := os.MkdirTemp("", "nexul-migrated-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "template.db")
	db, err := storage.OpenDB(path)
	if err != nil {
		return nil, err
	}
	if err := storage.Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if seedProject {
		if err := testutil.SeedGeneralProject(db); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	// A truncating checkpoint folds the WAL into the main file, so the bytes are the whole database.
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// DB opens a private copy of the migrated database, closed when the test ends.
func DB(t *testing.T) *sql.DB { return open(t, plain) }

// Store is DB wrapped in the repositories, with the fixed test key.
func Store(t *testing.T) *storage.Store { return storage.New(DB(t), []byte(key)) }

// SeededStore is Store with the general project of testutil.SeedGeneralProject already inserted.
func SeededStore(t *testing.T) *storage.Store { return storage.New(open(t, seeded), []byte(key)) }

func open(t *testing.T, template func() ([]byte, error)) *sql.DB {
	t.Helper()
	data, err := template()
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	db, err := storage.OpenDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}
