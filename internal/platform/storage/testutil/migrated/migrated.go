// Package migrated hands tests a copy of one fully migrated database, so a test skips replaying every migration.
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

// image is a migrated database seeded with the general project, built once per test binary.
var image = sync.OnceValues(func() ([]byte, error) {
	dir, err := os.MkdirTemp("", "nexul-migrated-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	db, err := storage.OpenDB(filepath.Join(dir, "source.db"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	if err := storage.Migrate(db); err != nil {
		return nil, err
	}
	if err := testutil.SeedGeneralProject(db); err != nil {
		return nil, err
	}
	out := filepath.Join(dir, "migrated.db")
	if _, err := db.Exec(`VACUUM INTO ?`, out); err != nil {
		return nil, err
	}
	// VACUUM INTO writes a rollback-journal file; the first connections of a copy would race to switch it to WAL.
	copied, err := storage.OpenDB(out)
	if err != nil {
		return nil, err
	}
	if err := copied.Ping(); err != nil {
		return nil, err
	}
	if err := copied.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(out)
})

// Copy writes the migrated database to path.
func Copy(t testing.TB, path string) {
	t.Helper()
	b, err := image()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0o600))
}

// DB opens a private copy of the migrated database, closed when the test ends.
func DB(t testing.TB) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	Copy(t, path)
	db, err := storage.OpenDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}
