package connectors_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// memAppConfigStore is a minimal in-memory AppConfigStore for handler tests.
type memAppConfigStore struct {
	rows map[string]connectors.AppConfig
}

func (m *memAppConfigStore) GetAppConfig(_ context.Context, connectorID string) (connectors.AppConfig, error) {
	c, ok := m.rows[connectorID]
	if !ok {
		return connectors.AppConfig{ConnectorID: connectorID}, nil
	}
	return c, nil
}

func (m *memAppConfigStore) SetAppConfig(_ context.Context, c connectors.AppConfig) error {
	m.rows[c.ConnectorID] = c
	return nil
}

// fakeOwnerGate is a canned OwnerGate for handler tests: userID "owner-1"
// holds the instance-admin bit, everyone else doesn't.
type fakeOwnerGate struct{}

func (fakeOwnerGate) CanCreateWorkspace(_ context.Context, userID string) (bool, error) {
	return userID == "owner-1", nil
}

type fakeOAuthClient struct {
	configured  bool
	exchangeTS  *connectors.TokenSet
	exchangeErr error
	revoked     []string
}

func (f *fakeOAuthClient) Configured() bool { return f.configured }
func (f *fakeOAuthClient) AuthorizeURL(state string) string {
	return "https://provider.example/authorize?state=" + state
}
func (f *fakeOAuthClient) Exchange(context.Context, string) (*connectors.TokenSet, error) {
	return f.exchangeTS, f.exchangeErr
}
func (f *fakeOAuthClient) Refresh(context.Context, string) (*connectors.TokenSet, error) {
	return f.exchangeTS, nil
}
func (f *fakeOAuthClient) Revoke(_ context.Context, accessToken string) error {
	f.revoked = append(f.revoked, accessToken)
	return nil
}

type memStore struct {
	rows map[string]connectors.Credentials
}

func (m *memStore) GetCredentials(_ context.Context, connectorID string) (connectors.Credentials, error) {
	c, ok := m.rows[connectorID]
	if !ok {
		return connectors.Credentials{}, apperrs.ErrNotFound
	}
	return c, nil
}

func (m *memStore) SaveCredentials(_ context.Context, c connectors.Credentials) error {
	m.rows[c.ConnectorID] = c
	return nil
}

func (m *memStore) DeleteCredentials(_ context.Context, connectorID string) error {
	delete(m.rows, connectorID)
	return nil
}

// withUserID injects a fake authenticated user id the way the composition
// root's real middleware would (connectors.WithUserID), simulating
// RequireAuth for the authenticated routes under test.
func withUserID(userID string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(connectors.WithUserID(r.Context(), userID)))
	})
}

// fakeSettings satisfies connectors.SettingsReader with a canned instance URL.
type fakeSettings struct{ url string }

func (f fakeSettings) GetInstanceURL(context.Context) (string, error) { return f.url, nil }

