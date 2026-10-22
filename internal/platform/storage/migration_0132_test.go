package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/pairing"
)

// TestMigration0132_KeepsWhereRunsWentAndHowManyAtOnce upgrades a person with three computers, a default, a fallback
// project, a project link and two workspaces' runs at once: the default goes first and the rest newest first, each
// computer takes the most runs at once any of their workspaces allowed, the fallback becomes the default computer's
// chat folder, and the link keeps its model but no longer pins its computer.
func TestMigration0132_KeepsWhereRunsWentAndHowManyAtOnce(t *testing.T) {
	db := migrateBefore(t, "0132")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1), ('u-bob', 'bob', 1, 1);
INSERT INTO workspaces (id, name, slug, created_at, updated_at, auto_play_runs_at_once) VALUES ('ws-busy', 'Busy', 'busy', 1, 1, 3);
UPDATE workspaces SET auto_play_runs_at_once = 2 WHERE id = 'workspace-default';
INSERT INTO workspace_members (user_id, workspace_id, created_at) VALUES ('u-alice', 'workspace-default', 1), ('u-alice', 'ws-busy', 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO pairing_computers (id, user_id, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at) VALUES
    ('c-old', 'u-alice', 'old', '', 'sealed', 2000000000, '', 1, 1),
    ('c-main', 'u-alice', 'main', '', 'sealed', 2000000000, '', 3, 3),
    ('c-new', 'u-alice', 'new', '', 'sealed', 2000000000, '', 5, 5),
    ('c-bob', 'u-bob', 'bob', '', 'sealed', 2000000000, '', 1, 1);
INSERT INTO pairing_user_defaults (user_id, default_computer_id, fallback_project_id, provider, model, start_in)
VALUES ('u-alice', 'c-main', 't3-chat', 'claude', 'opus', 'worktree');
INSERT INTO pairing_project_links (user_id, project_id, computer_id, harness_project_id, provider, model, start_in, updated_at)
VALUES ('u-alice', 'p-web', 'c-old', 't3-web', 'codex', 'gpt', 'folder', 10);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0132 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	alice, err := s.Pairing.ListComputers(ctx, "u-alice")
	require.NoError(t, err)
	var order, chats []string
	var runs []int
	for _, c := range alice {
		order, chats, runs = append(order, c.ID), append(chats, c.ChatProjectID), append(runs, c.RunsAtOnce)
	}
	assert.Equal(t, []string{"c-main", "c-new", "c-old"}, order, "the default first, then newest first")
	assert.Equal(t, []string{"t3-chat", "", ""}, chats, "the fallback project is the default computer's chat folder")
	assert.Equal(t, []int{3, 3, 3}, runs, "the most any of their workspaces allowed")
	bob, err := s.Pairing.ListComputers(ctx, "u-bob")
	require.NoError(t, err)
	assert.Equal(t, []any{0, 1}, []any{bob[0].Position, bob[0].RunsAtOnce}, "someone in no workspace gets one run at once")

	defaults, err := s.Pairing.GetDefaults(ctx, "u-alice")
	require.NoError(t, err)
	assert.Equal(t, pairing.Defaults{Provider: "claude", Model: "opus", StartIn: pairing.StartInWorktree, Placement: pairing.PlacementInOrder}, defaults)

	link, err := s.Pairing.GetProjectLink(ctx, "u-alice", "p-web")
	require.NoError(t, err)
	assert.Equal(t, []any{"codex", "gpt", pairing.StartInFolder}, []any{link.Provider, link.Model, link.StartIn})
	assert.Empty(t, link.Computers, "the project follows the order now")
	assert.Equal(t, []pairing.Checkout{{ComputerID: "c-old", HarnessProjectID: "t3-web"}}, link.Checkouts, "where its code was stays known")

	var limitColumns int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('workspaces') WHERE name = 'auto_play_runs_at_once'`).Scan(&limitColumns))
	assert.Zero(t, limitColumns, "runs at once is no longer the workspace's")
}
