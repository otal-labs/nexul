package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0133_AChosenComputerBecomesAListOfOne upgrades an auto play that runs on a chosen computer, one that has
// no pick yet, and a run queued for the chosen computer: each reads back as a list, its person and its switch unchanged,
// and an auto play with no computer gets an empty list rather than a list holding an empty id.
func TestMigration0133_AChosenComputerBecomesAListOfOne(t *testing.T) {
	db := migrateBefore(t, "0133")
	_, err := db.Exec(`
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('play-fix', 'workspace-default', 'Fix in prod', 'ticket', '', 'Fix.', 1, 'review', '[]', '', 'u-1', 1, 1);
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, run_on, computer_id, run_as, created_at, updated_at) VALUES
    ('ap-picked', 'play-fix', 'workspace-default', 1, 'ticket.created', 'computer', 'c-box', 'u-alice', 1, 1),
    ('ap-waiting', 'play-fix', 'workspace-default', 0, 'ticket.unblocked', 'computer', '', '', 1, 1),
    ('ap-developer', 'play-fix', 'workspace-default', 1, 'ticket.developer_set', 'developer', '', '', 1, 1);
INSERT INTO play_queue (id, workspace_id, project_id, target_type, target_id, play_id, play_label, auto_play_id, person_id, run_on, moment, via, priority, status, queued_at, not_before, computer_id) VALUES
    ('q-box', 'workspace-default', 'p-1', 'ticket', 't-1', 'play-fix', 'Fix', 'ap-picked', 'u-alice', 'computer', 'ticket.created', 'web', 1, 'queued', 1, 1, 'c-box'),
    ('q-dev', 'workspace-default', 'p-1', 'ticket', 't-2', 'play-fix', 'Fix', 'ap-developer', 'u-dev', 'developer', 'ticket.developer_set', 'web', 1, 'queued', 1, 1, '');`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0133 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	listed, err := s.Plays.ListAutoPlays(ctx, []string{"play-fix"})
	require.NoError(t, err)
	got := map[string]*plays.AutoPlay{}
	for _, a := range listed {
		got[a.ID] = a
	}
	picked := got["ap-picked"]
	assert.Equal(t, []any{[]string{"c-box"}, "c-box", "u-alice", true, plays.RunOnComputer},
		[]any{picked.ComputerIDs, picked.ComputerID, picked.RunAs, picked.Enabled, picked.RunOn}, "the one computer is the whole list, still switched on and run as the same person")
	assert.Equal(t, []string{}, got["ap-waiting"].ComputerIDs, "no computer is an empty list")
	assert.Equal(t, []string{}, got["ap-developer"].ComputerIDs)

	box, err := s.PlayQueue.GetQueueItem(ctx, "q-box")
	require.NoError(t, err)
	assert.Equal(t, []any{[]string{"c-box"}, "c-box"}, []any{box.ComputerIDs, box.ComputerID}, "a run already waiting for the computer keeps it")
	dev, err := s.PlayQueue.GetQueueItem(ctx, "q-dev")
	require.NoError(t, err)
	assert.Equal(t, []string{}, dev.ComputerIDs)
}
