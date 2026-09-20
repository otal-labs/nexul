package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestHandler() (*Handler, *fakeRepo, *fakeProvider) {
	repo := newFakeRepo()
	p := newFakeProvider()
	return NewHandler(newTestService(repo, p, nil)), repo, p
}

func newTestHandlerRoutes() http.Handler {
	h, _, _ := newTestHandler()
	return h.Routes()
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_ListZones(t *testing.T) {
	rec := doJSON(t, newTestHandlerRoutes(), http.MethodGet, "/api/dns/zones", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var zones []Zone
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &zones))
	require.Len(t, zones, 1)
	assert.Equal(t, "example.com", zones[0].Name)
}

func TestHandler_ListZones_BadTokenIsFatal(t *testing.T) {
	h, _, p := newTestHandler()
	p.zonesErr = apperrs.ErrUnauthorized
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/zones", nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_CreateRecord(t *testing.T) {
	h, repo, _ := newTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/zones/z1/records", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 300})
	require.Equal(t, http.StatusCreated, rec.Code)
	var rec_ Record
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rec_))
	assert.Equal(t, "api", rec_.Name)
	assert.Equal(t, []string{TopicRecordChanged}, repo.topics(), "record_changed enqueued to outbox")
}

func TestHandler_CreateRecord_ValidationError(t *testing.T) {
	rec := doJSON(t, newTestHandlerRoutes(), http.MethodPost, "/api/dns/zones/z1/records", RecordInput{Type: "SRV", Name: "x", Content: "y"})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_DeleteRecord(t *testing.T) {
	h, _, p := newTestHandler()
	created, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
	require.NoError(t, err)
	rec := doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/zones/z1/records/"+created.ID, nil)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_CheckPropagation_UnknownRecord(t *testing.T) {
	rec := doJSON(t, newTestHandlerRoutes(), http.MethodGet, "/api/dns/zones/z1/records/ghost/propagation", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_CreateInstanceRecord(t *testing.T) {
	h, repo, _ := newTestHandler()
	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/instance-record", map[string]string{
		"zone_id": "z1", "zone": "example.com", "type": "A", "target": "10.0.0.1",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, []string{TopicRecordChanged}, repo.topics())
}

func TestHandler_CreateInstanceRecord_Mismatch(t *testing.T) {
	rec := doJSON(t, newTestHandlerRoutes(), http.MethodPost, "/api/dns/instance-record", map[string]string{
		"zone_id": "z1", "zone": "other.com", "type": "A", "target": "10.0.0.1",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_ServiceHostnames(t *testing.T) {
	h, _, _ := newTestHandler()
	in := ServiceHostnameInput{Service: "api", Hostname: "api.example.com", ZoneID: "z1", Zone: "example.com", Type: RecordA, Target: "1.2.3.4"}

	rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/service-hostnames", in)
	require.Equal(t, http.StatusCreated, rec.Code)
	var sh ServiceHostname
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sh))
	assert.Equal(t, "api.example.com", sh.Hostname)

	rec = doJSON(t, h.Routes(), http.MethodGet, "/api/dns/service-hostnames/api", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sh))
	assert.Equal(t, "api", sh.Service)

	rec = doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/service-hostnames/api", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doJSON(t, h.Routes(), http.MethodGet, "/api/dns/service-hostnames/api", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Verify(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		rec := doJSON(t, newTestHandlerRoutes(), http.MethodPost, "/api/dns/verify", nil)
		require.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("bad credentials surface", func(t *testing.T) {
		h, _, p := newTestHandler()
		p.verifyErr = apperrs.ErrUnauthorized
		rec := doJSON(t, h.Routes(), http.MethodPost, "/api/dns/verify", nil)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandler_ListRecords(t *testing.T) {
	h, _, p := newTestHandler()
	_, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
	require.NoError(t, err)
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/zones/z1/records", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var records []Record
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &records))
	require.Len(t, records, 1)
	assert.Equal(t, "api", records[0].Name)
}

func TestHandler_UpdateRecord(t *testing.T) {
	h, _, p := newTestHandler()
	created, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
	require.NoError(t, err)
	rec := doJSON(t, h.Routes(), http.MethodPatch, "/api/dns/zones/z1/records/"+created.ID, RecordInput{Type: RecordA, Name: "api", Content: "5.6.7.8", TTL: 60})
	require.Equal(t, http.StatusOK, rec.Code)
	var got Record
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "5.6.7.8", got.Content)
}

func TestHandler_CheckPropagation_OK(t *testing.T) {
	h, _, p := newTestHandler()
	created, err := p.CreateRecord(context.Background(), "z1", RecordInput{Type: RecordA, Name: "api", Content: "1.2.3.4", TTL: 1})
	require.NoError(t, err)
	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/zones/z1/records/"+created.ID+"/propagation", nil)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_ServiceHostname_CRUD(t *testing.T) {
	h, _, _ := newTestHandler()
	in := ServiceHostnameInput{Service: "api", Hostname: "api.example.com", ZoneID: "z1", Zone: "example.com", Type: RecordA, Target: "1.2.3.4"}

	rec := doJSON(t, h.Routes(), http.MethodGet, "/api/dns/service-hostnames", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var list []ServiceHostname
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Len(t, list, 0)

	rec = doJSON(t, h.Routes(), http.MethodPost, "/api/dns/service-hostnames", in)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doJSON(t, h.Routes(), http.MethodGet, "/api/dns/service-hostnames", nil)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Len(t, list, 1)

	rec = doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/service-hostnames/api", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doJSON(t, h.Routes(), http.MethodDelete, "/api/dns/service-hostnames/api", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_ServiceHostname_MismatchSurfaced(t *testing.T) {
	rec := doJSON(t, newTestHandlerRoutes(), http.MethodPost, "/api/dns/service-hostnames", ServiceHostnameInput{
		Service: "api", Hostname: "api.other.com", ZoneID: "z1", Zone: "example.com", Type: RecordA, Target: "1.2.3.4",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
