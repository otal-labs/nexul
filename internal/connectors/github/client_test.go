package github

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
// shape as internal/connectors' own test fake, reimplemented here since this
// package can't import connectors' internal test file.
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

// fixedInstanceURL is a canned InstanceURLReader for tests.
type fixedInstanceURL struct {
	url string
	err error
}

func (f fixedInstanceURL) GetInstanceURL(context.Context) (string, error) {
	return f.url, f.err
}

func testConfig() connectors.AppConfig {
	return connectors.AppConfig{
		ConnectorID:  "github",
		ClientID:     "Iv1.0000000000000001",
		ClientSecret: "shh-secret",
		AppSlug:      "nexul-test-app",
	}
}

func TestOAuthClient_Configured(t *testing.T) {
	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)
	assert.True(t, c.Configured())

	incomplete := testConfig()
	incomplete.AppSlug = ""
	store2 := newMemAppConfigStore(incomplete)
	c2 := New("github", store2, fixedInstanceURL{url: "http://localhost:5173"}, nil)
	assert.False(t, c2.Configured(), "missing app slug must not read as configured")
}

// TestOAuthClient_AuthorizeURL proves AuthorizeURL builds the classic
// /login/oauth/authorize web-flow URL, not the install URL (which dead-ends
// on already-installed apps), with the client id, redirect_uri from the
// current instance URL, and state.
func TestOAuthClient_AuthorizeURL(t *testing.T) {
	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	u := c.AuthorizeURL("nonce-1:user-1")
	assert.Contains(t, u, "https://github.com/login/oauth/authorize?")
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	q := parsed.Query()
	assert.Equal(t, testConfig().ClientID, q.Get("client_id"))
	assert.Equal(t, "nonce-1:user-1", q.Get("state"))
	assert.Equal(t, "http://localhost:5173/auth/connectors/github/callback", q.Get("redirect_uri"))
}

// TestOAuthClient_AuthorizeURL_UsesCurrentAppConfig proves AuthorizeURL
// builds from whatever AppConfig is currently in the store, not a value
// captured at construction: rotating the client id takes effect on the very
// next call, no restart (T13a/T3's "no restart" property).
func TestOAuthClient_AuthorizeURL_UsesCurrentAppConfig(t *testing.T) {
	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	u1 := c.AuthorizeURL("s1")
	assert.Contains(t, u1, "client_id="+testConfig().ClientID)

	rotated := testConfig()
	rotated.ClientID = "rotated-client-id"
	require.NoError(t, store.SetAppConfig(context.Background(), rotated))

	u2 := c.AuthorizeURL("s2")
	assert.Contains(t, u2, "client_id=rotated-client-id", "second call must use the rotated client id, not a cached value")
}

// installationsHandler fakes GET /user/installations, returning totalCount
// entries. It also records the Authorization header it received so tests
// can assert Exchange sent the freshly minted token, not the old one.
func installationsHandler(t *testing.T, totalCount int, gotAuth *string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if gotAuth != nil {
			*gotAuth = r.Header.Get("Authorization")
		}
		installs := make([]map[string]int, totalCount)
		for i := range installs {
			installs[i] = map[string]int{"id": i + 1}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count":   totalCount,
			"installations": installs,
		})
	}
}

// TestOAuthClient_Exchange_SucceedsWhenInstalled proves Exchange returns a
// TokenSet when the post-exchange GET /user/installations check finds at
// least one installation.
func TestOAuthClient_Exchange_SucceedsWhenInstalled(t *testing.T) {
	mux := http.NewServeMux()
	var gotAuth string
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "Iv1.0000000000000001", r.Form.Get("client_id"))
		assert.Equal(t, "shh-secret", r.Form.Get("client_secret"))
		assert.Equal(t, "auth-code", r.Form.Get("code"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at-1", "refresh_token": "rt-1", "expires_in": 28800,
		})
	})
	mux.HandleFunc("/user/installations", installationsHandler(t, 1, &gotAuth))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	oauthTokenURL = srv.URL + "/login/oauth/access_token"
	apiBaseURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	ts, err := c.Exchange(context.Background(), "auth-code")
	require.NoError(t, err)
	assert.Equal(t, "at-1", ts.AccessToken)
	assert.Equal(t, "rt-1", ts.RefreshToken)
	assert.Equal(t, "Bearer at-1", gotAuth, "installations check must use the just-exchanged token")
}

