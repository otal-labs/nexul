package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	token, err := s.Sign("user-1")
	require.NoError(t, err)

	userID, err := s.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
}

func TestVerify_RejectsTampered(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	token, err := s.Sign("user-1")
	require.NoError(t, err)

	userID, err := s.Verify(token + "x")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	assert.Empty(t, userID)
}

func TestVerify_RejectsExpired(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	now := time.Unix(1_700_000_000, 0)
	s.cfg.Now = func() time.Time { return now.Add(-2 * time.Hour) }
	token, err := s.Sign("user-1")
	require.NoError(t, err)

	s.cfg.Now = func() time.Time { return now }
	userID, err := s.Verify(token)
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	assert.Empty(t, userID)
}

func TestVerify_MalformedToken(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	for _, tok := range []string{"", "no-dot", "a.", ".b", "garbage!!"} {
		userID, err := s.Verify(tok)
		assert.ErrorIs(t, err, apperrs.ErrUnauthorized, "token %q", tok)
		assert.Empty(t, userID)
	}
}

func TestLogin_ExchangesCodeAndPersistsUser(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{token: "at", user: ghUser("42", "onik97")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	userID, err := s.Verify(token)
	require.NoError(t, err)

	user, err := users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, ProviderGitHub, user.Provider)
	assert.Equal(t, "42", user.ProviderUserID)
	assert.Equal(t, "onik97", user.Login)
	assert.Equal(t, "Name onik97", user.Name)
	assert.Equal(t, "https://avatar/onik97", user.AvatarURL)
}

func TestLogin_RejectsBadCode(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{token: "at", user: ghUser("42", "onik97")})
	_, err := s.Login(context.Background(), "bad-code")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrUnauthorized) || errors.Is(err, apperrs.ErrRetryable))
}

func TestLogin_SyncsProviderFieldsOnResignIn(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{token: "at", user: ghUser("42", "renamed")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	userID, err := s.Verify(token)
	require.NoError(t, err)
	user, err := users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "renamed", user.Login)

	s.cfg.GitHub = &fakeGitHub{token: "at", user: ghUser("42", "onik97")}
	token, err = s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	userID, err = s.Verify(token)
	require.NoError(t, err)
	user, err = users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "onik97", user.Login)
}

func TestLogin_FreshInstanceFirstSignInUnrestricted(t *testing.T) {
	s, _, allowlist, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "first-user")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Empty(t, allowlist.set)

	st, err := s.Me(context.Background(), mustVerify(t, s, token))
	require.NoError(t, err)
	assert.True(t, st.NeedsOwnerWizard)
	assert.False(t, st.NeedsFirstLoginWizard)
}

func TestLogin_FirstUserRace_AllowsOneUser(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "first-user")})
	other := NewService(Config{Secret: []byte("test-secret"), Users: users, GitHub: &fakeGitHub{user: ghUser("2", "second-user")}, Settings: newFakeSettings(), Allowlist: newFakeAllowlist(), Now: time.Now})
	results := make(chan error, 2)
	go func() {
		_, err := s.Login(context.Background(), "good-code")
		results <- err
	}()
	go func() {
		_, err := other.Login(context.Background(), "good-code")
		results <- err
	}()
	var successes int
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	assert.Equal(t, 1, successes)
	accounts, err := users.ListUsers(context.Background())
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
}

