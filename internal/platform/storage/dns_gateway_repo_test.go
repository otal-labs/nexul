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

func TestDNSRepo_GatewayRoundTrip(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()

	_, err := repo.GetGateway(ctx, "g1")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound), "no gateway before save")

	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	g := dns.Gateway{
		ID: "g1", Kind: dns.GatewayTunnel, DockerNetwork: "net1", Machine: "host1", Networks: []string{"net1", "net2"},
		ServiceID: "svc1", ServiceName: "cloudflared-prod", TunnelID: "t1",
		ZoneID: "z1", Zone: "example.com", CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.SaveGateway(ctx, g))

	got, err := repo.GetGateway(ctx, "g1")
	require.NoError(t, err)
	assert.Equal(t, dns.GatewayTunnel, got.Kind)
	assert.Equal(t, "net1", got.DockerNetwork)
	assert.Equal(t, "host1", got.Machine)
	assert.ElementsMatch(t, []string{"net1", "net2"}, got.Networks)
	assert.Equal(t, "t1", got.TunnelID)

	byNet, err := repo.GetGatewayByNetwork(ctx, "net1")
	require.NoError(t, err)
	assert.Equal(t, "g1", byNet.ID)

	_, err = repo.GetGatewayByNetwork(ctx, "no-such-network")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))

	list, err := repo.ListGateways(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	byMachine, err := repo.ListGatewaysByMachine(ctx, "host1")
	require.NoError(t, err)
	assert.Len(t, byMachine, 1)

	byMachine, err = repo.ListGatewaysByMachine(ctx, "no-such-machine")
	require.NoError(t, err)
	assert.Empty(t, byMachine)
}

func TestDNSRepo_GatewayNetworkIsUnique(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()

	require.NoError(t, repo.SaveGateway(ctx, dns.Gateway{ID: "g1", Kind: dns.GatewayProxy, DockerNetwork: "net1"}))
	err := repo.SaveGateway(ctx, dns.Gateway{ID: "g2", Kind: dns.GatewayProxy, DockerNetwork: "net1"})
	require.Error(t, err, "the docker_network UNIQUE constraint backstops the use-case-level check")
}

func TestDNSRepo_GatewayDelete(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()

	require.NoError(t, repo.SaveGateway(ctx, dns.Gateway{ID: "g1", Kind: dns.GatewayProxy, DockerNetwork: "net1"}))
	require.NoError(t, repo.DeleteGateway(ctx, "g1", eventbus.OutboxEvent{
		ID: "e1", Topic: dns.TopicGatewayChanged, Payload: dns.GatewayChangedEvent{GatewayID: "g1", Action: "deleted"},
	}))

	_, err := repo.GetGateway(ctx, "g1")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))

	err = repo.DeleteGateway(ctx, "g1")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound), "deleting an absent gateway is an error")
}

func TestDNSRepo_ExposureRoundTrip(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()
	require.NoError(t, repo.SaveGateway(ctx, dns.Gateway{ID: "g1", Kind: dns.GatewayTunnel, DockerNetwork: "net1"}))

	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	e := dns.Exposure{
		ID: "e1", GatewayID: "g1", Hostname: "app.example.com", Service: "app", ServiceID: "c-app", Port: 8080,
		ZoneID: "z1", Zone: "example.com", RecordID: "r1", CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.SaveExposure(ctx, e))

	got, err := repo.GetExposure(ctx, "e1")
	require.NoError(t, err)
	assert.Equal(t, "app.example.com", got.Hostname)
	assert.Equal(t, "c-app", got.ServiceID)
	assert.Equal(t, 8080, got.Port)

	byGateway, err := repo.ListExposuresByGateway(ctx, "g1")
	require.NoError(t, err)
	assert.Len(t, byGateway, 1)

	byServiceName, err := repo.ListExposuresByServiceName(ctx, "app")
	require.NoError(t, err)
	assert.Len(t, byServiceName, 1)

	byServiceName, err = repo.ListExposuresByServiceName(ctx, "unrelated")
	require.NoError(t, err)
	assert.Empty(t, byServiceName)

	byService, err := repo.ListExposuresByService(ctx, "c-app")
	require.NoError(t, err)
	assert.Len(t, byService, 1)

	byService, err = repo.ListExposuresByService(ctx, "c-unrelated")
	require.NoError(t, err)
	assert.Empty(t, byService)

	all, err := repo.ListExposures(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestDNSRepo_ExposureDelete(t *testing.T) {
	store := openDNSStore(t)
	repo := store.DNS
	ctx := context.Background()
	require.NoError(t, repo.SaveGateway(ctx, dns.Gateway{ID: "g1", Kind: dns.GatewayTunnel, DockerNetwork: "net1"}))
	require.NoError(t, repo.SaveExposure(ctx, dns.Exposure{ID: "e1", GatewayID: "g1", Hostname: "app.example.com", Service: "app", Port: 8080}))

	require.NoError(t, repo.DeleteExposure(ctx, "e1", eventbus.OutboxEvent{
		ID: "ev1", Topic: dns.TopicExposureChanged, Payload: dns.ExposureChangedEvent{ExposureID: "e1", Action: "deleted"},
	}))

	_, err := repo.GetExposure(ctx, "e1")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}
