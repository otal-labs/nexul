package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

type fakeOAuthClients struct {
	mu       sync.Mutex
	byID     map[string]OAuthClient
	getErr   error
	sweepErr error
	sweptTo  []time.Time
}

func (f *fakeOAuthClients) DeleteUngrantedOAuthClients(_ context.Context, before time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sweptTo = append(f.sweptTo, before)
	return f.sweepErr
}

func (f *fakeOAuthClients) CreateOAuthClient(_ context.Context, c OAuthClient) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[c.ID] = c
	return nil
}

func (f *fakeOAuthClients) GetOAuthClient(_ context.Context, id string) (*OAuthClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.byID[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &c, nil
}

const testRedirect = "http://localhost:4000/cb"

var testVerifier = strings.Repeat("a", 43)

func newOAuthHarness(t *testing.T) (*Service, *fakeUserStore, *fakeOAuthClients, string) {
	t.Helper()
	s, users, _ := newPATHarness()
	clients := &fakeOAuthClients{byID: map[string]OAuthClient{}}
	s.cfg.OAuthClients = clients
	userID := seedPATUser(t, users)
	return s, users, clients, userID
}

func mustClient(t *testing.T, s *Service) string {
	t.Helper()
	c, err := s.RegisterOAuthClient(t.Context(), "10.0.0.1", "Acme", []string{testRedirect})
	require.NoError(t, err)
	return c.ID
}

func authorizeRequest(clientID string) OAuthRequest {
	return OAuthRequest{
		ClientID: clientID, RedirectURI: testRedirect, ResponseType: "code",
		CodeChallenge: oauthS256(testVerifier), CodeChallengeMethod: "S256", State: "s1",
	}
}

func oauthS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func approvedCode(t *testing.T, s *Service, userID, clientID string) string {
	t.Helper()
	redirect, err := s.DecideOAuth(t.Context(), userID, authorizeRequest(clientID), true)
	require.NoError(t, err)
	u, err := url.Parse(redirect)
	require.NoError(t, err)
	return u.Query().Get("code")
}

func TestOAuth_Off_RefusesEveryStep(t *testing.T) {
	s, _, _ := newPATHarness()
	ctx := t.Context()
	_, err := s.RegisterOAuthClient(ctx, "a", "x", []string{testRedirect})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.OAuthConsent(ctx, OAuthRequest{})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.ExchangeOAuthCode(ctx, "c", "code", "", testVerifier)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.RefreshOAuth(ctx, "c", "depr_x")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	h := NewHandler(s).OAuthRoutes()
	for _, path := range []string{"/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, http.StatusNotFound, rec.Code, path)
	}
}

func TestRegisterOAuthClient_NamesAndLimits(t *testing.T) {
	s, _, _, _ := newOAuthHarness(t)
	ctx := t.Context()

	c, err := s.RegisterOAuthClient(ctx, "a", "   ", []string{testRedirect})
	require.NoError(t, err)
	assert.Equal(t, "MCP client", c.Name, "a client with no name gets a plain one")
	c, err = s.RegisterOAuthClient(ctx, "a", strings.Repeat("é", 150), []string{testRedirect})
	require.NoError(t, err)
	assert.Len(t, []rune(c.Name), 100)

	_, err = s.RegisterOAuthClient(ctx, "a", "x", []string{"https://a.example.com", "https://b.example.com", "https://c.example.com", "https://d.example.com", "https://e.example.com", "https://f.example.com"})
	assert.Equal(t, "invalid_redirect_uri", oauthCode0(err))
	_, err = s.RegisterOAuthClient(ctx, "a", "x", []string{"https://" + strings.Repeat("a", 2000)})
	assert.Equal(t, "invalid_redirect_uri", oauthCode0(err))
	_, err = s.RegisterOAuthClient(ctx, "a", "x", []string{"https:///no-host"})
	assert.Equal(t, "invalid_redirect_uri", oauthCode0(err))
	_, err = s.RegisterOAuthClient(ctx, "a", "x", []string{"http://[::1]:9/cb"})
	require.NoError(t, err, "IPv6 loopback is a loopback")

	for range oauthRegistrations {
		_, _ = s.RegisterOAuthClient(ctx, "10.9.9.9", "x", []string{testRedirect})
	}
	_, err = s.RegisterOAuthClient(ctx, "10.9.9.9", "x", []string{testRedirect})
	require.ErrorIs(t, err, apperrs.ErrRateLimited, "registrations are throttled per address")
	_, err = s.RegisterOAuthClient(ctx, "10.9.9.10", "x", []string{testRedirect})
	require.NoError(t, err, "another address is not")
}

func TestRegisterOAuthClient_SweepsUngrantedClientsADayOld(t *testing.T) {
	s, _, clients, _ := newOAuthHarness(t)
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	s.cfg.Now = func() time.Time { return now }

	mustClient(t, s)
	assert.Equal(t, []time.Time{now.Add(-24 * time.Hour)}, clients.sweptTo)

	clients.sweepErr = errors.New("disk full")
	_, err := s.RegisterOAuthClient(t.Context(), "10.0.0.1", "Acme", []string{testRedirect})
	require.ErrorContains(t, err, "sweep ungranted oauth clients")
}

func oauthCode0(err error) string {
	var oe *OAuthError
	if errors.As(err, &oe) {
		return oe.Code
	}
	return ""
}

func TestOAuthConsent_RefusesWhatItMustNotFollow(t *testing.T) {
	s, _, clients, _ := newOAuthHarness(t)
	clientID := mustClient(t, s)
	ctx := t.Context()

	got, err := s.OAuthConsent(ctx, authorizeRequest(clientID))
	require.NoError(t, err)
	assert.Equal(t, &OAuthConsent{ClientName: "Acme", RedirectURI: testRedirect}, got)
	noRedirect := authorizeRequest(clientID)
	noRedirect.RedirectURI = ""
	got, err = s.OAuthConsent(ctx, noRedirect)
	require.NoError(t, err, "a client that registered one redirect may leave it out")
	assert.Equal(t, testRedirect, got.RedirectURI)

	tests := []struct {
		name  string
		patch func(*OAuthRequest)
	}{
		{"unknown client", func(r *OAuthRequest) { r.ClientID = "nobody" }},
		{"unregistered redirect", func(r *OAuthRequest) { r.RedirectURI = "http://localhost:4000/other" }},
		{"implicit flow", func(r *OAuthRequest) { r.ResponseType = "token" }},
		{"plain PKCE", func(r *OAuthRequest) { r.CodeChallengeMethod = "plain" }},
		{"short challenge", func(r *OAuthRequest) { r.CodeChallenge = "abc" }},
		{"challenge outside base64url", func(r *OAuthRequest) { r.CodeChallenge = strings.Repeat("+", 43) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := authorizeRequest(clientID)
			tt.patch(&req)
			_, err := s.OAuthConsent(ctx, req)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}

	clients.getErr = errors.New("disk on fire")
	_, err = s.OAuthConsent(ctx, authorizeRequest(clientID))
	require.Error(t, err)
	require.NotErrorIs(t, err, apperrs.ErrInvalid)
}

func TestDecideOAuth_ViewingAsSomeoneElse_IsRefused(t *testing.T) {
	s, _, _, userID := newOAuthHarness(t)
	clientID := mustClient(t, s)
	ctx := WithViewedUser(t.Context(), &User{ID: userID}, &identity.ViewerRef{ID: "u-owner"})
	_, err := s.DecideOAuth(ctx, userID, authorizeRequest(clientID), true)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestDecideOAuth_KeepsTheClientsOwnQuery(t *testing.T) {
	s, _, _, userID := newOAuthHarness(t)
	c, err := s.RegisterOAuthClient(t.Context(), "a", "x", []string{"https://agent.example.com/cb?tenant=1"})
	require.NoError(t, err)
	req := authorizeRequest(c.ID)
	req.RedirectURI, req.State = "https://agent.example.com/cb?tenant=1", ""
	redirect, err := s.DecideOAuth(t.Context(), userID, req, false)
	require.NoError(t, err)
	assert.Equal(t, "https://agent.example.com/cb?tenant=1&error=access_denied", redirect)
}

func TestExchangeOAuthCode_ErrorsFirst(t *testing.T) {
	s, users, clients, userID := newOAuthHarness(t)
	clientID := mustClient(t, s)
	ctx := t.Context()

	_, err := s.ExchangeOAuthCode(ctx, clientID, "never-issued", testRedirect, testVerifier)
	assert.Equal(t, "invalid_grant", oauthCode0(err))
	_, err = s.ExchangeOAuthCode(ctx, clientID, approvedCode(t, s, userID, clientID), "http://localhost:4000/other", testVerifier)
	assert.Equal(t, "invalid_grant", oauthCode0(err), "the redirect must repeat the authorize request's")
	_, err = s.ExchangeOAuthCode(ctx, clientID, approvedCode(t, s, userID, clientID), testRedirect, "short")
	assert.Equal(t, "invalid_grant", oauthCode0(err), "a malformed verifier")

	code := approvedCode(t, s, userID, clientID)
	s.cfg.Now = func() time.Time { return time.Now().Add(2 * oauthCodeTTL) }
	_, err = s.ExchangeOAuthCode(ctx, clientID, code, testRedirect, testVerifier)
	assert.Equal(t, "invalid_grant", oauthCode0(err), "a code past its minute")
	s.cfg.Now = time.Now

	code = approvedCode(t, s, userID, clientID)
	delete(clients.byID, clientID)
	_, err = s.ExchangeOAuthCode(ctx, clientID, code, testRedirect, testVerifier)
	assert.Equal(t, "invalid_client", oauthCode0(err))

	clientID = mustClient(t, s)
	code = approvedCode(t, s, userID, clientID)
	require.NoError(t, users.SetAccountStatus(ctx, userID, AccountDisabled))
	_, err = s.ExchangeOAuthCode(ctx, clientID, code, testRedirect, testVerifier)
	assert.Equal(t, "invalid_grant", oauthCode0(err), "a disabled account gets no grant")
}

func TestRefreshOAuth_ErrorsFirst(t *testing.T) {
	s, users, _, userID := newOAuthHarness(t)
	clientID := mustClient(t, s)
	ctx := t.Context()
	tokens, err := s.ExchangeOAuthCode(ctx, clientID, approvedCode(t, s, userID, clientID), testRedirect, testVerifier)
	require.NoError(t, err)

	_, err = s.RefreshOAuth(ctx, clientID, tokens.AccessToken)
	assert.Equal(t, "invalid_grant", oauthCode0(err), "an access token is no refresh token")
	_, err = s.RefreshOAuth(ctx, "another", tokens.RefreshToken)
	assert.Equal(t, "invalid_grant", oauthCode0(err), "another client's refresh")

	require.NoError(t, users.SetAccountStatus(ctx, userID, AccountDisabled))
	_, err = s.RefreshOAuth(ctx, clientID, tokens.RefreshToken)
	assert.Equal(t, "invalid_grant", oauthCode0(err), "a disabled account cannot refresh")
}

func TestOAuthCodes_CapAndSweep(t *testing.T) {
	codes := &oauthCodes{m: map[string]oauthCode{}}
	now := time.Now()
	for range oauthCodeCap {
		_, err := codes.put(now, oauthCode{})
		require.NoError(t, err)
	}
	_, err := codes.put(now, oauthCode{})
	require.ErrorIs(t, err, apperrs.ErrRateLimited)
	_, err = codes.put(now.Add(oauthCodeTTL), oauthCode{})
	require.NoError(t, err, "expired codes are swept to make room")
}

func TestOAuthToken_Endpoint_AnswersInRFC6749Shape(t *testing.T) {
	s, _, _, userID := newOAuthHarness(t)
	clientID := mustClient(t, s)
	h := NewHandler(s).OAuthRoutes()
	post := func(body string, basic string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if basic != "" {
			req.SetBasicAuth(basic, "")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := post("grant_type=authorization_code", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"error":"invalid_request"`)
	rec = post("%zz", "")
	assert.Contains(t, rec.Body.String(), `"error":"invalid_request"`, "a body that is no form")

	form := url.Values{"grant_type": {"authorization_code"}, "code": {approvedCode(t, s, userID, clientID)}, "redirect_uri": {testRedirect}, "code_verifier": {testVerifier}}
	rec = post(form.Encode(), clientID)
	require.Equal(t, http.StatusOK, rec.Code, "client_id may come as HTTP Basic: %s", rec.Body.String())

	delete(s.cfg.OAuthClients.(*fakeOAuthClients).byID, clientID)
	form.Set("code", "x")
	rec = post(form.Encode(), clientID)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(`not json`))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Contains(t, rec.Body.String(), `"error":"invalid_client_metadata"`)

	for range oauthRegistrations {
		_, _ = s.RegisterOAuthClient(t.Context(), "192.0.2.1", "x", []string{testRedirect})
	}
	req = httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(`{"redirect_uris":["`+testRedirect+`"]}`))
	req.RemoteAddr = "192.0.2.1:1234"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "a refusal that is not an OAuth error keeps the gateway's mapping")
}

func TestRequireMCPAuth_NamesTheMetadataOnlyWhenSignInIsOn(t *testing.T) {
	s, _, _ := newPATHarness()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	rec := httptest.NewRecorder()
	s.RequireMCPAuth(next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	assert.Equal(t, `Bearer realm="nexul"`, rec.Header().Get("WWW-Authenticate"))

	s.cfg.OAuthClients = &fakeOAuthClients{byID: map[string]OAuthClient{}}
	rec = httptest.NewRecorder()
	s.RequireMCPAuth(next).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), `resource_metadata="http://example.com/.well-known/oauth-protected-resource/mcp"`)
}

func TestGrant_AuthenticatesOnlyWhereSignInIssuedIt(t *testing.T) {
	s, _, _, userID := newOAuthHarness(t)
	clientID := mustClient(t, s)
	tokens, err := s.ExchangeOAuthCode(t.Context(), clientID, approvedCode(t, s, userID, clientID), testRedirect, testVerifier)
	require.NoError(t, err)
	var grant string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		grant = GrantFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	call := func(h http.Handler, target string, header bool) int {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if header {
			req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusUnauthorized, call(s.RequireAuth(next), "/api/auth/me", true))
	assert.Equal(t, http.StatusUnauthorized, call(s.RequireWS(next), "/ws/events?token="+tokens.AccessToken, false))
	assert.Equal(t, http.StatusOK, call(s.RequireMCPAuth(next), "/mcp", true))
	assert.Equal(t, clientID, grant)
}
