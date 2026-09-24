package dns_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/dns/cloudflare"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeCloudflare keeps just enough Cloudflare state to prove a computer tunnel's create and teardown leave nothing behind.
type fakeCloudflare struct {
	mu      sync.Mutex
	calls   []string
	seq     int
	tunnels map[string]bool // id → a connector is connected
	ingress map[string][]map[string]any
	records map[string]map[string]any
	apps    map[string]string
	fail    map[string]int
}

func newFakeCloudflare() *fakeCloudflare {
	return &fakeCloudflare{
		tunnels: map[string]bool{}, ingress: map[string][]map[string]any{},
		records: map[string]map[string]any{}, apps: map[string]string{}, fail: map[string]int{},
	}
}

func (f *fakeCloudflare) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	op, id := route(r.Method, strings.TrimPrefix(r.URL.Path, "/client/v4/"))
	f.calls = append(f.calls, op)
	if status := f.fail[op]; status != 0 {
		reply(w, status, nil, "injected failure")
		return
	}
	f.serve(w, op, id, body)
}

// routes names the Cloudflare operation behind each request; the capture group is the id it acts on.
var routes = []struct {
	method string
	path   *regexp.Regexp
	op     string
}{
	{http.MethodGet, regexp.MustCompile(`^zones$`), "list_zones"},
	{http.MethodPost, regexp.MustCompile(`^zones/[^/]+/dns_records$`), "create_record"},
	{http.MethodDelete, regexp.MustCompile(`^zones/[^/]+/dns_records/([^/]+)$`), "delete_record"},
	{http.MethodPost, regexp.MustCompile(`/cfd_tunnel$`), "create_tunnel"},
	{http.MethodGet, regexp.MustCompile(`/cfd_tunnel/([^/]+)/configurations$`), "get_config"},
	{http.MethodPut, regexp.MustCompile(`/cfd_tunnel/([^/]+)/configurations$`), "put_config"},
	{http.MethodDelete, regexp.MustCompile(`/cfd_tunnel/([^/]+)/connections$`), "delete_connections"},
	{http.MethodGet, regexp.MustCompile(`/cfd_tunnel/([^/]+)/token$`), "tunnel_token"},
	{http.MethodGet, regexp.MustCompile(`/cfd_tunnel/([^/]+)$`), "get_tunnel"},
	{http.MethodPatch, regexp.MustCompile(`/cfd_tunnel/([^/]+)$`), "rotate"},
	{http.MethodDelete, regexp.MustCompile(`/cfd_tunnel/([^/]+)$`), "delete_tunnel"},
	{http.MethodPost, regexp.MustCompile(`/access/service_tokens$`), "create_service_token"},
	{http.MethodPost, regexp.MustCompile(`/access/apps$`), "create_app"},
	{http.MethodDelete, regexp.MustCompile(`/access/apps/([^/]+)$`), "delete_app"},
}

func route(method, path string) (string, string) {
	for _, r := range routes {
		m := r.path.FindStringSubmatch(path)
		if r.method != method || m == nil {
			continue
		}
		return r.op, m[len(m)-1]
	}
	return "unknown " + method + " " + path, ""
}

func (f *fakeCloudflare) serve(w http.ResponseWriter, op, id string, body map[string]any) {
	switch op {
	case "list_zones":
		reply(w, 200, []map[string]string{{"id": "z-root", "name": "example.com"}, {"id": "z-dev", "name": "dev.example.com"}}, "")
	case "create_record":
		f.seq++
		recID := fmt.Sprintf("rec-%d", f.seq)
		f.records[recID] = body
		reply(w, 200, map[string]any{"id": recID, "type": body["type"], "name": body["name"], "content": body["content"]}, "")
	case "delete_record":
		f.remove(w, f.records, id)
	case "create_tunnel":
		f.seq++
		tunID := fmt.Sprintf("tun-%d", f.seq)
		f.tunnels[tunID] = false
		reply(w, 200, map[string]any{"id": tunID, "name": body["name"], "status": "inactive", "token": "tok-" + tunID}, "")
	default:
		f.serveTunnel(w, op, id, body)
	}
}

