package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

const tunnelGatewayArgs = `{"kind":"tunnel","machine":"host1","docker_network":"net1","project_id":"p1","zone_id":"z1","zone":"example.com","tunnel_id":"t1"}`

func seedGateway(f *toolFakes, g Gateway) {
	f.repo.gateways[g.ID] = &g
}

func seedExposure(f *toolFakes, e Exposure) {
	f.repo.exposures[e.ID] = &e
}

func TestGatewayAndExposureTools_Errors(t *testing.T) {
	runToolErrors(t, []toolError{
		{name: "an unknown kind is invalid", tool: "gateway_create", want: apperrs.ErrInvalid,
			args: `{"kind":"bridge","machine":"host1","docker_network":"net1","project_id":"p1","zone_id":"z1","zone":"example.com"}`},
		{name: "a tunnel gateway without a tunnel is invalid", tool: "gateway_create", want: apperrs.ErrInvalid,
			args: `{"kind":"tunnel","machine":"host1","docker_network":"net1","project_id":"p1","zone_id":"z1","zone":"example.com"}`},
		{name: "the old target argument is rejected", tool: "gateway_create", want: apperrs.ErrInvalid,
			args: `{"kind":"tunnel","target":"host1","docker_network":"net1","project_id":"p1","zone_id":"z1","zone":"example.com","tunnel_id":"t1"}`},
		{name: "ports are rejected, not dropped", tool: "gateway_create", want: apperrs.ErrInvalid,
			args: `{"kind":"tunnel","machine":"host1","docker_network":"net1","project_id":"p1","zone_id":"z1","zone":"example.com","tunnel_id":"t1","ports":["80:80"]}`},
		{name: "an unknown tunnel is not found", tool: "gateway_create", want: apperrs.ErrNotFound,
			args: `{"kind":"tunnel","machine":"host1","docker_network":"net1","project_id":"p1","zone_id":"z1","zone":"example.com","tunnel_id":"ghost"}`},
		{name: "a network that has a gateway conflicts", tool: "gateway_create", want: apperrs.ErrConflict, args: tunnelGatewayArgs,
			setup: func(f *toolFakes) {
				seedGateway(f, Gateway{ID: "g0", Kind: GatewayProxy, DockerNetwork: "net1", Machine: "host1"})
			}},
		{name: "deleting an unknown gateway is not found", tool: "gateway_delete", args: `{"id":"ghost"}`, want: apperrs.ErrNotFound},
		{name: "deleting a gateway with exposures conflicts", tool: "gateway_delete", args: `{"id":"g1"}`, want: apperrs.ErrConflict,
			setup: func(f *toolFakes) {
				seedGateway(f, Gateway{ID: "g1", Kind: GatewayProxy, DockerNetwork: "net1", Machine: "host1"})
				seedExposure(f, Exposure{ID: "e1", GatewayID: "g1", Hostname: "app.example.com", ServiceID: "c-app", Port: 80})
			}},
		{name: "an exposure without a container is invalid", tool: "exposure_create", want: apperrs.ErrInvalid,
			args: `{"hostname":"app.example.com","port":8080,"zone_id":"z1","zone":"example.com"}`},
		{name: "an exposure to port zero is invalid", tool: "exposure_create", want: apperrs.ErrInvalid,
			args: `{"hostname":"app.example.com","service_id":"c-app","port":0,"zone_id":"z1","zone":"example.com"}`},
		{name: "an exposure to an unknown container is not found", tool: "exposure_create", want: apperrs.ErrNotFound,
			args: `{"hostname":"app.example.com","service_id":"ghost","port":8080,"zone_id":"z1","zone":"example.com"}`},
		{name: "a record write the token may not make is forbidden", tool: "exposure_create", want: apperrs.ErrForbidden,
			args: `{"hostname":"app.example.com","service_id":"c-app","port":8080,"zone_id":"z1","zone":"example.com","gateway_id":"g1"}`,
			setup: func(f *toolFakes) {
				seedGateway(f, Gateway{ID: "g1", Kind: GatewayTunnel, TunnelID: "t1", DockerNetwork: "net1", Machine: "host1"})
				f.records.createErr = apperrs.ErrForbidden
			}},
		{name: "deleting an unknown exposure is not found", tool: "exposure_delete", args: `{"id":"ghost"}`, want: apperrs.ErrNotFound},
		{name: "a failed gateway read surfaces", tool: "gateway_list", args: `{}`, want: errBoom,
			setup: func(f *toolFakes) { f.repo.gatewayErr = errBoom }},
		{name: "a failed exposure read surfaces", tool: "exposure_list", args: `{}`, want: errBoom,
			setup: func(f *toolFakes) { f.repo.exposureErr = errBoom }},
	})
}

