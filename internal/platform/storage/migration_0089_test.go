package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0089_ARunFromBeforeKeepsNoLocation_ANewRunKeepsItsOwn upgrades with a run waiting on its answer, which
// has no recorded location and so follows the starter's link as before; a run started after records where it ran.
func TestMigration0089_ARunFromBeforeKeepsNoLocation_ANewRunKeepsItsOwn(t *testing.T) {
	db := migrateBefore(t, "0089")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1);
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, starter_id, via, state, started_at, computer_id, harness_session_id)
    VALUES ('trail-waiting', 'ws-acme', 'play-1', 'Fix with AI', 'ticket', 't-1', 'p-1', 'u-alice', 'web', 'waiting', 1, 'c-1', 'th-1');
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0089_play_trail_location.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0089_play_trail_location", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	old, err := s.PlayTrails.GetTrail(ctx, "trail-waiting")
	require.NoError(t, err)
	assert.Equal(t, plays.TrailWaiting, old.State)
	assert.Equal(t, "th-1", old.HarnessSessionID)
	assert.Empty(t, old.HarnessProjectID)
	assert.False(t, old.Worktree)

	require.NoError(t, s.PlayTrails.CreateTrail(ctx, &plays.Trail{
		ID: "trail-new", WorkspaceID: "ws-acme", PlayID: "play-1", PlayLabel: "Fix with AI", TargetType: plays.TargetTicket, TargetID: "t-2",
		ProjectID: "p-1", StarterID: "u-alice", Via: plays.ViaWeb, State: plays.TrailStarting, StartedAt: time.Unix(2, 0).UTC(),
		ComputerID: "c-1", HarnessProjectID: "t3-app", Worktree: true,
	}))
	fresh, err := s.PlayTrails.GetTrail(ctx, "trail-new")
	require.NoError(t, err)
	assert.Equal(t, "t3-app", fresh.HarnessProjectID)
	assert.True(t, fresh.Worktree)
}
