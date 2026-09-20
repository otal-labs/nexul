package topology

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHandler(t *testing.T) *Handler {
	svc := NewService(newFakeRepo())
	return NewHandler(svc)
}

func getCanvas(t *testing.T, rec *httptest.ResponseRecorder) *Canvas {
	t.Helper()
	var c Canvas
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &c))
	return &c
}

func TestRoutes_Get_EmptyCanvas(t *testing.T) {
	h := newHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/topology", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	c := getCanvas(t, rec)
	assert.Equal(t, 2, c.SchemaVersion)
	assert.Empty(t, c.Nodes)
	assert.Empty(t, c.Edges)
}

func TestRoutes_AddNode_ThenGet(t *testing.T) {
	h := newHandler(t).Routes()

	body := `{"id":"net-main","type":"network","position":{"x":1,"y":2},"data":{"name":"main"}}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/topology/nodes", strings.NewReader(body)))
	assert.Equal(t, http.StatusOK, rec.Code)
	c := getCanvas(t, rec)
	require.Len(t, c.Nodes, 1)
	assert.Equal(t, "net-main", c.Nodes[0].ID)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/topology?environment=default", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	c = getCanvas(t, rec)
	require.Len(t, c.Nodes, 1)
	assert.Equal(t, "main", c.Nodes[0].Data.Name)
}

func TestRoutes_AddExternalNode_RequiresLabel(t *testing.T) {
	h := newHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/topology/nodes",
		strings.NewReader(`{"id":"ext-cf","type":"external","position":{"x":0,"y":0},"data":{"name":"Cloudflare"}}`)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRoutes_RemoveServiceNode_IsRejected(t *testing.T) {
	h := newHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/topology/nodes",
		strings.NewReader(`{"id":"svc-a","type":"service","position":{"x":0,"y":0},"data":{"service_id":"svc-a","name":"A","status":"healthy"}}`)))
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/topology/nodes/svc-a", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRoutes_AddNode_Invalid(t *testing.T) {
	h := newHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/topology/nodes",
		strings.NewReader(`{"id":"","data":{}}`)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRoutes_RemoveNode_Missing(t *testing.T) {
	h := newHandler(t).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/topology/nodes/nope", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRoutes_AddEdge_ThenRemove(t *testing.T) {
	h := newHandler(t).Routes()

	for _, node := range []string{
		`{"id":"net-db","type":"network","position":{"x":0,"y":0},"data":{"name":"db-net"}}`,
		`{"id":"net-api","type":"network","position":{"x":0,"y":0},"data":{"name":"api-net"}}`,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/topology/nodes", strings.NewReader(node)))
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	edge := `{"id":"e1","source":"net-api","target":"net-db","type":"relation","data":{"kind":"depends_on"}}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/topology/edges", strings.NewReader(edge)))
	assert.Equal(t, http.StatusOK, rec.Code)
	c := getCanvas(t, rec)
	require.Len(t, c.Edges, 1)
	assert.Equal(t, "depends_on", string(c.Edges[0].Data.Kind))

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/topology/edges/e1", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	c = getCanvas(t, rec)
	assert.Empty(t, c.Edges)
}

func TestRoutes_Update_FullCanvas(t *testing.T) {
	h := newHandler(t).Routes()
	body := `{"schema_version":2,"nodes":[{"id":"net-a","type":"network","position":{"x":1,"y":1},"data":{"name":"A"}}],"edges":[]}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/topology", strings.NewReader(body)))
	assert.Equal(t, http.StatusOK, rec.Code)
	c := getCanvas(t, rec)
	require.Len(t, c.Nodes, 1)
}
