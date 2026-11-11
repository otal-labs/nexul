package auth

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

// rejectingTransport answers a token request with 401 invalid_client for each client secret in rejected.
type rejectingTransport struct {
	mu       sync.Mutex
	rejected map[string]bool
	requests int
}

func (r *rejectingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	_ = req.ParseForm()
	r.mu.Lock()
	r.requests++
	r.mu.Unlock()
	if r.rejected[req.PostForm.Get("client_secret")] {
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_client"}`)), Header: http.Header{}}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"at"}`)), Header: http.Header{}}, nil
}

func TestExchange_AClientRejectionReReadsTheSecretOnceAndRetriesOnce(t *testing.T) {
	tests := []struct {
		name     string
		rejected map[string]bool
		wantErr  bool
	}{
		{"the new value is accepted", map[string]bool{"old-secret": true}, false},
		{"the new value is rejected too", map[string]bool{"old-secret": true, "new-secret": true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
			_, err := settings.SetProviderOAuth(t.Context(), ProviderGoogle, "g-id", googleSecretRef)
			require.NoError(t, err)
			fake := &secretreftest.Fake{Values: map[string]string{googleSecretRef: "old-secret"}}
			s.cfg.Secrets = secretsOver(t, fake)
			transport := &rejectingTransport{rejected: tt.rejected}
			s.httpClient = &http.Client{Transport: transport}
			client, err := s.providerClient(t.Context(), ProviderGoogle)
			require.NoError(t, err)
			fake.Values[googleSecretRef] = "new-secret"

			grant, err := s.exchangeCode(t.Context(), ProviderGoogle, client, "code")

			assert.Equal(t, 2, transport.requests, "one rejected try and one retry, never a third")
			assert.Len(t, fake.Calls(), 2, "the secret was read at first and once more after the rejection")
			if tt.wantErr {
				require.ErrorIs(t, err, apperrs.ErrUnauthorized)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "at", grant.AccessToken)
		})
	}
}
