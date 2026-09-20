package inprocess

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

func newDeadLetterStore(t *testing.T) deadletter.Storer {
	return testutil.NewStore(t).DeadLetters
}

func TestDeadLetter_PersistsOnExhaustion(t *testing.T) {
	store := newDeadLetterStore(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := NewDeadLetter(store, log)

	exhausted := &RetryExhausted{Err: errBoom, Attempts: 4}
	err := mw(func(ctx context.Context, ev eventbus.Event) error {
		return exhausted
	})(context.Background(), eventbus.Event{ID: "e1", Topic: "t.created", Payload: json.RawMessage(`{"n":1}`)})
	require.NoError(t, err)

	dl, gerr := store.Get(context.Background(), "e1")
	require.NoError(t, gerr)
	assert.Equal(t, 4, dl.Attempts)
	assert.Equal(t, "t.created", dl.Topic)
	assert.Equal(t, exhausted.Error(), dl.Error)
	assert.JSONEq(t, `{"n":1}`, string(dl.Payload))
}

func TestDeadLetter_PlainError_DefaultsToOneAttempt(t *testing.T) {
	store := newDeadLetterStore(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := NewDeadLetter(store, log)

	err := mw(func(ctx context.Context, ev eventbus.Event) error {
		return errBoom
	})(context.Background(), eventbus.Event{ID: "e1", Topic: "t", Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)

	dl, gerr := store.Get(context.Background(), "e1")
	require.NoError(t, gerr)
	assert.Equal(t, 1, dl.Attempts)
}

func TestDeadLetter_ContextCancellationIsNotDeadLettered(t *testing.T) {
	store := newDeadLetterStore(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := NewDeadLetter(store, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := mw(func(ctx context.Context, ev eventbus.Event) error {
		return context.Canceled
	})(ctx, eventbus.Event{ID: "e2", Topic: "t"})
	require.ErrorIs(t, err, context.Canceled)

	_, gerr := store.Get(context.Background(), "e2")
	require.ErrorIs(t, gerr, apperrs.ErrNotFound)
}

func TestDeadLetter_SuccessIsNotDeadLettered(t *testing.T) {
	store := newDeadLetterStore(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := NewDeadLetter(store, log)

	require.NoError(t, mw(func(ctx context.Context, ev eventbus.Event) error {
		return nil
	})(context.Background(), eventbus.Event{ID: "e3", Topic: "t"}))

	_, gerr := store.Get(context.Background(), "e3")
	assert.ErrorIs(t, gerr, apperrs.ErrNotFound)
}

func TestDeadLetter_PersistFailure_ReturnsError(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef")).DeadLetters
	require.NoError(t, db.Close())
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := NewDeadLetter(store, log)

	err := mw(func(ctx context.Context, ev eventbus.Event) error {
		return errBoom
	})(context.Background(), eventbus.Event{ID: "e4", Topic: "t"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dead letter")
}
