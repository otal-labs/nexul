package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// TestMigration0096_AQueuedRunFromBeforeHasNoPickAndTakesOne upgrades a database with a run already waiting: it keeps
// every field, starts with no computer picked, and a pick then sticks to it.
func TestMigration0096_AQueuedRunFromBeforeHasNoPickAndTakesOne(t *testing.T) {
	db := migrateBefore(t, "0096")
	_, err := db.Exec(`
INSERT INTO play_queue (id, workspace_id, project_id, target_type, target_id, play_id, play_label, auto_play_id, person_id,
  run_on, moment, via, priority, status, reason, trail_id, queued_at, decided_at, not_before)
VALUES ('q-1', 'workspace-default', 'p-1', 'ticket', 't-1', 'play-1', 'Fix it', 'a-1', 'u-dev',
  'developer', 'ticket.unblocked', 'web', 1, 'queued', 'no_default_computer', '', 100, NULL, 160);`)
	require.NoError(t, err)
	require.NoError(t, Migrate(db), "0096 and every later migration apply on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))

	it, err := s.PlayQueue.GetQueueItem(t.Context(), "q-1")
	require.NoError(t, err)
	assert.Equal(t, []any{plays.QueueQueued, "no_default_computer", "Fix it", "", ""}, []any{it.Status, it.Reason, it.PlayLabel, it.ComputerID, it.HarnessProjectID})

	it.ComputerID, it.HarnessProjectID, it.Reason = "c-laptop", "t3-project", ""
	require.NoError(t, s.PlayQueue.MoveQueueItem(t.Context(), it, plays.QueueQueued))
	it, err = s.PlayQueue.GetQueueItem(t.Context(), "q-1")
	require.NoError(t, err)
	assert.Equal(t, []any{"c-laptop", "t3-project"}, []any{it.ComputerID, it.HarnessProjectID})

	require.NoError(t, s.PlayQueue.WakePerson(t.Context(), "u-dev", time.Unix(120, 0)))
	it, err = s.PlayQueue.GetQueueItem(t.Context(), "q-1")
	require.NoError(t, err)
	assert.Equal(t, time.Unix(120, 0).UTC(), it.NotBefore, "a woken run is due at once")
}
