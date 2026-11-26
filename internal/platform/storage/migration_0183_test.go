package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0183_AnAutoPlayOnComputersRunsAsItsPicker upgrades three auto plays alice picked computers for, one on
// her own desk, one on a computer bob shares with her under Run agents, one on a share bob has since cut to See, plus a
// queued run: each runs as alice on her order, the live share joins her order after her own computers, the cut one does
// not, and the queued run keeps its person.
func TestMigration0183_AnAutoPlayOnComputersRunsAsItsPicker(t *testing.T) {
	db := migrateBefore(t, "0183")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1), ('u-bob', 'bob', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO pairing_computers (id, user_id, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at, position) VALUES
    ('c-desk', 'u-alice', 'desk', '', 'sealed', 2000000000, '', 1, 1, 0),
    ('c-laptop', 'u-alice', 'laptop', '', 'sealed', 2000000000, '', 1, 1, 1),
    ('c-client', 'u-bob', 'client', '', 'sealed', 2000000000, '', 1, 1, 0),
    ('c-gone', 'u-bob', 'gone', '', 'sealed', 2000000000, '', 1, 1, 1);
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at) VALUES
    ('computer', 'c-client', 'person', 'u-alice', '["computers:see","computers:run_agents"]', '[]', 1, 1),
    ('computer', 'c-gone', 'person', 'u-alice', '["computers:see"]', '[]', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('play-crash', 'workspace-default', 'Investigate', 'stack', '', 'Look.', 1, NULL, '[]', '', 'u-alice', 1, 1);
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, run_on, computer_id, computer_ids, run_as, memory_ids, created_at, updated_at) VALUES
    ('ap-own', 'play-crash', 'workspace-default', 1, 'stack.service_crashed', 'computer', 'c-desk', '["c-desk"]', 'u-alice', '[]', 1, 1),
    ('ap-shared', 'play-crash', 'workspace-default', 1, 'stack.service_crashed', 'computer', 'c-client', '["c-client","c-desk"]', 'u-alice', '[]', 1, 1),
    ('ap-cut', 'play-crash', 'workspace-default', 1, 'stack.service_crashed', 'computer', 'c-gone', '["c-gone"]', 'u-alice', '[]', 1, 1),
    ('ap-dev', 'play-crash', 'workspace-default', 0, 'stack.service_crashed', 'developer', '', '[]', '', '[]', 1, 1);
INSERT INTO play_queue (id, workspace_id, project_id, target_type, target_id, play_id, play_label, auto_play_id, person_id, run_on, moment, via, priority, status, queued_at, not_before, computer_id, computer_ids) VALUES
    ('q-1', 'workspace-default', 'p-web', 'stack', 's-1', 'play-crash', 'Investigate', 'ap-shared', 'u-alice', 'computer', 'stack.service_crashed', 'web', 1, 'queued', 1, 1, 'c-client', '["c-client","c-desk"]');`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0183 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	for _, id := range []string{"ap-own", "ap-shared", "ap-cut"} {
		a, err := s.Plays.GetAutoPlay(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, []any{plays.RunOnPerson, "u-alice", true, []string{}}, []any{a.RunOn, a.RunAs, a.Enabled, a.ComputerIDs}, id)
	}
	dev, err := s.Plays.GetAutoPlay(ctx, "ap-dev")
	require.NoError(t, err)
	assert.Equal(t, plays.RunOnDeveloper, dev.RunOn, "an auto play that never named computers is left as it was")

	shares, err := s.Pairing.ListOrderShares(ctx, "u-alice")
	require.NoError(t, err)
	assert.Equal(t, []pairing.OrderShare{{ComputerID: "c-client", Position: 2}}, shares,
		"the live share joins after her two own computers; the one cut to See and her own desk do not")
	none, err := s.Pairing.ListOrderShares(ctx, "u-bob")
	require.NoError(t, err)
	assert.Empty(t, none)

	queued, err := s.PlayQueue.GetQueueItem(ctx, "q-1")
	require.NoError(t, err)
	assert.Equal(t, []any{plays.RunOnPerson, "u-alice", []string{}}, []any{queued.RunOn, queued.PersonID, queued.ComputerIDs})
}
