package dns

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// newExposureFixture wires a gateway service plus a tunnel gateway on host1/net1 and a container lookup with
// "app"/"api" (host1, net1) and "other" (host2, net2), mirroring what a create-exposure caller needs already
// set up.
func newExposureFixture(t *testing.T) (*Service, *fakeRepo, *fakeTunnelProvider, *fakeContainerLookup, *Gateway) {
	t.Helper()
	repo := newFakeRepo()
	enc, err := encryptTunnelTokenForTest([]byte("0123456789abcdef0123456789abcdef"), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "prod", Token: enc}))
	tunnel := newFakeTunnelProvider()
	containers := newFakeContainerLookup()
	containers.add("app", ExposureTarget{ContainerID: "c-app", Name: "app", StackID: "s-app", ProjectID: "p1", Machine: "host1", Networks: []string{"net1"}, Running: true})
	containers.add("api", ExposureTarget{ContainerID: "c-api", Name: "api", StackID: "s-api", ProjectID: "p1", Machine: "host1", Networks: []string{"net1"}, Running: true})
	containers.add("other", ExposureTarget{ContainerID: "c-other", Name: "other", StackID: "s-other", ProjectID: "p1", Machine: "host2", Networks: []string{"net2"}, Running: true})
	s := newGatewayService(repo, tunnel, &fakeProvisioner{}, containers)
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayTunnel, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		TunnelID: "t1", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)
	return s, repo, tunnel, containers, g
}

