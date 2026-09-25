package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestHandler_NotConfigured(t *testing.T) {
	s := NewService(Config{Secret: []byte("x")})
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "not configured")
}

// TestHandler_GithubGateReflectsLiveSettings proves the /auth/github 503 gate reads Settings.Configured() from the store on every request (T3), not a startup-time snapshot: clearing the stored credentials mid-session flips the very next request to 503, and restoring them flips it back, no restart involved.
func TestHandler_GithubGateReflectsLiveSettings(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	settings.st.GitHubOAuthClientID = "client"
	settings.st.GitHubOAuthClientSecret = "secret"
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github", nil))
	assert.Equal(t, http.StatusFound, rec.Code, "configured at harness setup")

	settings.st.GitHubOAuthClientID = ""
	settings.st.GitHubOAuthClientSecret = ""
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "cleared credentials must gate immediately")
	assert.Contains(t, rec.Body.String(), "not configured")

	_, err := settings.SetGitHubOAuth(context.Background(), "back-id", "back-secret")
	require.NoError(t, err)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github", nil))
	assert.Equal(t, http.StatusFound, rec.Code, "restored credentials must unblock immediately")
}

func TestHandler_StartOAuth(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	settings.st.GitHubOAuthClientID = "client"
	settings.st.GitHubOAuthClientSecret = "secret"
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github", nil))
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "https://github.com/login/oauth/authorize")
	assert.Contains(t, rec.Header().Get("Location"), "state=")

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, stateCookie, cookies[0].Name)
}

func TestHandler_AuthenticatedAcceptance_ReturnsGrantDetails(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	session, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	userID := mustVerify(t, s, session)
	s.SetInvitationGate(&fakeInvitationGate{token: "raw-token", invitation: &InvitationAcceptance{InvitationID: "inv-1", InstanceName: "Nexul"}})
	s.SetOAuthHandoffStore(&fakeOAuthHandoffStore{})
	h := NewHandler(s).Routes()
	req := httptest.NewRequest(http.MethodPost, "/api/invitations/acceptance", strings.NewReader(`{"token":"raw-token"}`))
	req.Header.Set("Authorization", "Bearer "+session)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body InvitationAcceptance
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "inv-1", body.InvitationID)
	assert.NotEmpty(t, body.AcceptanceToken)
	assert.Equal(t, userID, body.AuthenticatedUser.ID)
	_, err = users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
}

func TestHandler_StartOAuth_UsesConfiguredHTTPSForSecureCookie(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	settings.st.GitHubOAuthClientID = "client"
	settings.st.GitHubOAuthClientSecret = "secret"
	settings.st.InstanceURL = "https://nexul.example"
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/github", nil))
	require.Equal(t, http.StatusFound, rec.Code)
	require.Len(t, rec.Result().Cookies(), 1)
	assert.True(t, rec.Result().Cookies()[0].Secure)
}

func TestHandler_CallbackGET_StateMismatch(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	h := NewHandler(s).Routes()

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=c&state=stale", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "other"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_CallbackGET_NoStateCookie(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/callback?code=c&state=st8", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_CallbackGET_NoCode(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	h := NewHandler(s).Routes()

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?state=st8", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "st8"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CallbackGET_ExchangeFails(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97"), exErr: apperrs.ErrUnauthorized})
	h := NewHandler(s).Routes()

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=bad&state=st8", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "st8"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_CallbackPOST_BadJSON(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/callback", strings.NewReader(`{`)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_CallbackPOST_ExchangeFails(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97"), exErr: apperrs.ErrUnauthorized})
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/callback",
		strings.NewReader(`{"code":"bad"}`)))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_CallbackGET_ExchangeAndRedirect(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	h := NewHandler(s).Routes()

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=c&state=st8", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "st8"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusFound, rec.Code)
	loc := rec.Header().Get("Location")
	assert.Contains(t, loc, "/login?token=")
	token := strings.TrimPrefix(loc, "http://example.com/login?token=")
	userID, err := s.Verify(token)
	require.NoError(t, err)
	user, err := users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "onik97", user.Login)

	assert.True(t, rec.Result().Cookies()[0].MaxAge < 0, "state cookie should be cleared")
}