func (f *fakeCloudflare) serveTunnel(w http.ResponseWriter, op, id string, body map[string]any) {
	_, exists := f.tunnels[id]
	switch op {
	case "put_config", "get_config", "get_tunnel", "rotate", "delete_connections", "tunnel_token":
		if !exists {
			reply(w, 404, nil, "tunnel not found")
			return
		}
	}
	switch op {
	case "put_config":
		cfg, _ := body["config"].(map[string]any)
		rules, _ := cfg["ingress"].([]any)
		f.ingress[id] = nil
		for _, r := range rules {
			f.ingress[id] = append(f.ingress[id], r.(map[string]any))
		}
		reply(w, 200, map[string]any{"config": map[string]any{"ingress": f.ingress[id]}}, "")
	case "get_config":
		reply(w, 200, map[string]any{"config": map[string]any{"ingress": f.ingress[id]}}, "")
	case "get_tunnel":
		status := "inactive"
		if f.tunnels[id] {
			status = "healthy"
		}
		reply(w, 200, map[string]any{"id": id, "status": status}, "")
	case "rotate":
		f.tunnels[id] = false
		reply(w, 200, map[string]any{"id": id, "token": "rotated-" + id}, "")
	case "delete_connections":
		reply(w, 200, nil, "")
	case "tunnel_token":
		reply(w, 200, "tok-"+id, "")
	case "delete_tunnel":
		if f.tunnels[id] {
			reply(w, 400, nil, "cannot delete a tunnel with active connections")
			return
		}
		delete(f.ingress, id)
		f.remove(w, f.tunnels, id)
	default:
		f.serveAccess(w, op, id, body)
	}
}

func (f *fakeCloudflare) serveAccess(w http.ResponseWriter, op, id string, body map[string]any) {
	switch op {
	case "create_service_token":
		reply(w, 200, map[string]any{"id": "st-1", "client_id": "cid.access", "client_secret": "secret"}, "")
	case "create_app":
		f.seq++
		appID := fmt.Sprintf("app-%d", f.seq)
		f.apps[appID] = body["domain"].(string)
		reply(w, 200, map[string]any{"id": appID}, "")
	default:
		f.remove(w, f.apps, id)
	}
}

func drop[V any](m map[string]V, id string) bool {
	_, ok := m[id]
	delete(m, id)
	return ok
}

func (f *fakeCloudflare) remove(w http.ResponseWriter, m any, id string) {
	found := false
	switch m := m.(type) {
	case map[string]bool:
		found = drop(m, id)
	case map[string]string:
		found = drop(m, id)
	case map[string]map[string]any:
		found = drop(m, id)
	}
	if !found {
		reply(w, 404, nil, "not found")
		return
	}
	reply(w, 200, map[string]string{"id": id}, "")
}

