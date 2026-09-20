package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/integrations"
)

func newTestIntegrationsStore(t *testing.T) *Store {
	t.Helper()
	return newTestStore(t)
}

func TestIntegrationInstallsRepo(t *testing.T) {
	store := newTestIntegrationsStore(t)
	repo := store.IntegrationInstalls

	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	install := &integrations.Install{
		ID:            "i1",
		Name:          "discord",
		TrustTier:     integrations.TrustCommunity,
		WebhookURL:    "https://example.com/hook",
		WebhookSecret: "secret",
		Scopes:        []integrations.Scope{integrations.ScopeEventsRead},
		CreatedBy:     "owner-1",
		CreatedAt:     now,
	}
	require.NoError(t, repo.Create(ctx, install))

	t.Run("get by id", func(t *testing.T) {
		got, err := repo.GetByID(ctx, "i1")
		require.NoError(t, err)
		assert.Equal(t, "discord", got.Name)
		assert.Equal(t, "secret", got.WebhookSecret)
		assert.Equal(t, []integrations.Scope{integrations.ScopeEventsRead}, got.Scopes)
		assert.Equal(t, now, got.CreatedAt)
	})

	t.Run("list", func(t *testing.T) {
		got, err := repo.List(ctx)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("get unknown is not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "nope")
		require.Error(t, err)
	})

	t.Run("revoke + verify", func(t *testing.T) {
		require.NoError(t, repo.Revoke(ctx, "i1"))
		got, err := repo.GetByID(ctx, "i1")
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt)
	})

	t.Run("revoke unknown is not found", func(t *testing.T) {
		err := repo.Revoke(ctx, "nope")
		require.Error(t, err)
	})

	t.Run("revoke twice is not found", func(t *testing.T) {
		err := repo.Revoke(ctx, "i1")
		require.Error(t, err)
	})
}

