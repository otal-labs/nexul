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

func TestHTTPGoogleClient_Exchange(t *testing.T) {
	c := NewHTTPGoogleClient("", "", "https://x/auth/google/callback", http.DefaultClient)
	_, err := c.Exchange(context.Background(), "code")
	assert.True(t, errors.Is(err, apperrs.ErrInvalid), "not configured")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "https://x/auth/google/callback", r.Form.Get("redirect_uri"))
		w.Header().Set("Content-Type", "application/json")
		if r.Form.Get("code") == "bad" {
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"at"}`))
	}))
	defer srv.Close()
	c = NewHTTPGoogleClient("id", "secret", "https://x/auth/google/callback", srv.Client())
	c.tokenURL = srv.URL

	token, err := c.Exchange(context.Background(), "good")
	require.NoError(t, err)
	assert.Equal(t, "at", token)

	_, err = c.Exchange(context.Background(), "bad")
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))

	c.tokenURL = "http://127.0.0.1:1/token"
	_, err = c.Exchange(context.Background(), "good")
	assert.True(t, errors.Is(err, apperrs.ErrRetryable))
}

func TestHTTPGoogleClient_FetchUser(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{"verified", `{"sub":"s1","email":"Client@Example.com","email_verified":true,"name":"Client","picture":"https://p/x"}`, nil},
		{"unverified email", `{"sub":"s1","email":"a@b.c","email_verified":false}`, apperrs.ErrUnauthorized},
		{"missing email", `{"sub":"s1","email_verified":true}`, apperrs.ErrUnauthorized},
		{"invalid json", `nope`, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewHTTPGoogleClient("id", "secret", "", srv.Client())
			c.userURL = srv.URL
			u, err := c.FetchUser(context.Background(), "at")
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, &ProviderUser{ID: "s1", Login: "client@example.com", Name: "Client", AvatarURL: "https://p/x"}, u)
		})
	}
}