func TestLogin_RejectsNonAllowlistedUser(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, ownerToken)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	s.cfg.GitHub = &fakeGitHub{user: ghUser("2", "bob")}
	_, err = s.Login(context.Background(), "good-code")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, err = users.GetUserByProvider(context.Background(), ProviderGitHub, "2")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestLogin_AllowlistedUserPasses(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), mustVerify(t, s, ownerToken), "https://deploy.example.com"))
	_, _, err = users.UpsertUser(context.Background(), &User{ID: "bob-id", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	require.NoError(t, err)

	s.cfg.GitHub = &fakeGitHub{user: ghUser("2", "bob")}
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	st, err := s.Me(context.Background(), mustVerify(t, s, token))
	require.NoError(t, err)
	assert.False(t, st.NeedsOwnerWizard)
	assert.True(t, st.NeedsFirstLoginWizard)
}

func TestLogin_OwnerBypassesAllowlist(t *testing.T) {
	s, _, allowlist, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	token2, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	require.NotEmpty(t, token2)
	assert.Empty(t, allowlist.set)
}

func TestIsLoginAllowlisted(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), mustVerify(t, s, ownerToken), "https://deploy.example.com"))
	require.NoError(t, s.AddMember(context.Background(), mustVerify(t, s, ownerToken), "Bob"))

	allowed, err := s.IsLoginAllowlisted(context.Background(), "bob")
	require.NoError(t, err)
	assert.True(t, allowed, "allowlist check should be case-insensitive, matching AddMember's own normalization")

	allowed, err = s.IsLoginAllowlisted(context.Background(), "nobody")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestUserIDForLogin(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, ownerToken)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	id, found, err := s.UserIDForLogin(context.Background(), "owner")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, ownerID, id)

	_, found, err = s.UserIDForLogin(context.Background(), "never-signed-in")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestLogin_ResolvesPendingInvitesForNewUser(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, ownerToken)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))
	s.cfg.GitHub = &fakeGitHub{user: ghUser("2", "bob")}
	_, err = s.Login(context.Background(), "good-code")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestLogin_DisabledUserCannotResignIn(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	userID := mustVerify(t, s, token)
	require.NoError(t, users.SetAccountStatus(context.Background(), userID, AccountDisabled))
	_, err = s.Login(context.Background(), "good-code")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestConfigured(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "x")})
	settings.st.GitHubOAuthClientID = "client"
	settings.st.GitHubOAuthClientSecret = "secret"
	configured, err := s.Configured(context.Background())
	require.NoError(t, err)
	assert.True(t, configured)

	notCfg := NewService(Config{Secret: []byte("x")})
	configured, err = notCfg.Configured(context.Background())
	require.NoError(t, err)
	assert.False(t, configured)
}

// TestConfigured_ReflectsLiveSettings proves the 503 gate reads Settings from the store on every call, not a startup-time snapshot (T3): flipping the stored credentials changes what the very next call reports.
func TestConfigured_ReflectsLiveSettings(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "x")})

	settings.st.GitHubOAuthClientID = ""
	settings.st.GitHubOAuthClientSecret = ""
	configured, err := s.Configured(context.Background())
	require.NoError(t, err)
	assert.False(t, configured, "cleared credentials must be reflected immediately")

	_, err = settings.SetGitHubOAuth(context.Background(), "new-id", "new-secret")
	require.NoError(t, err)
	configured, err = s.Configured(context.Background())
	require.NoError(t, err)
	assert.True(t, configured, "newly stored credentials must be reflected immediately")
}

func TestSetMentionChipTemplate_UpdatesTemplate(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	st, err := s.SetMentionChipTemplate(context.Background(), "user-1", "{ticket.Project} {ticket.Ticket}")
	require.NoError(t, err)
	assert.Equal(t, "{ticket.Project} {ticket.Ticket}", st.MentionChipTemplate)

	got, err := s.cfg.Settings.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "{ticket.Project} {ticket.Ticket}", got.MentionChipTemplate)
}

// TestSetMentionChipTemplate_RequiresPermission proves the write is gated on workspaces:write (spec.md section 5): a caller the gate denies is rejected with ErrForbidden and the stored template is left untouched.
func TestSetMentionChipTemplate_RequiresPermission(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	s.cfg.MentionLayout.(*fakeMentionLayoutGate).SetAllowed(false)

	_, err := s.SetMentionChipTemplate(context.Background(), "user-1", "{ticket.Project} {ticket.Ticket}")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Equal(t, "{ticket.Ticket} {ticket.Status}", settings.st.MentionChipTemplate, "denied write must not change the stored template")
}

// TestSetMentionChipTemplate_NilGateFailsClosed proves a Service built without a MentionLayout gate (a construction bug, not user input) refuses the write rather than silently allowing it.
func TestSetMentionChipTemplate_NilGateFailsClosed(t *testing.T) {
	s := NewService(Config{Secret: []byte("x"), Settings: newFakeSettings()})
	_, err := s.SetMentionChipTemplate(context.Background(), "user-1", "{ticket.Ticket}")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestAuthorizeURL(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "x")})
	settings.st.GitHubOAuthClientID = "client"
	settings.st.GitHubOAuthClientSecret = "secret"
	u, err := s.AuthorizeURL(context.Background(), "st8")
	require.NoError(t, err)
	assert.Contains(t, u, "client_id=client")
	assert.Contains(t, u, "redirect_uri=")
	assert.Contains(t, u, "state=st8")
	assert.Contains(t, u, "scope=read%3Auser")
}

