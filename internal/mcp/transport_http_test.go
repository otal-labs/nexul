package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func post(t *testing.T, s *Server, body, accept, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestServeHTTP_JSONResponse(t *testing.T) {
	rec := post(t, newTestServer(), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, "application/json", "application/json")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Nil(t, resp.Error)
	assert.Equal(t, "1", string(resp.ID))
	result := decodeResult[initializeResult](t, &resp)
	assert.Equal(t, ProtocolVersion, result.ProtocolVersion)
}

func TestServeHTTP_SSEResponse(t *testing.T) {
	rec := post(t, newTestServer(), `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"value":"hi"}}}`, "text/event-stream", "application/json")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "event: message")
	assert.Contains(t, rec.Body.String(), `"hi"`)
}

func TestServeHTTP_SSEFromMultiAccept(t *testing.T) {
	rec := post(t, newTestServer(), `{"jsonrpc":"2.0","id":1,"method":"ping"}`, "application/json, text/event-stream", "application/json")
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
}

func TestServeHTTP_NotificationIs202(t *testing.T) {
	rec := post(t, newTestServer(), `{"jsonrpc":"2.0","method":"notifications/initialized"}`, "application/json", "application/json")
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestServeHTTP_ToolErrorCode(t *testing.T) {
	rec := post(t, newTestServer(), `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"boom"}}`, "application/json", "application/json")
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeNotFound, resp.Error.Code)
}

func TestServeHTTP_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()
	newTestServer().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestServeHTTP_UnsupportedMediaType(t *testing.T) {
	rec := post(t, newTestServer(), `{"jsonrpc":"2.0","id":1,"method":"ping"}`, "application/json", "text/plain")
	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestServeHTTP_ParseError(t *testing.T) {
	rec := post(t, newTestServer(), `{not json`, "application/json", "application/json")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeParseError, resp.Error.Code)
}

func TestServeHTTP_HeaderMismatch(t *testing.T) {
	t.Run("Mcp-Method mismatch is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Mcp-Method", "tools/call")
		rec := httptest.NewRecorder()
		newTestServer().ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp Response
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeHeaderMismatch, resp.Error.Code)
	})
	t.Run("Mcp-Name mismatch is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo"}}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Mcp-Method", "tools/call")
		req.Header.Set("Mcp-Name", "boom")
		rec := httptest.NewRecorder()
		newTestServer().ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp Response
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotNil(t, resp.Error)
		assert.Equal(t, CodeHeaderMismatch, resp.Error.Code)
	})
	t.Run("absent headers still work", func(t *testing.T) {
		rec := post(t, newTestServer(), `{"jsonrpc":"2.0","id":1,"method":"ping"}`, "application/json", "application/json")
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("matching headers still work", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"value":"hi"}}}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Mcp-Method", "tools/call")
		req.Header.Set("Mcp-Name", "echo")
		rec := httptest.NewRecorder()
		newTestServer().ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestServeHTTP_OversizedBody(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"x":"` + strings.Repeat("a", maxBodyBytes) + `"}}`
	rec := post(t, newTestServer(), body, "application/json", "application/json")
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}