func TestIntegrationTokensRepo(t *testing.T) {
	store := newTestIntegrationsStore(t)
	installRepo := store.IntegrationInstalls
	tokenRepo := store.IntegrationTokens

	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	require.NoError(t, installRepo.Create(ctx, &integrations.Install{
		ID: "i1", Name: "x", TrustTier: integrations.TrustVerified, WebhookSecret: "s",
		Scopes: []integrations.Scope{integrations.ScopeDocsRead}, CreatedBy: "u", CreatedAt: now,
	}))

	require.NoError(t, tokenRepo.Create(ctx, &integrations.IntegrationToken{
		ID: "t1", InstallID: "i1", TokenHash: "hash1", Prefix: "abc123", CreatedAt: now,
	}))

	t.Run("get by hash", func(t *testing.T) {
		got, err := tokenRepo.GetByHash(ctx, "hash1")
		require.NoError(t, err)
		assert.Equal(t, "i1", got.InstallID)
	})

	t.Run("get unknown hash", func(t *testing.T) {
		_, err := tokenRepo.GetByHash(ctx, "nope")
		require.Error(t, err)
	})

	t.Run("list by install", func(t *testing.T) {
		got, err := tokenRepo.ListByInstall(ctx, "i1")
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("revoke by install", func(t *testing.T) {
		require.NoError(t, tokenRepo.RevokeByInstall(ctx, "i1"))
		got, err := tokenRepo.GetByHash(ctx, "hash1")
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt)
	})
}

func TestIntegrationSubscriptionsRepo(t *testing.T) {
	store := newTestIntegrationsStore(t)
	installRepo := store.IntegrationInstalls
	subRepo := store.IntegrationSubs

	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	require.NoError(t, installRepo.Create(ctx, &integrations.Install{
		ID: "i1", Name: "x", TrustTier: integrations.TrustVerified, WebhookSecret: "s",
		Scopes: []integrations.Scope{integrations.ScopeEventsRead}, CreatedBy: "u", CreatedAt: now,
	}))

	require.NoError(t, subRepo.Add(ctx, integrations.Subscription{InstallID: "i1", Topic: "ticket.created", CreatedAt: now}))

	t.Run("duplicate conflicts", func(t *testing.T) {
		err := subRepo.Add(ctx, integrations.Subscription{InstallID: "i1", Topic: "ticket.created", CreatedAt: now})
		require.Error(t, err)
	})

	t.Run("list by install", func(t *testing.T) {
		got, err := subRepo.ListByInstall(ctx, "i1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "ticket.created", got[0].Topic)
	})

	t.Run("list by topic", func(t *testing.T) {
		got, err := subRepo.ListByTopic(ctx, "ticket.created")
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("remove", func(t *testing.T) {
		require.NoError(t, subRepo.Remove(ctx, "i1", "ticket.created"))
		got, err := subRepo.ListByInstall(ctx, "i1")
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("remove missing is not found", func(t *testing.T) {
		err := subRepo.Remove(ctx, "i1", "ticket.created")
		require.Error(t, err)
	})
}

func TestIntegrationDeliveriesRepo(t *testing.T) {
	store := newTestIntegrationsStore(t)
	installRepo := store.IntegrationInstalls
	deliveryRepo := store.IntegrationDeliveries

	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	require.NoError(t, installRepo.Create(ctx, &integrations.Install{
		ID: "i1", Name: "x", TrustTier: integrations.TrustVerified, WebhookSecret: "s",
		Scopes: []integrations.Scope{integrations.ScopeEventsRead}, CreatedBy: "u", CreatedAt: now,
	}))

	delivery := &integrations.Delivery{
		ID: "d1", InstallID: "i1", Topic: "ticket.created", EventID: "evt-1",
		Payload: []byte(`{"delivery_id":"d1"}`), Signature: "sha256=abc", URL: "https://example.com",
		Status: integrations.DeliveryPending, Attempts: 0, NextAttemptAt: now, CreatedAt: now,
	}
	require.NoError(t, deliveryRepo.Create(ctx, delivery))

	t.Run("duplicate event is idempotent", func(t *testing.T) {
		dup := *delivery
		dup.ID = "d2"
		require.NoError(t, deliveryRepo.Create(ctx, &dup))
		got, err := deliveryRepo.ListByInstall(ctx, "i1")
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("due returns pending + failed", func(t *testing.T) {
		due, err := deliveryRepo.Due(ctx, 10, now.Add(time.Minute).Unix())
		require.NoError(t, err)
		require.Len(t, due, 1)
	})

	t.Run("future next_attempt not due", func(t *testing.T) {
		require.NoError(t, deliveryRepo.MarkFailed(ctx, "d1", 1, now.Add(time.Hour).Unix()))
		due, err := deliveryRepo.Due(ctx, 10, now.Add(time.Minute).Unix())
		require.NoError(t, err)
		assert.Empty(t, due)
	})

	t.Run("mark delivered", func(t *testing.T) {
		require.NoError(t, deliveryRepo.MarkDelivered(ctx, "d1", now.Add(time.Hour).Unix()))
		got, err := deliveryRepo.ListByInstall(ctx, "i1")
		require.NoError(t, err)
		assert.Equal(t, integrations.DeliveryDelivered, got[0].Status)
		require.NotNil(t, got[0].DeliveredAt)
	})

	t.Run("mark dead bumps attempts", func(t *testing.T) {
		require.NoError(t, deliveryRepo.MarkFailed(ctx, "d1", 1, now.Unix()))
		require.NoError(t, deliveryRepo.MarkDead(ctx, "d1"))
		got, err := deliveryRepo.ListByInstall(ctx, "i1")
		require.NoError(t, err)
		assert.Equal(t, integrations.DeliveryDead, got[0].Status)
		assert.Equal(t, 2, got[0].Attempts)
	})
}

func TestEventSchemasRepo(t *testing.T) {
	store := newTestIntegrationsStore(t)
	repo := store.EventSchemas
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Publish(ctx, integrations.SchemaEntry{Topic: "ticket.created", Version: 1, Schema: `{"type":"object"}`, CreatedAt: now}))

	t.Run("publish is idempotent", func(t *testing.T) {
		require.NoError(t, repo.Publish(ctx, integrations.SchemaEntry{Topic: "ticket.created", Version: 1, Schema: `{"type":"string"}`, CreatedAt: now}))
		entries, err := repo.Catalog(ctx)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, `{"type":"object"}`, entries[0].Schema)
	})

	t.Run("catalog lists all versions", func(t *testing.T) {
		require.NoError(t, repo.Publish(ctx, integrations.SchemaEntry{Topic: "ticket.created", Version: 2, Schema: `{"type":"object","x":1}`, CreatedAt: now}))
		entries, err := repo.Catalog(ctx)
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})
}

func TestAuditRepo(t *testing.T) {
	store := newTestIntegrationsStore(t)
	repo := store.Audit
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Append(ctx, integrations.AuditEntry{
		ID: "a1", ActorType: "integration", ActorID: "install-1", TokenID: "tok-1", Action: "GET /api/tickets", CreatedAt: now,
	}))
	require.NoError(t, repo.Append(ctx, integrations.AuditEntry{
		ID: "a2", ActorType: "user", ActorID: "owner-1", Action: "GET /api/docs", CreatedAt: now.Add(time.Second),
	}))

	t.Run("list newest first", func(t *testing.T) {
		entries, err := repo.List(ctx, 10)
		require.NoError(t, err)
		require.Len(t, entries, 2)
		assert.Equal(t, "a2", entries[0].ID)
		assert.Equal(t, "tok-1", entries[1].TokenID)
	})

	t.Run("limit respected", func(t *testing.T) {
		entries, err := repo.List(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})
}
