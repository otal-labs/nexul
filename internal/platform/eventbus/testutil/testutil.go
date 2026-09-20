// Package testutil provides shared test helpers for the eventbus packages. It
// is exempt from the coverage gate (see the testing standard).
package testutil

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/otal-labs/nexul/internal/platform/storage"
)

// NewDB opens an isolated on-disk SQLite DB in a per-test temp dir, closed via t.Cleanup.
func NewDB(t testing.TB) *sql.DB {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// testEncKey is a fixed 32-byte AES key for tests.
var testEncKey = []byte("0123456789abcdef0123456789abcdef")

// NewStore returns a storage.Store over a fresh migrated DB.
func NewStore(t testing.TB) *storage.Store {
	db := NewDB(t)
	if err := storage.Migrate(db); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return storage.New(db, testEncKey)
}

// DiscardLogger returns a slog logger that writes nowhere, for tests that
// don't want log noise.
func DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
