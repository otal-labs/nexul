package storage

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func openDNSStore(t *testing.T) *Store {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return New(db, testEncKey)
}

func TestDNSRepo_ServiceHostnameLifecycle(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	sh := dns.ServiceHostname{
		Service: "api", Hostname: "api.example.com", ZoneID: "z1", Zone: "example.com",
		RecordID: "r1", Type: dns.RecordA, Content: "1.2.3.4", CreatedAt: now,
	}

	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: dns.TopicRecordChanged, Payload: map[string]any{"action": "created"}}
	require.NoError(t, repo.UpsertServiceHostname(ctx, sh, evt))

	got, err := repo.GetServiceHostname(ctx, "api")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com", got.Hostname)

	list, err := repo.ListServiceHostnames(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	row, err := store.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, row, 1)
	assert.Equal(t, dns.TopicRecordChanged, row[0].Topic)

	// Upsert replaces the association and appends another outbox row.
	sh.Content = "5.6.7.8"
	require.NoError(t, repo.UpsertServiceHostname(ctx, sh, eventbus.OutboxEvent{ID: "evt-2", Topic: dns.TopicRecordChanged, Payload: map[string]any{"action": "updated"}}))
	got, err = repo.GetServiceHostname(ctx, "api")
	require.NoError(t, err)
	assert.Equal(t, "5.6.7.8", got.Content)

	delEvt := eventbus.OutboxEvent{ID: "evt-3", Topic: dns.TopicRecordChanged, Payload: map[string]any{"action": "deleted"}}
	require.NoError(t, repo.DeleteServiceHostname(ctx, "api", delEvt))
	_, err = repo.GetServiceHostname(ctx, "api")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))

	// Deleting an absent association is not found.
	err = repo.DeleteServiceHostname(ctx, "api", delEvt)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestDNSRepo_RecordChangedWritesOutboxRow(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()
	payload := dns.RecordChangedEvent{ZoneID: "z1", RecordID: "r1", Action: "deleted"}
	require.NoError(t, repo.RecordChanged(ctx, eventbus.OutboxEvent{ID: "evt-x", Topic: dns.TopicRecordChanged, Payload: payload}))

	row, err := store.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	require.Len(t, row, 1)
	var got dns.RecordChangedEvent
	require.NoError(t, json.Unmarshal(row[0].Payload, &got))
	assert.Equal(t, "deleted", got.Action)
}
