package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0071_EveryWorkspaceGetsClarifyViaAIOnce upgrades a database with two workspaces, one of whose plays
// were renamed or deleted, and checks each gets the Clarify play with the code default's instructions, once.
func TestMigration0071_EveryWorkspaceGetsClarifyViaAIOnce(t *testing.T) {
	db := migrateBefore(t, "0071")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-old', 'Old', 'old', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('p-mine', 'ws-old', 'Clarify via AI', 'doc', '', 'my own', 1, NULL, '[]', '', '', 1, 1);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0071 applies on top, as an upgrade would")
	require.NoError(t, applyMigration(db, "0071_again", migration0071(t)), "a second run adds nothing")

	var want plays.Builtin
	for _, b := range plays.Builtins() {
		if b.Key == plays.ClarifyKey {
			want = b
		}
	}
	s := New(db, testEncKey)
	// 0086 later renames the second of two same-named plays, here the seeded one.
	for ws, label := range map[string]string{"workspace-default": want.Label, "ws-old": want.Label + " (2)"} {
		list, err := s.Plays.List(t.Context(), ws)
		require.NoError(t, err)
		var clarify []*plays.Play
		for _, p := range list {
			if p.BuiltinKey == plays.ClarifyKey {
				clarify = append(clarify, p)
			}
		}
		require.Len(t, clarify, 1, "%s has one Clarify play", ws)
		p := clarify[0]
		assert.Equal(t, label, p.Label)
		assert.Equal(t, plays.TypeDoc, p.Type)
		assert.Equal(t, want.Description, p.Description)
		assert.Equal(t, want.Instructions, p.Instructions, "the migration carries the code default verbatim")
		assert.True(t, p.Enabled)
		assert.Nil(t, p.ShowWhenStage)
	}
	mine, err := s.Plays.Get(t.Context(), "p-mine")
	require.NoError(t, err)
	assert.Equal(t, "my own", mine.Instructions, "a person's play of the same name is left alone")
	assert.Empty(t, mine.BuiltinKey)
}

func migration0071(t *testing.T) string {
	t.Helper()
	script, err := migrationFS.ReadFile("migrations/0071_clarify_play.sql")
	require.NoError(t, err)
	return string(script)
}