func TestHandler_CallbackGET_SPAOrigin(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	s.cfg.SPAOrigin = "https://deploy.example.com"
	h := NewHandler(s).Routes()

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=c&state=st8", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "st8"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.True(t, strings.HasPrefix(rec.Header().Get("Location"), "https://deploy.example.com/login?token="))
}

func TestHandler_CallbackPOST(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/callback",
		strings.NewReader(`{"code":"c"}`)))
	assert.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Token string `json:"token"`
	}
	require.NoError(t, decodeJSON(rec, &body))
	userID, err := s.Verify(body.Token)
	require.NoError(t, err)
	user, err := users.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "onik97", user.Login)
}

func TestHandler_CallbackPOST_EmptyCode(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/auth/callback", strings.NewReader(`{}`)))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func seedOwner(t *testing.T, users *fakeUserStore, id, providerID, login string) {
	t.Helper()
	_, _, err := users.UpsertUser(context.Background(), &User{ID: id, Provider: ProviderGitHub, ProviderUserID: providerID, Login: login})
	require.NoError(t, err)
	require.NoError(t, users.SetCanCreateWorkspace(context.Background(), id, true))
}

// fakeGitHubAppVerifier answers Bootstrap's live App check with err.
type fakeGitHubAppVerifier struct {
	err    error
	calls  int
	checks []string
}

func (f *fakeGitHubAppVerifier) VerifyGitHubApp(context.Context, string, string, string) error {
	f.calls++
	return f.err
}

func (f *fakeGitHubAppVerifier) VerifyGitHubAppCheck(_ context.Context, check, _, _, _ string) error {
	f.checks = append(f.checks, check)
	return f.err
}

