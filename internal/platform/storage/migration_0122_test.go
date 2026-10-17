package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0122_AddsRequestMinutesToAnInstanceWithData upgrades a database that already holds a stack: the table
// appears empty beside it, a minute with no counts stays NULL, a stack's minutes follow it when it is deleted, and
// running the script again changes nothing.
func TestMigration0122_AddsRequestMinutesToAnInstanceWithData(t *testing.T) {
	db := migrateBefore(t, "0122")
	_, err := db.Exec(`INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at)
    VALUES ('st-acme', 'acme-web', 'acme-web', 'box', 'compose', 1, 1)`)
	require.NoError(t, err)

	migrateUpTo(t, db, "0124")

	var minutes, stacks int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stack_request_minutes`).Scan(&minutes))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stacks`).Scan(&stacks))
	assert.Zero(t, minutes)
	assert.Equal(t, 1, stacks, "existing rows are untouched")

	_, err = db.Exec(`INSERT INTO stack_request_minutes (stack_id, minute) VALUES ('st-acme', 60)`)
	require.NoError(t, err)
	var requests *int64
	require.NoError(t, db.QueryRow(`SELECT requests FROM stack_request_minutes`).Scan(&requests))
	assert.Nil(t, requests, "a minute with no counts is NULL, not zero")
	_, err = db.Exec(`INSERT INTO stack_request_minutes (stack_id, minute) VALUES ('st-acme', 60)`)
	assert.Error(t, err, "a stack has one row a minute")

	_, err = db.Exec(`DELETE FROM stacks WHERE id = 'st-acme'`)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM stack_request_minutes`).Scan(&minutes))
	assert.Zero(t, minutes, "a deleted stack takes its minutes with it")

	script, err := migrationFS.ReadFile("migrations/0122_stack_request_minutes.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(script))
	require.NoError(t, err, "a second run is a no-op")
}