func newTestHandler(t *testing.T, oauth *fakeOAuthClient) (*connectors.Handler, *memStore) {
	t.Helper()
	store := &memStore{rows: map[string]connectors.Credentials{}}
	svc := connectors.NewService(connectors.Config{
		Store:    store,
		Settings: fakeSettings{url: "https://spa.example/"},
		Registry: []connectors.Connector{
			{ID: "github", Name: "GitHub", Description: "d", Category: "development", OAuth: oauth},
			{ID: "google", Name: "Google", Description: "d", Category: "productivity", OAuth: nil},
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	})
	return connectors.NewHandler(svc), store
}

func TestConnectFlow_StartCallbackList(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true, exchangeTS: &connectors.TokenSet{
		AccessToken: "at1", RefreshToken: "rt1", ExpiresIn: time.Hour,
	}}
	h, store := newTestHandler(t, oauth)

	authed := withUserID("user-1", h.Routes())
	pub := h.PublicRoutes()

	// 1. start: authenticated, mints a state cookie, returns the fake's
	// authorize URL for the SPA to navigate to.
	startReq := httptest.NewRequest("GET", "/api/connectors/github/oauth/start", nil)
	startRec := httptest.NewRecorder()
	authed.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("oauth start status = %d, body %s", startRec.Code, startRec.Body.String())
	}
	cookies := startRec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "nexul_connector_oauth_state" {
		t.Fatalf("oauth start cookies = %+v, want one nexul_connector_oauth_state cookie", cookies)
	}
	stateCookie := cookies[0]
	if !strings.Contains(startRec.Body.String(), "https://provider.example/authorize?state=") {
		t.Fatalf("oauth start body = %s, want the fake's authorize URL", startRec.Body.String())
	}

	// 2. callback: the provider redirects the browser back with the same
	// state; we verify it via the cookie, complete the exchange, and
	// redirect into the SPA.
	callbackURL := "/auth/connectors/github/callback?code=abc&state=" + url.QueryEscape(stateCookie.Value)
	cbReq := httptest.NewRequest("GET", callbackURL, nil)
	cbReq.AddCookie(stateCookie)
	cbRec := httptest.NewRecorder()
	pub.ServeHTTP(cbRec, cbReq)
	if cbRec.Code != http.StatusFound {
		t.Fatalf("oauth callback status = %d, body %s", cbRec.Code, cbRec.Body.String())
	}
	loc := cbRec.Header().Get("Location")
	// Must target the configured instance URL (where the SPA actually lives),
	// not the API host the provider redirect landed on.
	if loc != "https://spa.example/settings?connector=github&connected=1" {
		t.Fatalf("oauth callback redirect = %q, want the instance URL's /settings with connector=github", loc)
	}

	// 3. the stored credential now shows up in List, connected by the user
	// that started the flow.
	cred, err := store.GetCredentials(context.Background(), "github")
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if cred.AccessToken != "at1" || cred.ConnectedBy != "user-1" {
		t.Fatalf("stored cred = %+v, want at1/user-1", cred)
	}

	listReq := httptest.NewRequest("GET", "/api/connectors", nil)
	listRec := httptest.NewRecorder()
	authed.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"configured":true`) || !strings.Contains(listRec.Body.String(), "user-1") {
		t.Fatalf("list body = %s, want configured:true and user-1", listRec.Body.String())
	}
}

func TestOAuthCallback_StateMismatch_RedirectsToSPAWithError(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true}
	h, store := newTestHandler(t, oauth)
	pub := h.PublicRoutes()

	req := httptest.NewRequest("GET", "/auth/connectors/github/callback?code=abc&state=wrong", nil)
	req.AddCookie(&http.Cookie{Name: "nexul_connector_oauth_state", Value: "right"})
	rec := httptest.NewRecorder()
	pub.ServeHTTP(rec, req)
	// A browser navigation, not an API call: the failure must land back in
	// the SPA as ?error=, never a JSON body on the API origin.
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 into the SPA on state mismatch, body %s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://spa.example/settings?connector=github&error=") {
		t.Fatalf("redirect = %q, want the SPA settings page with an error param", loc)
	}
	if _, err := store.GetCredentials(context.Background(), "github"); err == nil {
		t.Fatal("state mismatch must not store credentials")
	}
}

// TestOAuthCallback_NotInstalled_ForwardsToInstall proves the first-run flow:
// an authorized-but-not-installed exchange forwards the browser to the
// provider's install page with a freshly minted state cookie, so the
// post-install redirect completes the same callback.
func TestOAuthCallback_NotInstalled_ForwardsToInstall(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true, exchangeErr: &connectors.NotInstalledError{
		InstallURL: "https://github.com/apps/my-app/installations/new?redirect_uri=cb",
	}}
	h, _ := newTestHandler(t, oauth)
	pub := h.PublicRoutes()

	req := httptest.NewRequest("GET", "/auth/connectors/github/callback?code=abc&state="+url.QueryEscape("nonce-1:user-1"), nil)
	req.AddCookie(&http.Cookie{Name: "nexul_connector_oauth_state", Value: "nonce-1:user-1"})
	rec := httptest.NewRecorder()
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 to the install page, body %s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil || loc.Host != "github.com" || !strings.Contains(loc.Path, "/apps/my-app/installations/new") {
		t.Fatalf("redirect = %q, want the provider install URL", rec.Header().Get("Location"))
	}
	state := loc.Query().Get("state")
	if state == "" || state == "nonce-1:user-1" || !strings.HasSuffix(state, ":user-1") {
		t.Fatalf("install redirect state = %q, want a fresh nonce still carrying user-1", state)
	}
	var freshCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "nexul_connector_oauth_state" && c.MaxAge >= 0 && c.Value != "" {
			freshCookie = c
		}
	}
	if freshCookie == nil || freshCookie.Value != state {
		t.Fatalf("cookies = %+v, want a fresh state cookie matching the install URL's state %q", rec.Result().Cookies(), state)
	}
}

func TestOAuthCallback_ExchangeFailure_RedirectsToSPAWithError(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true, exchangeErr: errors.New("provider exploded")}
	h, _ := newTestHandler(t, oauth)
	pub := h.PublicRoutes()

	req := httptest.NewRequest("GET", "/auth/connectors/github/callback?code=abc&state="+url.QueryEscape("nonce-1:user-1"), nil)
	req.AddCookie(&http.Cookie{Name: "nexul_connector_oauth_state", Value: "nonce-1:user-1"})
	rec := httptest.NewRecorder()
	pub.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 into the SPA, body %s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://spa.example/settings?connector=github&error=") || !strings.Contains(loc, "provider+exploded") {
		t.Fatalf("redirect = %q, want the SPA settings page carrying the failure message", loc)
	}
}

func TestDisconnectFlow_RevokesDeletesAndNoOpsWhenNotConnected(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true}
	h, store := newTestHandler(t, oauth)
	authed := withUserID("user-1", h.Routes())

	if err := store.SaveCredentials(context.Background(), connectors.Credentials{
		ConnectorID: "github", AccessToken: "at1",
	}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/connectors/github/disconnect", nil)
	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disconnect status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(oauth.revoked) != 1 || oauth.revoked[0] != "at1" {
		t.Fatalf("revoked = %v, want [at1]", oauth.revoked)
	}
	if _, err := store.GetCredentials(context.Background(), "github"); err == nil {
		t.Fatalf("credentials still present after disconnect")
	}

	// A never-connected connector's disconnect is a clean no-op (200, not an
	// error), even though it was never in the store to begin with.
	req2 := httptest.NewRequest("POST", "/api/connectors/google/disconnect", nil)
	rec2 := httptest.NewRecorder()
	authed.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("disconnect (never connected) status = %d, body %s, want 200 no-op", rec2.Code, rec2.Body.String())
	}
}

func newTestHandlerWithAppConfig(t *testing.T) (*connectors.Handler, *memAppConfigStore) {
	t.Helper()
	credStore := &memStore{rows: map[string]connectors.Credentials{}}
	appStore := &memAppConfigStore{rows: map[string]connectors.AppConfig{}}
	svc := connectors.NewService(connectors.Config{
		Store:          credStore,
		AppConfigStore: appStore,
		Owner:          fakeOwnerGate{},
		Registry: []connectors.Connector{
			{ID: "github", Name: "GitHub", Description: "d", Category: "development"},
		},
	})
	return connectors.NewHandler(svc), appStore
}

// TestSetAppConfig_OwnerAllowed_NonOwnerForbidden proves PUT
// /api/connectors/{id}/app-config requires owner permission: the owner's
// request succeeds and is stored, a non-owner's request is rejected and
// leaves storage untouched.
func TestSetAppConfig_OwnerAllowed_NonOwnerForbidden(t *testing.T) {
	h, appStore := newTestHandlerWithAppConfig(t)

	body := strings.NewReader(`{"client_id":"Iv1.abc","client_secret":"shh","base_url":""}`)
	ownerReq := httptest.NewRequest("PUT", "/api/connectors/github/app-config", body)
	ownerReq = ownerReq.WithContext(connectors.WithUserID(ownerReq.Context(), "owner-1"))
	ownerRec := httptest.NewRecorder()
	h.Routes().ServeHTTP(ownerRec, ownerReq)
	if ownerRec.Code != http.StatusOK {
		t.Fatalf("owner app-config status = %d, body %s", ownerRec.Code, ownerRec.Body.String())
	}
	if !strings.Contains(ownerRec.Body.String(), `"configured":true`) {
		t.Fatalf("owner app-config body = %s, want configured:true", ownerRec.Body.String())
	}
	if strings.Contains(ownerRec.Body.String(), "shh") {
		t.Fatalf("owner app-config body = %s, must never echo the client secret", ownerRec.Body.String())
	}
	cfg, _ := appStore.GetAppConfig(context.Background(), "github")
	if cfg.ClientSecret != "shh" {
		t.Fatalf("stored client secret = %q, want shh", cfg.ClientSecret)
	}

	nonOwnerBody := strings.NewReader(`{"client_id":"Iv1.other","client_secret":"nope","base_url":""}`)
	nonOwnerReq := httptest.NewRequest("PUT", "/api/connectors/github/app-config", nonOwnerBody)
	nonOwnerReq = nonOwnerReq.WithContext(connectors.WithUserID(nonOwnerReq.Context(), "not-owner"))
	nonOwnerRec := httptest.NewRecorder()
	h.Routes().ServeHTTP(nonOwnerRec, nonOwnerReq)
	if nonOwnerRec.Code != http.StatusForbidden {
		t.Fatalf("non-owner app-config status = %d, want 403, body %s", nonOwnerRec.Code, nonOwnerRec.Body.String())
	}
	cfgAfter, _ := appStore.GetAppConfig(context.Background(), "github")
	if cfgAfter.ClientID != "Iv1.abc" {
		t.Fatalf("non-owner request must not overwrite stored config, got %+v", cfgAfter)
	}
}

// fakeManualVerifier is a canned Verifier for the handler-level manual-save
// tests (ticket 10).
type fakeManualVerifier struct {
	err error
}

func (f *fakeManualVerifier) Verify(context.Context, map[string]string) error { return f.err }

func newTestHandlerWithManual(t *testing.T, verifier connectors.Verifier) (*connectors.Handler, *memStore) {
	t.Helper()
	store := &memStore{rows: map[string]connectors.Credentials{}}
	svc := connectors.NewService(connectors.Config{
		Store: store,
		Registry: []connectors.Connector{
			{ID: "livekit", Name: "LiveKit", Description: "d", Category: "communication", Manual: []connectors.CredentialField{
				{Key: "ws_url", Label: "WebSocket URL"},
				{Key: "api_key", Label: "API key"},
				{Key: "api_secret", Label: "API secret", Secret: true},
			}, Verify: verifier},
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	})
	return connectors.NewHandler(svc), store
}

// TestSaveManual_VerifiesStoresAndReturnsStatus proves POST
// /api/connectors/{id}/manual runs Verify before storing, then responds
// with the safe CredentialStatus (never the submitted secret values).
func TestSaveManual_VerifiesStoresAndReturnsStatus(t *testing.T) {
	h, store := newTestHandlerWithManual(t, &fakeManualVerifier{})
	authed := withUserID("user-1", h.Routes())

	body := strings.NewReader(`{"ws_url":"wss://lk.example.com","api_key":"key1","api_secret":"shh"}`)
	req := httptest.NewRequest("POST", "/api/connectors/livekit/manual", body)
	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save manual status = %d, body %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "shh") {
		t.Fatalf("response body = %s, must never echo the submitted secret", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"configured":true`) {
		t.Fatalf("response body = %s, want configured:true", rec.Body.String())
	}
	cred, err := store.GetCredentials(context.Background(), "livekit")
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if cred.ManualFields["api_secret"] != "shh" {
		t.Fatalf("stored ManualFields = %+v, want api_secret=shh", cred.ManualFields)
	}
}