func TestHandler_Bootstrap(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	seeder := &fakeConnectorAppSeeder{}
	s.cfg.ConnectorApps = seeder
	verifier := &fakeGitHubAppVerifier{}
	s.cfg.GitHubApp = verifier
	h := NewHandler(s).Routes()

	t.Run("credentials GitHub rejects are not stored", func(t *testing.T) {
		verifier.err = fmt.Errorf("%w: GitHub rejected the client secret", apperrs.ErrInvalid)
		defer func() { verifier.err = nil }()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap",
			strings.NewReader(`{"instance_url":"https://deploy.example.com","client_id":"gh-id","client_secret":"wrong","app_slug":"gh-slug"}`)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "rejected the client secret")
		st, err := s.cfg.Settings.Get(context.Background())
		require.NoError(t, err)
		assert.False(t, st.Configured(), "a rejected App must leave the instance unconfigured")
		assert.Nil(t, seeder.seeded)
	})

	t.Run("verify runs the named check and stores nothing", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap/verify?check=slug",
			strings.NewReader(`{"client_id":"gh-id","client_secret":"gh-secret","app_slug":"gh-slug"}`)))
		assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.Equal(t, []string{"slug"}, verifier.checks)
		st, err := s.cfg.Settings.Get(context.Background())
		require.NoError(t, err)
		assert.False(t, st.Configured(), "verify must not configure the instance")
		assert.Nil(t, seeder.seeded)
	})

	t.Run("verify surfaces GitHub's verdict", func(t *testing.T) {
		verifier.err = fmt.Errorf("%w: GitHub rejected the client secret", apperrs.ErrInvalid)
		defer func() { verifier.err = nil }()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap/verify?check=secret",
			strings.NewReader(`{"client_id":"gh-id","client_secret":"wrong","app_slug":"gh-slug"}`)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "rejected the client secret")
	})

	t.Run("status false before bootstrap", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Configured bool `json:"configured"`
		}
		require.NoError(t, decodeJSON(rec, &body))
		assert.False(t, body.Configured)
	})

	t.Run("missing app slug rejected", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap",
			strings.NewReader(`{"instance_url":"https://deploy.example.com","client_id":"gh-id","client_secret":"gh-secret"}`)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("numeric App ID rejected as slug", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap",
			strings.NewReader(`{"instance_url":"https://deploy.example.com","client_id":"gh-id","client_secret":"gh-secret","app_slug":"123456"}`)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("bootstrap succeeds and persists", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap",
			strings.NewReader(`{"instance_url":"https://deploy.example.com","client_id":"gh-id","client_secret":"gh-secret","app_slug":"gh-slug"}`)))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "gh-secret", "raw client secret must never be returned")
		var body map[string]any
		require.NoError(t, decodeJSON(rec, &body))
		assert.Equal(t, "gh-id", body["client_id"])
		assert.Equal(t, true, body["configured"])

		st, err := s.cfg.Settings.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "https://deploy.example.com", st.InstanceURL)
		assert.Equal(t, "gh-id", st.GitHubOAuthClientID)
		assert.Equal(t, "gh-secret", st.GitHubOAuthClientSecret)
		// The same GitHub App backs the connectors' install flow, so bootstrap must seed its app config so the owner wizard's Connect step works.
		assert.Equal(t, []string{"gh-id", "gh-secret", "gh-slug"}, seeder.seeded)
	})

	t.Run("status true after bootstrap", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Configured bool `json:"configured"`
		}
		require.NoError(t, decodeJSON(rec, &body))
		assert.True(t, body.Configured)
	})

	t.Run("status says reconfigurable while nobody has logged in", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil))
		var body struct {
			Reconfigurable bool `json:"reconfigurable"`
		}
		require.NoError(t, decodeJSON(rec, &body))
		assert.True(t, body.Reconfigurable)
	})

	t.Run("second bootstrap before any login replaces a wrong App", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap",
			strings.NewReader(`{"instance_url":"https://other.example.com","client_id":"other-id","client_secret":"other-secret","app_slug":"other-slug"}`)))
		assert.Equal(t, http.StatusOK, rec.Code)

		st, err := s.cfg.Settings.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "other-id", st.GitHubOAuthClientID)
	})

	t.Run("bootstrap conflicts once a user exists", func(t *testing.T) {
		seedOwner(t, users, "u1", "1", "onik97")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap",
			strings.NewReader(`{"instance_url":"https://third.example.com","client_id":"third-id","client_secret":"third-secret","app_slug":"third-slug"}`)))
		assert.Equal(t, http.StatusConflict, rec.Code)

		st, err := s.cfg.Settings.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "other-id", st.GitHubOAuthClientID, "a live instance must not have its App swapped")

		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap/verify?check=slug",
			strings.NewReader(`{"client_id":"third-id","client_secret":"third-secret","app_slug":"third-slug"}`)))
		assert.Equal(t, http.StatusConflict, rec.Code, "a live instance must not proxy GitHub checks for strangers")

		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil))
		var body struct {
			Reconfigurable bool `json:"reconfigurable"`
		}
		require.NoError(t, decodeJSON(rec, &body))
		assert.False(t, body.Reconfigurable)
	})
}

func TestHandler_Me(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")

	rec := httptest.NewRecorder()
	protectedHandler(s, users).ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/me", "u1", ""))
	assert.Equal(t, http.StatusOK, rec.Code)
	var st OnboardingStatus
	require.NoError(t, decodeJSON(rec, &st))
	require.NotNil(t, st.User)
	assert.Equal(t, "owner", st.User.Login)
	assert.True(t, st.User.CanCreateWorkspace)
	assert.False(t, st.NeedsOwnerWizard)
}

func TestHandler_CompleteOwnerWizard(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "owner"})
	require.NoError(t, err)
	h := protectedHandler(s, users)

	t.Run("invalid url", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/onboarding/owner", "u1", `{"instance_url":"nope"}`))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("grants owner and saves url", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/onboarding/owner", "u1", `{"instance_url":"https://deploy.example.com"}`))
		assert.Equal(t, http.StatusOK, rec.Code)
		user, err := users.GetUserByID(context.Background(), "u1")
		require.NoError(t, err)
		assert.True(t, user.CanCreateWorkspace)
		st, err := settings.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "https://deploy.example.com", st.InstanceURL)
	})
}

