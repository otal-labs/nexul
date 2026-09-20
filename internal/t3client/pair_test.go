package t3client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// The real T3 server is never reachable in CI (ticket 11's brief); every
// case here runs against an httptest.Server standing in for it.

func TestHarness_Exchange_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/oauth/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "urn:ietf:params:oauth:grant-type:token-exchange", r.Form.Get("grant_type"))
		assert.Equal(t, "one-time-token", r.Form.Get("subject_token"))
		assert.Equal(t, "urn:t3:params:oauth:token-type:environment-bootstrap", r.Form.Get("subject_token_type"))
		assert.Equal(t, "urn:ietf:params:oauth:token-type:access_token", r.Form.Get("requested_token_type"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "bearer-xyz", "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
			"token_type": "Bearer", "expires_in": 2592000, "scope": clientScopes,
		})
	}))
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	result, err := h.exchange(context.Background(), srv.URL, "one-time-token")
	require.NoError(t, err)
	assert.Equal(t, "bearer-xyz", result.BearerToken)
	assert.Equal(t, 30*24*3600, int(result.ExpiresIn.Seconds()))
}

func TestHarness_Exchange_DefaultsExpiryWhenMissing(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "bearer-xyz", "token_type": "Bearer"})
	}))
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	result, err := h.exchange(context.Background(), srv.URL, "tok")
	require.NoError(t, err)
	assert.Equal(t, 30*24*3600, int(result.ExpiresIn.Seconds()))
}

func TestHarness_Exchange_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid_grant: one-time token expired"))
	}))
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	_, err := h.exchange(context.Background(), srv.URL, "expired-token")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestHarness_Exchange_NoAccessToken(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"token_type": "Bearer"})
	}))
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	_, err := h.exchange(context.Background(), srv.URL, "tok")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestHarness_Exchange_Unreachable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // closed immediately: connecting now fails, simulating a sleeping computer.

	h := NewHarness(Options{HTTPClient: srv.Client()})
	_, err := h.exchange(context.Background(), srv.URL, "tok")
	require.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestHarness_Version_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, wellKnownEnvironmentPath, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"environmentId": "env-1", "label": "home", "platform": "linux",
			"serverVersion": "0.0.34", "capabilities": []string{},
		})
	}))
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	v, err := h.Version(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "0.0.34", v)
}

func TestHarness_Version_MissingField(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"environmentId": "env-1"})
	}))
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	_, err := h.Version(context.Background(), srv.URL)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestNewHarness_NilClientDefaultsToHTTPDefault(t *testing.T) {
	t.Parallel()
	h := NewHarness(Options{})
	assert.Equal(t, http.DefaultClient, h.httpClient())
}

func TestHarness_Pair_ExchangesThenReadsVersion(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "bearer-xyz", "expires_in": 3600})
	})
	mux.HandleFunc(wellKnownEnvironmentPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"serverVersion": "0.0.34"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	result, err := h.Pair(context.Background(), srv.URL, "tok")
	require.NoError(t, err)
	assert.Equal(t, harness.PairResult{BearerToken: "bearer-xyz", ExpiresIn: time.Hour, Version: "0.0.34"}, result)
}
