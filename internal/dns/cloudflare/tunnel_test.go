package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// tunnelAPI stubs the account-scoped tunnel endpoints; WithAccountID pins the account so these tests never call GET /accounts.
type tunnelAPI struct {
	mu        sync.Mutex
	reqs      []string
	create    any
	list      any
	get       any
	rotate    any
	token     string
	status    int
	errBody   string
	failConns bool
	// ingress is the tunnel's stored configuration, mutated by PUT
	// .../configurations and served back by GET .../configurations — lets
	// tests exercise the real read-modify-write round trip.
	ingress []map[string]any
}

func (f *tunnelAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.reqs = append(f.reqs, r.Method+" "+r.URL.Path)
	status := f.status
	errBody := f.errBody
	f.mu.Unlock()
	if status != 0 {
		w.WriteHeader(status)
		if errBody != "" {
			_, _ = w.Write([]byte(errBody))
		}
		return
	}
	if r.Method == http.MethodDelete && hasSuffix(r.URL.Path, "/connections") && f.failConns {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":1000,"message":"boom"}],"result":null}`))
		return
	}
	if hasSuffix(r.URL.Path, "/configurations") {
		f.serveConfigurations(w, r)
		return
	}
	writeJSON(w, map[string]any{"success": true, "result": f.resultFor(r), "errors": []any{}})
}

