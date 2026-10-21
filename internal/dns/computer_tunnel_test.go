package dns_test

import (
	"context"
	"encoding/json"
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
)

// fakeCloudflare keeps just enough Cloudflare state to prove a computer tunnel's teardown leaves nothing behind.
type fakeCloudflare struct {
	mu      sync.Mutex
	calls   []string
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
	{http.MethodDelete, regexp.MustCompile(`^zones/[^/]+/dns_records/([^/]+)$`), "delete_record"},
	{http.MethodDelete, regexp.MustCompile(`/cfd_tunnel/([^/]+)/connections$`), "delete_connections"},
	{http.MethodGet, regexp.MustCompile(`/cfd_tunnel/([^/]+)$`), "get_tunnel"},
	{http.MethodPatch, regexp.MustCompile(`/cfd_tunnel/([^/]+)$`), "rotate"},
	{http.MethodDelete, regexp.MustCompile(`/cfd_tunnel/([^/]+)$`), "delete_tunnel"},
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
	case "delete_record":
		f.remove(w, f.records, id)
	default:
		f.serveTunnel(w, op, id, body)
	}
}

func (f *fakeCloudflare) serveTunnel(w http.ResponseWriter, op, id string, body map[string]any) {
	_, exists := f.tunnels[id]
	switch op {
	case "get_tunnel", "rotate", "delete_connections":
		if !exists {
			reply(w, 404, nil, "tunnel not found")
			return
		}
	}
	switch op {
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
	case "delete_tunnel":
		if f.tunnels[id] {
			reply(w, 400, nil, "cannot delete a tunnel with active connections")
			return
		}
		delete(f.ingress, id)
		f.remove(w, f.tunnels, id)
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

// leftover is a computer tunnel pairing through one made: its tunnel, route, DNS record and Access app, still connected.
func (f *fakeCloudflare) leftover() dns.ComputerTunnel {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tunnels["tun-1"] = true
	f.ingress["tun-1"] = []map[string]any{{"hostname": "laptop-a1b2c3d4.example.com", "service": "http://127.0.0.1:3773"}}
	f.records["rec-1"] = map[string]any{"type": "CNAME", "name": "laptop-a1b2c3d4"}
	f.apps["app-1"] = "laptop-a1b2c3d4.example.com"
	return dns.ComputerTunnel{TunnelID: "tun-1", Hostname: "laptop-a1b2c3d4.example.com", ZoneID: "z-root", RecordID: "rec-1", AccessAppID: "app-1"}
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

func TestComputerTunnel_Delete_LeavesNothingBehind(t *testing.T) {
	svc, fake := newComputerTunnelService(t, "https://example.com")
	ct := fake.leftover()

	require.NoError(t, svc.DeleteComputerTunnel(t.Context(), ct))
	fake.order(t, "rotate", "delete_tunnel", "delete_record", "delete_app")
	fake.empty(t)

	require.NoError(t, svc.DeleteComputerTunnel(t.Context(), ct), "a second teardown finds everything gone and succeeds")
}

func TestComputerTunnel_Delete_StopsWhenTheComputerCannotBeDisconnected(t *testing.T) {
	svc, fake := newComputerTunnelService(t, "https://example.com")
	ct := fake.leftover()
	fake.fail["rotate"] = http.StatusInternalServerError

	require.Error(t, svc.DeleteComputerTunnel(t.Context(), ct))
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
			ct := fake.leftover()
			fake.fail[step] = http.StatusInternalServerError
			require.Error(t, svc.DeleteComputerTunnel(t.Context(), ct))
		})
	}
}
