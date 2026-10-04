package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0068_EveryWorkspaceGainsTheDraftingPlayOnce upgrades a database with the default workspace and one that
// edited its Interview play; each gains the Draft interview play with the code default, and nothing else changes.
func TestMigration0068_EveryWorkspaceGainsTheDraftingPlayOnce(t *testing.T) {
	var want plays.Builtin
	for _, b := range plays.Builtins() {
		if b.Key == plays.DraftInterviewKey {
			want = b
		}
	}
	db := migrateBefore(t, "0068")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-own', 'Own', 'own', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, created_by, created_at, updated_at, builtin_key) VALUES
    ('p-own', 'ws-own', 'Interview', 'interview', 'Ours.', 'Interview them our way.', 1, NULL, '[]', '', 1, 1, 'interview');`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0068 applies on top, as an upgrade would")
	require.NoError(t, Migrate(db), "and a second pass adds nothing")

	for _, ws := range []string{"workspace-default", "ws-own"} {
		var n int
		var label, typ, desc, instructions string
		require.NoError(t, db.QueryRow(`SELECT COUNT(*), label, type, description, instructions FROM plays WHERE workspace_id = ? AND builtin_key = ?`, ws, want.Key).
			Scan(&n, &label, &typ, &desc, &instructions))
		assert.Equal(t, 1, n, ws)
		assert.Equal(t, []string{want.Label, string(want.Type), want.Description, want.Instructions}, []string{label, typ, desc, instructions}, ws)
	}
	var got string
	require.NoError(t, db.QueryRow(`SELECT instructions FROM plays WHERE id = 'p-own'`).Scan(&got))
	assert.Equal(t, "Interview them our way.", got, "an edited Interview play is untouched")
}
