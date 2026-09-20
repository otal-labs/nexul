package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeAPI is a minimal Cloudflare v4 API stub: canned responses per path (page-aware for pagination tests), requests recorded for assertions.
type fakeAPI struct {
	mu           []string
	bodies       []string
	zones        any
	zonesPage2   any
	records      any
	recordsPage2 any
	statusCode   int
	errBody      string
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu = append(f.mu, r.Method+" "+r.URL.Path)
	body, _ := io.ReadAll(r.Body)
	f.bodies = append(f.bodies, string(body))
	if f.statusCode != 0 {
		w.WriteHeader(f.statusCode)
		if f.errBody != "" {
			_, _ = w.Write([]byte(f.errBody))
		}
		return
	}
	writeJSON(w, map[string]any{
		"success": true,
		"result":  f.resultFor(r),
		"errors":  []any{},
	})
}

func (f *fakeAPI) resultFor(r *http.Request) any {
	page := r.URL.Query().Get("page")
	switch r.URL.Path {
	case "/client/v4/user/tokens/verify":
		return map[string]any{"id": "tok", "status": "active"}
	case "/client/v4/zones":
		if page == "2" {
			return f.zonesPage2
		}
		return f.zones
	case "/client/v4/zones/z1/dns_records", "/client/v4/zones/z1/dns_records/r1":
		if page == "2" {
			return f.recordsPage2
		}
		return f.records
	default:
		return f.records
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func newTestClient(t *testing.T, api *fakeAPI) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(api)
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	c := New("test-token", WithBaseURL(u))
	return c, srv
}

func TestClient_Verify(t *testing.T) {
	api := &fakeAPI{zones: []any{}}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	require.NoError(t, c.Verify(context.Background()))
	assert.Contains(t, api.mu[0], "user/tokens/verify")
}

func TestClient_Verify_BadToken(t *testing.T) {
	api := &fakeAPI{statusCode: http.StatusUnauthorized, errBody: `{"success":false,"errors":[{"code":10000,"message":"Invalid token"}],"result":null}`}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	err := c.Verify(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestClient_ListZones(t *testing.T) {
	api := &fakeAPI{zones: []map[string]any{{"id": "z1", "name": "example.com", "status": "active"}}}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	zones, err := c.ListZones(context.Background())
	require.NoError(t, err)
	require.Len(t, zones, 1)
	assert.Equal(t, "example.com", zones[0].Name)
}

func TestClient_ListZones_NotFound(t *testing.T) {
	api := &fakeAPI{statusCode: http.StatusNotFound, errBody: `{"success":false,"errors":[{"code":1001,"message":"Zone not found"}],"result":null}`}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	_, err := c.ListZones(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestClient_ListZones_Transient(t *testing.T) {
	api := &fakeAPI{statusCode: http.StatusBadGateway, errBody: `{"success":false,"errors":[{"code":1000,"message":"boom"}],"result":null}`}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	_, err := c.ListZones(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestClient_NetworkFailure_IsRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	srv.Close() // accept, then the connection dies
	c := New("tok", WithBaseURL(u))
	_, err = c.ListZones(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestClient_ListRecords(t *testing.T) {
	api := &fakeAPI{records: []map[string]any{{"id": "r1", "type": "A", "name": "api.example.com", "content": "1.2.3.4", "ttl": 1}}}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	records, err := c.ListRecords(context.Background(), "z1")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "api.example.com", records[0].Name)
}

func TestClient_CreateRecord(t *testing.T) {
	api := &fakeAPI{records: map[string]any{"id": "r1", "type": "A", "name": "api", "content": "1.2.3.4", "ttl": 300}}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	rec, err := c.CreateRecord(context.Background(), "z1", dns.RecordInput{Type: dns.RecordA, Name: "api", Content: "1.2.3.4", TTL: 300, Proxied: true})
	require.NoError(t, err)
	assert.Equal(t, "r1", rec.ID)
	assert.Equal(t, "z1", rec.ZoneID)
	assert.Contains(t, api.mu[0], "POST /client/v4/zones/z1/dns_records")
	assert.Contains(t, api.bodies[0], `"proxied":true`)
}

func TestClient_UpdateRecord(t *testing.T) {
	api := &fakeAPI{records: map[string]any{"id": "r1", "type": "A", "name": "api", "content": "5.6.7.8", "ttl": 60}}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	rec, err := c.UpdateRecord(context.Background(), "z1", "r1", dns.RecordInput{Type: dns.RecordA, Name: "api", Content: "5.6.7.8", TTL: 60})
	require.NoError(t, err)
	assert.Equal(t, "5.6.7.8", rec.Content)
	assert.Contains(t, api.mu[0], "PATCH /client/v4/zones/z1/dns_records/r1")
}

func TestClient_ListZones_Paginates(t *testing.T) {
	one := make([]map[string]any, 50)
	for i := range one {
		one[i] = map[string]any{"id": string(rune('a' + i)), "name": "zone", "status": "active"}
	}
	api := &fakeAPI{zones: one, zonesPage2: []map[string]any{{"id": "z2", "name": "second.com", "status": "active"}}}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	zones, err := c.ListZones(context.Background())
	require.NoError(t, err)
	assert.Len(t, zones, 51, "page 2 appended to page 1")
	assert.Equal(t, "second.com", zones[50].Name)
}

func TestClient_ListRecords_Paginates(t *testing.T) {
	one := make([]map[string]any, 100)
	for i := range one {
		one[i] = map[string]any{"id": "r", "type": "A", "name": "rec", "content": "1.2.3.4", "ttl": 1}
	}
	api := &fakeAPI{records: one, recordsPage2: []map[string]any{{"id": "r2", "type": "A", "name": "rec2", "content": "1.2.3.4", "ttl": 1}}}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	records, err := c.ListRecords(context.Background(), "z1")
	require.NoError(t, err)
	assert.Len(t, records, 101)
	assert.Equal(t, "rec2", records[100].Name)
}

func TestClient_Options(t *testing.T) {
	api := &fakeAPI{zones: []any{}}
	srv := httptest.NewServer(api)
	defer srv.Close()
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)

	c := New("tok", WithBaseURL(u), WithHTTPClient(&http.Client{}), WithResolver(stubResolver{host: []string{"1.2.3.4"}}))
	require.NoError(t, c.Verify(context.Background()))
	require.NoError(t, c.CheckPropagation(context.Background(), "z1", dns.Record{Type: dns.RecordA, Name: "api", Content: "1.2.3.4"}))
}

func TestClient_DeleteRecord_MissingIsNoOp(t *testing.T) {
	api := &fakeAPI{statusCode: http.StatusNotFound, errBody: `{"success":false,"errors":[{"code":1001,"message":"not found"}],"result":null}`}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	require.NoError(t, c.DeleteRecord(context.Background(), "z1", "ghost"), "delete of an absent record is idempotent")
}

func TestClient_DeleteRecord_FailureSurfaces(t *testing.T) {
	api := &fakeAPI{statusCode: http.StatusUnauthorized, errBody: `{"success":false,"errors":[{"code":10000,"message":"bad token"}],"result":null}`}
	c, srv := newTestClient(t, api)
	defer srv.Close()
	err := c.DeleteRecord(context.Background(), "z1", "r1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

type stubResolver struct {
	host  []string
	cname string
	txt   []string
}

func (s stubResolver) LookupHost(_ context.Context, _ string) ([]string, error) { return s.host, nil }
func (s stubResolver) LookupCNAME(_ context.Context, _ string) (string, error)  { return s.cname, nil }
func (s stubResolver) LookupTXT(_ context.Context, _ string) ([]string, error)  { return s.txt, nil }

func TestClient_CheckPropagation(t *testing.T) {
	tests := []struct {
		name    string
		rec     dns.Record
		resolve dnsResolver
		wantErr bool
	}{
		{"A propagated", dns.Record{Type: dns.RecordA, Name: "api.example.com", Content: "1.2.3.4"},
			stubResolver{host: []string{"1.2.3.4"}}, false},
		{"A not yet propagated", dns.Record{Type: dns.RecordA, Name: "api.example.com", Content: "1.2.3.4"},
			stubResolver{host: []string{"5.6.7.8"}}, true},
		{"CNAME propagated", dns.Record{Type: dns.RecordCNAME, Name: "api.example.com", Content: "target.example.net"},
			stubResolver{cname: "target.example.net."}, false},
		{"TXT propagated", dns.Record{Type: dns.RecordTXT, Name: "_challenge.example.com", Content: "abc"},
			stubResolver{txt: []string{"abc"}}, false},
		{"unsupported type", dns.Record{Type: "SRV", Name: "x", Content: "y"},
			stubResolver{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{resolve: tt.resolve}
			err := c.CheckPropagation(context.Background(), "z1", tt.rec)
			if tt.wantErr {
				require.Error(t, err)
				if tt.rec.Type != "SRV" {
					assert.True(t, errors.Is(err, apperrs.ErrRetryable), "unpropagated is retryable: %v", err)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
