package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A joiner replays the newest snapshot plus what came after it; a reset leaves nothing and a seq above all it held.
func TestMemoryStore_ReplayAndReset(t *testing.T) {
	store := NewMemoryStore()
	ctx := t.Context()
	_, err := store.AppendUpdate(ctx, "room", "alice", KindUpdate, "before")
	require.NoError(t, err)
	snap, err := store.AppendUpdate(ctx, "room", "alice", KindSnapshot, "snap")
	require.NoError(t, err)
	after, err := store.AppendUpdate(ctx, "room", "bob", KindUpdate, "after")
	require.NoError(t, err)
	require.NoError(t, store.TrimUpdates(ctx, "room", snap))

	replay, err := store.LoadReplay(ctx, "room")
	require.NoError(t, err)
	require.NotNil(t, replay.Snapshot)
	assert.Equal(t, "snap", replay.Snapshot.Payload)
	require.Len(t, replay.Increments, 1, "an increment after the snapshot survives the trim to the snapshot's base")
	assert.Equal(t, "after", replay.Increments[0].Payload)
	assert.Equal(t, after, replay.Seq)

	seq, err := store.Reset(ctx, "room")
	require.NoError(t, err)
	assert.Greater(t, seq, after)
	replay, err = store.LoadReplay(ctx, "room")
	require.NoError(t, err)
	assert.Equal(t, Replay{Seq: seq}, replay)

	empty, err := store.LoadReplay(ctx, "never-joined")
	require.NoError(t, err)
	assert.Equal(t, Replay{}, empty)
}
