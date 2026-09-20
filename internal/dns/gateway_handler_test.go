package dns

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGatewayTestHandler wires a Handler over fakes with a tunnel gateway
// already created, so exposure tests have a valid gateway id to route
// through.
func newGatewayTestHandler(t *testing.T) (*Handler, *Gateway) {
	t.Helper()
	repo := newFakeRepo()
	enc, err := encryptTunnelTokenForTest([]byte("0123456789abcdef0123456789abcdef"), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, repo.SaveTunnel(context.Background(), Tunnel{ID: "t1", Name: "prod", Token: enc}))
	containers := newFakeContainerLookup()
	containers.add("app", ExposureTarget{ContainerID: "c-app", Name: "app", StackID: "s-app", ProjectID: "p1", Machine: "host1", Networks: []string{"net1"}})
	containers.add("other", ExposureTarget{ContainerID: "c-other", Name: "other", StackID: "s-other", ProjectID: "p1", Machine: "host2", Networks: []string{"net2"}})
	s := newGatewayService(repo, newFakeTunnelProvider(), &fakeProvisioner{}, containers)
	g, err := s.CreateGateway(context.Background(), CreateGatewayInput{
		Kind: GatewayTunnel, DockerNetwork: "net1", ZoneID: "z1", Zone: "example.com",
		TunnelID: "t1", ProjectID: "p1", Target: "host1",
	})
	require.NoError(t, err)
	return NewHandler(s), g
}

func TestHandler_CreateAndListGateways(t *testing.T) {
	h, _ := newGatewayTestHandler(t)
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/gateways", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var got []Gateway
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Len(t, got, 1)
}

func TestHandler_DeleteGateway(t *testing.T) {
	h, g := newGatewayTestHandler(t)
	rec := doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/gateways/"+g.ID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_CreateAndListExposures(t *testing.T) {
	h, g := newGatewayTestHandler(t)
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/exposures", map[string]any{
		"gateway_id": g.ID, "hostname": "app.example.com", "service_id": "c-app", "port": 8080, "zone_id": "z1", "zone": "example.com",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var created Exposure
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "app.example.com", created.Hostname)

	rec = doJSON(t, h.Routes(), http.MethodGet, "/api/dns/exposures", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var list []Exposure
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Len(t, list, 1)

	rec = doJSON(t, h.Routes(), http.MethodGet, "/api/dns/services/app/exposures", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var summaries []ExposureSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summaries))
	require.Len(t, summaries, 1)
	assert.Equal(t, 8080, summaries[0].Port)

	rec = doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/exposures/"+created.ID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_CreateExposure_MachineMismatchIsBadRequest(t *testing.T) {
	h, g := newGatewayTestHandler(t)
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/exposures", map[string]any{
		"gateway_id": g.ID, "hostname": "other.example.com", "service_id": "c-other", "port": 8080, "zone_id": "z1", "zone": "example.com",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
