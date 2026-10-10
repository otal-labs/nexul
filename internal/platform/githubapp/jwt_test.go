package githubapp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestVerifyKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pkcs1 := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	tests := []struct {
		name   string
		key    string
		status int
		body   string
		want   error
	}{
		{"GitHub's own PKCS#1 key for this App passes", pkcs1, http.StatusOK, `{"client_id":"Iv1.acme"}`, nil},
		{"a PKCS#8 key passes too", string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})), http.StatusOK, `{"client_id":"Iv1.acme"}`, nil},
		{"a key of another App is invalid", pkcs1, http.StatusOK, `{"client_id":"Iv1.globex"}`, apperrs.ErrInvalid},
		{"a key GitHub refuses is invalid", pkcs1, http.StatusUnauthorized, `{}`, apperrs.ErrInvalid},
		{"text that is no PEM is invalid", "not a key", http.StatusOK, `{}`, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/app", r.URL.Path)
				assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ey"), "authenticates with a JWT")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body)) // test server: write errors are irrelevant
			}))
			t.Cleanup(srv.Close)
			err := VerifyKey(t.Context(), srv.Client(), srv.URL, "Iv1.acme", tt.key)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}
}
