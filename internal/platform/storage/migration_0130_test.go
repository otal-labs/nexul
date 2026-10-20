package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0130_AutoPlaysDropMemoriesDeletedBeforeIt keeps the ids of memories that still exist, in their order.
func TestMigration0130_AutoPlaysDropMemoriesDeletedBeforeIt(t *testing.T) {
	db := migrateBefore(t, "0130")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	seedPlays(t, s, "play-1")
	ctx := t.Context()
	seedProject(t, s, "p-1", "ws-1", "ACM")
	for _, id := range []string{"mem-1", "mem-2"} {
		m := newTestMemory(id, "p-1")
		m.WorkspaceID = "ws-1"
		require.NoError(t, s.Memories.Create(ctx, m, ""))
	}
	untouched := newTestAutoPlay("ap-kept", "play-1", time.Now())
	untouched.MemoryIDs = []string{"mem-2", "mem-1"}
	require.NoError(t, s.Plays.CreateAutoPlay(ctx, untouched))
	stale := newTestAutoPlay("ap-stale", "play-1", time.Now())
	stale.MemoryIDs = []string{"mem-gone", "mem-2", "mem-also-gone"}
	require.NoError(t, s.Plays.CreateAutoPlay(ctx, stale))

	require.NoError(t, Migrate(db))

	kept, err := s.Plays.GetAutoPlay(ctx, "ap-kept")
	require.NoError(t, err)
	assert.Equal(t, []string{"mem-2", "mem-1"}, kept.MemoryIDs)
	cleaned, err := s.Plays.GetAutoPlay(ctx, "ap-stale")
	require.NoError(t, err)
	assert.Equal(t, []string{"mem-2"}, cleaned.MemoryIDs)
}
