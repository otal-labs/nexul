package main

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/tickets"
)

// TestEventDedupe_LivePushAndFanoutWriteNoProcessedRows: the production subscriptions record a processed_events row
// only for consumers a redelivery could harm; the live push and the integration fan-out already survive one.
func TestEventDedupe_LivePushAndFanoutWriteNoProcessedRows(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	ctx := t.Context()
	key := []byte("0123456789abcdef0123456789abcdef")
	store := storage.New(db, key)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: string(key)}
	svc := wireCoreServices(cfg, store, key, bus, logger)
	wireDomainEventSubscriptions(ctx, bus, svc, logger)
	wireIntegrationFanout(ctx, bus, store, svc, logger)
	wireLiveHubAndAgent(ctx, bus, store, svc, logger)
	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Interval: time.Hour, Wake: store.Commits().Next, Logger: logger})
	go func() { _ = relay.Run(ctx) }()

	const events = 3
	for i := range events {
		_, err := db.Exec(`INSERT INTO outbox (id, topic, payload) VALUES (?, ?, ?)`, fmt.Sprintf("ev-%d", i), tickets.TopicUpdated, `{"ticket":{"id":"t-1"}}`)
		require.NoError(t, err)
	}
	store.Commits().Notify()

	count := func(consumer string) (n int) {
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM processed_events WHERE event_id LIKE ?`, consumer+":%").Scan(&n))
		return n
	}
	require.Eventually(t, func() bool { return count("notifications") == events }, 5*time.Second, 10*time.Millisecond, "a deduped consumer records every event")
	require.NoError(t, bus.Close())

	assert.Zero(t, count("live.push"))
	assert.Zero(t, count("integrations.fanout"))
}
