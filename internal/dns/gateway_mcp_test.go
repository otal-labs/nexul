package dns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPTools_GatewayAndExposureLifecycle(t *testing.T) {
	repo := newFakeRepo()
	enc, err := encryptTunnelTokenForTest([]byte("0123456789abcdef0123456789abcdef"), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "prod", Token: enc}))
	containers := newFakeContainerLookup()
	containers.add("app", ExposureTarget{ContainerID: "c-app", Name: "app", StackID: "s-app", ProjectID: "p1", Machine: "host1", Networks: []string{"net1"}})
	s := newGatewayService(repo, newFakeTunnelProvider(), &fakeProvisioner{}, containers)
	tools := MCPTools(s)

	create := toolCall(t, "dns_gateway_create", tools...)
	got, err := create(context.Background(), map[string]any{
		"kind": "tunnel", "docker_network": "net1", "machine": "host1", "zone_id": "z1", "zone": "example.com",
		"tunnel_id": "t1", "project_id": "p1", "target": "host1",
	})
	require.NoError(t, err)
	g, ok := got.(*Gateway)
	require.True(t, ok)
	assert.Equal(t, GatewayTunnel, g.Kind)

	list := toolCall(t, "dns_gateway_list", tools...)
	gotList, err := list(context.Background(), map[string]any{})
	require.NoError(t, err)
	assert.Len(t, gotList, 1)

	exposeCreate := toolCall(t, "exposure_create", tools...)
	gotExp, err := exposeCreate(context.Background(), map[string]any{
		"gateway_id": g.ID, "hostname": "app.example.com", "service_id": "c-app", "port": float64(8080),
		"zone_id": "z1", "zone": "example.com",
	})
	require.NoError(t, err)
	e, ok := gotExp.(*Exposure)
	require.True(t, ok)
	assert.Equal(t, "app.example.com", e.Hostname)

	expList := toolCall(t, "dns_exposure_list", tools...)
	gotExpList, err := expList(context.Background(), map[string]any{})
	require.NoError(t, err)
	assert.Len(t, gotExpList, 1)

	deleteExp := toolCall(t, "exposure_delete", tools...)
	_, err = deleteExp(context.Background(), map[string]any{"exposure_id": e.ID})
	require.NoError(t, err)

	deleteGw := toolCall(t, "dns_gateway_delete", tools...)
	_, err = deleteGw(context.Background(), map[string]any{"gateway_id": g.ID})
	require.NoError(t, err)
}
