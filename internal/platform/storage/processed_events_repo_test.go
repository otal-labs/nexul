package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessedEventsRepo_NotProcessed_Initially(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	got, err := s.ProcessedEvents.Seen(context.Background(), "evt-1")
	require.NoError(t, err)
	assert.False(t, got)
}

func TestProcessedEventsRepo_Record_IsProcessed(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.ProcessedEvents.Record(context.Background(), "evt-1"))

	got, err := s.ProcessedEvents.Seen(context.Background(), "evt-1")
	require.NoError(t, err)
	assert.True(t, got)
}

func TestProcessedEventsRepo_Record_Idempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.ProcessedEvents.Record(context.Background(), "evt-1"))
	require.NoError(t, s.ProcessedEvents.Record(context.Background(), "evt-1"))
}
