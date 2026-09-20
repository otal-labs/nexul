package dns

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func tunnelTools(repo *fakeRepo, tunnel *fakeTunnelProvider, prov *fakeProvisioner) []mcptool.Tool {
	return MCPTools(newTunnelService(repo, tunnel, prov))
}

func TestMCPTools_TunnelCreate(t *testing.T) {
	repo := newFakeRepo()
	call := toolCall(t, "dns_tunnel_create", tunnelTools(repo, newFakeTunnelProvider(), nil)...)
	got, err := call(context.Background(), map[string]any{"name": "tunnel-1"})
	require.NoError(t, err)
	tunnel, ok := got.(*Tunnel)
	require.True(t, ok)
	assert.Equal(t, "tunnel-1", tunnel.Name)
	assert.NotContains(t, tunnel.Token, "tunnel-token", "token never echoed")
	_, err = repo.GetTunnel(context.Background(), "tunnel-1")
	require.NoError(t, err)
}

func TestMCPTools_TunnelCreate_MissingNameIsInvalid(t *testing.T) {
	call := toolCall(t, "dns_tunnel_create", tunnelTools(newFakeRepo(), newFakeTunnelProvider(), nil)...)
	_, err := call(context.Background(), map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestMCPTools_TunnelListAndGet(t *testing.T) {
	repo := newFakeRepo()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one", CreatedAt: time.Now()}))
	tools := tunnelTools(repo, newFakeTunnelProvider(), nil)

	list := toolCall(t, "dns_tunnel_list", tools...)
	got, err := list(context.Background(), map[string]any{})
	require.NoError(t, err)
	assert.Len(t, got, 1)

	get := toolCall(t, "dns_tunnel_get", tools...)
	gotT, err := get(context.Background(), map[string]any{"tunnel_id": "t1"})
	require.NoError(t, err)
	tunnel, ok := gotT.(*Tunnel)
	require.True(t, ok)
	assert.Equal(t, "t1", tunnel.ID)
}

func TestMCPTools_TunnelRoute(t *testing.T) {
	repo := newFakeRepo()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	call := toolCall(t, "dns_tunnel_route", tunnelTools(repo, newFakeTunnelProvider(), nil)...)
	got, err := call(context.Background(), map[string]any{
		"tunnel_id": "t1", "hostname": "app.example.com", "zone_id": "z1", "zone": "example.com", "service": "http://localhost:80",
	})
	require.NoError(t, err)
	tunnel, ok := got.(*Tunnel)
	require.True(t, ok)
	assert.Equal(t, "app.example.com", tunnel.Hostname)
}

func TestMCPTools_TunnelRotate(t *testing.T) {
	repo := newFakeRepo()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	call := toolCall(t, "dns_tunnel_rotate", tunnelTools(repo, newFakeTunnelProvider(), nil)...)
	got, err := call(context.Background(), map[string]any{"tunnel_id": "t1"})
	require.NoError(t, err)
	tunnel, ok := got.(*Tunnel)
	require.True(t, ok)
	assert.Equal(t, "t1", tunnel.ID)
}

func TestMCPTools_TunnelDelete(t *testing.T) {
	repo := newFakeRepo()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	call := toolCall(t, "dns_tunnel_delete", tunnelTools(repo, newFakeTunnelProvider(), nil)...)
	got, err := call(context.Background(), map[string]any{"tunnel_id": "t1"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"status": "deleted"}, got)
	_, err = repo.GetTunnel(context.Background(), "t1")
	require.Error(t, err)
}

func TestMCPTools_TunnelProvisionAgent(t *testing.T) {
	repo := newFakeRepo()
	enc, err := encryptTunnelTokenForTest(testKey(), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one", Token: enc}))
	prov := &fakeProvisioner{}
	call := toolCall(t, "dns_tunnel_provision_agent", tunnelTools(repo, newFakeTunnelProvider(), prov)...)
	got, err := call(context.Background(), map[string]any{
		"tunnel_id": "t1", "project_id": "p1", "target": "10.0.0.1",
	})
	require.NoError(t, err)
	provisioned, ok := got.(*AgentProvisioned)
	require.True(t, ok)
	assert.Equal(t, "svc-1", provisioned.ServiceID)
	require.Len(t, prov.calls, 1)
	assert.Equal(t, "the-tunnel-secret", prov.calls[0].Env["TUNNEL_TOKEN"])
}

func TestMCPTools_ProvisionReverseProxy(t *testing.T) {
	prov := &fakeProvisioner{}
	call := toolCall(t, "dns_provision_reverse_proxy", tunnelTools(newFakeRepo(), newFakeTunnelProvider(), prov)...)
	got, err := call(context.Background(), map[string]any{
		"project_id": "p1", "target": "10.0.0.1", "strategy": "run", "ports": []any{"80:80", "443:443"}, "image": "traefik:v3",
	})
	require.NoError(t, err)
	provisioned, ok := got.(*AgentProvisioned)
	require.True(t, ok)
	assert.Equal(t, "svc-1", provisioned.ServiceID)
	require.Len(t, prov.calls, 1)
	assert.Equal(t, []string{"80:80", "443:443"}, prov.calls[0].Ports)
}
