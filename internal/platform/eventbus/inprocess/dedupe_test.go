package inprocess

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/processed"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

func newDedupeStore(t *testing.T) processed.Storer {
	return testutil.NewStore(t).ProcessedEvents
}

func TestDedupe_ProcessesFirstAndSkipsDuplicate(t *testing.T) {
	store := newDedupeStore(t)
	mw := NewDedupe(store)
	calls := 0
	h := mw(func(ctx context.Context, ev eventbus.Event) error {
		calls++
		return nil
	})
	ctx := context.Background()
	require.NoError(t, h(ctx, eventbus.Event{ID: "dup-1"}))
	require.NoError(t, h(ctx, eventbus.Event{ID: "dup-1"}))
	assert.Equal(t, 1, calls)

	seen, err := store.Seen(ctx, "dup-1")
	require.NoError(t, err)
	assert.True(t, seen)
}

func TestDedupe_FailedHandlerIsNotRecorded(t *testing.T) {
	store := newDedupeStore(t)
	mw := NewDedupe(store)
	calls := 0
	h := mw(func(ctx context.Context, ev eventbus.Event) error {
		calls++
		if calls == 1 {
			return errBoom
		}
		return nil
	})
	ctx := context.Background()
	require.Error(t, h(ctx, eventbus.Event{ID: "retry-1"}))
	require.NoError(t, h(ctx, eventbus.Event{ID: "retry-1"}))
	assert.Equal(t, 2, calls)

	seen, err := store.Seen(ctx, "retry-1")
	require.NoError(t, err)
	assert.True(t, seen)
}

func TestDedupe_StoreError_ReturnsWrappedError(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef")).ProcessedEvents
	require.NoError(t, db.Close())
	mw := NewDedupe(store)
	err := mw(func(ctx context.Context, ev eventbus.Event) error {
		return nil
	})(context.Background(), eventbus.Event{ID: "e"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dedupe")
}