// serveConfigurations backs the read-modify-write round trip: PUT replaces the stored ingress, GET (and the
// PUT response) echo it back.
func (f *tunnelAPI) serveConfigurations(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Method == http.MethodPut {
		var body struct {
			Config struct {
				Ingress []map[string]any `json:"ingress"`
			} `json:"config"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.ingress = body.Config.Ingress
	}
	writeJSON(w, map[string]any{"success": true, "result": map[string]any{
		"config": map[string]any{"ingress": f.ingress},
	}, "errors": []any{}})
}

// resultFor picks the stubbed response body for every other tunnel endpoint by method and path.
func (f *tunnelAPI) resultFor(r *http.Request) any {
	switch {
	case r.Method == http.MethodPost && hasSuffix(r.URL.Path, "/cfd_tunnel"):
		return f.create
	case r.Method == http.MethodGet && hasSuffix(r.URL.Path, "/token"):
		return f.token
	case r.Method == http.MethodGet && hasSuffix(r.URL.Path, "/cfd_tunnel"):
		return f.list
	case r.Method == http.MethodGet:
		return f.get
	case r.Method == http.MethodPatch:
		return f.rotate
	default:
		return f.get
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func newTunnelClient(t *testing.T, api *tunnelAPI) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(api)
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	c := New("test-token", WithBaseURL(u), WithAccountID("acct-1"))
	return c, srv
}

func TestClient_CreateTunnel(t *testing.T) {
	api := &tunnelAPI{create: map[string]any{
		"id": "t1", "name": "api-tunnel", "account_tag": "acct-1", "status": "inactive", "token": "eyJ0b2tlbiJ9",
	}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	tunnel, err := c.CreateTunnel(context.Background(), "api-tunnel")
	require.NoError(t, err)
	assert.Equal(t, "t1", tunnel.ID)
	assert.Equal(t, "api-tunnel", tunnel.Name)
	assert.Equal(t, "eyJ0b2tlbiJ9", tunnel.Token)
	assert.Equal(t, "acct-1", tunnel.AccountID)
	require.Len(t, api.reqs, 1)
	assert.Contains(t, api.reqs[0], "POST /client/v4/accounts/acct-1/cfd_tunnel")
}

func TestClient_CreateTunnel_BadTokenIsInvalidInput(t *testing.T) {
	api := &tunnelAPI{status: http.StatusUnauthorized, errBody: `{"success":false,"errors":[{"code":10000,"message":"Invalid token"}],"result":null}`}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	_, err := c.CreateTunnel(context.Background(), "api-tunnel")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestClient_ListTunnels(t *testing.T) {
	api := &tunnelAPI{list: []map[string]any{
		{"id": "t1", "name": "one", "account_tag": "acct-1", "status": "healthy"},
		{"id": "t2", "name": "two", "account_tag": "acct-1", "status": "inactive"},
	}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	tunnels, err := c.ListTunnels(context.Background())
	require.NoError(t, err)
	require.Len(t, tunnels, 2)
	assert.Equal(t, "one", tunnels[0].Name)
}

func TestClient_GetTunnel(t *testing.T) {
	api := &tunnelAPI{get: map[string]any{"id": "t1", "name": "one", "account_tag": "acct-1", "status": "healthy"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	tunnel, err := c.GetTunnel(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "healthy", tunnel.Status)
}

func TestClient_DeleteTunnel(t *testing.T) {
	api := &tunnelAPI{get: map[string]any{"id": "t1"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	require.NoError(t, c.DeleteTunnel(context.Background(), "t1"))
	assert.Contains(t, api.reqs[0], "DELETE /client/v4/accounts/acct-1/cfd_tunnel/t1")
}

func TestClient_DeleteTunnel_MissingIsNoOp(t *testing.T) {
	api := &tunnelAPI{status: http.StatusNotFound, errBody: `{"success":false,"errors":[{"code":1001,"message":"not found"}],"result":null}`}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	require.NoError(t, c.DeleteTunnel(context.Background(), "ghost"), "delete of an absent tunnel is idempotent")
}

func TestClient_RouteTunnelHostname(t *testing.T) {
	api := &tunnelAPI{get: map[string]any{"id": "t1"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	require.NoError(t, c.RouteTunnelHostname(context.Background(), "t1", "app.example.com", "http://localhost:80"))
	assert.Contains(t, api.reqs[0], "GET /client/v4/accounts/acct-1/cfd_tunnel/t1/configurations")
	assert.Contains(t, api.reqs[1], "PUT /client/v4/accounts/acct-1/cfd_tunnel/t1/configurations")
}

// TestClient_RouteTunnelHostname_SecondHostnameSurvivesFirst is the
// regression test for the replace-only bug: routing a second hostname on the
// same tunnel used to PUT a two-rule ingress (the new hostname plus the
// catch-all), silently dropping the first hostname. Read-modify-write means
// both survive.
func TestClient_RouteTunnelHostname_SecondHostnameSurvivesFirst(t *testing.T) {
	api := &tunnelAPI{get: map[string]any{"id": "t1"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()

	require.NoError(t, c.RouteTunnelHostname(context.Background(), "t1", "app.example.com", "http://app:80"))
	require.NoError(t, c.RouteTunnelHostname(context.Background(), "t1", "api.example.com", "http://api:8080"))

	ingress, err := c.tunnelIngress(context.Background(), "acct-1", "t1")
	require.NoError(t, err)
	require.Len(t, ingress, 2, "both hostnames survive")
	assert.Equal(t, "app.example.com", ingress[0].Hostname)
	assert.Equal(t, "http://app:80", ingress[0].Service)
	assert.Equal(t, "api.example.com", ingress[1].Hostname)
	assert.Equal(t, "http://api:8080", ingress[1].Service)
}

// TestClient_RouteTunnelHostname_UpsertsExistingHostname confirms re-routing
// the same hostname updates its rule in place rather than duplicating it.
func TestClient_RouteTunnelHostname_UpsertsExistingHostname(t *testing.T) {
	api := &tunnelAPI{get: map[string]any{"id": "t1"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()

	require.NoError(t, c.RouteTunnelHostname(context.Background(), "t1", "app.example.com", "http://app:80"))
	require.NoError(t, c.RouteTunnelHostname(context.Background(), "t1", "app.example.com", "http://app:8081"))

	ingress, err := c.tunnelIngress(context.Background(), "acct-1", "t1")
	require.NoError(t, err)
	require.Len(t, ingress, 1)
	assert.Equal(t, "http://app:8081", ingress[0].Service)
}

// TestClient_RemoveTunnelHostname_KeepsOtherRules is the regression test for
// route removal: removing one hostname's ingress rule must not clobber the
// others routed on the same tunnel.
func TestClient_RemoveTunnelHostname_KeepsOtherRules(t *testing.T) {
	api := &tunnelAPI{get: map[string]any{"id": "t1"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()

	require.NoError(t, c.RouteTunnelHostname(context.Background(), "t1", "app.example.com", "http://app:80"))
	require.NoError(t, c.RouteTunnelHostname(context.Background(), "t1", "api.example.com", "http://api:8080"))
	require.NoError(t, c.RemoveTunnelHostname(context.Background(), "t1", "app.example.com"))

	ingress, err := c.tunnelIngress(context.Background(), "acct-1", "t1")
	require.NoError(t, err)
	require.Len(t, ingress, 1)
	assert.Equal(t, "api.example.com", ingress[0].Hostname)
}

func TestClient_RotateTunnelCredentials(t *testing.T) {
	api := &tunnelAPI{rotate: map[string]any{"id": "t1", "token": "eyJuZXcifQ"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	token, err := c.RotateTunnelCredentials(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "eyJuZXcifQ", token)
	// The rotate PATCH plus the force-disconnect DELETE.
	assert.Contains(t, api.reqs[0], "PATCH /client/v4/accounts/acct-1/cfd_tunnel/t1")
	assert.Contains(t, api.reqs[1], "DELETE /client/v4/accounts/acct-1/cfd_tunnel/t1/connections")
}

func TestClient_RotateTunnelCredentials_EmptyTokenIsRetryable(t *testing.T) {
	api := &tunnelAPI{rotate: map[string]any{"id": "t1"}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	_, err := c.RotateTunnelCredentials(context.Background(), "t1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestClient_RotateTunnelCredentials_DisconnectFailureSurfaces(t *testing.T) {
	api := &tunnelAPI{rotate: map[string]any{"id": "t1", "token": "eyJuZXcifQ"}, failConns: true}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	token, err := c.RotateTunnelCredentials(context.Background(), "t1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	assert.Equal(t, "eyJuZXcifQ", token, "the new token is still returned")
}

func TestClient_TunnelToken(t *testing.T) {
	api := &tunnelAPI{token: "eyJjdXJyZW50In0"}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	token, err := c.TunnelToken(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "eyJjdXJyZW50In0", token)
}

func TestClient_AccountIDResolvesFromAccounts(t *testing.T) {
	api := &tunnelAPI{create: map[string]any{
		"id": "t1", "name": "api-tunnel", "account_tag": "acct-9", "status": "inactive", "token": "tok",
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasSuffix(r.URL.Path, "/accounts") {
			writeJSON(w, map[string]any{"success": true, "result": []map[string]any{{"id": "acct-9", "name": "My Account"}}, "errors": []any{}})
			return
		}
		writeJSON(w, map[string]any{"success": true, "result": api.create, "errors": []any{}})
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	c := New("test-token", WithBaseURL(u)) // no WithAccountID → resolve via GET /accounts
	tunnel, err := c.CreateTunnel(context.Background(), "api-tunnel")
	require.NoError(t, err)
	assert.Equal(t, "acct-9", tunnel.AccountID)
}

// A DNS+Tunnel token usually can't list accounts, but every zone it can read names its account.
func TestClient_AccountID_FallsBackToZoneAccount(t *testing.T) {
	api := &tunnelAPI{create: map[string]any{
		"id": "t1", "name": "api-tunnel", "account_tag": "acct-from-zone", "status": "inactive", "token": "tok",
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasSuffix(r.URL.Path, "/accounts") {
			writeJSON(w, map[string]any{"success": true, "result": []map[string]any{}, "errors": []any{}})
			return
		}
		if hasSuffix(r.URL.Path, "/zones") {
			writeJSON(w, map[string]any{"success": true, "result": []map[string]any{{"id": "z1", "name": "example.com", "account": map[string]any{"id": "acct-from-zone"}}}, "errors": []any{}})
			return
		}
		assert.Contains(t, r.URL.Path, "/accounts/acct-from-zone/cfd_tunnel")
		writeJSON(w, map[string]any{"success": true, "result": api.create, "errors": []any{}})
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	c := New("test-token", WithBaseURL(u))
	tunnel, err := c.CreateTunnel(context.Background(), "api-tunnel")
	require.NoError(t, err)
	assert.Equal(t, "acct-from-zone", tunnel.AccountID)
}

func TestClient_AccountID_NoAccountAndNoZoneIsInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"success": true, "result": []map[string]any{}, "errors": []any{}})
	}))
	defer srv.Close()
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	c := New("test-token", WithBaseURL(u))
	_, err = c.CreateTunnel(context.Background(), "api-tunnel")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	assert.Contains(t, err.Error(), "Zone: Read")
}

var _ = json.Marshal

func TestClient_ListTunnelHostnames_DropsCatchAll(t *testing.T) {
	api := &tunnelAPI{get: map[string]any{"id": "t1"}, ingress: []map[string]any{
		{"hostname": "app.example.com", "service": "http://web:80"},
		{"hostname": "api.example.com", "service": "http://api:8080"},
		{"service": "http_status:404"},
	}}
	c, srv := newTunnelClient(t, api)
	defer srv.Close()
	routes, err := c.ListTunnelHostnames(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, []dns.TunnelRoute{
		{Hostname: "app.example.com", Service: "http://web:80"},
		{Hostname: "api.example.com", Service: "http://api:8080"},
	}, routes)
}
