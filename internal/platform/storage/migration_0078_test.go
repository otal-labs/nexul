package storage

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func queryPlan(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	var steps []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
		steps = append(steps, detail)
	}
	require.NoError(t, rows.Err())
	return strings.Join(steps, "; ")
}

// TestMigration0078_TicketListsReadInCreatedOrderWithoutASort upgrades a database with tickets: they come back in
// creation order, and neither ticket list sorts every row in a temporary b-tree any more.
func TestMigration0078_TicketListsReadInCreatedOrderWithoutASort(t *testing.T) {
	db := migrateBefore(t, "0078")
	_, err := db.Exec(`
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO tickets (id, project_id, title, status, created_at, updated_at) VALUES
    ('t-b', 'p-web', 'Second', 'open', 2, 2), ('t-a', 'p-web', 'First', 'open', 1, 1), ('t-c', NULL, 'Third', 'open', 3, 3);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0078 and every later migration apply on top, as an upgrade would")

	var ids string
	require.NoError(t, db.QueryRow(`SELECT group_concat(id) FROM (SELECT id FROM tickets ORDER BY created_at)`).Scan(&ids))
	assert.Equal(t, "t-a,t-b,t-c", ids)
	assert.NotContains(t, queryPlan(t, db, `SELECT * FROM tickets ORDER BY created_at`), "TEMP B-TREE")
	assert.NotContains(t, queryPlan(t, db, `SELECT * FROM tickets WHERE project_id = ? ORDER BY created_at`, "p-web"), "TEMP B-TREE")
}
