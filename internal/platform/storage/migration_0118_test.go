package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0118_AddsServiceOutagesToAnInstanceWithData upgrades a database that already holds a stack and its
// services: the table appears empty beside them, a stack's outages follow it when it is deleted, a service holds at
// most one open outage while closed ones stack up, and running the script again changes nothing.
func TestMigration0118_AddsServiceOutagesToAnInstanceWithData(t *testing.T) {
	db := migrateBefore(t, "0118")
	_, err := db.Exec(`
INSERT INTO stacks (id, name, slug, machine, strategy, created_at, updated_at)
    VALUES ('st-acme', 'acme-web', 'acme-web', 'box', 'compose', 1, 1);
INSERT INTO services (id, stack_id, name) VALUES ('svc-web', 'st-acme', 'web');`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	var outages, services int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM service_outages`).Scan(&outages))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM services`).Scan(&services))
	assert.Zero(t, outages)
	assert.Equal(t, 1, services, "existing rows are untouched")

	insert := `INSERT INTO service_outages (id, stack_id, service, container, started_at, ended_at) VALUES (?, 'st-acme', 'web', 'acme-web-web-1', 10, ?)`
	_, err = db.Exec(insert, "closed-1", 20)
	require.NoError(t, err)
	_, err = db.Exec(insert, "closed-2", 30)
	require.NoError(t, err)
	_, err = db.Exec(insert, "open-1", nil)
	require.NoError(t, err)
	_, err = db.Exec(insert, "open-2", nil)
	assert.Error(t, err, "a service never has two open outages")

	_, err = db.Exec(`DELETE FROM stacks WHERE id = 'st-acme'`)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM service_outages`).Scan(&outages))
	assert.Zero(t, outages, "a deleted stack takes its outages with it")

	script, err := migrationFS.ReadFile("migrations/0118_service_outages.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(script))
	require.NoError(t, err, "a second run is a no-op")
}
