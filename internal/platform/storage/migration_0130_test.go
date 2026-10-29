package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0130_AutoPlaysDropMemoriesDeletedBeforeIt keeps the ids of memories that still exist, in their order.
func TestMigration0130_AutoPlaysDropMemoriesDeletedBeforeIt(t *testing.T) {
	db := migrateBefore(t, "0130")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	seedPlays(t, s, "play-1")
	ctx := t.Context()
	// Raw SQL at this migration's schema: the repos write the newest one.
	_, err := db.Exec(`INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-1', 'Project ACM', 'ACM', 1, 'ws-1', 1, 1);
INSERT INTO memories (id, workspace_id, project_id, title, body, version, created_at, updated_at) VALUES
    ('mem-1', 'ws-1', 'p-1', 'A', '', 1, 1, 1), ('mem-2', 'ws-1', 'p-1', 'B', '', 1, 1, 1)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO auto_plays (id, play_id, workspace_id, moment, run_on, memory_ids, created_at, updated_at) VALUES
    ('ap-kept', 'play-1', 'ws-1', 'doc.changed', 'causer', '["mem-2","mem-1"]', 1, 1),
    ('ap-stale', 'play-1', 'ws-1', 'doc.changed', 'causer', '["mem-gone","mem-2","mem-also-gone"]', 1, 1)`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	kept, err := s.Plays.GetAutoPlay(ctx, "ap-kept")
	require.NoError(t, err)
	assert.Equal(t, []string{"mem-2", "mem-1"}, kept.MemoryIDs)
	cleaned, err := s.Plays.GetAutoPlay(ctx, "ap-stale")
	require.NoError(t, err)
	assert.Equal(t, []string{"mem-2"}, cleaned.MemoryIDs)
}
