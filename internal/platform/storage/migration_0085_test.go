package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0085_TheDecisionsCheckBecomesASeededPlay upgrades a workspace with the check switched on, plays already
// holding its label and its " (2)" in other cases, and one failed check on a ticket, beside the default workspace with
// it off; labels are already unique per workspace without case, as on an instance that ran that migration first.
func TestMigration0085_TheDecisionsCheckBecomesASeededPlay(t *testing.T) {
	db := migrateBefore(t, "0085")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at, decisions_check_enabled) VALUES ('ws-on', 'On', 'on', 1, 1, 1);
CREATE UNIQUE INDEX unique_play_labels ON plays(workspace_id, label COLLATE NOCASE);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('play-mine', 'ws-on', 'decisions check', 'ticket', '', 'Mine.', 1, 'review', '[]', '', 'u-1', 1, 1),
    ('play-mine-2', 'ws-on', 'DECISIONS CHECK (2)', 'ticket', '', 'Mine too.', 1, 'review', '[]', '', 'u-1', 1, 1);
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, starter_id, via, state, started_at, last_error)
    VALUES ('trail-old', 'ws-on', 'decisions-check', 'Decisions check', 'ticket', 't-1', 'p-1', 'u-1', 'web', 'failed', 1, 'offline');
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0085_decisions_check_play.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0085_decisions_check_play", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	var code plays.Builtin
	for _, b := range plays.Builtins() {
		if b.Key == plays.DecisionsCheckKey {
			code = b
		}
	}
	done := plays.StageDone
	for _, tt := range []struct {
		ws, label string
		enabled   bool
	}{
		{"workspace-default", "Decisions check", false},
		{"ws-on", "Decisions check (3)", true},
	} {
		list, err := s.Plays.List(ctx, tt.ws)
		require.NoError(t, err)
		var check *plays.Play
		for _, p := range list {
			if p.BuiltinKey == plays.DecisionsCheckKey {
				check = p
			}
		}
		require.NotNil(t, check, tt.ws)
		assert.Equal(t, []any{tt.label, plays.TypeTicket, &done, true, code.Description, code.Instructions},
			[]any{check.Label, check.Type, check.ShowWhenStage, check.Enabled, check.Description, check.Instructions}, tt.ws)

		autoPlays, err := s.Plays.ListAutoPlays(ctx, []string{check.ID})
		require.NoError(t, err)
		require.Len(t, autoPlays, 1, tt.ws)
		a := autoPlays[0]
		assert.Equal(t, []any{tt.enabled, plays.MomentTicketEnteredStage, &done, plays.RunOnCauser, plays.LevelNormal, 0},
			[]any{a.Enabled, a.Moment, a.MomentStage, a.RunOn, a.Priority.Otherwise, a.OnceWithinMinutes}, "the switch carries over in %s", tt.ws)
		assert.Empty(t, a.Conditions.Groups)
		require.NoError(t, a.Validate(plays.TypeTicket))

		if tt.ws == "ws-on" {
			trail, err := s.PlayTrails.GetTrail(ctx, "trail-old")
			require.NoError(t, err)
			assert.Equal(t, check.ID, trail.PlayID, "an old check's trail names the play now")
		}
	}

	mine, err := s.Plays.Get(ctx, "play-mine")
	require.NoError(t, err)
	assert.Equal(t, "decisions check", mine.Label, "a play that held the label keeps it")

	rows, err := db.Query(`PRAGMA foreign_key_check`)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rows.Close()) })
	assert.False(t, rows.Next(), "no row points at nothing")
}
