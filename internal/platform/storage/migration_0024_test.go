package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0024_SwapsOldToolNamesInEditedPlays(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at)
		VALUES ('p-edited', 'workspace-default', 'My interview', 'interview', '', ?, 1, NULL, '[]', '', 0, 0)`,
		"Open it with `memory_create_interview`, then call `ticket_test_pass`; keep ticket_search plain.")
	require.NoError(t, err)

	migration, err := migrationFS.ReadFile("migrations/0024_consolidated_mcp_tool_names.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)

	var got string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT instructions FROM plays WHERE id = 'p-edited'`).Scan(&got))
	assert.Equal(t, "Open it with `memory_create` with `kind` `interview`, then call `ticket_test_report` with `outcome` `pass`; keep ticket_search plain.", got,
		"a merged tool keeps the argument that made the old one distinct, and only backticked names are swapped")
}
