package inprocess

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/eventbus/processed"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
)

// TestPublishPanic_DeadLetter_Replay is the ws-02 acceptance scenario:
// publish -> handler panics -> dead letter written -> replay succeeds.
func TestPublishPanic_DeadLetter_Replay(t *testing.T) {
	ctx := context.Background()
	st := testutil.NewStore(t)
	pStore := st.ProcessedEvents
	dlStore := st.DeadLetters

	bus := New(Options{
		Logger:           testutil.DiscardLogger(),
		RetryMaxAttempts: 2,
		RetryBackoff:     NewBackoff(time.Millisecond, time.Millisecond, nil),
		DedupeStore:      pStore,
		DeadLetterStore:  dlStore,
	})
	require.NoError(t, bus.Subscribe(ctx, "order.created", func(ctx context.Context, ev eventbus.Event) error {
		panic("handler bug")
	}))
	require.NoError(t, bus.Publish(ctx, "order.created", map[string]any{"order": 7}))

	require.Eventually(t, func() bool {
		rows, err := dlStore.List(ctx, 10, 0)
		return err == nil && len(rows) == 1
	}, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, bus.Close())

	rows, err := dlStore.List(ctx, 10, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	dl := rows[0]
	assert.Equal(t, "order.created", dl.Topic)
	assert.Equal(t, 2, dl.Attempts)
	assert.Contains(t, dl.Error, "panic")

	bus2 := New(Options{
		Logger:           testutil.DiscardLogger(),
		RetryMaxAttempts: 2,
		RetryBackoff:     NewBackoff(time.Millisecond, time.Millisecond, nil),
		DedupeStore:      pStore,
		DeadLetterStore:  dlStore,
	})
	defer func() { require.NoError(t, bus2.Close()) }()
	got := make(chan eventbus.Event, 1)
	require.NoError(t, bus2.Subscribe(ctx, "order.created", func(ctx context.Context, ev eventbus.Event) error {
		got <- ev
		return nil
	}))

	require.NoError(t, deadletter.Replay(ctx, dlStore, bus2, dl.ID))

	select {
	case ev := <-got:
		var p map[string]any
		require.NoError(t, json.Unmarshal(ev.Payload, &p))
		assert.Equal(t, float64(7), p["order"])
	case <-time.After(time.Second):
		t.Fatal("replayed event not delivered")
	}
	_, err = dlStore.Get(ctx, dl.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

// TestBus_Publish_FansOutToAllSubscribers_WithDedupe is the ws-33 regression
// for the fan-out defect: with a DedupeStore (the production configuration),
// every subscriber of a topic must receive every event. Before the fix the
// first subscriber to run recorded the shared event ID and every other
// subscriber saw "seen" and skipped.
func TestBus_Publish_FansOutToAllSubscribers_WithDedupe(t *testing.T) {
	ctx := context.Background()
	st := testutil.NewStore(t)
	bus := New(Options{
		Logger:      testutil.DiscardLogger(),
		DedupeStore: st.ProcessedEvents,
	})
	defer func() { require.NoError(t, bus.Close()) }()

	const subs, events = 3, 20
	var hits [subs]atomic.Int32
	for i := 0; i < subs; i++ {
		i := i
		require.NoError(t, bus.Subscribe(ctx, "fanout.t", func(ctx context.Context, ev eventbus.Event) error {
			hits[i].Add(1)
			return nil
		}))
	}
	for i := 0; i < events; i++ {
		require.NoError(t, bus.Publish(ctx, "fanout.t", i))
	}

	require.Eventually(t, func() bool {
		for i := 0; i < subs; i++ {
			if hits[i].Load() != events {
				return false
			}
		}
		return true
	}, 5*time.Second, 10*time.Millisecond)
}

// TestBus_Dedupe_ScopedPerSubscriber proves the dedupe middleware is scoped
// per subscriber: each subscriber of the same topic processes an event exactly
// once, while a redelivery to the same subscriber is still skipped.
func TestBus_Dedupe_ScopedPerSubscriber(t *testing.T) {
	ctx := context.Background()
	st := testutil.NewStore(t)
	bus := New(Options{
		Logger:      testutil.DiscardLogger(),
		DedupeStore: st.ProcessedEvents,
	})
	defer func() { require.NoError(t, bus.Close()) }()

	var a, b atomic.Int32
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "consumer-a", "dup.t", func(ctx context.Context, ev eventbus.Event) error {
		a.Add(1)
		return nil
	}))
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "consumer-b", "dup.t", func(ctx context.Context, ev eventbus.Event) error {
		b.Add(1)
		return nil
	}))

	require.NoError(t, bus.PublishWithID(ctx, "e1", "dup.t", "x"))
	require.Eventually(t, func() bool { return a.Load() == 1 && b.Load() == 1 }, time.Second, 10*time.Millisecond)

	require.NoError(t, bus.PublishWithID(ctx, "e1", "dup.t", "x"))
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(1), a.Load())
	assert.Equal(t, int32(1), b.Load())

	require.NoError(t, bus.PublishWithID(ctx, "e2", "dup.t", "x"))
	require.Eventually(t, func() bool { return a.Load() == 2 && b.Load() == 2 }, time.Second, 10*time.Millisecond)
}

