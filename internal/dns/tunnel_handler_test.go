package dns

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTunnelTestHandler wires a Handler over fakes with a tunnel provider.
func newTunnelTestHandler() (*Handler, *fakeRepo, *fakeTunnelProvider, *fakeProvisioner) {
	repo := newFakeRepo()
	tunnel := newFakeTunnelProvider()
	prov := &fakeProvisioner{}
	return NewHandler(newTunnelService(repo, tunnel, prov)), repo, tunnel, prov
}

func TestHandler_CreateTunnel(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels", map[string]string{"name": "tunnel-1"})
	require.Equal(t, http.StatusCreated, rec.Code)
	var got Tunnel
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "tunnel-1", got.Name)
	assert.NotContains(t, got.Token, "tunnel-token", "token must never be serialized")
	_, err := repo.GetTunnel(context.Background(), "tunnel-1")
	require.NoError(t, err, "tunnel stored")
}

func TestHandler_ListTunnels(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one", CreatedAt: time.Now()}))
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/tunnels", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var got []Tunnel
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Len(t, got, 1)
}

func TestHandler_ListTunnels_RepoError(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	repo.tunnelErr = errBoom
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/tunnels", nil)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_GetTunnel(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/tunnels/t1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var got Tunnel
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "t1", got.ID)
}

func TestHandler_TunnelStatus(t *testing.T) {
	h, repo, tunnel, _ := newTunnelTestHandler()
	live, err := tunnel.CreateTunnel(context.Background(), "instance")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: live.ID, Name: "instance", Token: "enc"}))
	tunnel.mu.Lock()
	tunnel.tunnels[live.ID].Status = "healthy"
	tunnel.mu.Unlock()

	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/tunnels/"+live.ID+"/status", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var got Tunnel
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "healthy", got.Status)
	assert.NotContains(t, rec.Body.String(), "enc")
}

func TestHandler_RouteTunnelHostname(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/t1/route", map[string]any{
		"hostname": "app.example.com", "zone_id": "z1", "zone": "example.com", "service": "http://localhost:80",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var got Tunnel
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "app.example.com", got.Hostname)
}

func TestHandler_RotateTunnelCredentials(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/t1/rotate", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	stored, err := repo.GetTunnel(context.Background(), "t1")
	require.NoError(t, err)
	assert.NotContains(t, stored.Token, "rotated-token-t1")
}

func TestHandler_DeleteTunnel(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	rec := doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/tunnels/t1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	_, err := repo.GetTunnel(context.Background(), "t1")
	require.Error(t, err)
}

func TestHandler_DeleteTunnel_RefusedWhileInUse(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	require.NoError(t, repo.SaveGateway(context.Background(), Gateway{ID: "g1", Kind: GatewayTunnel, DockerNetwork: "web", TunnelID: "t1"}))
	require.NoError(t, repo.SaveExposure(context.Background(), Exposure{ID: "e1", GatewayID: "g1", Hostname: "app.example.com"}))
	rec := doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/tunnels/t1", nil)
	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "app.example.com")
	assert.Contains(t, rec.Body.String(), "gateway on network web")
}

func TestHandler_ProvisionTunnelAgent(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	enc, err := encryptTunnelTokenForTest(testKey(), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one", Token: enc}))
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/t1/agent", map[string]any{
		"target": "10.0.0.1", "strategy": "run", "docker_network": "nexul",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var got AgentProvisioned
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "svc-1", got.ServiceID)
}

func TestHandler_ProvisionReverseProxy(t *testing.T) {
	h := NewHandler(newGatewayService(newFakeRepo(), newFakeTunnelProvider(), &fakeProvisioner{}, newFakeContainerLookup()))
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/reverse-proxy", map[string]any{
		"target": "10.0.0.1", "strategy": "run", "docker_network": "nexul",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var got AgentProvisioned
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "svc-1", got.ServiceID)
}

func TestHandler_TunnelRoutes_RefuseWhatCannotWork(t *testing.T) {
	route := map[string]any{"hostname": "app.example.com", "zone_id": "z1", "zone": "example.com", "service": "http://localhost:80"}
	tests := []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"a blank tunnel name", http.MethodPost, "/api/dns/tunnels", map[string]string{"name": " "}, http.StatusBadRequest},
		{"reading a tunnel that does not exist", http.MethodGet, "/api/dns/tunnels/nope", nil, http.StatusNotFound},
		{"routing a hostname to a tunnel that does not exist", http.MethodPost, "/api/dns/tunnels/ghost/route", route, http.StatusNotFound},
		{"rotating a tunnel that does not exist", http.MethodPost, "/api/dns/tunnels/ghost/rotate", nil, http.StatusNotFound},
		{"provisioning an agent on a tunnel that does not exist", http.MethodPost, "/api/dns/tunnels/ghost/agent", map[string]any{"target": "10.0.0.1"}, http.StatusNotFound},
		{"an agent request that is not an object", http.MethodPost, "/api/dns/tunnels/t1/agent", "not-an-object", http.StatusBadRequest},
		{"a reverse proxy request that is not an object", http.MethodPost, "/api/dns/reverse-proxy", "not-an-object", http.StatusBadRequest},
		{"deleting a tunnel that does not exist", http.MethodDelete, "/api/dns/tunnels/nope", nil, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, _, _ := newTunnelTestHandler()
			rec := doJSON(t, h.Routes(), tt.method, tt.path, tt.body)
			require.Equal(t, tt.want, rec.Code)
		})
	}
}
