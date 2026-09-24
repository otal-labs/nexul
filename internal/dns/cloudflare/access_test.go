package cloudflare

import (
	"encoding/json"
	"io"
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

type recorded struct {
	status int
	body   string
}

// accessFixture replays recorded Cloudflare responses keyed by "METHOD path" and keeps each request body it saw.
type accessFixture struct {
	mu      sync.Mutex
	replies map[string]recorded
	bodies  map[string]map[string]any
}

func (f *accessFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	raw, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	var body map[string]any
	if json.Unmarshal(raw, &body) == nil {
		f.bodies[key] = body
	}
	reply, ok := f.replies[key]
	f.mu.Unlock()
	if !ok {
		reply = recorded{http.StatusNotFound, `{"success":false,"errors":[{"code":7003,"message":"Could not route to ` + r.URL.Path + `"}]}`}
	}
	w.WriteHeader(reply.status)
	_, _ = w.Write([]byte(reply.body))
}

func newAccessClient(t *testing.T, replies map[string]recorded) (*Client, *accessFixture) {
	t.Helper()
	f := &accessFixture{replies: replies, bodies: map[string]map[string]any{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	return New("tok", WithBaseURL(base), WithAccountID("acct-1")), f
}

const (
	appsPath   = "/client/v4/accounts/acct-1/access/apps"
	tokensPath = "/client/v4/accounts/acct-1/access/service_tokens"
	// Assumed shape of the pre-onboarding answer; the phrase is what zeroTrustMissing keys on.
	zeroTrustOff = `{"success":false,"errors":[{"code":9999,"message":"access.api.error.not_found: Unable to find your Access organization"}]}`
	forbidden    = `{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`
	tokenResult  = `{"success":true,"errors":[],"result":{"id":"st-1","name":"Nexul","client_id":"cid.access","client_secret":"sec-1","expires_at":null}}`
)

var _ dns.AccessProvider = (*Client)(nil)

func TestAccess_ZeroTrustDisabled_ReturnsDistinctError(t *testing.T) {
	c, _ := newAccessClient(t, map[string]recorded{
		"POST " + appsPath:   {http.StatusNotFound, zeroTrustOff},
		"POST " + tokensPath: {http.StatusBadRequest, zeroTrustOff},
	})
	_, err := c.CreateAccessApp(t.Context(), "laptop-ab12cd34.example.com", "st-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, dns.ErrZeroTrustDisabled)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "enable Zero Trust")

	_, err = c.CreateServiceToken(t.Context(), "Nexul")
	assert.ErrorIs(t, err, dns.ErrZeroTrustDisabled)
}

func TestAccess_ErrorPaths(t *testing.T) {
	tests := []struct {
		name  string
		reply recorded
		call  func(c *Client) error
		want  error
	}{
		{"create app forbidden", recorded{http.StatusForbidden, forbidden},
			func(c *Client) error { _, err := c.CreateAccessApp(t.Context(), "h.example.com", "st-1"); return err }, apperrs.ErrInvalid},
		{"create app empty id", recorded{http.StatusOK, `{"success":true,"result":{}}`},
			func(c *Client) error { _, err := c.CreateAccessApp(t.Context(), "h.example.com", "st-1"); return err }, apperrs.ErrRetryable},
		{"create app bad result", recorded{http.StatusOK, `{"success":true,"result":"nope"}`},
			func(c *Client) error { _, err := c.CreateAccessApp(t.Context(), "h.example.com", "st-1"); return err }, nil},
		{"create token missing secret", recorded{http.StatusOK, `{"success":true,"result":{"id":"st-1","client_id":"cid"}}`},
			func(c *Client) error { _, err := c.CreateServiceToken(t.Context(), "Nexul"); return err }, apperrs.ErrRetryable},
		{"create token bad result", recorded{http.StatusOK, `{"success":true,"result":[]}`},
			func(c *Client) error { _, err := c.CreateServiceToken(t.Context(), "Nexul"); return err }, nil},
		{"create token server error", recorded{http.StatusInternalServerError, `{"success":false,"errors":[{"code":1,"message":"boom"}]}`},
			func(c *Client) error { _, err := c.CreateServiceToken(t.Context(), "Nexul"); return err }, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newAccessClient(t, map[string]recorded{
				"POST " + appsPath:   tt.reply,
				"POST " + tokensPath: tt.reply,
			})
			err := tt.call(c)
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestAccess_DeleteForbidden_IsNotSwallowed(t *testing.T) {
	c, _ := newAccessClient(t, map[string]recorded{
		"DELETE " + appsPath + "/app-1":  {http.StatusForbidden, forbidden},
		"DELETE " + tokensPath + "/st-1": {http.StatusForbidden, forbidden},
	})
	assert.ErrorIs(t, c.DeleteAccessApp(t.Context(), "app-1"), apperrs.ErrInvalid)
	assert.ErrorIs(t, c.DeleteServiceToken(t.Context(), "st-1"), apperrs.ErrInvalid)
}

func TestAccess_NoAccount_FailsBeforeAnyAccessCall(t *testing.T) {
	f := &accessFixture{replies: map[string]recorded{
		"GET /client/v4/zones":    {http.StatusOK, `{"success":true,"result":[]}`},
		"GET /client/v4/accounts": {http.StatusOK, `{"success":true,"result":[]}`},
	}, bodies: map[string]map[string]any{}}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL + "/client/v4")
	require.NoError(t, err)
	_, err = New("tok", WithBaseURL(base)).CreateServiceToken(t.Context(), "Nexul")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestAccess_AppLifecycle(t *testing.T) {
	c, f := newAccessClient(t, map[string]recorded{
		"POST " + appsPath:              {http.StatusOK, `{"success":true,"errors":[],"result":{"id":"app-1","type":"self_hosted","domain":"laptop-ab12cd34.example.com"}}`},
		"DELETE " + appsPath + "/app-1": {http.StatusOK, `{"success":true,"errors":[],"result":{"id":"app-1"}}`},
		"DELETE " + appsPath + "/gone":  {http.StatusNotFound, `{"success":false,"errors":[{"code":12135,"message":"access.api.error.not_found"}]}`},
	})

	id, err := c.CreateAccessApp(t.Context(), "laptop-ab12cd34.example.com", "st-1")
	require.NoError(t, err)
	assert.Equal(t, "app-1", id)

	body := f.bodies["POST "+appsPath]
	assert.Equal(t, "self_hosted", body["type"])
	assert.Equal(t, "laptop-ab12cd34.example.com", body["domain"])
	assert.Equal(t, true, body["service_auth_401_redirect"])
	policy := body["policies"].([]any)[0].(map[string]any)
	assert.Equal(t, "non_identity", policy["decision"])
	include := policy["include"].([]any)[0].(map[string]any)
	assert.Equal(t, map[string]any{"token_id": "st-1"}, include["service_token"])

	require.NoError(t, c.DeleteAccessApp(t.Context(), "app-1"))
	require.NoError(t, c.DeleteAccessApp(t.Context(), "gone"), "an already-absent app deletes cleanly")
}

func TestAccess_ServiceTokenLifecycle(t *testing.T) {
	c, f := newAccessClient(t, map[string]recorded{
		"POST " + tokensPath:                  {http.StatusOK, tokenResult},
		"POST " + tokensPath + "/st-1/rotate": {http.StatusOK, `{"success":true,"errors":[],"result":{"id":"st-1","client_id":"cid.access","client_secret":"sec-2"}}`},
		"DELETE " + tokensPath + "/st-1":      {http.StatusOK, `{"success":true,"errors":[],"result":{"id":"st-1"}}`},
		"DELETE " + tokensPath + "/st-absent": {http.StatusNotFound, `{"success":false,"errors":[{"code":12135,"message":"access.api.error.not_found"}]}`},
	})

	tok, err := c.CreateServiceToken(t.Context(), "Nexul")
	require.NoError(t, err)
	assert.Equal(t, dns.ServiceToken{ID: "st-1", ClientID: "cid.access", ClientSecret: "sec-1"}, *tok)
	assert.Equal(t, map[string]any{"name": "Nexul", "duration": "forever"}, f.bodies["POST "+tokensPath])

	rotated, err := c.RotateServiceToken(t.Context(), "st-1")
	require.NoError(t, err)
	assert.Equal(t, "sec-2", rotated.ClientSecret)
	assert.Equal(t, "cid.access", rotated.ClientID)

	require.NoError(t, c.DeleteServiceToken(t.Context(), "st-1"))
	require.NoError(t, c.DeleteServiceToken(t.Context(), "st-absent"))
}

func TestZeroTrustMissing(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"cloudflare: Unable to find your Access organization", true},
		{"access.api.error.organization_not_found", true},
		{"Zero Trust is not enabled for this account", true},
		{"cloudflare: Authentication error", false},
		{"access.api.error.not_found", false},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, tt.want, zeroTrustMissing(tt.msg))
		})
	}
}
