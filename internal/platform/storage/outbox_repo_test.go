package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// insertOutboxTestRow seeds a row directly; production goes through insertOutboxRow in a domain repo's transaction.
func insertOutboxTestRow(t *testing.T, s *Store, id, topic string, payload []byte) {
	t.Helper()
	_, err := s.db.Exec(`INSERT INTO outbox (id, topic, payload) VALUES (?, ?, ?)`, id, topic, payload)
	require.NoError(t, err)
}

func TestOutboxRepo_Pending_ReturnsUnpublishedInOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	insertOutboxTestRow(t, s, "evt-1", "doc.created", []byte(`{"a":1}`))
	insertOutboxTestRow(t, s, "evt-2", "ticket.created", []byte(`{"b":2}`))

	got, err := s.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "evt-1", got[0].ID)
	assert.Equal(t, "doc.created", got[0].Topic)
	assert.Equal(t, []byte(`{"a":1}`), got[0].Payload)
	assert.Equal(t, "evt-2", got[1].ID)
}

func TestOutboxRepo_Pending_RespectsLimit(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	for _, id := range []string{"evt-1", "evt-2", "evt-3"} {
		insertOutboxTestRow(t, s, id, "doc.created", []byte(`{}`))
	}

	got, err := s.Outbox.Unpublished(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestOutboxRepo_MarkPublished_RemovesFromPending(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	insertOutboxTestRow(t, s, "evt-1", "doc.created", []byte(`{}`))
	require.NoError(t, s.Outbox.MarkPublished(context.Background(), "evt-1"))

	got, err := s.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestOutboxRepo_MarkPublished_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Outbox.MarkPublished(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
