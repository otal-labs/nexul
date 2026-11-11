package sso

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBrowser struct{ minted []string }

func (fakeBrowser) SPAOrigin(*http.Request) string  { return "https://nexul.example.com" }
func (fakeBrowser) SecureCookie(*http.Request) bool { return true }
func (b *fakeBrowser) SessionFor(_ *http.Request, userID string) (string, error) {
	b.minted = append(b.minted, userID)
	return "ses_" + userID, nil
}

func serve(t *testing.T, h http.Handler, method, target, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(asActor(t), method, target, strings.NewReader(body))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_Start_RedirectsToTheProviderWithTheStateInAHttpOnlyCookie(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	rec := serve(t, NewHandler(h.svc, &fakeBrowser{}).PublicRoutes(), http.MethodGet, "/auth/oidc", "")

	require.Equal(t, http.StatusFound, rec.Code)
	cookie := rec.Result().Cookies()[0]
	assert.Equal(t, stateCookie, cookie.Name)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "https://idp.test/authorize?state="+cookie.Value, rec.Header().Get("Location"))
}

func TestHandler_Callback(t *testing.T) {
	t.Parallel()
	state := &http.Cookie{Name: stateCookie, Value: "state-1"}
	tests := []struct {
		name     string
		query    string
		cookies  []*http.Cookie
		licensed bool
		setup    func(*harness)
		want     string
	}{
		{"a signed-in person lands on /login with their session", "state=state-1&code=c", []*http.Cookie{state}, true,
			func(h *harness) { h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-1", Status: "active"} },
			"https://nexul.example.com/login?token=ses_u-1"},
		{"no state cookie", "state=state-1&code=c", nil, true, nil, "/login?error=state_mismatch"},
		{"another browser's state", "state=state-2&code=c", []*http.Cookie{state}, true, nil, "/login?error=state_mismatch"},
		{"the provider's refusal", "state=state-1&error=access_denied", []*http.Cookie{state}, true, nil, "/login?error=access_denied"},
		{"another provider error", "state=state-1&error=server_error", []*http.Cookie{state}, true, nil, "/login?error=provider_error"},
		{"no code", "state=state-1", []*http.Cookie{state}, true, nil, "/login?error=sso_failed"},
		{"an unverified email", "state=state-1&code=c", []*http.Cookie{state}, true,
			func(h *harness) { h.idp.claims.EmailVerified = false }, "/login?error=sso_email_unverified"},
		{"a disabled account", "state=state-1&code=c", []*http.Cookie{state}, true,
			func(h *harness) { h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-1", Status: "disabled"} }, "/login?error=account_disabled"},
		{"a removed account", "state=state-1&code=c", []*http.Cookie{state}, true,
			func(h *harness) { h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-1", Status: "removed"} }, "/login?error=account_removed"},
		{"a role that can no longer be joined", "state=state-1&code=c", []*http.Cookie{state}, true,
			func(h *harness) { h.accounts.roles["member"] = false }, "/login?error=sso_not_ready"},
		{"single sign-on off for lack of a license", "state=state-1&code=c", []*http.Cookie{state}, false, nil, "/login?error=not_configured"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tt.licensed)
			if tt.setup != nil {
				tt.setup(h)
			}
			b := &fakeBrowser{}
			rec := serve(t, NewHandler(h.svc, b).PublicRoutes(), http.MethodGet, "/auth/oidc/callback?"+tt.query, "", tt.cookies...)

			require.Equal(t, http.StatusFound, rec.Code)
			loc := rec.Header().Get("Location")
			assert.True(t, strings.HasSuffix(loc, tt.want) || loc == tt.want, "went to %s, want %s", loc, tt.want)
			if !strings.Contains(tt.want, "token=") {
				assert.Empty(t, b.minted, "no session starts on a refused sign-in")
			}
		})
	}
}

func TestHandler_Callback_UsesTheStateCookieOnce(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-1", Status: "active"}
	rec := serve(t, NewHandler(h.svc, &fakeBrowser{}).PublicRoutes(), http.MethodGet, "/auth/oidc/callback?state=s&code=c", "",
		&http.Cookie{Name: stateCookie, Value: "s"})

	cleared := rec.Result().Cookies()
	require.Len(t, cleared, 1)
	assert.Equal(t, stateCookie, cleared[0].Name)
	assert.Negative(t, cleared[0].MaxAge, "the cookie is cleared, so the same callback cannot be replayed")
}

func TestHandler_PublicState_NeedsNoSession(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/sso", nil)
	rec := httptest.NewRecorder()
	NewHandler(h.svc, &fakeBrowser{}).PublicRoutes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"active":true,"name":"Acme SSO","only":false}`, rec.Body.String())
}

func TestHandler_Settings_NeverReturnTheClientSecret_AndAnswerRefusalsWithTheirStatus(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	routes := NewHandler(h.svc, &fakeBrowser{}).Routes()

	rec := serve(t, routes, http.MethodGet, "/api/instance/sso", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), verifiedSecret)
	assert.Contains(t, rec.Body.String(), `"client_secret_set":true`)

	rec = serve(t, routes, http.MethodPut, "/api/instance/sso", `{"issuer_url":"http://insecure.example.com"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, routes, http.MethodPut, "/api/instance/sso", `{"display_name":"Acme"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), verifiedSecret)

	rec = serve(t, routes, http.MethodPost, "/api/instance/sso/verify?check=issuer", `{"issuer_url":"https://idp.example.com"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"detail":"Found the sign-in endpoints at idp.example.com"}`, rec.Body.String())
	rec = serve(t, routes, http.MethodPost, "/api/instance/sso/verify?check=secret", `{"client_secret":"plain"}`)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = serve(t, routes, http.MethodDelete, "/api/instance/sso", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"unlocked":true,"configured":false,"issuer_url":"","client_id":"","client_secret_set":false,"display_name":"","join_role_id":"","only":false,"redirect_uri":"https://nexul.example.com/auth/oidc/callback","active":false}`, rec.Body.String())
}
