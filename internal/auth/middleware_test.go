package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAuth_AcceptableCredentials(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)
	raw, _, err := s.MintPAT(context.Background(), u, "ci")
	require.NoError(t, err)
	session, err := s.Sign(u)
	require.NoError(t, err)

	h := s.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromCtx(r.Context())
		if user == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(user.ID))
	}))

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"session token", session, http.StatusOK},
		{"personal access token", raw, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			assert.Equal(t, tt.want, rec.Code)
			if tt.want == http.StatusOK {
				assert.Equal(t, u, rec.Body.String(), "request must act as the token's user")
			}
		})
	}
}

func TestRequireAuth_Rejections(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)
	raw, pat, err := s.MintPAT(context.Background(), u, "ci")
	require.NoError(t, err)
	require.NoError(t, s.RevokePAT(context.Background(), u, pat.ID))

	h := s.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	valid, err := s.Sign(u)
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"garbage token", "nonsense", http.StatusUnauthorized},
		{"tampered session", valid + "x", http.StatusUnauthorized},
		{"revoked pat", raw, http.StatusUnauthorized},
		{"unknown pat", patPrefix + "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			assert.Equal(t, tt.want, rec.Code)
			assert.Equal(t, `Bearer realm="nexul"`, rec.Header().Get("WWW-Authenticate"), "RFC 6750 challenge on every 401")
		})
	}
}

func TestRequireAuth_DisabledUserCannotUseExistingCredentials(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)
	session, err := s.Sign(u)
	require.NoError(t, err)
	raw, _, err := s.MintPAT(context.Background(), u, "ci")
	require.NoError(t, err)
	require.NoError(t, users.SetAccountStatus(context.Background(), u, AccountDisabled))

	h := s.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, token := range []string{session, raw} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuth_NoPATStoreConfigured(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "owner"})
	require.NoError(t, err)

	h := s.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+patPrefix+"something")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
