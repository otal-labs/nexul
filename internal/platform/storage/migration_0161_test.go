package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0161_ACheckoutFromBeforeIsKeptForTheRepositoryItClones upgrades a project whose order names one
// computer and whose other computer only held a checkout, plus a play and a queued run: the order and both checkouts
// read back, each checkout under no repository, and the play and the run work on the app.
func TestMigration0161_ACheckoutFromBeforeIsKeptForTheRepositoryItClones(t *testing.T) {
	db := migrateBefore(t, "0161")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO pairing_computers (id, user_id, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at) VALUES
    ('c-main', 'u-alice', 'main', '', 'sealed', 2000000000, '', 1, 1),
    ('c-spare', 'u-alice', 'spare', '', 'sealed', 2000000000, '', 2, 2);
INSERT INTO pairing_project_links (user_id, project_id, updated_at) VALUES ('u-alice', 'p-web', 10);
INSERT INTO pairing_project_computers (user_id, project_id, computer_id, position, harness_project_id) VALUES
    ('u-alice', 'p-web', 'c-main', 0, 't3-web-main'),
    ('u-alice', 'p-web', 'c-spare', NULL, 't3-web-spare');
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('play-fix', 'workspace-default', 'Fix', 'ticket', '', 'Fix.', 1, 'review', '[]', '', 'u-alice', 1, 1);
INSERT INTO play_queue (id, workspace_id, project_id, target_type, target_id, play_id, play_label, auto_play_id, person_id, run_on, moment, via, priority, status, queued_at, not_before) VALUES
    ('q-1', 'workspace-default', 'p-web', 'ticket', 't-1', 'play-fix', 'Fix', '', 'u-alice', 'developer', 'automation', 'automation', 1, 'queued', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0161 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	link, err := s.Pairing.GetProjectLink(ctx, "u-alice", "p-web")
	require.NoError(t, err)
	assert.Equal(t, []string{"c-main"}, link.Computers, "the order keeps only the computer it named")
	assert.Equal(t, []pairing.Checkout{
		{ComputerID: "c-main", HarnessProjectID: "t3-web-main"},
		{ComputerID: "c-spare", HarnessProjectID: "t3-web-spare"},
	}, link.Checkouts)

	play, err := s.Plays.Get(ctx, "play-fix")
	require.NoError(t, err)
	assert.Equal(t, plays.RepositoryApp, play.Repository)
	queued, err := s.PlayQueue.GetQueueItem(ctx, "q-1")
	require.NoError(t, err)
	assert.Empty(t, queued.Repository)
}
