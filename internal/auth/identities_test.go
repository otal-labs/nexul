package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// newLinkHarness signs an owner in through GitHub and turns Google on, so a link round trip can run against fakes.
func newLinkHarness(t *testing.T) (*Service, *fakeUserStore, string) {
	t.Helper()
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	_, err := settings.Set(context.Background(), "https://deploy.example.com")
	require.NoError(t, err)
	_, err = settings.SetProviderOAuth(context.Background(), ProviderGoogle, "g-id", "g-secret")
	require.NoError(t, err)
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	userID := mustVerify(t, s, token)
	s.cfg.Google = &fakeGitHub{token: "at", user: &ProviderUser{ID: "sub-9", Login: "owner@example.com", Name: "Owner"}}
	return s, users, userID
}

func TestStartIdentityLink_Refusals(t *testing.T) {
	s, _, userID := newLinkHarness(t)
	_, _, err := s.StartIdentityLink(context.Background(), "", ProviderGoogle)
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, _, err = s.StartIdentityLink(context.Background(), userID, Provider("facebook"))
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, _, err = s.StartIdentityLink(context.Background(), userID, ProviderDiscord)
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "a provider the instance has not turned on cannot be linked")
}

func TestCompleteIdentityLink_AttachesToTheUserWhoStartedIt(t *testing.T) {
	s, users, userID := newLinkHarness(t)
	authorizeURL, state, err := s.StartIdentityLink(context.Background(), userID, ProviderGoogle)
	require.NoError(t, err)
	assert.Contains(t, authorizeURL, "state="+state)
	assert.True(t, IsLinkState(state))

	require.NoError(t, s.CompleteIdentityLink(context.Background(), ProviderGoogle, state, "good-code"))

	identities, err := s.ListIdentities(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, identities, 2)
	assert.Equal(t, ProviderGitHub, identities[0].Provider)
	assert.Equal(t, ProviderGoogle, identities[1].Provider)
	assert.Equal(t, "owner@example.com", identities[1].Login)

	token, err := s.LoginWith(context.Background(), ProviderGoogle, "good-code")
	require.NoError(t, err)
	assert.Equal(t, userID, mustVerify(t, s, token), "signing in with the linked account lands on the same user")
	user, err := users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "owner", user.Login, "the handle follows the first identity, not the one used last")
}

func TestCompleteIdentityLink_RejectsBadStates(t *testing.T) {
	s, _, userID := newLinkHarness(t)
	_, state, err := s.StartIdentityLink(context.Background(), userID, ProviderGoogle)
	require.NoError(t, err)
	tests := []struct {
		name     string
		state    string
		provider Provider
	}{
		{"not a link state", "plain-state", ProviderGoogle},
		{"tampered signature", state[:len(state)-2] + "zz", ProviderGoogle},
		{"another provider's callback", state, ProviderGitHub},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CompleteIdentityLink(context.Background(), tt.provider, tt.state, "good-code")
			assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
		})
	}
	t.Run("expired", func(t *testing.T) {
		now := s.cfg.Now()
		s.cfg.Now = func() time.Time { return now.Add(stateMaxAge + time.Second) }
		err := s.CompleteIdentityLink(context.Background(), ProviderGoogle, state, "good-code")
		assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
}

func TestCompleteIdentityLink_SomeoneElsesAccount_ChangesNothing(t *testing.T) {
	s, users, userID := newLinkHarness(t)
	_, _, err := users.UpsertUser(context.Background(), &Identity{UserID: "other", Provider: ProviderGoogle, ProviderUserID: "sub-9", Login: "owner@example.com"})
	require.NoError(t, err)
	_, state, err := s.StartIdentityLink(context.Background(), userID, ProviderGoogle)
	require.NoError(t, err)

	err = s.CompleteIdentityLink(context.Background(), ProviderGoogle, state, "good-code")
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "another user")

	mine, err := s.ListIdentities(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, mine, 1)
	theirs, err := s.ListIdentities(context.Background(), "other")
	require.NoError(t, err)
	assert.Len(t, theirs, 1)
}

func TestUnlinkIdentity(t *testing.T) {
	s, _, userID := newLinkHarness(t)
	err := s.UnlinkIdentity(context.Background(), userID, ProviderGitHub)
	assert.ErrorIs(t, err, apperrs.ErrConflict, "the only sign-in stays")
	err = s.UnlinkIdentity(context.Background(), userID, ProviderGoogle)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)

	_, state, err := s.StartIdentityLink(context.Background(), userID, ProviderGoogle)
	require.NoError(t, err)
	require.NoError(t, s.CompleteIdentityLink(context.Background(), ProviderGoogle, state, "good-code"))
	require.NoError(t, s.UnlinkIdentity(context.Background(), userID, ProviderGitHub))

	_, err = s.Login(context.Background(), "good-code")
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized, "the unlinked account no longer signs in")
	token, err := s.LoginWith(context.Background(), ProviderGoogle, "good-code")
	require.NoError(t, err)
	assert.Equal(t, userID, mustVerify(t, s, token))
}

func TestHandler_IdentityRoutes(t *testing.T) {
	s, users, userID := newLinkHarness(t)
	pats := newFakePATStore()
	s.cfg.PATs = pats
	protected := protectedHandler(s, users)
	public := NewHandler(s).Routes()

	t.Run("a personal access token has no say over sign-in accounts", func(t *testing.T) {
		patRaw, _, err := s.MintPAT(context.Background(), userID, "ci")
		require.NoError(t, err)
		rec := doRequest(protected, http.MethodGet, "/api/auth/identities", "Bearer "+patRaw, "")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/identities", userID, ""))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"provider":"github"`)
	assert.NotContains(t, rec.Body.String(), `"1"`, "the provider's own id stays server-side")

	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/identities/link", userID, `{"provider":"google"}`))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var start struct {
		URL string `json:"url"`
	}
	require.NoError(t, decodeJSON(rec, &start))
	assert.Contains(t, start.URL, googleAuthorizeURL)
	var stateCookieValue string
	for _, c := range rec.Result().Cookies() {
		if c.Name == stateCookie {
			stateCookieValue = c.Value
		}
	}
	require.True(t, IsLinkState(stateCookieValue))

	t.Run("callback without the cookie is refused", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=good-code&state="+stateCookieValue, nil)
		rec := httptest.NewRecorder()
		public.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=good-code&state="+stateCookieValue, nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: stateCookieValue})
	rec = httptest.NewRecorder()
	public.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "https://deploy.example.com/settings/profile?provider=google&linked=1", rec.Header().Get("Location"))

	t.Run("a second link of the same provider lands back with the reason", func(t *testing.T) {
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/identities/link", userID, `{"provider":"google"}`))
		require.Equal(t, http.StatusOK, rec.Code)
		var again string
		for _, c := range rec.Result().Cookies() {
			if c.Name == stateCookie {
				again = c.Value
			}
		}
		req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=good-code&state="+again, nil)
		req.AddCookie(&http.Cookie{Name: stateCookie, Value: again})
		rec = httptest.NewRecorder()
		public.ServeHTTP(rec, req)
		require.Equal(t, http.StatusFound, rec.Code)
		location := rec.Header().Get("Location")
		assert.True(t, strings.HasPrefix(location, "https://deploy.example.com/settings/profile?provider=google&error="), location)
		assert.Contains(t, location, "already+linked")
	})

	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/identities/github", userID, ""))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/identities/google", userID, ""))
	assert.Equal(t, http.StatusConflict, rec.Code, "the last sign-in is refused server-side")
}
