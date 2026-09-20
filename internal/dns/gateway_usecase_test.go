package dns

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestService_CreateGateway_Tunnel(t *testing.T) {
	repo := newFakeRepo()
	enc, err := encryptTunnelTokenForTest([]byte("0123456789abcdef0123456789abcdef"), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "prod", Token: enc}))
	prov := &fakeProvisioner{}
	s := newGatewayService(repo, newFakeTunnelProvider(), prov, newFakeContainerLookup())

	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayTunnel, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		TunnelID: "t1", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)
	assert.Equal(t, GatewayTunnel, g.Kind)
	assert.Equal(t, "net1", g.DockerNetwork)
	assert.Equal(t, "host1", g.Machine)
	assert.Equal(t, []string{"net1"}, g.Networks, "the home network already counts as joined")
	assert.Equal(t, "t1", g.TunnelID)
	assert.Equal(t, "cloudflared-prod", g.ServiceName)
	assert.Equal(t, "svc-1", g.ServiceID)
	require.Len(t, prov.calls, 1)
	assert.Equal(t, "net1", prov.calls[0].DockerNetwork)
	assert.NotEmpty(t, prov.calls[0].Env["TUNNEL_TOKEN"], "cloudflared gets the decrypted tunnel token")
	assert.Contains(t, repo.topics(), TopicGatewayChanged)
}

func TestService_CreateGateway_Proxy(t *testing.T) {
	repo := newFakeRepo()
	prov := &fakeProvisioner{}
	s := newGatewayService(repo, newFakeTunnelProvider(), prov, newFakeContainerLookup())

	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		ServerAddress: "203.0.113.10", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)
	assert.Equal(t, GatewayProxy, g.Kind)
	assert.Equal(t, "203.0.113.10", g.ServerAddress)
	require.Len(t, prov.calls, 1)
	spec := prov.calls[0]
	assert.Equal(t, traefikImage, spec.Image)
	assert.ElementsMatch(t, []string{"80:80", "443:443"}, spec.Ports)
	assert.Contains(t, spec.Mounts, dockerSocketMount)
	assert.Equal(t, "false", spec.Env["TRAEFIK_PROVIDERS_DOCKER_EXPOSEDBYDEFAULT"])
}

func TestService_CreateGateway_NetworkAlreadyHasGateway(t *testing.T) {
	repo := newFakeRepo()
	s := newGatewayService(repo, newFakeTunnelProvider(), &fakeProvisioner{}, newFakeContainerLookup())
	in := CreateGatewayInput{Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com", ServerAddress: "1.2.3.4", ProjectID: "p1", Target: "host1"}
	_, err := s.CreateGateway(context.Background(), in)
	require.NoError(t, err)

	_, err = s.CreateGateway(context.Background(), in)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrConflict))
}

func TestService_CreateGateway_Validation(t *testing.T) {
	s := newGatewayService(newFakeRepo(), newFakeTunnelProvider(), &fakeProvisioner{}, newFakeContainerLookup())

	t.Run("bad kind", func(t *testing.T) {
		_, err := s.CreateGateway(context.Background(), CreateGatewayInput{Kind: "vpn", DockerNetwork: "n", ZoneID: "z", Zone: "z.com", ProjectID: "p", Target: "t"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("tunnel kind needs tunnel id", func(t *testing.T) {
		_, err := s.CreateGateway(context.Background(), CreateGatewayInput{Kind: GatewayTunnel, DockerNetwork: "n", ZoneID: "z", Zone: "z.com", ProjectID: "p", Target: "t"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})

	t.Run("proxy kind needs server address", func(t *testing.T) {
		_, err := s.CreateGateway(context.Background(), CreateGatewayInput{Kind: GatewayProxy, DockerNetwork: "n", ZoneID: "z", Zone: "z.com", ProjectID: "p", Target: "t"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestService_ListAndGetGateway(t *testing.T) {
	repo := newFakeRepo()
	s := newGatewayService(repo, newFakeTunnelProvider(), &fakeProvisioner{}, newFakeContainerLookup())
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		ServerAddress: "1.2.3.4", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)

	list, err := s.ListGateways(context.Background())
	require.NoError(t, err)
	assert.Len(t, list, 1)

	got, err := s.GetGateway(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Equal(t, g.ID, got.ID)

	_, err = s.GetGateway(context.Background(), "ghost")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestService_DeleteGateway_Deprovisions(t *testing.T) {
	repo := newFakeRepo()
	prov := &fakeProvisioner{}
	s := newGatewayService(repo, newFakeTunnelProvider(), prov, newFakeContainerLookup())
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		ServerAddress: "1.2.3.4", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)

	require.NoError(t, s.DeleteGateway(context.Background(), g.ID))
	assert.Contains(t, prov.deprovisionCalls, g.ServiceID)
	_, err = repo.GetGateway(context.Background(), g.ID)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestService_DeleteGateway_FailsWithExposures(t *testing.T) {
	repo := newFakeRepo()
	s := newGatewayService(repo, newFakeTunnelProvider(), &fakeProvisioner{}, newFakeContainerLookup())
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		ServerAddress: "1.2.3.4", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)
	require.NoError(t, repo.SaveExposure(context.Background(), Exposure{ID: "e1", GatewayID: g.ID, Hostname: "app.example.com", Service: "app", Port: 8080}))

	err = s.DeleteGateway(context.Background(), g.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrConflict))
}

func TestService_GatewayForStackNetwork(t *testing.T) {
	repo := newFakeRepo()
	containers := newFakeContainerLookup()
	s := newGatewayService(repo, newFakeTunnelProvider(), &fakeProvisioner{}, containers)
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		ServerAddress: "1.2.3.4", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)
	containers.add("gateway-stack", ExposureTarget{
		ContainerID: "gwc1", Name: "gateway-container", StackID: g.ServiceID, Machine: "host1",
	})

	container, found, err := s.GatewayForStackNetwork(context.Background(), "net1")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "gateway-container", container)

	_, found, err = s.GatewayForStackNetwork(context.Background(), "unrelated-net")
	require.NoError(t, err)
	assert.False(t, found)
}
