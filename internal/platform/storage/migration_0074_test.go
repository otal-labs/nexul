package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0074_TheFirstWorkspaceInheritsTheSharedCanvas upgrades a database holding the shared canvas: the first
// workspace gets a copy, the shared row stays as the service registry, and another workspace gets nothing.
func TestMigration0074_TheFirstWorkspaceInheritsTheSharedCanvas(t *testing.T) {
	db := migrateBefore(t, "0074")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-other', 'Other', 'other', 1, 1);
INSERT INTO topology (environment, schema_version, canvas, updated_at) VALUES ('default', 2, '{"schema_version":2,"nodes":[{"id":"net"}],"edges":[]}', 7);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	canvases := map[string]string{}
	rows, err := db.Query(`SELECT environment, canvas FROM topology`)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	for rows.Next() {
		var env, canvas string
		require.NoError(t, rows.Scan(&env, &canvas))
		canvases[env] = canvas
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]string{
		"default":           `{"schema_version":2,"nodes":[{"id":"net"}],"edges":[]}`,
		"workspace-default": `{"schema_version":2,"nodes":[{"id":"net"}],"edges":[]}`,
	}, canvases)
}

// TestMigration0074_AFreshInstanceHasNoCanvasToCopy runs on a database that never saved a canvas.
func TestMigration0074_AFreshInstanceHasNoCanvasToCopy(t *testing.T) {
	db := migrateBefore(t, "0074")
	require.NoError(t, Migrate(db))
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM topology`).Scan(&n))
	assert.Zero(t, n)
}
