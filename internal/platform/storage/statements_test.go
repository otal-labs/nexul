package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"

	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
)

// newCountedStore is newTestStore over a driver connection that records every statement.
func newCountedStore(t *testing.T) (*Store, *testutil.Statements) {
	t.Helper()
	data, err := os.ReadFile(migratedTemplate(t))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	inner, err := sqlite.NewConnector(path + "?" + pragmas)
	require.NoError(t, err)
	db, st := testutil.OpenCounted(inner)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return New(db, testEncKey), st
}
