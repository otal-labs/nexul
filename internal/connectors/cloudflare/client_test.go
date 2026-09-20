package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// memAppConfigStore is a minimal in-memory connectors.AppConfigStore, same
// shape as internal/connectors/github's own test fake (that package can't
// import connectors' internal test file, and neither can this one).
type memAppConfigStore struct {
	rows map[string]connectors.AppConfig
}

func newMemAppConfigStore(cfg connectors.AppConfig) *memAppConfigStore {
	return &memAppConfigStore{rows: map[string]connectors.AppConfig{cfg.ConnectorID: cfg}}
}

func (m *memAppConfigStore) GetAppConfig(_ context.Context, connectorID string) (connectors.AppConfig, error) {
	c, ok := m.rows[connectorID]
	if !ok {
		return connectors.AppConfig{ConnectorID: connectorID}, nil
	}
	return c, nil
}

func (m *memAppConfigStore) SetAppConfig(_ context.Context, c connectors.AppConfig) error {
	m.rows[c.ConnectorID] = c
	return nil
}

type fixedInstanceURL struct {
	url string
	err error
}

func (f fixedInstanceURL) GetInstanceURL(context.Context) (string, error) {
	return f.url, f.err
}

func testConfig() connectors.AppConfig {
	return connectors.AppConfig{ConnectorID: "cloudflare", ClientID: "cid", ClientSecret: "csec"}
}

func TestOAuthClient_Configured(t *testing.T) {
	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)
	assert.True(t, c.Configured())

	incomplete := testConfig()
	incomplete.ClientSecret = ""
	store2 := newMemAppConfigStore(incomplete)
	c2 := New("cloudflare", store2, fixedInstanceURL{url: "http://localhost:5173"}, nil)
	assert.False(t, c2.Configured())
}

func TestOAuthClient_AuthorizeURL(t *testing.T) {
	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	u := c.AuthorizeURL("state-1")
	assert.Contains(t, u, "https://dash.cloudflare.com/oauth2/auth?")
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	q := parsed.Query()
	assert.Equal(t, "code", q.Get("response_type"))
	assert.Equal(t, "cid", q.Get("client_id"))
	assert.Equal(t, "state-1", q.Get("state"))
	assert.Equal(t, "http://localhost:5173/auth/connectors/cloudflare/callback", q.Get("redirect_uri"))
	assert.Contains(t, q.Get("scope"), "DNS Read")
	assert.Contains(t, q.Get("scope"), "DNS Write")
}

// TestOAuthClient_AuthorizeURL_UsesCurrentAppConfig proves AuthorizeURL
// builds from whatever AppConfig is currently in the store, not a value
// captured at construction (T13a/T3's "no restart" property).
func TestOAuthClient_AuthorizeURL_UsesCurrentAppConfig(t *testing.T) {
	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	u1 := c.AuthorizeURL("s1")
	assert.Contains(t, u1, "client_id=cid")

	rotated := testConfig()
	rotated.ClientID = "rotated-client-id"
	require.NoError(t, store.SetAppConfig(context.Background(), rotated))

	u2 := c.AuthorizeURL("s2")
	assert.Contains(t, u2, "client_id=rotated-client-id", "second call must use the rotated client id, not a cached value")
}

func TestOAuthClient_Exchange(t *testing.T) {
	var gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "code-1", r.Form.Get("code"))
		assert.Equal(t, "http://localhost:5173/auth/connectors/cloudflare/callback", r.Form.Get("redirect_uri"))
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "refresh_token": "rt", "expires_in": 3600})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	ts, err := c.Exchange(context.Background(), "code-1")
	require.NoError(t, err)
	assert.Equal(t, "at", ts.AccessToken)
	assert.Equal(t, "rt", ts.RefreshToken)
	assert.Equal(t, 3600, int(ts.ExpiresIn.Seconds()))
	assert.Equal(t, "cid", gotUser, "token exchange must authenticate with HTTP basic auth")
	assert.Equal(t, "csec", gotPass)
}

func TestOAuthClient_Exchange_NotConfigured(t *testing.T) {
	store := newMemAppConfigStore(connectors.AppConfig{ConnectorID: "cloudflare"})
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestOAuthClient_Exchange_RejectedGrantIsUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestOAuthClient_Exchange_ServerErrorIsRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "upstream"})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestOAuthClient_Refresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "old-rt", r.Form.Get("refresh_token"))
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at2", "refresh_token": "rt2", "expires_in": 1800})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	ts, err := c.Refresh(context.Background(), "old-rt")
	require.NoError(t, err)
	assert.Equal(t, "at2", ts.AccessToken)
	assert.Equal(t, "rt2", ts.RefreshToken)
}

// TestOAuthClient_Refresh_UsesCurrentAppConfig proves Refresh reads the store
// fresh rather than a value captured at construction.
func TestOAuthClient_Refresh_UsesCurrentAppConfig(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, gotSecret, _ = r.BasicAuth()
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "expires_in": 100})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Refresh(context.Background(), "rt")
	require.NoError(t, err)
	assert.Equal(t, "csec", gotSecret)

	rotated := testConfig()
	rotated.ClientSecret = "rotated-secret"
	require.NoError(t, store.SetAppConfig(context.Background(), rotated))

	_, err = c.Refresh(context.Background(), "rt")
	require.NoError(t, err)
	assert.Equal(t, "rotated-secret", gotSecret, "second call must use the rotated secret, not a cached one")
}

func TestOAuthClient_Revoke(t *testing.T) {
	store := newMemAppConfigStore(testConfig())
	c := New("cloudflare", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)
	require.NoError(t, c.Revoke(context.Background(), "at"), "Cloudflare has no revoke endpoint; revocation is a dashboard action")
}
