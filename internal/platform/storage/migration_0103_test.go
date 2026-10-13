package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0103_ASetupMarkerKeepsItsStateWithNoFinisher: an upgraded workspace keeps whether its setup is done,
// and names nobody as having finished it.
func TestMigration0103_ASetupMarkerKeepsItsStateWithNoFinisher(t *testing.T) {
	db := migrateBefore(t, "0103")
	_, err := db.Exec(`INSERT INTO workspaces (id, name, slug, created_at, updated_at, access_setup_finished) VALUES ('ws-waiting', 'Acme', 'acme', 1, 1, 0)`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	assert.Equal(t, []string{"workspace-default|1||0", "ws-waiting|0||0"},
		lines(t, db, `SELECT id || '|' || access_setup_finished || '|' || access_setup_by || '|' || access_setup_at FROM workspaces ORDER BY 1`))
}
