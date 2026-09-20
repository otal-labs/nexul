package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
)

func newTestDeadLetter(id string) deadletter.DeadLetter {
	return deadletter.DeadLetter{ID: id, Topic: "doc.created", Payload: []byte(`{}`), Error: "handler panic", Attempts: 3}
}

func TestDeadLettersRepo_Put_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.DeadLetters.Put(context.Background(), newTestDeadLetter("dl-1")))
	err := s.DeadLetters.Put(context.Background(), newTestDeadLetter("dl-1"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestDeadLettersRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.DeadLetters.Get(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeadLettersRepo_Put_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestDeadLetter("dl-1")
	require.NoError(t, s.DeadLetters.Put(context.Background(), want))

	got, err := s.DeadLetters.Get(context.Background(), "dl-1")
	require.NoError(t, err)
	assert.Equal(t, "doc.created", got.Topic)
	assert.Equal(t, []byte(`{}`), got.Payload)
	assert.Equal(t, "handler panic", got.Error)
	assert.Equal(t, 3, got.Attempts)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestDeadLettersRepo_List_Paginates(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	for _, id := range []string{"dl-1", "dl-2", "dl-3"} {
		require.NoError(t, s.DeadLetters.Put(context.Background(), newTestDeadLetter(id)))
	}

	page, err := s.DeadLetters.List(context.Background(), 2, 0)
	require.NoError(t, err)
	require.Len(t, page, 2)

	rest, err := s.DeadLetters.List(context.Background(), 2, 2)
	require.NoError(t, err)
	require.Len(t, rest, 1)
}

func TestDeadLettersRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.DeadLetters.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeadLettersRepo_Delete_RemovesLetter(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.DeadLetters.Put(context.Background(), newTestDeadLetter("dl-1")))
	require.NoError(t, s.DeadLetters.Delete(context.Background(), "dl-1"))
	_, err := s.DeadLetters.Get(context.Background(), "dl-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