// TestBus_Dedupe_StableConsumerAcrossRestart proves the subscriber identity
// survives a restart: a consumer that already processed an event (on a
// previous bus, sharing the store) skips the redelivery, while a new consumer
// of the same topic still processes it. This is the outbox crash-redelivery
// semantics the outbox relay depends on.
func TestBus_Dedupe_StableConsumerAcrossRestart(t *testing.T) {
	ctx := context.Background()
	st := testutil.NewStore(t)
	pStore := st.ProcessedEvents

	bus1 := New(Options{Logger: testutil.DiscardLogger(), DedupeStore: pStore})
	var processed1 atomic.Int32
	require.NoError(t, bus1.SubscribeWithConsumer(ctx, "deploy.domain", "deploy.status_changed", func(ctx context.Context, ev eventbus.Event) error {
		processed1.Add(1)
		return nil
	}))
	require.NoError(t, bus1.PublishWithID(ctx, "e1", "deploy.status_changed", "x"))
	require.Eventually(t, func() bool { return processed1.Load() == 1 }, time.Second, 10*time.Millisecond)
	require.NoError(t, bus1.Close())

	bus2 := New(Options{Logger: testutil.DiscardLogger(), DedupeStore: pStore})
	defer func() { require.NoError(t, bus2.Close()) }()
	var live, automations atomic.Int32
	require.NoError(t, bus2.SubscribeWithConsumer(ctx, "deploy.domain", "deploy.status_changed", func(ctx context.Context, ev eventbus.Event) error {
		live.Add(1)
		return nil
	}))
	require.NoError(t, bus2.SubscribeWithConsumer(ctx, "automations", "deploy.status_changed", func(ctx context.Context, ev eventbus.Event) error {
		automations.Add(1)
		return nil
	}))

	require.NoError(t, bus2.PublishWithID(ctx, "e1", "deploy.status_changed", "x"))
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, int32(0), live.Load(), "consumer that processed e1 before the restart must skip it")
	assert.Equal(t, int32(1), automations.Load(), "new consumer must still process e1")
}

// TestBus_FanOut_SubscribersRunIndependently proves each subscriber has its
// own dispatch goroutine: two subscribers on the same topic both start
// handling a fanned-out event, neither blocked behind the other.
func TestBus_FanOut_SubscribersRunIndependently(t *testing.T) {
	ctx := context.Background()
	bus := New(Options{Logger: testutil.DiscardLogger()})
	defer func() { require.NoError(t, bus.Close()) }()

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	h := func(ctx context.Context, ev eventbus.Event) error {
		started <- struct{}{}
		<-release
		return nil
	}
	require.NoError(t, bus.Subscribe(ctx, "throttle.t", h))
	require.NoError(t, bus.Subscribe(ctx, "throttle.t", h))

	// One event fans out to both subscribers; if they shared one dispatch
	// goroutine only the first subscriber's handler would start.
	require.NoError(t, bus.Publish(ctx, "throttle.t", "1"))

	require.Eventually(t, func() bool { return len(started) == 2 }, time.Second, 10*time.Millisecond)
	close(release)
}

