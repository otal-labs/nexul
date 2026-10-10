package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/hostcred"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

type manifestConverter struct {
	calls int
	app   auth.GitHubAppCredentials
	err   error
}

func (c *manifestConverter) Convert(_ context.Context, code string) (auth.GitHubAppCredentials, error) {
	c.calls++
	if code != "conversion-code" {
		return auth.GitHubAppCredentials{}, errors.New("unexpected registration code")
	}
	return c.app, c.err
}

func TestGitHubManifest_ThroughSetupConsumesTheInitiatingPassAndStoresNoPublicSecrets(t *testing.T) {
	db, err := storage.OpenDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, storage.Migrate(db))
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	_, err = store.Settings.Set(t.Context(), "https://nexul.example.com")
	require.NoError(t, err)
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	require.NoError(t, store.SetupCodes.ReplaceSetupCode(t.Context(), hostcred.Hash("setup-code"), now, now.Add(time.Hour)))
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	converter := &manifestConverter{app: auth.GitHubAppCredentials{ClientID: "Iv1.acme", ClientSecret: "fixture-secret", Slug: "nexul-acme", PrivateKey: pemKey}}
	svc := auth.NewService(auth.Config{Secret: []byte("fixture-signing-key"), Users: store.Users, Settings: store.Settings, SetupCodes: store.SetupCodes, GitHubManifest: converter, GitHubManifestStore: store.GitHubManifests, Now: func() time.Time { return now }})
	pass, err := svc.UnlockSetup(t.Context(), "127.0.0.1", "setup-code")
	require.NoError(t, err)
	other, err := svc.UnlockSetup(t.Context(), "127.0.0.2", "setup-code")
	require.NoError(t, err)
	h := svc.RequireAuth(auth.NewHandler(svc).SetupRoutes())
	call := func(path, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	start := func() string {
		rec := call("/api/setup/github-app/start", pass.Token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var out struct {
			URL      string         `json:"url"`
			Manifest map[string]any `json:"manifest"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		u, err := url.Parse(out.URL)
		require.NoError(t, err)
		assert.Equal(t, "github.com", u.Host)
		assert.Equal(t, "https://nexul.example.com/auth/github/manifest/callback", out.Manifest["redirect_url"])
		assert.Equal(t, true, out.Manifest["request_oauth_on_install"])
		assert.NotContains(t, rec.Body.String(), pass.Token)
		return u.Query().Get("state")
	}
	finish := func(state, token string) *httptest.ResponseRecorder {
		body, err := json.Marshal(map[string]string{"state": state, "code": "conversion-code"})
		require.NoError(t, err)
		return call("/api/setup/github-app/callback", token, string(body))
	}
	assert.Equal(t, http.StatusUnauthorized, call("/api/setup/github-app/start", "", "").Code)
	state := start()
	returned := httptest.NewRecorder()
	auth.NewHandler(svc).Routes().ServeHTTP(returned, httptest.NewRequest(http.MethodGet, "/auth/github/manifest/callback?code=conversion-code&state="+url.QueryEscape(state), nil))
	require.Equal(t, http.StatusSeeOther, returned.Code)
	landing, err := url.Parse(returned.Header().Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, landing.RawQuery, "the temporary code cannot become a script or API request's referrer")
	assert.Contains(t, landing.Fragment, "code=conversion-code")
	assert.Equal(t, "no-referrer", returned.Header().Get("Referrer-Policy"))
	wrong := finish(state, other.Token)
	assert.Equal(t, http.StatusBadRequest, wrong.Code, wrong.Body.String())
	assert.Zero(t, converter.calls, "a different setup pass cannot exchange the code")
	forged := finish(state+"x", pass.Token)
	assert.Equal(t, http.StatusBadRequest, forged.Code)
	assert.Zero(t, converter.calls)
	good := finish(state, pass.Token)
	require.Equal(t, http.StatusNoContent, good.Code, good.Body.String())
	assert.NotContains(t, good.Body.String(), "fixture-secret")
	cfg, err := store.ConnectorAppConfig.GetAppConfig(t.Context(), "github")
	require.NoError(t, err)
	assert.Equal(t, pemKey, cfg.PrivateKey)
	settings, err := store.Settings.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Iv1.acme", settings.GitHubOAuthClientID)
	assert.Equal(t, "fixture-secret", settings.GitHubOAuthClientSecret)
	var ciphertext string
	require.NoError(t, db.QueryRow(`SELECT private_key FROM connector_app_config WHERE connector_id='github'`).Scan(&ciphertext))
	assert.NotContains(t, ciphertext, pemKey)
	assert.Equal(t, http.StatusBadRequest, finish(state, pass.Token).Code, "a callback can run only once")
	assert.Equal(t, 1, converter.calls)
	state = start()
	now = now.Add(16 * time.Minute)
	assert.Equal(t, http.StatusBadRequest, finish(state, pass.Token).Code, "registration expires before the setup pass")
	assert.Equal(t, 1, converter.calls)
	state = start()
	converter.err = errors.New("provider unavailable")
	require.Equal(t, http.StatusInternalServerError, finish(state, pass.Token).Code)
	assert.Equal(t, http.StatusBadRequest, finish(state, pass.Token).Code, "a failed conversion is consumed too")
	assert.Equal(t, 2, converter.calls)
	cfg, err = store.ConnectorAppConfig.GetAppConfig(t.Context(), "github")
	require.NoError(t, err)
	assert.Equal(t, pemKey, cfg.PrivateKey, "a failed conversion preserves the registered App")
	state = start()
	converter.err = nil
	converter.app.ClientID = "Iv1.new"
	converter.app.Slug = "nexul-fail"
	_, err = db.Exec(`CREATE TRIGGER reject_manifest BEFORE UPDATE ON connector_app_config WHEN NEW.app_slug='nexul-fail' BEGIN SELECT RAISE(ABORT, 'fixture storage failure'); END`)
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, finish(state, pass.Token).Code)
	settings, err = store.Settings.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Iv1.acme", settings.GitHubOAuthClientID, "connector failure rolls back sign-in credentials too")
	cfg, err = store.ConnectorAppConfig.GetAppConfig(t.Context(), "github")
	require.NoError(t, err)
	assert.Equal(t, "nexul-acme", cfg.AppSlug)

	state = start()
	_, err = store.Users.CreateFirstUser(t.Context(), &auth.Identity{Provider: auth.ProviderGitHub, ProviderUserID: "42", UserID: "alice", Login: "alice"})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, finish(state, pass.Token).Code, "first sign-in closes pending setup callbacks")

}