func TestAuthorizeURL_NotConfigured(t *testing.T) {
	s := NewService(Config{Secret: []byte("x")})
	_, err := s.AuthorizeURL(context.Background(), "st8")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestAuthorizeURL_UsesCurrentSettings(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "x")})

	_, err := settings.SetGitHubOAuth(context.Background(), "rotated-id", "rotated-secret")
	require.NoError(t, err)
	u, err := s.AuthorizeURL(context.Background(), "st8")
	require.NoError(t, err)
	assert.Contains(t, u, "client_id=rotated-id", "must build from the current DB value, not a startup snapshot")
}

// TestGithubClient_BuildsFromCurrentSettings proves the GitHub API client itself is built fresh per request from whatever Settings currently holds (T3), not from a client constructed once at Service startup: changing the stored credentials between two calls changes what the very next call builds.
func TestGithubClient_BuildsFromCurrentSettings(t *testing.T) {
	settings := newFakeSettings()
	s := NewService(Config{Secret: []byte("x"), Settings: settings})

	_, err := s.providerClient(context.Background(), ProviderGitHub)
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "no credentials stored yet")

	_, err = settings.SetGitHubOAuth(context.Background(), "id-1", "secret-1")
	require.NoError(t, err)
	client1, err := s.providerClient(context.Background(), ProviderGitHub)
	require.NoError(t, err)
	hc1, ok := client1.(*HTTPGitHubClient)
	require.True(t, ok)
	assert.Equal(t, "id-1", hc1.clientID)
	assert.Equal(t, "secret-1", hc1.clientSecret)

	_, err = settings.SetGitHubOAuth(context.Background(), "id-2", "secret-2")
	require.NoError(t, err)
	client2, err := s.providerClient(context.Background(), ProviderGitHub)
	require.NoError(t, err)
	hc2, ok := client2.(*HTTPGitHubClient)
	require.True(t, ok)
	assert.Equal(t, "id-2", hc2.clientID, "second call must use the rotated credentials")
	assert.Equal(t, "secret-2", hc2.clientSecret)
	assert.NotSame(t, hc1, hc2, "client must be built fresh per request, not cached from startup")
}

// TestGithubClient_ExplicitOverrideBypassesSettings proves cfg.GitHub, when set (tests, local dev seams), always wins over Settings: this is the seam tests use to swap in a fake without touching the DB-backed credentials.
func TestGithubClient_ExplicitOverrideBypassesSettings(t *testing.T) {
	override := &fakeGitHub{user: ghUser("1", "onik97")}
	s := NewService(Config{Secret: []byte("x"), GitHub: override, Settings: newFakeSettings()})
	client, err := s.providerClient(context.Background(), ProviderGitHub)
	require.NoError(t, err)
	assert.Same(t, GitHubClient(override), client)
}

func TestRequireAuth(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	user, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "onik97"})
	require.NoError(t, err)
	require.NotNil(t, user)

	token, err := s.Sign("u1")
	require.NoError(t, err)

	t.Run("missing", func(t *testing.T) {
		inner := &captureHandler{}
		rec := doRequest(s.RequireAuth(inner), "GET", "/api/docs", "", "")
		assert.Equal(t, 401, rec.Code)
		assert.Nil(t, inner.user)
	})

	t.Run("invalid", func(t *testing.T) {
		inner := &captureHandler{}
		rec := doRequest(s.RequireAuth(inner), "GET", "/api/docs", "Bearer garbage", "")
		assert.Equal(t, 401, rec.Code)
		assert.Nil(t, inner.user)
	})

	t.Run("valid", func(t *testing.T) {
		inner := &captureHandler{}
		rec := doRequest(s.RequireAuth(inner), "GET", "/api/docs", "Bearer "+token, "")
		assert.Equal(t, 200, rec.Code)
		require.NotNil(t, inner.user)
		assert.Equal(t, "u1", inner.user.ID)
		assert.Equal(t, "onik97", inner.user.Login)
	})

	t.Run("unknown user", func(t *testing.T) {
		ghost, err := s.Sign("ghost")
		require.NoError(t, err)
		inner := &captureHandler{}
		rec := doRequest(s.RequireAuth(inner), "GET", "/api/docs", "Bearer "+ghost, "")
		assert.Equal(t, 401, rec.Code)
		assert.Nil(t, inner.user)
	})
}

