package github

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// appJWTClaims returns the claims of the JWT r carries, or nil after failing t when it is not signed by key.
func appJWTClaims(t *testing.T, r *http.Request, key *rsa.PublicKey) map[string]any {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), ".")
	if !assert.Len(t, parts, 3, "the App authenticates with a JWT") {
		return nil
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !assert.NoError(t, err) || !assert.NoError(t, rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig), "signed with the App's private key") {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	if !assert.NoError(t, err) || !assert.NoError(t, json.Unmarshal(raw, &claims)) {
		return nil
	}
	return claims
}

func TestApp_InstallationToken_ExchangesAJWTAndReusesTheTokenUntilNearExpiry(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	var mints atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST /app/installations/42/access_tokens", r.Method+" "+r.URL.Path)
		claims := appJWTClaims(t, r, &key.PublicKey)
		assert.Equal(t, "Iv1.acme", claims["iss"], "the client ID is the issuer")
		n := mints.Add(1)
		if n == 3 {
			_, _ = fmt.Fprint(w, `{"token":"","expires_at":"2099-01-01T00:00:00Z"}`)
			return
		}
		_, _ = fmt.Fprintf(w, `{"token":"ghs_%d","expires_at":%q}`, n, now.Add(time.Hour).Format(time.RFC3339)) // test server: write errors are irrelevant
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	app := NewApp("Iv1.acme", key, func() time.Time { return now }, WithBaseURL(u))

	tok, err := app.InstallationToken(t.Context(), 42)
	require.NoError(t, err)
	assert.Equal(t, "ghs_1", tok)

	now = now.Add(50 * time.Minute)
	tok, err = app.InstallationToken(t.Context(), 42)
	require.NoError(t, err)
	assert.Equal(t, "ghs_1", tok, "a live token is reused")

	now = now.Add(6 * time.Minute)
	tok, err = app.InstallationToken(t.Context(), 42)
	require.NoError(t, err)
	assert.Equal(t, "ghs_2", tok, "a token within minutes of expiring is replaced")
	assert.Equal(t, int32(2), mints.Load())
	now = now.Add(time.Hour)
	_, err = app.InstallationToken(t.Context(), 42)
	require.Error(t, err, "an empty App credential must never create an anonymous client")
}

func TestApp_ListInstallationRepos_ReadsEveryInstallationWithItsOwnToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /app/installations", func(w http.ResponseWriter, r *http.Request) {
		appJWTClaims(t, r, &key.PublicKey)
		_, _ = fmt.Fprint(w, `[{"id":1,"account":{"login":"acme","type":"Organization"},"repository_selection":"all"},
			{"id":2,"account":{"login":"globex","type":"User"},"repository_selection":"all"}]`) // test server: write errors are irrelevant
	})
	mux.HandleFunc("POST /app/installations/{id}/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"token":"ghs_%s","expires_at":"2099-01-01T00:00:00Z"}`, r.PathValue("id")) // test server: write errors are irrelevant
	})
	mux.HandleFunc("GET /installation/repositories", func(w http.ResponseWriter, r *http.Request) {
		owner := map[string]string{"Bearer ghs_1": "acme", "Bearer ghs_2": "globex"}[r.Header.Get("Authorization")]
		assert.NotEmpty(t, owner, "each installation is read with its own token")
		_, _ = fmt.Fprintf(w, `{"total_count":1,"repositories":[%s]}`, repoJSON(owner, "api")) // test server: write errors are irrelevant
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	repos, err := NewApp("Iv1.acme", key, nil, WithBaseURL(u)).ListInstallationRepos(t.Context())
	require.NoError(t, err)
	require.Len(t, repos, 2)
	assert.Equal(t, "acme", repos[0].Owner)
	assert.Equal(t, "globex", repos[1].Owner, "an installation on an account the connected user cannot open is still read")
}

func TestApp_ARefusedOrSuspendedInstallationLeavesTheOthersListed(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /app/installations", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `[{"id":1,"account":{"id":11,"login":"acme","type":"Organization"},"repository_selection":"all"},
			{"id":2,"account":{"id":12,"login":"globex","type":"Organization"},"repository_selection":"selected"},
			{"id":3,"account":{"id":13,"login":"initech","type":"User"},"repository_selection":"all","suspended_at":"2026-10-01T00:00:00Z"}]`) // test server: write errors are irrelevant
	})
	mux.HandleFunc("POST /app/installations/{id}/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		assert.NotEqual(t, "3", r.PathValue("id"), "a suspended installation is never asked for a token")
		if r.PathValue("id") != "1" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = fmt.Fprint(w, `{"message":"This installation has been suspended"}`) // test server: write errors are irrelevant
			return
		}
		_, _ = fmt.Fprint(w, `{"token":"ghs_1","expires_at":"2099-01-01T00:00:00Z"}`) // test server: write errors are irrelevant
	})
	mux.HandleFunc("GET /installation/repositories", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"total_count":1,"repositories":[%s]}`, repoJSON("acme", "api")) // test server: write errors are irrelevant
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	app := NewApp("Iv1.acme", key, nil, WithBaseURL(u))

	repos, err := app.ListInstallationRepos(t.Context())
	require.NoError(t, err, "one client's refusal never fails every workspace's list")
	require.Len(t, repos, 1)
	assert.Equal(t, int64(11), repos[0].AccountID, "a repository carries its installation's account")

	installs, err := app.ListInstallations(t.Context())
	require.NoError(t, err)
	require.Len(t, installs, 3)
	assert.Empty(t, installs[0].Problem)
	assert.Contains(t, installs[1].Problem, "refused")
	assert.Contains(t, installs[2].Problem, "suspended")
}
