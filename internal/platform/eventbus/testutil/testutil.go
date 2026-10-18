// Package testutil provides shared test helpers for the eventbus packages. It
// is exempt from the coverage gate (see the testing standard).
package testutil

import (
	"io"
	"log/slog"
	"testing"

	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

// NewStore returns a storage.Store over a private copy of the migrated database.
func NewStore(t testing.TB) *storage.Store {
	return storage.New(migrated.DB(t), []byte("0123456789abcdef0123456789abcdef"))
}

// DiscardLogger returns a slog logger that writes nowhere, for tests that
// don't want log noise.
func DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
