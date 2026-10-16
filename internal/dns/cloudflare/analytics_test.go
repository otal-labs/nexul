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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type graphqlStub struct {
	status   int
	response string
	path     string
	auth     string
	request  struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
}

func (g *graphqlStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.path, g.auth = r.URL.Path, r.Header.Get("Authorization")
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &g.request)
	if g.status != 0 {
		w.WriteHeader(g.status)
	}
	_, _ = io.WriteString(w, g.response)
}

func analyticsClient(t *testing.T, stub *graphqlStub) *Client {
	t.Helper()
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	return New("test-token", WithBaseURL(u))
}

const groupsResponse = `{"data":{"viewer":{"zones":[{"httpRequestsAdaptiveGroups":[
 {"count":40,"dimensions":{"datetimeMinute":"2026-10-16T10:39:00Z","clientRequestHTTPHost":"App.Example.com","edgeResponseStatus":200}},
 {"count":3,"dimensions":{"datetimeMinute":"2026-10-16T10:39:00Z","clientRequestHTTPHost":"app.example.com","edgeResponseStatus":502}},
 {"count":2,"dimensions":{"datetimeMinute":"2026-10-16T10:39:00Z","clientRequestHTTPHost":"app.example.com","edgeResponseStatus":404}},
 {"count":9,"dimensions":{"datetimeMinute":"2026-10-16T10:40:00Z","clientRequestHTTPHost":"app.example.com","edgeResponseStatus":599}},
 {"count":5,"dimensions":{"datetimeMinute":"2026-10-16T10:39:00Z","clientRequestHTTPHost":"api.example.com","edgeResponseStatus":500}}
]}]}},"errors":null}`

func TestClient_HostnameRequests_SumsGroupsPerHostAndMinuteAndCountsEvery5xx(t *testing.T) {
	stub := &graphqlStub{response: groupsResponse}
	c := analyticsClient(t, stub)
	from := time.Date(2026, 10, 16, 10, 39, 0, 0, time.UTC)

	got, err := c.HostnameRequests(context.Background(), "z1", []string{"app.example.com", "api.example.com"}, from, from.Add(2*time.Minute))

	require.NoError(t, err)
	assert.Equal(t, []dns.HostMinute{
		{Hostname: "api.example.com", Minute: from, Requests: 5, Errors: 5},
		{Hostname: "app.example.com", Minute: from, Requests: 45, Errors: 3},
		{Hostname: "app.example.com", Minute: from.Add(time.Minute), Requests: 9, Errors: 9},
	}, got)
	assert.Equal(t, "/client/v4/graphql", stub.path)
	assert.Equal(t, "Bearer test-token", stub.auth, "the connector's own token, nothing new")
	assert.Equal(t, map[string]any{
		"zone": "z1", "from": "2026-10-16T10:39:00Z", "to": "2026-10-16T10:41:00Z", "hosts": []any{"app.example.com", "api.example.com"},
	}, stub.request.Variables)
	assert.Contains(t, stub.request.Query, "httpRequestsAdaptiveGroups")
	assert.Contains(t, stub.request.Query, "clientRequestHTTPHost_in")
	assert.Contains(t, stub.request.Query, "edgeResponseStatus")
}

func TestClient_HostnameRequests_NoTrafficIsAnEmptyAnswer(t *testing.T) {
	c := analyticsClient(t, &graphqlStub{response: `{"data":{"viewer":{"zones":[{"httpRequestsAdaptiveGroups":[]}]}},"errors":null}`})
	got, err := c.HostnameRequests(context.Background(), "z1", []string{"app.example.com"}, time.Now(), time.Now())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestClient_HostnameRequests_Failures(t *testing.T) {
	tests := []struct {
		name      string
		stub      graphqlStub
		wantInval bool
		wantRetry bool
		contains  string
	}{
		{name: "a token without Analytics Read", stub: graphqlStub{response: `{"data":null,"errors":[{"message":"zone 'z1' does not have access to the path","extensions":{"code":"authz"}}]}`}, wantInval: true, contains: "Analytics"},
		{name: "rejected outright", stub: graphqlStub{status: http.StatusForbidden, response: `{}`}, wantInval: true},
		{name: "a server error", stub: graphqlStub{status: http.StatusBadGateway, response: `oops`}, wantRetry: true},
		{name: "another GraphQL error", stub: graphqlStub{response: `{"errors":[{"message":"query is too expensive"}]}`}, wantRetry: true, contains: "too expensive"},
		{name: "a body that is not JSON", stub: graphqlStub{response: `<html>`}, contains: "decode"},
		{name: "a minute that is not a time", stub: graphqlStub{response: `{"data":{"viewer":{"zones":[{"httpRequestsAdaptiveGroups":[{"count":1,"dimensions":{"datetimeMinute":"soon"}}]}]}}}`}, contains: "minute"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := analyticsClient(t, &tt.stub)
			_, err := c.HostnameRequests(context.Background(), "z1", nil, time.Now(), time.Now())
			require.Error(t, err)
			assert.Equal(t, tt.wantInval, apperrsIs(err, apperrs.ErrInvalid))
			assert.Equal(t, tt.wantRetry, apperrsIs(err, apperrs.ErrRetryable))
			if tt.contains != "" {
				assert.ErrorContains(t, err, tt.contains)
			}
		})
	}
}

func TestClient_HostnameRequests_AnUnreachableAPIIsRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	u, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	srv.Close()
	_, err = New("tok", WithBaseURL(u)).HostnameRequests(context.Background(), "z1", nil, time.Now(), time.Now())
	assert.ErrorIs(t, err, apperrs.ErrRetryable)
}

func apperrsIs(err, target error) bool { return errors.Is(err, target) }
