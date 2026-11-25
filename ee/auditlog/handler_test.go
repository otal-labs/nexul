package auditlog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, h http.Handler, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(asOwner(t), method, "/api/instance/audit", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_SaveReadAndRemove(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	routes := NewHandler(h.svc).Routes()

	rec := serve(t, routes, http.MethodPut, `{"retention_days":365,"stream_url":"https://logs.example.com"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var saved Saved
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))
	assert.Equal(t, 365, saved.RetentionDays)
	assert.NotEmpty(t, saved.NewSecret, "the browser is shown the made secret once")

	rec = serve(t, routes, http.MethodGet, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "new_secret")
	assert.NotContains(t, rec.Body.String(), saved.NewSecret)

	rec = serve(t, routes, http.MethodDelete, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"unlocked":true,"retention_days":45,"stream_url":"","stream_secret_set":false,"stream_enabled":false,"streaming":false,"pending":0,"dead":0}`, rec.Body.String())
}

func TestHandler_RefusalsCarryTheirStatus(t *testing.T) {
	t.Parallel()
	routes := NewHandler(newHarness(t, true).svc).Routes()
	assert.Equal(t, http.StatusBadRequest, serve(t, routes, http.MethodPut, `{"retention_days":10}`).Code)
	assert.Equal(t, http.StatusBadRequest, serve(t, routes, http.MethodPut, `{"retention_days":`).Code)

	locked := NewHandler(newHarness(t, false).svc).Routes()
	assert.Equal(t, http.StatusForbidden, serve(t, locked, http.MethodPut, `{"retention_days":90}`).Code)
	assert.Equal(t, http.StatusOK, serve(t, locked, http.MethodGet, "").Code, "a read with the key locked still works")
}