func TestService_CreateExposure_Tunnel(t *testing.T) {
	s, repo, tunnel, _, g := newExposureFixture(t)

	e, err := s.CreateExposure(context.Background(), CreateExposureInput{
		GatewayID: g.ID, Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "app.example.com", e.Hostname)
	assert.Equal(t, "c-app", e.ServiceID)
	assert.Equal(t, "app", e.Service, "the legacy name column carries the resolved container name")
	assert.NotEmpty(t, e.RecordID)

	require.Len(t, tunnel.routeCalls, 1)
	assert.Equal(t, "t1", tunnel.routeCalls[0].TunnelID)
	assert.Equal(t, "app.example.com", tunnel.routeCalls[0].Hostname)
	assert.Equal(t, "http://app:8080", tunnel.routeCalls[0].Service, "origin helper builds http://<container-name>:<port>")
	assert.Contains(t, repo.topics(), TopicExposureChanged)
}

func TestService_CreateExposure_LegacyServiceNameFallback(t *testing.T) {
	s, _, tunnel, _, g := newExposureFixture(t)

	e, err := s.CreateExposure(context.Background(), CreateExposureInput{
		GatewayID: g.ID, Hostname: "app.example.com", Service: "app", Port: 8080, ZoneID: "z1", Zone: "example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "c-app", e.ServiceID, "the deprecated name resolves through the stack to its single container")
	require.Len(t, tunnel.routeCalls, 1)
}

// TestService_CreateExposure_TwoHostnamesOnOneTunnelSurvive is the
// gateway-level regression for the read-modify-write fix: two exposures on
// the same tunnel gateway must both route distinct hostnames, not clobber
// each other's local records or the tunnel's ingress rule.
func TestService_CreateExposure_TwoHostnamesOnOneTunnelSurvive(t *testing.T) {
	s, _, tunnel, _, g := newExposureFixture(t)

	_, err := s.CreateExposure(context.Background(), CreateExposureInput{GatewayID: g.ID, Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com"})
	require.NoError(t, err)
	_, err = s.CreateExposure(context.Background(), CreateExposureInput{GatewayID: g.ID, Hostname: "api.example.com", ServiceID: "c-api", Port: 9090, ZoneID: "z1", Zone: "example.com"})
	require.NoError(t, err)

	exps, err := s.ListExposures(context.Background())
	require.NoError(t, err)
	require.Len(t, exps, 2, "both exposures persist locally")

	require.Len(t, tunnel.routeCalls, 2)
	assert.Equal(t, "app.example.com", tunnel.routeCalls[0].Hostname)
	assert.Equal(t, "api.example.com", tunnel.routeCalls[1].Hostname)
}

func TestService_CreateExposure_MachineMismatchIsInvalid(t *testing.T) {
	s, _, _, _, g := newExposureFixture(t)

	_, err := s.CreateExposure(context.Background(), CreateExposureInput{
		GatewayID: g.ID, Hostname: "other.example.com", ServiceID: "c-other", Port: 8080, ZoneID: "z1", Zone: "example.com",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestService_CreateExposure_Proxy(t *testing.T) {
	repo := newFakeRepo()
	containers := newFakeContainerLookup()
	containers.add("web", ExposureTarget{ContainerID: "c-web", Name: "web", StackID: "s-web", ProjectID: "p1", Machine: "host1", Networks: []string{"net1"}, Running: true})
	s := newGatewayService(repo, newFakeTunnelProvider(), &fakeProvisioner{}, containers)
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		ServerAddress: "203.0.113.10", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)

	e, err := s.CreateExposure(context.Background(), CreateExposureInput{
		GatewayID: g.ID, Hostname: "web.example.com", ServiceID: "c-web", Port: 3000, ZoneID: "z1", Zone: "example.com",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, e.RecordID)
}

// TestService_CreateExposure_ReusesGatewayOnMachine covers the wizard's reach-step default: no gateway_id
// reuses the machine's existing gateway, preferring one already on the target's network.
func TestService_CreateExposure_ReusesGatewayOnMachine(t *testing.T) {
	s, _, tunnel, _, g := newExposureFixture(t)

	e, err := s.CreateExposure(context.Background(), CreateExposureInput{
		Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, g.ID, e.GatewayID, "the only gateway on host1 already sits on net1")
	require.Len(t, tunnel.routeCalls, 1)
}

// TestService_CreateExposure_ProvisionsGatewayWhenNoneExists covers the wizard's other branch: no gateway on
// the target's machine at all, so one is provisioned — proxy, since no Cloudflare token is configured.
func TestService_CreateExposure_ProvisionsGatewayWhenNoneExists(t *testing.T) {
	repo := newFakeRepo()
	containers := newFakeContainerLookup()
	containers.add("web", ExposureTarget{ContainerID: "c-web", Name: "web", StackID: "s-web", ProjectID: "p1", Machine: "host9", Networks: []string{"net9"}, Running: false})
	prov := &fakeProvisioner{}
	cfg := Config{
		Repo: repo, Provider: newFakeProvider(), TunnelProvider: newFakeTunnelProvider(), Provisioner: prov,
		Containers: containers, EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
		Settings: &fakeSettings{instanceURL: "https://deploy.example.com"},
		// No Cloudflare token configured: a proxy gateway is auto-provisioned, not a tunnel.
		Tokens: &fakeTokenProvider{err: apperrs.ErrNotFound},
		Now:    func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) },
	}
	s := NewService(cfg)

	e, err := s.CreateExposure(context.Background(), CreateExposureInput{
		Hostname: "web.example.com", ServiceID: "c-web", Port: 3000, ZoneID: "z1", Zone: "example.com",
	})
	require.NoError(t, err)
	require.NotEmpty(t, e.GatewayID)

	g, err := s.GetGateway(context.Background(), e.GatewayID)
	require.NoError(t, err)
	assert.Equal(t, GatewayProxy, g.Kind, "no tunnel is configured, so the default falls back to proxy")
	assert.Equal(t, "net9", g.DockerNetwork, "provisioned on the target's own network")
	assert.Equal(t, "host9", g.Machine)
	assert.Equal(t, "deploy.example.com", g.ServerAddress, "defaults to the instance's own host")
}

// TestService_CreateExposure_JoinsGatewayNetworks covers the network-union + immediate-runner-join step: a
// target on a network the gateway hasn't joined yet gets unioned in, and — since it's running — the runner
// seam is asked to join it right away.
func TestService_CreateExposure_JoinsGatewayNetworks(t *testing.T) {
	repo := newFakeRepo()
	containers := newFakeContainerLookup()
	containers.add("app", ExposureTarget{ContainerID: "c-app", Name: "app", StackID: "s-app", ProjectID: "p1", Machine: "host1", Networks: []string{"net1", "net2"}, Running: true})
	joiner := &fakeRunnerJoiner{}
	s := newGatewayServiceWithJoiner(repo, newFakeTunnelProvider(), &fakeProvisioner{}, containers, joiner)
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		ServerAddress: "1.2.3.4", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)
	// The fixture's container lookup keys the gateway's own backing container by its stack id.
	containers.byStackID[g.ServiceID] = &ExposureTarget{ContainerID: "gwc1", Name: "gateway-container", StackID: g.ServiceID, Machine: "host1"}

	_, err = s.CreateExposure(context.Background(), CreateExposureInput{
		GatewayID: g.ID, Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com",
	})
	require.NoError(t, err)

	updated, err := s.GetGateway(context.Background(), g.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"net1", "net2"}, updated.Networks, "net2 is unioned in alongside the home network")

	require.Len(t, joiner.calls, 1)
	assert.Equal(t, "host1", joiner.calls[0].Machine)
	assert.Equal(t, "gateway-container", joiner.calls[0].GatewayContainer)
	assert.Equal(t, []string{"net2"}, joiner.calls[0].Networks, "only the newly-joined network is sent, not the whole set")
}

func TestService_ExposuresForService(t *testing.T) {
	s, _, _, _, g := newExposureFixture(t)
	_, err := s.CreateExposure(context.Background(), CreateExposureInput{GatewayID: g.ID, Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com"})
	require.NoError(t, err)

	summaries, err := s.ExposuresForService(context.Background(), "app")
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "app.example.com", summaries[0].Hostname)
	assert.Equal(t, 8080, summaries[0].Port)

	none, err := s.ExposuresForService(context.Background(), "unrelated")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestService_ExposuresForContainer(t *testing.T) {
	s, _, _, _, g := newExposureFixture(t)
	_, err := s.CreateExposure(context.Background(), CreateExposureInput{GatewayID: g.ID, Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com"})
	require.NoError(t, err)

	summaries, err := s.ExposuresForContainer(context.Background(), "c-app")
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, 8080, summaries[0].Port)

	none, err := s.ExposuresForContainer(context.Background(), "c-ghost")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestService_DeleteExposure_Tunnel(t *testing.T) {
	s, repo, tunnel, _, g := newExposureFixture(t)
	e, err := s.CreateExposure(context.Background(), CreateExposureInput{GatewayID: g.ID, Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com"})
	require.NoError(t, err)

	require.NoError(t, s.DeleteExposure(context.Background(), e.ID))
	require.Len(t, tunnel.removeCalls, 1)
	assert.Equal(t, "app.example.com", tunnel.removeCalls[0].Hostname)
	_, err = repo.GetExposure(context.Background(), e.ID)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

// TestService_DeleteExposure_OneOfTwoKeepsTheOther is the delete-side
// regression: removing one exposure's ingress rule must not disturb a
// sibling exposure routed on the same tunnel.
func TestService_DeleteExposure_OneOfTwoKeepsTheOther(t *testing.T) {
	s, repo, tunnel, _, g := newExposureFixture(t)
	e1, err := s.CreateExposure(context.Background(), CreateExposureInput{GatewayID: g.ID, Hostname: "app.example.com", ServiceID: "c-app", Port: 8080, ZoneID: "z1", Zone: "example.com"})
	require.NoError(t, err)
	_, err = s.CreateExposure(context.Background(), CreateExposureInput{GatewayID: g.ID, Hostname: "api.example.com", ServiceID: "c-api", Port: 9090, ZoneID: "z1", Zone: "example.com"})
	require.NoError(t, err)

	require.NoError(t, s.DeleteExposure(context.Background(), e1.ID))
	require.Len(t, tunnel.removeCalls, 1)
	assert.Equal(t, "app.example.com", tunnel.removeCalls[0].Hostname)

	exps, err := s.ListExposures(context.Background())
	require.NoError(t, err)
	require.Len(t, exps, 1)
	assert.Equal(t, "api.example.com", exps[0].Hostname, "the sibling exposure survives")
	_ = repo
}

// TestService_DeleteExposuresForStack covers the stack-delete cleanup: a branch-deploy exposure keyed only
// by the legacy service name, a wizard/UI exposure keyed only by a container id with a different service
// name, and an exposure that happens to match both lookups (deleted once, not twice) must all be released,
// while an exposure on an unrelated stack survives.
func TestService_DeleteExposuresForStack(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	provider := newFakeProvider()
	tunnel := newFakeTunnelProvider()
	enc, err := encryptTunnelTokenForTest([]byte("0123456789abcdef0123456789abcdef"), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(ctx, Tunnel{ID: "t1", Name: "prod", Token: enc}))
	require.NoError(t, repo.SaveGateway(ctx, Gateway{ID: "gw1", Kind: GatewayTunnel, TunnelID: "t1", ZoneID: "z1", Zone: "example.com", Machine: "host1"}))
	require.NoError(t, repo.SaveGateway(ctx, Gateway{ID: "gw2", Kind: GatewayTunnel, TunnelID: "t1", ZoneID: "z1", Zone: "example.com", Machine: "host2"}))
	s := NewService(Config{
		Repo: repo, Provider: provider, TunnelProvider: tunnel,
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
		Settings:      &fakeSettings{instanceURL: "https://deploy.example.com"},
		Tokens:        &fakeTokenProvider{token: "at"},
		Now:           func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) },
	})

	seed := func(id, gatewayID, hostname, serviceID, service string, port int) *Exposure {
		rec, err := provider.CreateRecord(ctx, "z1", RecordInput{Type: RecordCNAME, Name: hostname, Content: "t1.cfargotunnel.com", TTL: 1, Proxied: true})
		require.NoError(t, err)
		e := Exposure{ID: id, GatewayID: gatewayID, Hostname: hostname, ServiceID: serviceID, Service: service, Port: port, ZoneID: "z1", Zone: "example.com", RecordID: rec.ID}
		require.NoError(t, repo.SaveExposure(ctx, e))
		return &e
	}

	// Branch-deploy style: legacy row keyed only by the stack name, no service_id.
	seed("exp-legacy", "gw1", "stack.example.com", "", "stack", 8080)
	// Wizard/UI style: keyed by a container id, the service column holds the container's runtime name.
	seed("exp-wizard", "gw1", "worker.example.com", "c1", "worker", 9090)
	// Matches both lookups at once: stack name AND one of the passed container ids.
	seed("exp-dual", "gw1", "dual.example.com", "c1", "stack", 7070)
	// Unrelated stack's exposure must survive.
	unrelated := seed("exp-unrelated", "gw2", "sidecar.example.com", "c-other", "sidecar", 1234)

	require.NoError(t, s.DeleteExposuresForStack(ctx, "stack", []string{"c1"}))

	exps, err := s.ListExposures(ctx)
	require.NoError(t, err)
	require.Len(t, exps, 1, "only the unrelated exposure survives")
	assert.Equal(t, unrelated.ID, exps[0].ID)

	require.Len(t, tunnel.removeCalls, 3, "legacy, wizard, and dual are each removed exactly once")
	var removedHostnames []string
	for _, c := range tunnel.removeCalls {
		removedHostnames = append(removedHostnames, c.Hostname)
	}
	assert.ElementsMatch(t, []string{"stack.example.com", "worker.example.com", "dual.example.com"}, removedHostnames)

	remaining, err := provider.ListRecords(ctx, "z1")
	require.NoError(t, err)
	require.Len(t, remaining, 1, "only the unrelated exposure's DNS record remains")
	assert.Equal(t, unrelated.RecordID, remaining[0].ID)
}
