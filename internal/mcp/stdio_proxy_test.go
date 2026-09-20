package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProxyUpstream mimics the server's MCP HTTP transport: it requires a
// Bearer token (like RequireAuth does), answers JSON-RPC for non-notifications
// and 202 for notifications, and otherwise behaves like transport_http.go.
func newProxyUpstream(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var authed []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, `{"message":"unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}
		authed = append(authed, auth)
		body, _ := readBody(r)
		var req Request
		require.NoError(t, json.Unmarshal(body, &req))
		if len(req.ID) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeJSON(w, http.StatusOK, &Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]string{"echo": string(body)}})
	}))
	t.Cleanup(srv.Close)
	return srv, &authed
}

func readBody(r *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	return buf.Bytes(), err
}

func TestStdioProxy_RoundTrip(t *testing.T) {
	srv, authed := newProxyUpstream(t)
	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n")

	proxy := NewStdioProxy(srv.URL, "dep_pat123", nil)
	require.NoError(t, proxy.Serve(context.Background(), &in, &out))

	require.Len(t, *authed, 1)
	assert.Equal(t, "Bearer dep_pat123", (*authed)[0])

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 1)
	var resp Response
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &resp))
	assert.Equal(t, "1", string(resp.ID))
	assert.Nil(t, resp.Error)
}

func TestStdioProxy_NotificationProducesNoOutput(t *testing.T) {
	srv, _ := newProxyUpstream(t)
	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")

	proxy := NewStdioProxy(srv.URL, "dep_pat123", nil)
	require.NoError(t, proxy.Serve(context.Background(), &in, &out))
	assert.Empty(t, out.String())
}

func TestStdioProxy_Upstream401WrappedAsJSONRPC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}` + "\n")

	proxy := NewStdioProxy(srv.URL, "", nil)
	require.NoError(t, proxy.Serve(context.Background(), &in, &out))

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, 1)
	var resp Response
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeUnauthorized, resp.Error.Code)
}

func TestStdioProxy_TransportFailure(t *testing.T) {
	// Point at a closed port: the request must fail into a JSON-RPC error
	// rather than a raw socket error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	url := srv.URL
	srv.Close()

	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")

	proxy := NewStdioProxy(url, "dep_pat123", nil)
	require.NoError(t, proxy.Serve(context.Background(), &in, &out))

	var resp Response
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeInternalError, resp.Error.Code)
}

func TestStdioProxy_CancelledContextStops(t *testing.T) {
	srv, _ := newProxyUpstream(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var in, out bytes.Buffer
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")

	proxy := NewStdioProxy(srv.URL, "dep_pat123", nil)
	require.NoError(t, proxy.Serve(ctx, &in, &out))
	assert.Empty(t, out.String())
}