func reply(w http.ResponseWriter, status int, result any, errMsg string) {
	w.WriteHeader(status)
	out := map[string]any{"success": errMsg == "", "result": result, "errors": []any{}}
	if errMsg != "" {
		out["errors"] = []map[string]any{{"code": 1000, "message": errMsg}}
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (f *fakeCloudflare) connect(tunnelID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tunnels[tunnelID] = true
}

func (f *fakeCloudflare) empty(t *testing.T) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Empty(t, f.tunnels, "tunnels left behind")
	assert.Empty(t, f.ingress, "routes left behind")
	assert.Empty(t, f.records, "DNS records left behind")
	assert.Empty(t, f.apps, "Access apps left behind")
}

func (f *fakeCloudflare) order(t *testing.T, ops ...string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	last := -1
	for _, op := range ops {
		i := slices.Index(f.calls, op)
		require.GreaterOrEqual(t, i, 0, "%s never called in %v", op, f.calls)
		assert.Greater(t, i, last, "%s out of order in %v", op, f.calls)
		last = i
	}
}

type settingsAt string

func (s settingsAt) GetInstanceURL(_ context.Context) (string, error) { return string(s), nil }

func newComputerTunnelService(t *testing.T, instanceURL string) (*dns.Service, *fakeCloudflare) {
	t.Helper()
	fake := newFakeCloudflare()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	client := cloudflare.New("cf-token", cloudflare.WithBaseURL(base), cloudflare.WithAccountID("acct-1"))
	svc := dns.NewService(dns.Config{
		Repo: openStore(t).DNS, Provider: client, TunnelProvider: client, AccessProvider: client,
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"), Settings: settingsAt(instanceURL),
	})
	return svc, fake
}

func TestComputerTunnel_CreateStatusDelete_LeavesNothingBehind(t *testing.T) {
	svc, fake := newComputerTunnelService(t, "https://nexul.dev.example.com")

	ct, err := svc.CreateComputerTunnel(t.Context(), "Onik Laptop", 3773)
	require.NoError(t, err)
	assert.Regexp(t, `^onik-laptop-[a-z2-7]{8}\.dev\.example\.com$`, ct.Hostname, "longest matching zone wins")
	assert.Equal(t, "z-dev", ct.ZoneID)
	fake.order(t, "create_tunnel", "create_app", "put_config", "create_record")
	fake.mu.Lock()
	assert.Equal(t, ct.Hostname, fake.apps[ct.AccessAppID])
	assert.Equal(t, ct.Hostname, fake.ingress[ct.TunnelID][0]["hostname"])
	assert.Equal(t, "http://127.0.0.1:3773", fake.ingress[ct.TunnelID][0]["service"])
	rec := fake.records[ct.RecordID]
	assert.Equal(t, "CNAME", rec["type"])
	assert.Equal(t, ct.TunnelID+".cfargotunnel.com", rec["content"])
	assert.Equal(t, true, rec["proxied"])
	assert.Equal(t, strings.TrimSuffix(ct.Hostname, ".dev.example.com"), rec["name"])
	fake.mu.Unlock()

	token, err := svc.ComputerTunnelToken(t.Context(), ct.TunnelID)
	require.NoError(t, err)
	assert.Equal(t, "tok-"+ct.TunnelID, token)
	status, err := svc.ComputerTunnelStatus(t.Context(), ct.TunnelID)
	require.NoError(t, err)
	assert.Equal(t, "inactive", status)

	fake.connect(ct.TunnelID)
	status, err = svc.ComputerTunnelStatus(t.Context(), ct.TunnelID)
	require.NoError(t, err)
	assert.Equal(t, "healthy", status)

	require.NoError(t, svc.DeleteComputerTunnel(t.Context(), *ct))
	fake.order(t, "rotate", "delete_tunnel", "delete_record", "delete_app")
	fake.empty(t)

	require.NoError(t, svc.DeleteComputerTunnel(t.Context(), *ct), "a second teardown finds everything gone and succeeds")
}

func TestComputerTunnel_CreateFailure_RollsBackEverything(t *testing.T) {
	for _, step := range []string{"create_app", "put_config", "create_record"} {
		t.Run(step, func(t *testing.T) {
			svc, fake := newComputerTunnelService(t, "https://example.com")
			fake.fail[step] = http.StatusBadGateway

			_, err := svc.CreateComputerTunnel(t.Context(), "Laptop", 3773)
			require.Error(t, err)
			fake.empty(t)
		})
	}
}

func TestComputerTunnel_Create_RejectsWhatCannotWork(t *testing.T) {
	tests := []struct {
		name        string
		instanceURL string
		port        int
		fail        string
	}{
		{"port zero", "https://example.com", 0, ""},
		{"port too high", "https://example.com", 70000, ""},
		{"instance outside every zone", "https://nexul.other.org", 3773, ""},
		{"instance url unusable", "not a url", 3773, ""},
		{"zones unreadable", "https://example.com", 3773, "list_zones"},
		{"tunnel create refused", "https://example.com", 3773, "create_tunnel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, fake := newComputerTunnelService(t, tt.instanceURL)
			if tt.fail != "" {
				fake.fail[tt.fail] = http.StatusBadRequest
			}
			_, err := svc.CreateComputerTunnel(t.Context(), "Laptop", tt.port)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			fake.empty(t)
		})
	}
}

func TestComputerTunnel_Delete_StopsWhenTheComputerCannotBeDisconnected(t *testing.T) {
	svc, fake := newComputerTunnelService(t, "https://example.com")
	ct, err := svc.CreateComputerTunnel(t.Context(), "Laptop", 3773)
	require.NoError(t, err)
	fake.fail["rotate"] = http.StatusInternalServerError

	require.Error(t, svc.DeleteComputerTunnel(t.Context(), *ct))
	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.Contains(t, fake.tunnels, ct.TunnelID)
	assert.Contains(t, fake.records, ct.RecordID, "the record stays until the tunnel is gone")
	assert.Contains(t, fake.apps, ct.AccessAppID, "the hostname stays closed until the tunnel is gone")
}

func TestComputerTunnel_Delete_SurfacesEachStepsFailure(t *testing.T) {
	for _, step := range []string{"delete_tunnel", "delete_record", "delete_app"} {
		t.Run(step, func(t *testing.T) {
			svc, fake := newComputerTunnelService(t, "https://example.com")
			ct, err := svc.CreateComputerTunnel(t.Context(), "Laptop", 3773)
			require.NoError(t, err)
			fake.fail[step] = http.StatusInternalServerError
			require.Error(t, svc.DeleteComputerTunnel(t.Context(), *ct))
		})
	}
}

func TestComputerTunnel_StatusAndToken_Errors(t *testing.T) {
	svc, fake := newComputerTunnelService(t, "https://example.com")
	_, err := svc.ComputerTunnelStatus(t.Context(), " ")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = svc.ComputerTunnelToken(t.Context(), "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	fake.fail["get_tunnel"] = http.StatusInternalServerError
	fake.fail["tunnel_token"] = http.StatusInternalServerError
	_, err = svc.ComputerTunnelStatus(t.Context(), "tun-9")
	require.ErrorIs(t, err, apperrs.ErrRetryable)
	_, err = svc.ComputerTunnelToken(t.Context(), "tun-9")
	require.ErrorIs(t, err, apperrs.ErrRetryable)
}