// TestSaveManual_VerifyFails_400InlineNothingStored proves a failing
// provider verification surfaces as 400 with the provider's message and
// stores nothing.
func TestSaveManual_VerifyFails_400InlineNothingStored(t *testing.T) {
	h, store := newTestHandlerWithManual(t, &fakeManualVerifier{err: errors.New("bad api key")})
	authed := withUserID("user-1", h.Routes())

	body := strings.NewReader(`{"ws_url":"wss://lk.example.com","api_key":"key1","api_secret":"wrong"}`)
	req := httptest.NewRequest("POST", "/api/connectors/livekit/manual", body)
	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("save manual (verify fails) status = %d, want 400, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "bad api key") {
		t.Fatalf("response body = %s, want the provider's error inline", rec.Body.String())
	}
	if _, err := store.GetCredentials(context.Background(), "livekit"); err == nil {
		t.Fatal("failed verify must not store credentials")
	}
}

// POST /api/connectors/{id}/manual/verify answers 204 on a passing check and stores nothing.
func TestVerifyManual_PassesWithoutStoring(t *testing.T) {
	h, store := newTestHandlerWithManual(t, &fakeManualVerifier{})
	authed := withUserID("user-1", h.Routes())

	body := strings.NewReader(`{"ws_url":"wss://lk.example.com","api_key":"key1","api_secret":"shh"}`)
	req := httptest.NewRequest("POST", "/api/connectors/livekit/manual/verify", body)
	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("verify manual status = %d, want 204, body %s", rec.Code, rec.Body.String())
	}
	if _, err := store.GetCredentials(context.Background(), "livekit"); err == nil {
		t.Fatal("verify must not store credentials")
	}
}

