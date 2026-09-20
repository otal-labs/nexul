package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestHTTPGitHubClient_Exchange_NotConfigured(t *testing.T) {
	c := NewHTTPGitHubClient("", "", http.DefaultClient)
	_, err := c.Exchange(context.Background(), "code")
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestHTTPGitHubClient_Exchange_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"bad_verification_code"}`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.tokenURL = srv.URL
	_, err := c.Exchange(context.Background(), "code")
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestHTTPGitHubClient_Exchange_EmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.tokenURL = srv.URL
	_, err := c.Exchange(context.Background(), "code")
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestHTTPGitHubClient_Exchange_NetworkError(t *testing.T) {
	c := NewHTTPGitHubClient("id", "secret", http.DefaultClient)
	c.tokenURL = "http://127.0.0.1:1/token"
	_, err := c.Exchange(context.Background(), "code")
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestHTTPGitHubClient_Exchange_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.tokenURL = srv.URL
	_, err := c.Exchange(context.Background(), "code")
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestHTTPGitHubClient_Exchange_ReturnsAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at"}`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.tokenURL = srv.URL
	token, err := c.Exchange(context.Background(), "code")
	require.NoError(t, err)
	assert.Equal(t, "at", token)
}

func TestHTTPGitHubClient_FetchUser_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer at", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"login":"onik97","name":"Onik","avatar_url":"https://avatar/x"}`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.userURL = srv.URL
	gh, err := c.FetchUser(context.Background(), "at")
	require.NoError(t, err)
	assert.Equal(t, "42", gh.ID)
	assert.Equal(t, "onik97", gh.Login)
	assert.Equal(t, "Onik", gh.Name)
	assert.Equal(t, "https://avatar/x", gh.AvatarURL)
}

func TestHTTPGitHubClient_FetchUser_EmptyUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.userURL = srv.URL
	_, err := c.FetchUser(context.Background(), "at")
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestHTTPGitHubClient_FetchUser_MissingLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.userURL = srv.URL
	_, err := c.FetchUser(context.Background(), "at")
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
}

func TestHTTPGitHubClient_FetchUser_NetworkError(t *testing.T) {
	c := NewHTTPGitHubClient("id", "secret", http.DefaultClient)
	c.userURL = "http://127.0.0.1:1/user"
	_, err := c.FetchUser(context.Background(), "at")
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestHTTPGitHubClient_FetchUser_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`nope`))
	}))
	defer srv.Close()
	c := NewHTTPGitHubClient("id", "secret", srv.Client())
	c.userURL = srv.URL
	_, err := c.FetchUser(context.Background(), "at")
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}
