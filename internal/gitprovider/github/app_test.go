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
