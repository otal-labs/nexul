package dns

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routesFragment struct {
	Routers map[string]struct {
		Rule        string            `json:"rule"`
		EntryPoints []string          `json:"entryPoints"`
		Service     string            `json:"service"`
		TLS         map[string]string `json:"tls"`
	} `json:"routers"`
	Services map[string]struct {
		LoadBalancer struct {
			Servers []struct {
				URL string `json:"url"`
			} `json:"servers"`
		} `json:"loadBalancer"`
	} `json:"services"`
}

func parseRoutes(t *testing.T, raw string) routesFragment {
	t.Helper()
	var f routesFragment
	require.NoError(t, json.Unmarshal([]byte(raw), &f))
	return f
}

func TestProxyRoutes_RouteEachExposureToItsContainerOverTLS(t *testing.T) {
	f := parseRoutes(t, proxyRoutes([]*Exposure{{ID: "e1", Hostname: "web.example.com", Service: "webby-web-1", Port: 8000}}))

	name := RouterName("e1")
	require.Contains(t, f.Routers, name)
	r := f.Routers[name]
	assert.Equal(t, "Host(`web.example.com`)", r.Rule)
	assert.Equal(t, []string{"websecure"}, r.EntryPoints)
	assert.Equal(t, name, r.Service)
	assert.Equal(t, certResolver, r.TLS["certResolver"])
	require.Len(t, f.Services[name].LoadBalancer.Servers, 1)
	assert.Equal(t, "http://webby-web-1:8000", f.Services[name].LoadBalancer.Servers[0].URL)
}

func TestProxyRoutes_NoExposuresIsAnEmptyFragmentThatClearsTheOldRoutes(t *testing.T) {
	assert.JSONEq(t, `{"routers":{},"services":{}}`, proxyRoutes(nil))
}

// proxyFixture is a proxy gateway on host1 with its Traefik container "gw-traefik" and an app container on net2.
func proxyFixture(t *testing.T) (*Service, *fakeRepo, *fakeRunnerJoiner, *Gateway) {
	t.Helper()
	repo := newFakeRepo()
	containers := newFakeContainerLookup()
	containers.add("web", ExposureTarget{ContainerID: "c-web", Name: "web-1", StackID: "s-web", Machine: "host1", Networks: []string{"net2"}, Running: true})
	joiner := &fakeRunnerJoiner{}
	s := newGatewayServiceWithJoiner(repo, newFakeTunnelProvider(), &fakeProvisioner{}, containers, joiner)
	g, err := s.CreateGateway(t.Context(), CreateGatewayInput{
		Kind: GatewayProxy, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com", ServerAddress: "203.0.113.10", Target: "host1",
	})
	require.NoError(t, err)
	containers.byStackID[g.ServiceID] = &ExposureTarget{ContainerID: "gwc", Name: "gw-traefik", StackID: g.ServiceID, Machine: "host1"}
	return s, repo, joiner, g
}

var webExposure = CreateExposureInput{Hostname: "web.example.com", ServiceID: "c-web", Port: 8000, ZoneID: "z1", Zone: "example.com"}

func TestService_ProxyExposure_RoutesFollowCreateAndDelete(t *testing.T) {
	s, _, joiner, g := proxyFixture(t)
	in := webExposure
	in.GatewayID = g.ID

	e, err := s.CreateExposure(t.Context(), in)
	require.NoError(t, err)

	require.Len(t, joiner.routes, 1)
	assert.Equal(t, "host1", joiner.routes[0].Machine)
	assert.Equal(t, "gw-traefik", joiner.routes[0].GatewayContainer)
	created := parseRoutes(t, joiner.routes[0].Routes)
	assert.Equal(t, "Host(`web.example.com`)", created.Routers[RouterName(e.ID)].Rule)
	assert.Equal(t, "http://web-1:8000", created.Services[RouterName(e.ID)].LoadBalancer.Servers[0].URL)

	require.NoError(t, s.DeleteExposure(t.Context(), e.ID))

	require.Len(t, joiner.routes, 2)
	assert.JSONEq(t, `{"routers":{},"services":{}}`, joiner.routes[1].Routes, "the deleted exposure's route is gone")
}

func TestService_TunnelExposure_SendsNoRoutes(t *testing.T) {
	s, _, _, _, joiner := newAgentOnlyFixture(t)

	_, err := s.CreateExposure(t.Context(), whoamiExposure)
	require.NoError(t, err)

	assert.Empty(t, joiner.routes)
}

