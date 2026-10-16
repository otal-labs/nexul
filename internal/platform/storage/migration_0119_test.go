package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0119_AddsStackUnwatchersToAnInstanceWithData upgrades a database that already holds a stack: the table
// appears empty, so every existing stack stays watched by everyone who reads it, and a second run changes nothing.
func TestMigration0119_AddsStackUnwatchersToAnInstanceWithData(t *testing.T) {
	db := migrateBefore(t, "0119")
	_, err := db.Exec(`
INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at)
    VALUES ('st-acme', 'acme-web', 'acme-web', 'box', 'compose', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	var optOuts, stacks int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stack_unwatchers`).Scan(&optOuts))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stacks`).Scan(&stacks))
	assert.Zero(t, optOuts)
	assert.Equal(t, 1, stacks, "existing rows are untouched")

	_, err = db.Exec(`INSERT INTO stack_unwatchers (stack_id, user_id, created_at) VALUES ('st-acme', 'alice', 10)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO stack_unwatchers (stack_id, user_id, created_at) VALUES ('st-acme', 'alice', 20)`)
	assert.Error(t, err, "a person opts out of a stack once")

	script, err := migrationFS.ReadFile("migrations/0119_stack_unwatchers.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(script))
	require.NoError(t, err, "a second run is a no-op")
}
