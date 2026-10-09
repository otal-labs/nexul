package storage

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDB_BurstOfQueries_ReusesEveryConnection(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ctx := t.Context()

	conns := make([]*sql.Conn, 8)
	for i := range conns {
		conns[i], err = db.Conn(ctx)
		require.NoError(t, err)
	}
	for _, c := range conns {
		require.NoError(t, c.Close())
	}

	stats := db.Stats()
	assert.Zero(t, stats.MaxIdleClosed, "a connection released after a burst was closed and must be reopened next time")
	assert.Equal(t, 8, stats.Idle)
}