func TestGatewayCreate_TunnelDeploysCloudflaredOnTheMachine(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "gateway_create", tunnelGatewayArgs)
	require.NoError(t, err)
	g := got.(*Gateway)
	assert.Equal(t, GatewayTunnel, g.Kind)
	assert.Equal(t, "host1", g.Machine)
	require.Len(t, f.prov.calls, 1)
	assert.Equal(t, "host1", f.prov.calls[0].Target)
	assert.Equal(t, "net1", f.prov.calls[0].DockerNetwork)
	assert.Equal(t, "the-tunnel-secret", f.prov.calls[0].Env["TUNNEL_TOKEN"])
	assert.NotContains(t, asJSON(t, got), "the-tunnel-secret")
}

func TestGatewayCreate_ProxyDeploysTraefik(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "gateway_create",
		`{"kind":"proxy","machine":"host1","docker_network":"net1","project_id":"p1","zone_id":"z1","zone":"example.com","server_address":"203.0.113.10"}`)
	require.NoError(t, err)
	assert.Equal(t, "203.0.113.10", got.(*Gateway).ServerAddress)
	require.Len(t, f.prov.calls, 1)
	assert.Equal(t, traefikImage, f.prov.calls[0].Image)
	assert.Equal(t, []string{"80:80", "443:443"}, f.prov.calls[0].Ports)
}

func TestGatewayListAndDelete(t *testing.T) {
	f := newToolFakes(t)
	seedGateway(f, Gateway{ID: "g1", Kind: GatewayProxy, DockerNetwork: "net1", Machine: "host1", ServiceID: "s-gw"})

	got, err := f.call(t, "gateway_list", `{}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[*Gateway])
	require.Len(t, page.Items, 1)
	assert.Equal(t, "g1", page.Items[0].ID)

	got, err = f.call(t, "gateway_delete", `{"id":"g1"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone("g1"), got)
	assert.Equal(t, []string{"s-gw"}, f.prov.deprovisionCalls)
}

func TestExposureLifecycle(t *testing.T) {
	f := newToolFakes(t)
	seedGateway(f, Gateway{ID: "g1", Kind: GatewayTunnel, TunnelID: "t1", DockerNetwork: "net1", Networks: []string{"net1"}, Machine: "host1"})

	got, err := f.call(t, "exposure_create",
		`{"hostname":"App.example.com","service_id":"c-app","port":8080,"zone_id":"z1","zone":"example.com"}`)
	require.NoError(t, err)
	e := got.(*Exposure)
	assert.Equal(t, "g1", e.GatewayID, "without gateway_id the machine's gateway is reused")
	assert.Equal(t, "app.example.com", e.Hostname)
	assert.Equal(t, []routeCall{{TunnelID: "t1", Hostname: "app.example.com", Service: "http://app:8080"}}, f.tunnels.routeCalls)

	got, err = f.call(t, "exposure_delete", `{"id":"`+e.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone(e.ID), got)
	assert.Empty(t, f.repo.exposures)
}

func TestExposureList_Filters(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantIDs []string
	}{
		{"every exposure", `{}`, []string{"e1", "e2", "e3"}},
		{"by container", `{"service_id":"c-app"}`, []string{"e1", "e2"}},
		{"by gateway", `{"gateway_id":"g2"}`, []string{"e2", "e3"}},
		{"by both", `{"service_id":"c-app","gateway_id":"g2"}`, []string{"e2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newToolFakes(t)
			seedExposure(f, Exposure{ID: "e1", GatewayID: "g1", ServiceID: "c-app", Hostname: "a.example.com"})
			seedExposure(f, Exposure{ID: "e2", GatewayID: "g2", ServiceID: "c-app", Hostname: "b.example.com"})
			seedExposure(f, Exposure{ID: "e3", GatewayID: "g2", ServiceID: "c-api", Hostname: "c.example.com"})
			got, err := f.call(t, "exposure_list", tt.args)
			require.NoError(t, err)
			var ids []string
			for _, e := range got.(mcptool.Page[*Exposure]).Items {
				ids = append(ids, e.ID)
			}
			assert.ElementsMatch(t, tt.wantIDs, ids)
		})
	}
}
