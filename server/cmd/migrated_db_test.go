package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
)

// migratedDB is a migrated database seeded with the general project, built once per test binary.
var migratedDB = sync.OnceValues(func() ([]byte, error) {
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

// copyMigratedDB writes migratedDB to path, so a fixture skips parsing every migration again.
func copyMigratedDB(t *testing.T, path string) {
	t.Helper()
	b, err := migratedDB()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0o600))
}