func TestHandler_CompleteFirstLogin(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "member")})
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "member"})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	protectedHandler(s, users).ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/onboarding/profile", "u1", ""))
	assert.Equal(t, http.StatusOK, rec.Code)
	user, err := users.GetUserByID(context.Background(), "u1")
	require.NoError(t, err)
	assert.True(t, user.FirstLoginDone)
}

func TestHandler_Settings(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	_, err := settings.Set(context.Background(), "https://deploy.example.com")
	require.NoError(t, err)
	h := protectedHandler(s, users)

	t.Run("get", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/settings", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"instance_url":"https://deploy.example.com"`)
		assert.Contains(t, rec.Body.String(), `"oauth_callback":"https://deploy.example.com/auth/callback"`)
	})

	t.Run("update by owner", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/settings", "u1", `{"instance_url":"https://new.example.com"}`))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"instance_url":"https://new.example.com"`)
	})

	t.Run("update by non-owner forbidden", func(t *testing.T) {
		_, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/settings", "u2", `{"instance_url":"https://evil.example.com"}`))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_ConnectionToken(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	h := protectedHandler(s, users)

	t.Run("no instance url yet", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/connection-token", "u1", ""))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	_, err := settings.Set(context.Background(), "https://deploy.example.com")
	require.NoError(t, err)

	t.Run("issues a parsable token", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/connection-token", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		var ct ConnectionToken
		require.NoError(t, decodeJSON(rec, &ct))
		claims, err := parseConnectionToken(s, ct.Token)
		require.NoError(t, err)
		assert.Equal(t, "https://deploy.example.com", claims.InstanceURL)
	})

	t.Run("non-owner forbidden", func(t *testing.T) {
		_, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/connection-token", "u2", ""))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_PersonalAccessTokens(t *testing.T) {
	s, users, _ := newPATHarness()
	seedOwner(t, users, "u1", "1", "owner")
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	require.NoError(t, err)
	h := protectedHandler(s, users)

	var mintedRaw string
	t.Run("mint returns raw once", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/tokens", "u1", `{"name":"ci agent"}`))
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, decodeJSON(rec, &body))
		mintedRaw, _ = body["token"].(string)
		assert.True(t, strings.HasPrefix(mintedRaw, patPrefix), "raw token must carry the pat prefix")
		prefix, _ := body["prefix"].(string)
		assert.Equal(t, mintedRaw[len(mintedRaw)-6:], prefix)
	})

	t.Run("empty name is invalid", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/tokens", "u1", `{"name":""}`))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("list never exposes a usable credential", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/tokens", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), mintedRaw, "raw token must never be listable")
		assert.NotContains(t, rec.Body.String(), `"token_hash"`, "stored hash must never be listable")
		assert.Contains(t, rec.Body.String(), `"ci agent"`)
	})

	t.Run("any user can mint their own", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/tokens", "u2", `{"name":"personal"}`))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("revoke then minted token stops working", func(t *testing.T) {
		raw, pat, err := s.MintPAT(context.Background(), "u1", "temp")
		require.NoError(t, err)
		_, err = s.AuthenticatePAT(context.Background(), raw)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/tokens/"+pat.ID, "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		_, err = s.AuthenticatePAT(context.Background(), raw)
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})

	t.Run("revoke twice not found", func(t *testing.T) {
		_, pat, err := s.MintPAT(context.Background(), "u1", "temp2")
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/tokens/"+pat.ID, "u1", ""))
		require.Equal(t, http.StatusOK, rec.Code)
		rec = httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/tokens/"+pat.ID, "u1", ""))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("revoking another user's token not found", func(t *testing.T) {
		_, pat, err := s.MintPAT(context.Background(), "u1", "mine")
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/tokens/"+pat.ID, "u2", ""))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_Members(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	require.NoError(t, err)
	h := protectedHandler(s, users)

	t.Run("list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"members":[]`)
	})

	t.Run("add and list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/members", "u1", `{"login":"Bob"}`))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"bob"`)
	})

	t.Run("duplicate add conflicts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPost, "/api/auth/members", "u1", `{"login":"bob"}`))
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("remove", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/members/bob", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"members":[]`)
	})

	t.Run("remove absent member", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodDelete, "/api/auth/members/nobody", "u1", ""))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("non-owner forbidden", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members", "u2", ""))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_LookupMembers(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	require.NoError(t, err)
	h := protectedHandler(s, users)

	t.Run("unauthenticated rejected", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/auth/members/lookup?q=octo", nil)
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-owner forbidden", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members/lookup?q=octo", "u2", ""))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("too short query skips github", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members/lookup?q=a", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"matches":[]}`, rec.Body.String())
	})

	t.Run("email query skips github", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members/lookup?q=someone@x.io", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"matches":[]}`, rec.Body.String())
	})

	t.Run("github search with configured credentials", func(t *testing.T) {
		settings.st.GitHubOAuthClientID = "cid"
		settings.st.GitHubOAuthClientSecret = "csecret"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "octo in:login", r.URL.Query().Get("q"))
			assert.Equal(t, "5", r.URL.Query().Get("per_page"))
			assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
			user, pass, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "cid", user)
			assert.Equal(t, "csecret", pass)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"login":"octocat","avatar_url":"https://avatar/octocat"},{"login":"octodog","avatar_url":"https://avatar/octodog"}]}`))
		}))
		defer srv.Close()
		s.searchURL = srv.URL

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members/lookup?q=octo", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Matches []LoginMatch `json:"matches"`
		}
		require.NoError(t, decodeJSON(rec, &body))
		require.Len(t, body.Matches, 2)
		assert.Equal(t, "octocat", body.Matches[0].Login)
		assert.Equal(t, "https://avatar/octocat", body.Matches[0].AvatarURL)
		assert.Equal(t, "octodog", body.Matches[1].Login)

		settings.st.GitHubOAuthClientID = ""
		settings.st.GitHubOAuthClientSecret = ""
	})

	t.Run("github search unauthenticated when no credentials configured", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _, ok := r.BasicAuth()
			assert.False(t, ok)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[]}`))
		}))
		defer srv.Close()
		s.searchURL = srv.URL

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members/lookup?q=octo", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"matches":[]}`, rec.Body.String())
	})

	t.Run("github rate limited returns empty matches", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()
		s.searchURL = srv.URL

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/members/lookup?q=octo", "u1", ""))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"matches":[]}`, rec.Body.String())
	})
}

