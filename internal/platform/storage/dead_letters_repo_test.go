package storage

import (
	"context"
	"fmt"
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

func TestDeadLettersRepo_List_PagesCoverEveryLetterOnce(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	var want []string
	for i := range 7 {
		id := fmt.Sprintf("dl-%d", i)
		want = append(want, id)
		require.NoError(t, s.DeadLetters.Put(context.Background(), newTestDeadLetter(id)))
	}

	got := pageAll(t, 3, func(offset, limit int) ([]string, int) {
		page, err := s.DeadLetters.List(context.Background(), limit, offset)
		require.NoError(t, err)
		total, err := s.DeadLetters.Count(context.Background())
		require.NoError(t, err)
		ids := make([]string, len(page))
		for i, dl := range page {
			ids[i] = dl.ID
		}
		return ids, total
	})

	assert.ElementsMatch(t, want, got, "letters stored in the same second still page in a stable order")
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
