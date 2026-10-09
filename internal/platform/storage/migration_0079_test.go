package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0079_AuditPurgeAndListWalkTheCreatedIndex upgrades a database with audit rows: they survive, and
// neither the retention purge nor the newest-first list scans the whole table or sorts it.
func TestMigration0079_AuditPurgeAndListWalkTheCreatedIndex(t *testing.T) {
	db := migrateBefore(t, "0079")
	_, err := db.Exec(`INSERT INTO audit_log (id, actor_type, actor_id, action, created_at) VALUES
    ('a1', 'user', 'u-alice', 'GET /api/tickets', 1), ('a2', 'user', 'u-alice', 'PATCH /api/tickets/t1', 2);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0079 and every later migration apply on top, as an upgrade would")

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&n))
	assert.Equal(t, 2, n)
	purge := queryPlan(t, db, `DELETE FROM audit_log WHERE id IN (SELECT a.id FROM audit_log AS a WHERE a.created_at < ? LIMIT ?)`, 2, 500)
	assert.Contains(t, purge, "idx_audit_created")
	assert.NotContains(t, purge, "SCAN audit_log")
	list := queryPlan(t, db, `SELECT * FROM audit_log ORDER BY created_at DESC, id DESC LIMIT ?`, 50)
	assert.Contains(t, list, "idx_audit_created")
	assert.NotContains(t, list, "TEMP B-TREE")
}