func TestHandler_UpdateProfile(t *testing.T) {
	s, users, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "onik97", Name: "GitHub Name", AvatarURL: "https://avatar/gh"})
	require.NoError(t, err)
	h := protectedHandler(s, users)

	t.Run("round-trips the override and keeps provider fields", func(t *testing.T) {
		body := `{"display_name":"Manual Name","avatar_override_url":"data:image/png;base64,aGVsbG8="}`
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/profile", "u1", body))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var got User
		require.NoError(t, decodeJSON(rec, &got))
		require.NotNil(t, got.DisplayName)
		require.NotNil(t, got.AvatarOverrideURL)
		assert.Equal(t, "Manual Name", *got.DisplayName)
		assert.Equal(t, "data:image/png;base64,aGVsbG8=", *got.AvatarOverrideURL)
		assert.Equal(t, "GitHub Name", got.Name, "provider-sourced name is untouched")
		assert.Equal(t, "https://avatar/gh", got.AvatarURL, "provider-sourced avatar is untouched")
	})

	t.Run("rejects a non-data URI", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/profile", "u1",
			`{"avatar_override_url":"https://evil.example.com/x.png"}`))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects an oversized data URI", func(t *testing.T) {
		huge := strings.Repeat("A", (maxAvatarOverrideBytes/3+1)*4)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/profile", "u1",
			`{"avatar_override_url":"data:image/png;base64,`+huge+`"}`))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("a user can only ever set their own override", func(t *testing.T) {
		_, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/profile", "u2", `{"display_name":"Bob's Name"}`))
		require.Equal(t, http.StatusOK, rec.Code)

		u1, err := users.GetUserByID(context.Background(), "u1")
		require.NoError(t, err)
		require.NotNil(t, u1.DisplayName, "earlier override for u1 must be untouched")
		assert.Equal(t, "Manual Name", *u1.DisplayName)

		u2, err := users.GetUserByID(context.Background(), "u2")
		require.NoError(t, err)
		require.NotNil(t, u2.DisplayName)
		assert.Equal(t, "Bob's Name", *u2.DisplayName)
	})
}