func TestRequireAuth_NilUserStoreRejects(t *testing.T) {
	s := NewService(Config{Secret: []byte("x")})
	inner := &captureHandler{}
	rec := doRequest(s.RequireAuth(inner), "GET", "/api/docs", "Bearer x", "")
	assert.Equal(t, 401, rec.Code)
}

func TestRequireWS(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "onik97"})
	require.NoError(t, err)

	token, err := s.Sign("u1")
	require.NoError(t, err)

	t.Run("missing", func(t *testing.T) {
		inner := &captureHandler{}
		rec := doRequest(s.RequireWS(inner), "GET", "/ws/events", "", "")
		assert.Equal(t, 401, rec.Code)
		assert.Nil(t, inner.user)
	})

	t.Run("invalid", func(t *testing.T) {
		inner := &captureHandler{}
		rec := doRequest(s.RequireWS(inner), "GET", "/ws/events?token=garbage", "", "")
		assert.Equal(t, 401, rec.Code)
		assert.Nil(t, inner.user)
	})

	t.Run("valid query token", func(t *testing.T) {
		inner := &captureHandler{}
		rec := doRequest(s.RequireWS(inner), "GET", "/ws/events?token="+token, "", "")
		assert.Equal(t, 200, rec.Code)
		require.NotNil(t, inner.user)
		assert.Equal(t, "u1", inner.user.ID)
		assert.Equal(t, "onik97", inner.user.Login)
	})

	t.Run("unknown user", func(t *testing.T) {
		ghost, err := s.Sign("ghost")
		require.NoError(t, err)
		inner := &captureHandler{}
		rec := doRequest(s.RequireWS(inner), "GET", "/ws/events?token="+ghost, "", "")
		assert.Equal(t, 401, rec.Code)
		assert.Nil(t, inner.user)
	})
}

type captureHandler struct {
	user *User
}

func (c *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	c.user = UserFromCtx(r.Context())
}

func doRequest(h http.Handler, method, path, auth, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func mustVerify(t *testing.T, s *Service, token string) string {
	t.Helper()
	userID, err := s.Verify(token)
	require.NoError(t, err)
	return userID
}

// TestLoginWith_Google proves the second provider (ADR 0040) follows the same upsert → allowlist → sign path as GitHub, keyed by provider=google with the email as login so the owner allowlists clients by email.
func TestLoginWith_Google(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID, err := s.Verify(ownerToken)
	require.NoError(t, err)
	require.NoError(t, users.SetCanCreateWorkspace(context.Background(), ownerID, true))

	s.cfg.Google = &fakeGitHub{token: "at", user: &ProviderUser{ID: "sub-9", Login: "client@example.com", Name: "Client"}}

	_, _, err = users.UpsertUser(context.Background(), &User{ID: "google-id", Provider: ProviderGoogle, ProviderUserID: "sub-9", Login: "client@example.com"})
	require.NoError(t, err)
	token, err := s.LoginWith(context.Background(), ProviderGoogle, "good-code")
	require.NoError(t, err)
	id, err := s.Verify(token)
	require.NoError(t, err)
	u, err := users.GetUserByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, ProviderGoogle, u.Provider)
	assert.Equal(t, "sub-9", u.ProviderUserID)
	assert.Equal(t, "client@example.com", u.Login)
}

func TestLoginWith_UnknownProvider(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	_, err := s.LoginWith(context.Background(), Provider("gitlab"), "code")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestAuthorizeURLFor_Google(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "x")})
	_, err := settings.Set(context.Background(), "https://deploy.example.com/")
	require.NoError(t, err)

	_, err = s.AuthorizeURLFor(context.Background(), ProviderGoogle, "st8")
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "google is optional and off until credentials are stored")

	_, err = settings.SetProviderOAuth(context.Background(), ProviderGoogle, "g-id", "g-secret")
	require.NoError(t, err)
	u, err := s.AuthorizeURLFor(context.Background(), ProviderGoogle, "st8")
	require.NoError(t, err)
	assert.Contains(t, u, googleAuthorizeURL)
	assert.Contains(t, u, "client_id=g-id")
	assert.Contains(t, u, "redirect_uri=https%3A%2F%2Fdeploy.example.com%2Fauth%2Fgoogle%2Fcallback")
	assert.Contains(t, u, "scope=openid+email+profile")

	client, err := s.providerClient(context.Background(), ProviderGoogle)
	require.NoError(t, err)
	gc, ok := client.(*HTTPGoogleClient)
	require.True(t, ok)
	assert.Equal(t, "https://deploy.example.com/auth/google/callback", gc.redirectURI)
}

