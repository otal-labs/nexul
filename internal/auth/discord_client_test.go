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

func TestHTTPDiscordClient_Exchange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "https://x/auth/discord/callback", r.Form.Get("redirect_uri"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"at"}`))
	}))
	defer srv.Close()
	c := NewHTTPDiscordClient("id", "secret", "https://x/auth/discord/callback", srv.Client())
	c.tokenURL = srv.URL
	token, err := c.Exchange(context.Background(), "good")
	require.NoError(t, err)
	assert.Equal(t, "at", token)

	_, err = NewHTTPDiscordClient("", "", "", srv.Client()).Exchange(context.Background(), "good")
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestHTTPDiscordClient_FetchUser(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    *ProviderUser
		wantErr error
	}{
		{"verified with avatar", `{"id":"123","username":"clientname","global_name":"Client","avatar":"abc","email":"Client@Example.com","verified":true}`,
			&ProviderUser{ID: "123", Login: "client@example.com", Name: "Client", AvatarURL: "https://cdn.discordapp.com/avatars/123/abc.png"}, nil},
		{"falls back to username, no avatar", `{"id":"123","username":"clientname","avatar":null,"email":"a@b.c","verified":true}`,
			&ProviderUser{ID: "123", Login: "a@b.c", Name: "clientname"}, nil},
		{"unverified", `{"id":"123","username":"x","email":"a@b.c","verified":false}`, nil, apperrs.ErrUnauthorized},
		{"no email scope", `{"id":"123","username":"x"}`, nil, apperrs.ErrUnauthorized},
		{"invalid json", `nope`, nil, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer at", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			c := NewHTTPDiscordClient("id", "secret", "", srv.Client())
			c.userURL = srv.URL
			u, err := c.FetchUser(context.Background(), "at")
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, u)
		})
	}
}
