package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestDNSRepo_TunnelRoundTrip(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()

	_, err := repo.GetTunnel(ctx, "t1")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound), "no tunnel before save")

	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	tunnel := dns.Tunnel{
		ID: "t1", Name: "tunnel-1", AccountID: "acct-1", Status: "inactive",
		Hostname: "app.example.com", ZoneID: "z1", Zone: "example.com",
		RecordID: "r1", Service: "http://localhost:80", Token: "cipher-token",
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.SaveTunnel(ctx, tunnel))

	got, err := repo.GetTunnel(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "tunnel-1", got.Name)
	assert.Equal(t, "cipher-token", got.Token, "token blob round-trips untouched")
	assert.Equal(t, "app.example.com", got.Hostname)

	tunnels, err := repo.ListTunnels(ctx)
	require.NoError(t, err)
	assert.Len(t, tunnels, 1)
}

func TestDNSRepo_TunnelUpsert(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()

	require.NoError(t, repo.SaveTunnel(ctx, dns.Tunnel{ID: "t1", Name: "one"}))
	require.NoError(t, repo.SaveTunnel(ctx, dns.Tunnel{ID: "t1", Name: "renamed", Hostname: "app.example.com"}))

	got, err := repo.GetTunnel(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "renamed", got.Name)
	assert.Equal(t, "app.example.com", got.Hostname)

	tunnels, err := repo.ListTunnels(ctx)
	require.NoError(t, err)
	assert.Len(t, tunnels, 1, "upsert replaces, never duplicates")
}

func TestDNSRepo_TunnelSaveWithOutbox(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()

	evt := eventbus.OutboxEvent{ID: "e1", Topic: dns.TopicTunnelChanged, Payload: dns.TunnelChangedEvent{TunnelID: "t1", Action: "created"}}
	require.NoError(t, repo.SaveTunnel(ctx, dns.Tunnel{ID: "t1", Name: "one"}, evt))

	events, err := store.Outbox.Unpublished(ctx, 50)
	require.NoError(t, err)
	found := false
	for _, e := range events {
		if e.ID == "e1" && e.Topic == dns.TopicTunnelChanged {
			found = true
		}
	}
	assert.True(t, found, "tunnel_changed outbox row written in the same tx")
}

func TestDNSRepo_TunnelDelete(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()

	require.NoError(t, repo.SaveTunnel(ctx, dns.Tunnel{ID: "t1", Name: "one"}))
	require.NoError(t, repo.DeleteTunnel(ctx, "t1", eventbus.OutboxEvent{
		ID: "e1", Topic: dns.TopicTunnelChanged, Payload: dns.TunnelChangedEvent{TunnelID: "t1", Action: "deleted"},
	}))

	_, err := repo.GetTunnel(ctx, "t1")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))

	err = repo.DeleteTunnel(ctx, "t1")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound), "deleting an absent tunnel is an error")
}