func TestSetProviderOAuth(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "owner"})
	require.NoError(t, err)
	_, _, err = users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	require.NoError(t, err)
	require.NoError(t, users.SetCanCreateWorkspace(context.Background(), "u1", true))

	_, err = s.SetProviderOAuth(context.Background(), "u2", ProviderGoogle, "id", "secret")
	assert.ErrorIs(t, err, apperrs.ErrForbidden)

	_, err = s.SetProviderOAuth(context.Background(), "u1", ProviderGoogle, "id", "")
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "half-configured must be rejected")

	st, err := s.SetProviderOAuth(context.Background(), "u1", ProviderGoogle, " id ", " secret ")
	require.NoError(t, err)
	assert.True(t, st.ProviderConfigured(ProviderGoogle))
	assert.Equal(t, "id", st.GoogleOAuthClientID)
	ok, err := s.ProviderConfigured(context.Background(), ProviderGoogle)
	require.NoError(t, err)
	assert.True(t, ok)

	_, err = s.SetProviderOAuth(context.Background(), "u1", ProviderGoogle, "", "")
	require.NoError(t, err)
	assert.False(t, settings.st.ProviderConfigured(ProviderGoogle), "both empty disables google sign-in")
}

func TestLoginWith_Discord(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	ownerToken, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID, err := s.Verify(ownerToken)
	require.NoError(t, err)
	require.NoError(t, users.SetCanCreateWorkspace(context.Background(), ownerID, true))
	_, err = settings.Set(context.Background(), "https://deploy.example.com")
	require.NoError(t, err)

	_, err = s.AuthorizeURLFor(context.Background(), ProviderDiscord, "st8")
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "discord is off until credentials are stored")
	_, err = settings.SetProviderOAuth(context.Background(), ProviderDiscord, "d-id", "d-secret")
	require.NoError(t, err)
	u, err := s.AuthorizeURLFor(context.Background(), ProviderDiscord, "st8")
	require.NoError(t, err)
	assert.Contains(t, u, discordAuthorizeURL)
	assert.Contains(t, u, "redirect_uri=https%3A%2F%2Fdeploy.example.com%2Fauth%2Fdiscord%2Fcallback")
	assert.Contains(t, u, "scope=identify+email")
	client, err := s.providerClient(context.Background(), ProviderDiscord)
	require.NoError(t, err)
	_, ok := client.(*HTTPDiscordClient)
	assert.True(t, ok)

	s.cfg.Discord = &fakeGitHub{token: "at", user: &ProviderUser{ID: "snowflake", Login: "client@example.com"}}
	_, _, err = users.UpsertUser(context.Background(), &User{ID: "discord-id", Provider: ProviderDiscord, ProviderUserID: "snowflake", Login: "client@example.com"})
	require.NoError(t, err)
	token, err := s.LoginWith(context.Background(), ProviderDiscord, "good-code")
	require.NoError(t, err)
	id, err := s.Verify(token)
	require.NoError(t, err)
	got, err := users.GetUserByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, ProviderDiscord, got.Provider)
	assert.Equal(t, "snowflake", got.ProviderUserID)
}

func TestSetProviderOAuth_RefusesGitHub(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	_, err := s.SetProviderOAuth(context.Background(), "u1", ProviderGitHub, "id", "secret")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.SetProviderOAuth(context.Background(), "u1", Provider("gitlab"), "id", "secret")
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestLookupMembers_RequiresPermission(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "bob")})
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "bob"})
	require.NoError(t, err)
	_, err = s.LookupMembers(context.Background(), "u1", "octo")
	assert.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestLookupMembers_TransportError(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	s.searchURL = "http://127.0.0.1:1/search"
	_, err := s.LookupMembers(context.Background(), "u1", "octo")
	assert.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestLookupMembers_InvalidJSON(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()
	s.searchURL = srv.URL
	_, err := s.LookupMembers(context.Background(), "u1", "octo")
	assert.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestLookupMembers_EmptyMatchesAreNeverNil(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	matches, err := s.LookupMembers(context.Background(), "u1", "")
	require.NoError(t, err)
	assert.NotNil(t, matches)
	assert.Empty(t, matches)
}