func TestVerifyManual_ProviderRejects_400(t *testing.T) {
	h, _ := newTestHandlerWithManual(t, &fakeManualVerifier{err: errors.New("bad api key")})
	authed := withUserID("user-1", h.Routes())

	body := strings.NewReader(`{"ws_url":"wss://lk.example.com","api_key":"key1","api_secret":"wrong"}`)
	req := httptest.NewRequest("POST", "/api/connectors/livekit/manual/verify", body)
	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("verify manual status = %d, want 400, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "bad api key") {
		t.Fatalf("response body = %s, want the provider's error inline", rec.Body.String())
	}
}

// fakeCheckVerifier answers one named check at a time; failing lists the keys that should fail.
type fakeCheckVerifier struct {
	failing map[string]error
}

func (f *fakeCheckVerifier) Verify(context.Context, map[string]string) error { return nil }
func (f *fakeCheckVerifier) VerifyCheck(_ context.Context, _ map[string]string, key string) error {
	return f.failing[key]
}

func TestVerifyManual_SingleCheck(t *testing.T) {
	store := &memStore{rows: map[string]connectors.Credentials{}}
	svc := connectors.NewService(connectors.Config{
		Store: store,
		Registry: []connectors.Connector{
			{ID: "cloudflare", Name: "Cloudflare", Description: "d", Category: "infrastructure",
				Manual: []connectors.CredentialField{{Key: "api_token", Label: "API token", Secret: true}},
				Checks: []connectors.CredentialCheck{{Key: "token", Label: "Token is active"}, {Key: "dns_edit", Label: "Zone → DNS: Edit"}},
				Verify: &fakeCheckVerifier{failing: map[string]error{"dns_edit": errors.New("the token is missing Zone → DNS: Edit")}},
			},
		},
	})
	authed := withUserID("user-1", connectors.NewHandler(svc).Routes())

	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, httptest.NewRequest("POST", "/api/connectors/cloudflare/manual/verify?check=token", strings.NewReader(`{"api_token":"tok"}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("check token status = %d, want 204, body %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	authed.ServeHTTP(rec, httptest.NewRequest("POST", "/api/connectors/cloudflare/manual/verify?check=dns_edit", strings.NewReader(`{"api_token":"tok"}`)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "DNS: Edit") {
		t.Fatalf("check dns_edit = %d %s, want 400 naming the permission", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	authed.ServeHTTP(rec, httptest.NewRequest("POST", "/api/connectors/cloudflare/manual/verify?check=nope", strings.NewReader(`{"api_token":"tok"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown check status = %d, want 400", rec.Code)
	}
}

func TestVerifyManual_Unauthenticated_Rejected(t *testing.T) {
	h, _ := newTestHandlerWithManual(t, &fakeManualVerifier{})
	req := httptest.NewRequest("POST", "/api/connectors/livekit/manual/verify", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("verify manual (no user) status = %d, want 401", rec.Code)
	}
}

// TestSaveManual_Unauthenticated_Rejected proves the route requires an
// authenticated user, same as the other /api/connectors routes.
func TestSaveManual_Unauthenticated_Rejected(t *testing.T) {
	h, _ := newTestHandlerWithManual(t, &fakeManualVerifier{})
	req := httptest.NewRequest("POST", "/api/connectors/livekit/manual", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("save manual (no user) status = %d, want 401, body %s", rec.Code, rec.Body.String())
	}
}

// TestManualDisconnect_DeletesRow proves disconnect works for a manual
// connector the same way it does for an OAuth one (delete row, no revoke
// call to make since there's no OAuth client).
func TestManualDisconnect_DeletesRow(t *testing.T) {
	h, store := newTestHandlerWithManual(t, &fakeManualVerifier{})
	authed := withUserID("user-1", h.Routes())
	if err := store.SaveCredentials(context.Background(), connectors.Credentials{
		ConnectorID:  "livekit",
		ManualFields: map[string]string{"ws_url": "a", "api_key": "b", "api_secret": "c"},
	}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/connectors/livekit/disconnect", nil)
	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disconnect status = %d, body %s", rec.Code, rec.Body.String())
	}
	if _, err := store.GetCredentials(context.Background(), "livekit"); err == nil {
		t.Fatal("credentials still present after disconnect")
	}
}

func TestOAuthStart_ComingSoonConnector_Rejected(t *testing.T) {
	h, _ := newTestHandler(t, &fakeOAuthClient{configured: true})
	authed := withUserID("user-1", h.Routes())

	req := httptest.NewRequest("GET", "/api/connectors/google/oauth/start", nil)
	rec := httptest.NewRecorder()
	authed.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oauth start (coming soon) status = %d, want 400, body %s", rec.Code, rec.Body.String())
	}
}
