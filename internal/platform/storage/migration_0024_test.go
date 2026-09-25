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
		VALUES ('p-edited', 'workspace-default', 'My fix', 'ticket', '', 'Read it with `+"`ticket_get_links`"+`, then call `+"`ticket_test_pass`"+`; keep ticket_search plain.', 1, NULL, '[]', '', 0, 0)`)
	require.NoError(t, err)

	migration, err := migrationFS.ReadFile("migrations/0024_consolidated_mcp_tool_names.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)

	var got string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT instructions FROM plays WHERE id = 'p-edited'`).Scan(&got))
	assert.Equal(t, "Read it with `ticket_get`, then call `ticket_test_report`; keep ticket_search plain.", got,
		"only backticked tool names are swapped, so prose that happens to match a name is left alone")
}
