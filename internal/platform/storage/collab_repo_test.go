package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/collab"
)

// collabUpdatesFor inspects the raw rows behind the repo, so tests pin the
// trim/kind semantics directly.
func collabUpdatesFor(t *testing.T, s *Store, docID string) []collab.StoredUpdate {
	t.Helper()
	rows, err := s.db.Query(`SELECT seq, kind, actor_id, payload, created_at FROM collab_updates WHERE doc_id = ? ORDER BY seq`, docID)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var out []collab.StoredUpdate
	for rows.Next() {
		var u collab.StoredUpdate
		var created int64
		require.NoError(t, rows.Scan(&u.Seq, &u.Kind, &u.ActorID, &u.Payload, &created))
		u.CreatedAt = time.Unix(created, 0).UTC()
		out = append(out, u)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestCollabRepo_AppendAndReplay(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))

	seq1, err := s.Collab.AppendUpdate(ctx, "doc-1", "alice", collab.KindUpdate, "dXA9MQ==")
	require.NoError(t, err)
	seq2, err := s.Collab.AppendUpdate(ctx, "doc-1", "bob", collab.KindUpdate, "dXA9Mg==")
	require.NoError(t, err)
	_, err = s.Collab.AppendUpdate(ctx, "doc-1", "alice", collab.KindSnapshot, "c25hcA==")
	require.NoError(t, err)
	seq4, err := s.Collab.AppendUpdate(ctx, "doc-1", "carol", collab.KindUpdate, "dXA9NA==")
	require.NoError(t, err)

	replay, err := s.Collab.LoadReplay(ctx, "doc-1")
	require.NoError(t, err)
	require.NotNil(t, replay.Snapshot)
	assert.Equal(t, "c25hcA==", replay.Snapshot.Payload)
	assert.Equal(t, collab.KindSnapshot, replay.Snapshot.Kind)
	require.Len(t, replay.Increments, 1)
	assert.Equal(t, "dXA9NA==", replay.Increments[0].Payload)
	assert.Equal(t, seq4, replay.Seq)
	assert.Less(t, seq1, seq2, "seqs share one counter per doc")

	// No state at all for an untouched doc: empty replay, no snapshot.
	empty, err := s.Collab.LoadReplay(ctx, "doc-empty")
	require.NoError(t, err)
	assert.Nil(t, empty.Snapshot)
	assert.Empty(t, empty.Increments)
	assert.Zero(t, empty.Seq)
}

func TestCollabRepo_TrimKeepsSnapshotAndNewerIncrements(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))

	// alice applies seq 1 and 2, commits a snapshot covering them (baseSeq 2).
	_, err := s.Collab.AppendUpdate(ctx, "doc-1", "bob", collab.KindUpdate, "dXA9MQ==")
	require.NoError(t, err)
	_, err = s.Collab.AppendUpdate(ctx, "doc-1", "bob", collab.KindUpdate, "dXA9Mg==")
	require.NoError(t, err)
	_, err = s.Collab.AppendUpdate(ctx, "doc-1", "alice", collab.KindSnapshot, "c25hcA==")
	require.NoError(t, err)
	// bob keeps typing after alice's snapshot was computed.
	_, err = s.Collab.AppendUpdate(ctx, "doc-1", "bob", collab.KindUpdate, "dXA9NA==")
	require.NoError(t, err)

	require.NoError(t, s.Collab.TrimUpdates(ctx, "doc-1", 2))

	rows := collabUpdatesFor(t, s, "doc-1")
	require.Len(t, rows, 2, "snapshot + post-snapshot increment survive the trim")
	assert.Equal(t, collab.KindSnapshot, rows[0].Kind)
	assert.Equal(t, "dXA9NA==", rows[1].Payload)

	// Trimming only reaches baseSeq: an update the snapshot misses because it
	// landed after the author's base is never dropped.
	require.NoError(t, s.Collab.TrimUpdates(ctx, "doc-1", 2))
	rows = collabUpdatesFor(t, s, "doc-1")
	require.Len(t, rows, 2)
}

func TestCollabRepo_UpdateLogCascadesWithDocDelete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))
	_, err := s.Collab.AppendUpdate(ctx, "doc-1", "alice", collab.KindUpdate, "dXA9MQ==")
	require.NoError(t, err)

	require.NoError(t, s.Docs.Delete(ctx, "doc-1"))
	assert.Empty(t, collabUpdatesFor(t, s, "doc-1"))
}
