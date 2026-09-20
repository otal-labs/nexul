package outbox_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

var errPublish = errors.New("publish refused")

func insertRow(t *testing.T, db *sql.DB, id, topic string, published int, createdAt int64) {
	_, err := db.Exec(`INSERT INTO outbox (id, topic, payload, created_at, published) VALUES (?, ?, ?, ?, ?)`,
		id, topic, []byte(`{}`), createdAt, published)
	require.NoError(t, err)
}

type fakePublisher struct {
	mu        sync.Mutex
	calls     []string
	ids       []string
	failCount atomic.Int32
}

func (f *fakePublisher) Publish(ctx context.Context, topic string, payload any) error {
	return f.publish("", topic)
}

func (f *fakePublisher) PublishWithID(ctx context.Context, id, topic string, payload any) error {
	return f.publish(id, topic)
}

func (f *fakePublisher) publish(id, topic string) error {
	if f.failCount.Add(-1) >= 0 {
		return errPublish
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, topic)
	f.ids = append(f.ids, id)
	return nil
}

func (f *fakePublisher) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *fakePublisher) eventIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.ids))
	copy(out, f.ids)
	return out
}

func TestRelay_Run_PublishesAndMarks(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	insertRow(t, db, "e1", "a.created", 0, 1000)
	insertRow(t, db, "e2", "b.created", 0, 2000)
	insertRow(t, db, "e3", "c.created", 1, 3000)

	pub := &fakePublisher{}
	relay := outbox.NewRelay(s.Outbox, pub, outbox.RelayConfig{Interval: 5 * time.Millisecond, BatchSize: 10, Logger: testutil.DiscardLogger()})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()

	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(1) FROM outbox WHERE published = 0`).Scan(&n))
		return n == 0
	}, 2*time.Second, 5*time.Millisecond)

	assert.ElementsMatch(t, []string{"a.created", "b.created"}, pub.topics())
	cancel()
	require.NoError(t, <-done)
}

func TestRelay_Run_RetriesAfterPublishFailure(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	insertRow(t, db, "e1", "a.created", 0, 1000)

	pub := &fakePublisher{}
	pub.failCount.Store(1)
	relay := outbox.NewRelay(s.Outbox, pub, outbox.RelayConfig{Interval: 5 * time.Millisecond, BatchSize: 10, Logger: testutil.DiscardLogger()})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()

	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(1) FROM outbox WHERE published = 0`).Scan(&n))
		return n == 0
	}, 2*time.Second, 5*time.Millisecond)

	assert.Equal(t, []string{"a.created"}, pub.topics())
	cancel()
	require.NoError(t, <-done)
}

func TestRelay_Run_StopsOnContextCancel(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	pub := &fakePublisher{}
	relay := outbox.NewRelay(storage.New(db, []byte("0123456789abcdef0123456789abcdef")).Outbox, pub, outbox.RelayConfig{Interval: time.Hour, Logger: testutil.DiscardLogger()})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("relay did not stop on context cancel")
	}
}

func TestRelay_Run_FlushErrorIsLoggedAndRetried(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	insertRow(t, db, "e1", "t", 0, 1000)

	require.NoError(t, db.Close())
	pub := &fakePublisher{}
	relay := outbox.NewRelay(s.Outbox, pub, outbox.RelayConfig{Interval: 5 * time.Millisecond, BatchSize: 10, Logger: testutil.DiscardLogger()})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()

	time.Sleep(30 * time.Millisecond)
	cancel()
	require.NoError(t, <-done)
	assert.Empty(t, pub.topics())
}

// TestRelay_PublishWithID_UsesRowID is the ws-33 regression: the relay must
// publish each pending row under the row's own ID, so a crash-redelivered
// row keeps the same bus event ID as the first attempt and consumers can
// dedupe it.
func TestRelay_PublishWithID_UsesRowID(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	insertRow(t, db, "row-1", "a.created", 0, 1000)
	insertRow(t, db, "row-2", "b.created", 0, 2000)

	pub := &fakePublisher{}
	relay := outbox.NewRelay(s.Outbox, pub, outbox.RelayConfig{Interval: 5 * time.Millisecond, BatchSize: 10, Logger: testutil.DiscardLogger()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = relay.Run(ctx) }()

	require.Eventually(t, func() bool {
		return len(pub.eventIDs()) == 2
	}, 2*time.Second, 5*time.Millisecond)
	assert.ElementsMatch(t, []string{"row-1", "row-2"}, pub.eventIDs())
}

// TestRelay_CrashRedelivery_DedupeEndToEnd covers the ws-33 AC end-to-end with
// the real bus: an outbox row is published under its row ID; after a simulated
// crash (row unmarked) the relay re-emits the same ID; a subscriber that
// already processed it skips, a subscriber that had not yet processed it runs.
func TestRelay_CrashRedelivery_DedupeEndToEnd(t *testing.T) {
	db := testutil.NewDB(t)
	require.NoError(t, storage.Migrate(db))
	st := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))

	bus := inprocess.New(inprocess.Options{
		Logger:      testutil.DiscardLogger(),
		DedupeStore: st.ProcessedEvents,
	})
	defer func() { require.NoError(t, bus.Close()) }()
	ctx := context.Background()

	var a, b atomic.Int32
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "consumer-a", "ticket.created", func(ctx context.Context, ev eventbus.Event) error {
		a.Add(1)
		return nil
	}))

	relay := outbox.NewRelay(st.Outbox, bus, outbox.RelayConfig{Interval: 5 * time.Millisecond, BatchSize: 10, Logger: testutil.DiscardLogger()})
	insertRow(t, db, "row-1", "ticket.created", 0, 1000)

	runCtx1, cancel1 := context.WithCancel(context.Background())
	done1 := make(chan error, 1)
	go func() { done1 <- relay.Run(runCtx1) }()
	require.Eventually(t, func() bool { return a.Load() == 1 }, time.Second, 5*time.Millisecond)
	cancel1()
	require.NoError(t, <-done1)

	// Simulate a crash: the row was not marked published, so the relay
	// re-emits it on the next boot. consumer-b subscribes on this "restart".
	_, err := db.Exec(`UPDATE outbox SET published = 0 WHERE id = 'row-1'`)
	require.NoError(t, err)
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "consumer-b", "ticket.created", func(ctx context.Context, ev eventbus.Event) error {
		b.Add(1)
		return nil
	}))

	runCtx2, cancel2 := context.WithCancel(context.Background())
	done2 := make(chan error, 1)
	go func() { done2 <- relay.Run(runCtx2) }()
	require.Eventually(t, func() bool { return b.Load() == 1 }, time.Second, 5*time.Millisecond)
	cancel2()
	require.NoError(t, <-done2)

	assert.Equal(t, int32(1), a.Load(), "consumer that already processed the row must not process it again")
	assert.Equal(t, int32(1), b.Load(), "consumer that had not processed the row must process the redelivery")

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(1) FROM outbox WHERE id = 'row-1' AND published = 1`).Scan(&n))
	assert.Equal(t, 1, n, "row must be marked published after the successful re-publish")
}