// TestOAuthClient_Exchange_FailsWhenNotInstalled proves the "authorizing
// != installing" fix: a token that GET /user/installations reports zero
// installations for must not come back as a usable TokenSet, since a bare
// authorize-only token grants no repo access at all.
func TestOAuthClient_Exchange_FailsWhenNotInstalled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at-1", "expires_in": 28800})
	})
	mux.HandleFunc("/user/installations", installationsHandler(t, 0, nil))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	oauthTokenURL = srv.URL + "/login/oauth/access_token"
	apiBaseURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Exchange(context.Background(), "auth-code")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid), "zero installations must be a clear, non-retryable error")
	// The callback uses the carried install URL to forward the browser to
	// the one-shot install→authorize flow instead of dead-ending.
	var notInstalled *connectors.NotInstalledError
	require.True(t, errors.As(err, &notInstalled))
	assert.Contains(t, notInstalled.InstallURL, "https://github.com/apps/nexul-test-app/installations/new")
	assert.Contains(t, notInstalled.InstallURL, "redirect_uri=")
}

func TestOAuthClient_Exchange_RejectedGrantIsUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "bad_verification_code"})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Exchange(context.Background(), "bad-code")
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
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Exchange(context.Background(), "code")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

// TestOAuthClient_Refresh proves Refresh hits the token endpoint with
// grant_type=refresh_token and the current client id/secret.
func TestOAuthClient_Refresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "old-rt", r.Form.Get("refresh_token"))
		assert.Equal(t, "Iv1.0000000000000001", r.Form.Get("client_id"))
		assert.Equal(t, "shh-secret", r.Form.Get("client_secret"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at-2", "refresh_token": "rt-2", "expires_in": 28800,
		})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	ts, err := c.Refresh(context.Background(), "old-rt")
	require.NoError(t, err)
	assert.Equal(t, "at-2", ts.AccessToken)
	assert.Equal(t, "rt-2", ts.RefreshToken)
}

// TestOAuthClient_Refresh_UsesCurrentAppConfig proves Refresh (like every
// other method) reads the store fresh rather than a value captured at
// construction: rotating the client secret between two calls changes what
// the very next Refresh call sends.
func TestOAuthClient_Refresh_UsesCurrentAppConfig(t *testing.T) {
	var gotSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		gotSecret = r.Form.Get("client_secret")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "expires_in": 100})
	}))
	defer srv.Close()
	oauthTokenURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	_, err := c.Refresh(context.Background(), "rt")
	require.NoError(t, err)
	assert.Equal(t, "shh-secret", gotSecret)

	rotated := testConfig()
	rotated.ClientSecret = "rotated-secret"
	require.NoError(t, store.SetAppConfig(context.Background(), rotated))

	_, err = c.Refresh(context.Background(), "rt")
	require.NoError(t, err)
	assert.Equal(t, "rotated-secret", gotSecret, "second call must use the rotated secret, not a cached one")
}

// TestOAuthClient_Revoke proves Revoke hits DELETE
// /applications/{client_id}/token with HTTP Basic Auth and the access token
// in the body.
func TestOAuthClient_Revoke(t *testing.T) {
	var gotUser, gotPass string
	var gotBody map[string]string
	mux := http.NewServeMux()
	mux.HandleFunc("/applications/Iv1.0000000000000001/token", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		gotUser, gotPass, _ = r.BasicAuth()
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	apiBaseURL = srv.URL

	store := newMemAppConfigStore(testConfig())
	c := New("github", store, fixedInstanceURL{url: "http://localhost:5173"}, nil)

	err := c.Revoke(context.Background(), "at-to-revoke")
	require.NoError(t, err)
	assert.Equal(t, "Iv1.0000000000000001", gotUser)
	assert.Equal(t, "shh-secret", gotPass)
	assert.Equal(t, "at-to-revoke", gotBody["access_token"])
}
