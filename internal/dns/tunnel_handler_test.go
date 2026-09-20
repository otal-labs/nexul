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

func TestHandler_CreateTunnel_InvalidName(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels", map[string]string{"name": " "})
	require.Equal(t, http.StatusBadRequest, rec.Code)
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

func TestHandler_GetTunnel_Missing(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/tunnels/nope", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
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

func TestHandler_RouteTunnelHostname_MissingTunnel(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/ghost/route", map[string]any{
		"hostname": "app.example.com", "zone_id": "z1", "zone": "example.com", "service": "http://localhost:80",
	})
	require.Equal(t, http.StatusNotFound, rec.Code)
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

func TestHandler_RotateTunnelCredentials_Missing(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/ghost/rotate", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_ProvisionTunnelAgent_MissingTunnel(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/ghost/agent", map[string]any{
		"project_id": "p1", "target": "10.0.0.1",
	})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_ProvisionTunnelAgent_InvalidBody(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/t1/agent", "not-an-object")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_ProvisionReverseProxy_InvalidBody(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/reverse-proxy", "not-an-object")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_DeleteTunnel(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one"}))
	rec := doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/tunnels/t1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	_, err := repo.GetTunnel(context.Background(), "t1")
	require.Error(t, err)
}

func TestHandler_DeleteTunnel_Missing(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/tunnels/nope", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_ProvisionTunnelAgent(t *testing.T) {
	h, repo, _, _ := newTunnelTestHandler()
	enc, err := encryptTunnelTokenForTest(testKey(), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "one", Token: enc}))
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/tunnels/t1/agent", map[string]any{
		"project_id": "p1", "target": "10.0.0.1", "strategy": "run", "docker_network": "nexul",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var got AgentProvisioned
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "svc-1", got.ServiceID)
}

func TestHandler_ProvisionReverseProxy(t *testing.T) {
	h, _, _, _ := newTunnelTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/reverse-proxy", map[string]any{
		"project_id": "p1", "target": "10.0.0.1", "strategy": "run", "docker_network": "nexul",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var got AgentProvisioned
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "svc-1", got.ServiceID)
}
