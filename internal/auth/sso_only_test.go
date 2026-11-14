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

func ssoOnly(on *bool) func(context.Context) bool { return func(context.Context) bool { return *on } }

func TestLoginWith_WhileSSOIsTheOnlySignIn_RefusesEveryProviderAndTellsTheBrowserWhy(t *testing.T) {
	t.Parallel()
	s, users, _, _ := newTestHarness(&fakeGitHub{token: "at", user: ghUser("42", "onik97")})
	_, err := s.Login(t.Context(), "good-code")
	require.NoError(t, err, "the person signs in while the switch is off")
	users.setOwner(users.byKey[userKey(ProviderGitHub, "42")])
	on := true
	s.SetSSOOnly(ssoOnly(&on))

	for _, provider := range []Provider{ProviderGitHub, ProviderGoogle, ProviderDiscord} {
		s.cfg.Google, s.cfg.Discord = &fakeGitHub{token: "at", user: ghUser("42", "onik97")}, &fakeGitHub{token: "at", user: ghUser("42", "onik97")}
		_, err := s.LoginWith(t.Context(), provider, "good-code")
		require.ErrorIs(t, err, errSSOOnly, "%s", provider)
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
		assert.Equal(t, signInSSOOnly, signInErrorCode(err))
	}

	on = false
	_, err = s.Login(t.Context(), "good-code")
	require.NoError(t, err, "turning the switch off brings the other sign-ins back")
}

func TestFinishPasskeyLogin_WhileSSOIsTheOnlySignIn_LetsOnlyAnInstanceOwnersPasskeyIn(t *testing.T) {
	t.Parallel()
	h := newPasskeyHarness(t)
	h.users.insert(&Identity{UserID: "user-bob", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	alice := h.register(t, h.userID)
	bob := h.register(t, "user-bob")
	h.users.setOwner(h.userID)
	on := true
	h.s.SetSSOOnly(ssoOnly(&on))

	_, err := h.signIn(t, bob, "nexul.example.com", testOrigin)
	require.ErrorIs(t, err, errSSOOnly, "a passkey is a sign-in method like the others")

	token, err := h.signIn(t, alice, "nexul.example.com", testOrigin)
	require.NoError(t, err, "the instance Owner's passkey is the way back in when the provider breaks")
	assert.Equal(t, h.userID, mustVerify(t, h.s, token))
}

func TestInstanceOwnerHasPasskey(t *testing.T) {
	t.Parallel()
	h := newPasskeyHarness(t)
	h.users.insert(&Identity{UserID: "user-bob", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})

	has, err := h.s.InstanceOwnerHasPasskey(t.Context())
	require.NoError(t, err)
	assert.False(t, has, "an Owner with no passkey")
	h.users.setOwner(h.userID)
	h.register(t, "user-bob")
	has, err = h.s.InstanceOwnerHasPasskey(t.Context())
	require.NoError(t, err)
	assert.False(t, has, "somebody else's passkey is no way back in for the Owner")
	h.register(t, h.userID)
	has, err = h.s.InstanceOwnerHasPasskey(t.Context())
	require.NoError(t, err)
	assert.True(t, has)
}

func TestHandler_SSOSessionFor_StartsAnSSOSessionOnTheDeviceTheRequestCameFrom(t *testing.T) {
	t.Parallel()
	s, users, _, _ := newTestHarness(&fakeGitHub{token: "at", user: ghUser("42", "onik97")})
	_, err := s.Login(t.Context(), "good-code")
	require.NoError(t, err)
	userID := users.byKey[userKey(ProviderGitHub, "42")]
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/oidc/callback", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0")

	token, err := NewHandler(s).SSOSessionFor(req, userID)

	require.NoError(t, err)
	assert.Equal(t, userID, mustVerify(t, s, token))
	sessions, err := s.cfg.Sessions.ListSessionsByUser(t.Context(), userID)
	require.NoError(t, err)
	assert.Equal(t, "Linux", sessions[len(sessions)-1].Platform)
	assert.True(t, sessions[len(sessions)-1].SSO)
	assert.False(t, sessions[0].SSO, "the GitHub sign-in's session")
}

func TestExchangeConnectCode_WhileSSOIsTheOnlySignIn_LetsInOnlyACodeFromAnSSOSessionOrAnOwner(t *testing.T) {
	t.Parallel()
	s, codes, u := newConnectHarness(t)
	users := s.cfg.Users.(*fakeUserStore)
	issue := func(t *testing.T, from *Session) string {
		t.Helper()
		code, err := s.IssueConnectCode(withPrincipal(t.Context(), &User{ID: u}, from), u, "")
		require.NoError(t, err)
		return code.Code
	}
	githubCode := issue(t, &Session{ID: "web-github"})
	require.Len(t, codes.byHash, 1)
	on := true
	s.SetSSOOnly(ssoOnly(&on))

	_, err := s.ExchangeConnectCode(t.Context(), "203.0.113.7", githubCode, ConnectDevice{Model: "Pixel 8"})
	require.ErrorIs(t, err, errSSOOnly, "a code from a session SSO-only now refuses mints no phone session")

	token, err := s.ExchangeConnectCode(t.Context(), "203.0.113.7", issue(t, &Session{ID: "web-sso", SSO: true}), ConnectDevice{Model: "Pixel 8"})
	require.NoError(t, err)
	_, ses, err := s.AuthenticateSession(t.Context(), token, "")
	require.NoError(t, err)
	assert.True(t, ses.SSO, "the phone session counts as single sign-on, so turning the switch on again keeps it")

	users.setOwner(u)
	_, err = s.ExchangeConnectCode(t.Context(), "203.0.113.7", issue(t, &Session{ID: "web-github"}), ConnectDevice{Model: "Pixel 8"})
	require.NoError(t, err, "an instance Owner signs in outside single sign-on, so their phone does too")
}
