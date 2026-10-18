package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestHTTPGitHubClient_Exchange_Failures(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{"an error response", `{"error":"bad_verification_code"}`, apperrs.ErrUnauthorized},
		{"an empty token", `{}`, apperrs.ErrUnauthorized},
		{"a body that is not json", `not json`, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewHTTPGitHubClient("id", "secret", srv.Client())
			c.tokenURL = srv.URL
			_, err := c.Exchange(context.Background(), "code")
			assert.ErrorIs(t, err, tt.want)
		})
	}
	t.Run("a client with no credentials", func(t *testing.T) {
		_, err := NewHTTPGitHubClient("", "", http.DefaultClient).Exchange(context.Background(), "code")
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("a server that cannot be reached", func(t *testing.T) {
		c := NewHTTPGitHubClient("id", "secret", http.DefaultClient)
		c.tokenURL = "http://127.0.0.1:1/token"
		_, err := c.Exchange(context.Background(), "code")
		assert.ErrorIs(t, err, apperrs.ErrRetryable)
	})
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

func TestHTTPGitHubClient_FetchUser_Failures(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{"an empty user", `{}`, apperrs.ErrUnauthorized},
		{"a user with no login", `{"id":42}`, apperrs.ErrUnauthorized},
		{"a body that is not json", `nope`, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewHTTPGitHubClient("id", "secret", srv.Client())
			c.userURL = srv.URL
			_, err := c.FetchUser(context.Background(), "at")
			assert.ErrorIs(t, err, tt.want)
		})
	}
	t.Run("a server that cannot be reached", func(t *testing.T) {
		c := NewHTTPGitHubClient("id", "secret", http.DefaultClient)
		c.userURL = "http://127.0.0.1:1/user"
		_, err := c.FetchUser(context.Background(), "at")
		assert.ErrorIs(t, err, apperrs.ErrRetryable)
	})
}