// TestBus_PublishWithID_UsesSuppliedID proves PublishWithID stamps the given
// ID onto the envelope instead of a fresh one, so outbox redeliveries keep the
// row ID.
func TestBus_PublishWithID_UsesSuppliedID(t *testing.T) {
	bus := newTestBus()
	defer func() { require.NoError(t, bus.Close()) }()
	got := make(chan eventbus.Event, 1)
	require.NoError(t, bus.Subscribe(context.Background(), "t", func(ctx context.Context, ev eventbus.Event) error {
		got <- ev
		return nil
	}))
	require.NoError(t, bus.PublishWithID(context.Background(), "row-42", "t", "x"))
	select {
	case ev := <-got:
		assert.Equal(t, "row-42", ev.ID)
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
	require.Error(t, bus.PublishWithID(context.Background(), "", "t", "x"))
}

// TestDuplicateEvent_ProcessedOnce is the ws-02 acceptance scenario: the same
// event delivered twice must only run its handler once.
func TestDuplicateEvent_ProcessedOnce(t *testing.T) {
	ctx := context.Background()
	st := testutil.NewStore(t)
	pStore := st.ProcessedEvents

	bus := New(Options{Logger: testutil.DiscardLogger(), DedupeStore: pStore})
	defer func() { require.NoError(t, bus.Close()) }()
	got := make(chan eventbus.Event, 2)
	require.NoError(t, bus.Subscribe(ctx, "ticket.status_changed", func(ctx context.Context, ev eventbus.Event) error {
		got <- ev
		return nil
	}))

	ev := eventbus.Event{
		ID:        "dup-event-1",
		TraceID:   "tr-1",
		Topic:     "ticket.status_changed",
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(`{"status":"done"}`),
	}
	require.NoError(t, bus.publish(ctx, ev))
	require.NoError(t, bus.publish(ctx, ev))

	select {
	case <-got:
	case <-time.After(time.Second):
		t.Fatal("first delivery missing")
	}
	select {
	case <-got:
		t.Fatal("duplicate event was processed a second time")
	case <-time.After(150 * time.Millisecond):
	}
	seen, err := processed.Scope(pStore, "ticket.status_changed#0").Seen(ctx, "dup-event-1")
	require.NoError(t, err)
	assert.True(t, seen)
}

func TestBus_Close_DrainsInFlightHandler(t *testing.T) {
	bus := New(Options{DrainTimeout: time.Second})
	started := make(chan struct{})
	release := make(chan struct{})
	require.NoError(t, bus.Subscribe(context.Background(), "t", func(ctx context.Context, ev eventbus.Event) error {
		close(started)
		<-release
		return nil
	}))
	require.NoError(t, bus.Publish(context.Background(), "t", "x"))
	<-started

	done := make(chan error, 1)
	go func() { done <- bus.Close() }()
	select {
	case err := <-done:
		t.Fatalf("close returned while a handler was in flight: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("close did not return after drain")
	}
}

func TestBus_Close_TimesOutAndCancelsInFlight(t *testing.T) {
	bus := New(Options{DrainTimeout: 50 * time.Millisecond, Logger: testutil.DiscardLogger()})
	started := make(chan struct{})
	require.NoError(t, bus.Subscribe(context.Background(), "t", func(ctx context.Context, ev eventbus.Event) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}))
	require.NoError(t, bus.Publish(context.Background(), "t", "x"))
	<-started

	err := bus.Close()
	require.ErrorIs(t, err, eventbus.ErrDrainTimeout)
}
