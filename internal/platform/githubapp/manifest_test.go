package githubapp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type manifestTransport struct {
	target *url.URL
}

func (m manifestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	copy.URL.Scheme, copy.URL.Host = m.target.Scheme, m.target.Host
	return http.DefaultTransport.RoundTrip(copy)
}

func manifestHTTPClient(t *testing.T, address string) *http.Client {
	t.Helper()
	u, err := url.Parse(address)
	require.NoError(t, err)
	return &http.Client{Transport: manifestTransport{target: u}}
}

func TestManifestClient_ExchangesTheCodeAndRejectsUntrustedRepliesWithoutLeakingSecrets(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	app := ManifestApp{ClientID: "Iv1.acme", ClientSecret: "fixture-secret", Slug: "nexul-acme", PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))}
	raw, err := json.Marshal(app)
	require.NoError(t, err)
	for _, tt := range []struct {
		name   string
		status int
		body   string
		valid  bool
	}{
		{"generated credentials", http.StatusCreated, string(raw), true},
		{"provider refuses code", http.StatusUnprocessableEntity, `{"message":"fixture-secret"}`, false},
		{"provider redirects", http.StatusFound, string(raw), false},
		{"malformed reply", http.StatusCreated, `{"pem":"fixture-key"`, false},
		{"incomplete reply", http.StatusCreated, `{"client_id":"Iv1.acme"}`, false},
		{"invalid key", http.StatusCreated, `{"client_id":"Iv1.acme","client_secret":"fixture-secret","slug":"nexul-acme","pem":"fixture-key"}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST /app-manifests/conversion-code/conversions", r.Method+" "+r.URL.Path)
				assert.Empty(t, r.Header.Get("Authorization"), "the setup pass never reaches GitHub")
				w.Header().Set("Location", "https://example.com/redirect")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(srv.Close)
			got, err := (ManifestClient{Client: manifestHTTPClient(t, srv.URL)}).Convert(t.Context(), "conversion-code")
			if !tt.valid {
				require.Error(t, err)
				assert.NotContains(t, err.Error(), "fixture-secret")
				assert.NotContains(t, err.Error(), "fixture-key")
				assert.NotContains(t, err.Error(), "conversion-code")
				assert.Equal(t, ManifestApp{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, app, got)
		})
	}
	t.Run("transport failure hides the code-bearing URL", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		srv.Close()
		_, err := (ManifestClient{Client: manifestHTTPClient(t, srv.URL)}).Convert(t.Context(), "conversion-code")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "conversion-code")
	})
}
