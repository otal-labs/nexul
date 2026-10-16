package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0121_EveryWorkspaceGetsInvestigateACrashSwitchedOff upgrades two workspaces, one already holding a play
// under the shipped label in another case: each gets the stack play under its built-in key, with its auto play off and
// waiting for a computer, and the label stays unique per workspace.
func TestMigration0121_EveryWorkspaceGetsInvestigateACrashSwitchedOff(t *testing.T) {
	db := migrateBefore(t, "0121")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-taken', 'Taken', 'taken', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('play-mine', 'ws-taken', 'investigate a CRASH', 'ticket', '', 'Mine.', 1, 'review', '[]', '', 'u-1', 1, 1);
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, run_on, created_at, updated_at)
    VALUES ('ap-old', 'play-mine', 'ws-taken', 1, 'ticket.created', 'developer', 1, 1);
INSERT INTO play_queue (id, workspace_id, project_id, target_type, target_id, play_id, play_label, auto_play_id, run_on, moment, via, priority, status, queued_at, not_before)
    VALUES ('q-old', 'ws-taken', 'p-1', 'ticket', 't-1', 'play-mine', 'Mine', 'ap-old', 'developer', 'ticket.created', 'web', 1, 'queued', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	var code plays.Builtin
	for _, b := range plays.Builtins() {
		if b.Key == plays.InvestigateCrashKey {
			code = b
		}
	}
	for _, tt := range []struct{ ws, label string }{
		{"workspace-default", "Investigate a crash"},
		{"ws-taken", "Investigate a crash (2)"},
	} {
		list, err := s.Plays.List(ctx, tt.ws)
		require.NoError(t, err)
		var crash *plays.Play
		for _, p := range list {
			if p.BuiltinKey == plays.InvestigateCrashKey {
				crash = p
			}
		}
		require.NotNil(t, crash, tt.ws)
		assert.Equal(t, []any{tt.label, plays.TypeStack, (*plays.Stage)(nil), true, code.Description, code.Instructions},
			[]any{crash.Label, crash.Type, crash.ShowWhenStage, crash.Enabled, crash.Description, crash.Instructions}, tt.ws)
		require.NoError(t, crash.Validate())

		autoPlays, err := s.Plays.ListAutoPlays(ctx, []string{crash.ID})
		require.NoError(t, err)
		require.Len(t, autoPlays, 1, tt.ws)
		a := autoPlays[0]
		assert.Equal(t, []any{false, plays.MomentStackServiceCrashed, plays.RunOnComputer, "", "", plays.LevelNormal},
			[]any{a.Enabled, a.Moment, a.RunOn, a.ComputerID, a.RunAs, a.Priority.Otherwise}, "off, until a computer is picked in %s", tt.ws)
		require.NoError(t, a.Validate(plays.TypeStack))
	}

	mine, err := s.Plays.Get(ctx, "play-mine")
	require.NoError(t, err)
	assert.Equal(t, "investigate a CRASH", mine.Label, "a play that held the label keeps it")
	old, err := s.Plays.GetAutoPlay(ctx, "ap-old")
	require.NoError(t, err)
	assert.Equal(t, []any{true, plays.RunOnDeveloper, ""}, []any{old.Enabled, old.RunOn, old.ComputerID}, "existing auto plays are untouched")
	item, err := s.PlayQueue.GetQueueItem(ctx, "q-old")
	require.NoError(t, err)
	assert.Equal(t, []string{"", ""}, []string{item.Service, item.OutageID}, "queued runs gain empty stack columns")

	rows, err := db.Query(`PRAGMA foreign_key_check`)
	require.NoError(t, err)
	assert.False(t, rows.Next(), "no row points at a missing parent")
	require.NoError(t, rows.Close())
}
