package inprocess

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

func newTestBus() *Bus {
	return New(Options{Logger: testutil.DiscardLogger()})
}

func TestBus_Publish_DeliversEnvelope(t *testing.T) {
	bus := newTestBus()
	defer func() { require.NoError(t, bus.Close()) }()
	got := make(chan eventbus.Event, 1)
	topic := "doc.created"
	require.NoError(t, bus.Subscribe(context.Background(), topic, func(ctx context.Context, ev eventbus.Event) error {
		got <- ev
		return nil
	}))

	type payload struct{ Name string }
	require.NoError(t, bus.Publish(context.Background(), topic, payload{Name: "spec"}))

	select {
	case ev := <-got:
		assert.Equal(t, topic, ev.Topic)
		assert.NotEmpty(t, ev.ID)
		assert.NotEmpty(t, ev.TraceID)
		assert.False(t, ev.Timestamp.IsZero())
		var p payload
		require.NoError(t, json.Unmarshal(ev.Payload, &p))
		assert.Equal(t, "spec", p.Name)
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
}

func TestBus_Publish_FansOutToAllSubscribers(t *testing.T) {
	bus := newTestBus()
	defer func() { require.NoError(t, bus.Close()) }()
	got := make(chan eventbus.Event, 2)
	h := func(ctx context.Context, ev eventbus.Event) error {
		got <- ev
		return nil
	}
	require.NoError(t, bus.Subscribe(context.Background(), "t", h))
	require.NoError(t, bus.Subscribe(context.Background(), "t", h))
	require.NoError(t, bus.Publish(context.Background(), "t", "x"))

	for i := 0; i < 2; i++ {
		select {
		case <-got:
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d missed the event", i)
		}
	}
}

func TestBus_Publish_PropagatesTraceID(t *testing.T) {
	bus := newTestBus()
	defer func() { require.NoError(t, bus.Close()) }()
	got := make(chan eventbus.Event, 1)
	require.NoError(t, bus.Subscribe(context.Background(), "t", func(ctx context.Context, ev eventbus.Event) error {
		assert.Equal(t, ev.TraceID, logging.TraceIDFromCtx(ctx))
		got <- ev
		return nil
	}))

	ctx := logging.CtxWithTraceID(context.Background(), "trace-abc")
	require.NoError(t, bus.Publish(ctx, "t", "x"))
	select {
	case ev := <-got:
		assert.Equal(t, "trace-abc", ev.TraceID)
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
}

func TestBus_Publish_NoSubscribersIsNoop(t *testing.T) {
	bus := newTestBus()
	defer func() { require.NoError(t, bus.Close()) }()
	require.NoError(t, bus.Publish(context.Background(), "nobody.listens", "x"))
}

func TestBus_ClosedRejectsOperations(t *testing.T) {
	bus := newTestBus()
	require.NoError(t, bus.Close())
	require.NoError(t, bus.Close())

	ctx := context.Background()
	require.ErrorIs(t, bus.Publish(ctx, "t", "x"), eventbus.ErrClosed)
	require.ErrorIs(t, bus.Subscribe(ctx, "t", func(ctx context.Context, ev eventbus.Event) error { return nil }), eventbus.ErrClosed)
}

func TestBus_RetryableHandler_SucceedsAfterTransientFailure(t *testing.T) {
	bus := New(Options{
		Logger:           testutil.DiscardLogger(),
		RetryMaxAttempts: 3,
		RetryBackoff:     NewBackoff(time.Millisecond, time.Millisecond, nil),
	})
	defer func() { require.NoError(t, bus.Close()) }()
	var calls atomic.Int32
	require.NoError(t, bus.Subscribe(context.Background(), "t", func(ctx context.Context, ev eventbus.Event) error {
		calls.Add(1)
		if calls.Load() < 3 {
			return errBoom
		}
		return nil
	}))
	require.NoError(t, bus.Publish(context.Background(), "t", "x"))
	require.Eventually(t, func() bool { return calls.Load() == 3 }, time.Second, 10*time.Millisecond)
}