func TestService_ReconcileExposures_ProxyRouteFollowsTheRunningName(t *testing.T) {
	s, repo, joiner, g := proxyFixture(t)
	in := webExposure
	in.GatewayID = g.ID
	e, err := s.CreateExposure(t.Context(), in)
	require.NoError(t, err)
	joiner.routes = nil

	running := ExposureTarget{ContainerID: "c-web", Name: "web-renamed-1", StackID: "s-web", Machine: "host1", Networks: []string{"net2"}, Running: true}
	require.NoError(t, s.ReconcileExposures(t.Context(), []ExposureTarget{running}))

	require.Len(t, joiner.routes, 1)
	assert.Equal(t, "http://web-renamed-1:8000", parseRoutes(t, joiner.routes[0].Routes).Services[RouterName(e.ID)].LoadBalancer.Servers[0].URL)
	saved, err := repo.GetExposure(t.Context(), e.ID)
	require.NoError(t, err)
	assert.Equal(t, "web-renamed-1", saved.Service)

	require.NoError(t, s.ReconcileExposures(t.Context(), []ExposureTarget{running}))
	assert.Len(t, joiner.routes, 1, "a route already on the running name is left alone")
}

// TestService_ReconcileExposures_RedeployedGatewayGetsItsRoutesBack: a recreated Traefik container starts with only
// its base config, so the deploy that reports it healthy sends the exposures again, to the container it reports.
func TestService_ReconcileExposures_RedeployedGatewayGetsItsRoutesBack(t *testing.T) {
	s, _, joiner, g := proxyFixture(t)
	in := webExposure
	in.GatewayID = g.ID
	_, err := s.CreateExposure(t.Context(), in)
	require.NoError(t, err)
	joiner.routes = nil

	gatewayDeploy := ExposureTarget{ContainerID: "gwc", Name: "gw-traefik-new", StackID: g.ServiceID, Machine: "host1", Running: true}
	require.NoError(t, s.ReconcileExposures(t.Context(), []ExposureTarget{gatewayDeploy}))

	require.Len(t, joiner.routes, 1)
	assert.Equal(t, "gw-traefik-new", joiner.routes[0].GatewayContainer)
	assert.Contains(t, joiner.routes[0].Routes, "web.example.com")
}

// TestService_SyncMachineRoutes_OneFramePerProxyGatewayOnTheMachine: a runner connecting gets the current routes of each
// proxy gateway on its machine, none for a tunnel gateway, and nothing for a machine that has no gateway.
func TestService_SyncMachineRoutes_OneFramePerProxyGatewayOnTheMachine(t *testing.T) {
	repo := newFakeRepo()
	containers := newFakeContainerLookup()
	joiner := &fakeRunnerJoiner{}
	s := newGatewayServiceWithJoiner(repo, newFakeTunnelProvider(), &fakeProvisioner{}, containers, joiner)
	now := time.Now().UTC()
	for _, g := range []Gateway{
		{ID: "gw-a", Kind: GatewayProxy, Machine: "host1", DockerNetwork: "net-a", ServiceID: "s-a", CreatedAt: now, UpdatedAt: now},
		{ID: "gw-b", Kind: GatewayProxy, Machine: "host1", DockerNetwork: "net-b", ServiceID: "s-b", CreatedAt: now, UpdatedAt: now},
		{ID: "gw-t", Kind: GatewayTunnel, Machine: "host1", DockerNetwork: "net-t", ServiceID: "s-t", CreatedAt: now, UpdatedAt: now},
		{ID: "gw-c", Kind: GatewayProxy, Machine: "host2", DockerNetwork: "net-c", ServiceID: "s-c", CreatedAt: now, UpdatedAt: now},
	} {
		require.NoError(t, repo.SaveGateway(t.Context(), g))
		containers.byStackID[g.ServiceID] = &ExposureTarget{Name: "traefik-" + g.ID, StackID: g.ServiceID, Machine: g.Machine}
	}
	require.NoError(t, repo.SaveExposure(t.Context(), Exposure{ID: "e1", GatewayID: "gw-a", Hostname: "a.example.com", Service: "web-1", Port: 80, CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, s.SyncMachineRoutes(t.Context(), "host1"))

	require.Len(t, joiner.routes, 2)
	byContainer := map[string]string{}
	for _, c := range joiner.routes {
		assert.Equal(t, "host1", c.Machine)
		byContainer[c.GatewayContainer] = c.Routes
	}
	assert.Contains(t, byContainer["traefik-gw-a"], "a.example.com", "the gateway's current exposures")
	assert.JSONEq(t, `{"routers":{},"services":{}}`, byContainer["traefik-gw-b"], "a gateway with none gets the empty set, which clears stale routes")
	assert.NotContains(t, byContainer, "traefik-gw-t")

	joiner.routes = nil
	require.NoError(t, s.SyncMachineRoutes(t.Context(), "host3"))
	assert.Empty(t, joiner.routes)
}