func protectedHandler(s *Service, users *fakeUserStore) http.Handler {
	return s.RequireAuth(NewHandler(s).ProtectedRoutes())
}

func protectedRequest(t *testing.T, s *Service, users *fakeUserStore, method, path, userID, body string) *http.Request {
	t.Helper()
	if _, err := users.GetUserByID(context.Background(), userID); err != nil {
		t.Fatalf("user %s must exist in the fake store", userID)
	}
	token, err := s.Sign(userID)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func decodeJSON(rec *httptest.ResponseRecorder, v any) error {
	return json.NewDecoder(rec.Body).Decode(v)
}

func TestHandler_GoogleGateAndCallback(t *testing.T) {
	s, _, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "onik97")})
	_, err := settings.Set(context.Background(), "https://deploy.example.com")
	require.NoError(t, err)
	h := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "google is off until credentials are stored")

	_, err = settings.SetProviderOAuth(context.Background(), ProviderGoogle, "g-id", "g-secret")
	require.NoError(t, err)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "accounts.google.com")
	require.Len(t, rec.Result().Cookies(), 1)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil))
	assert.Contains(t, rec.Body.String(), `"google_configured":true`)

	s.cfg.Google = &fakeGitHub{token: "at", user: &ProviderUser{ID: "sub-1", Login: "client@example.com"}}
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=c&state=st8", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "st8"})
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "/login?token=")
}

func TestHandler_GoogleOAuthSettings(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	_, err := settings.Set(context.Background(), "https://deploy.example.com")
	require.NoError(t, err)
	h := protectedHandler(s, users)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/settings/oauth/google", "u1", `{"client_id":"g-id","client_secret":"g-secret"}`))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "g-secret", "raw client secret must never be returned")
	assert.Contains(t, rec.Body.String(), `"configured":true`)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/settings", "u1", ""))
	assert.Contains(t, rec.Body.String(), `"google_oauth_client_id":"g-id"`)
	assert.Contains(t, rec.Body.String(), `"google_oauth_callback":"https://deploy.example.com/auth/google/callback"`)
	assert.NotContains(t, rec.Body.String(), "g-secret")

	_, _, err = users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
	require.NoError(t, err)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/settings/oauth/google", "u2", `{"client_id":"x","client_secret":"y"}`))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_DiscordGateAndSettings(t *testing.T) {
	s, users, _, settings := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	seedOwner(t, users, "u1", "1", "owner")
	_, err := settings.Set(context.Background(), "https://deploy.example.com")
	require.NoError(t, err)
	public := NewHandler(s).Routes()
	protected := protectedHandler(s, users)

	rec := httptest.NewRecorder()
	public.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/discord", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/settings/oauth/discord", "u1", `{"client_id":"d-id","client_secret":"d-secret"}`))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "d-secret")
	assert.Contains(t, rec.Body.String(), `"provider":"discord"`)

	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodPut, "/api/auth/settings/oauth/github", "u1", `{"client_id":"x","client_secret":"y"}`))
	assert.Equal(t, http.StatusBadRequest, rec.Code, "github credentials belong to bootstrap, not this route")

	rec = httptest.NewRecorder()
	public.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/discord", nil))
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "discord.com/oauth2/authorize")

	rec = httptest.NewRecorder()
	public.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil))
	assert.Contains(t, rec.Body.String(), `"discord_configured":true`)

	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, protectedRequest(t, s, users, http.MethodGet, "/api/auth/settings", "u1", ""))
	assert.Contains(t, rec.Body.String(), `"discord_oauth_callback":"https://deploy.example.com/auth/discord/callback"`)

	s.cfg.Discord = &fakeGitHub{token: "at", user: &ProviderUser{ID: "snow", Login: "client@example.com"}}
	req := httptest.NewRequest(http.MethodGet, "/auth/discord/callback?code=c&state=st8", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "st8"})
	rec = httptest.NewRecorder()
	public.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "not allowlisted → rejected like any provider")
}
