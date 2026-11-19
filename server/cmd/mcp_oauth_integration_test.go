package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

const oauthRedirect = "http://127.0.0.1:33418/callback"

// oauthVerifier is a fixed PKCE verifier; oauthChallenge is its S256 challenge.
var oauthVerifier = strings.Repeat("v", 50)

func oauthChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func send(t *testing.T, h http.Handler, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func registerClient(t *testing.T, h http.Handler, redirect string) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, h, http.MethodPost, "/oauth/register", "application/json",
		`{"client_name":"Globex Agent","redirect_uris":["`+redirect+`"],"token_endpoint_auth_method":"none"}`)
}

func mustRegisterClient(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := registerClient(t, h, oauthRedirect)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return decodeBody[map[string]any](t, rec)["client_id"].(string)
}

func authorizeQuery(clientID string) url.Values {
	return url.Values{
		"client_id": {clientID}, "redirect_uri": {oauthRedirect}, "response_type": {"code"},
		"code_challenge": {oauthChallenge(oauthVerifier)}, "code_challenge_method": {"S256"}, "state": {"xyz"},
	}
}

// approve answers the consent page as the session's person and returns the code the client is sent back with.
func approve(t *testing.T, h http.Handler, session, clientID string) string {
	t.Helper()
	q := authorizeQuery(clientID)
	body, err := json.Marshal(map[string]any{
		"client_id": clientID, "redirect_uri": oauthRedirect, "response_type": "code",
		"code_challenge": q.Get("code_challenge"), "code_challenge_method": "S256", "state": "xyz", "approve": true,
	})
	require.NoError(t, err)
	rec := callAPI(t, h, http.MethodPost, "/api/auth/oauth/authorize", session, string(body))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	back, err := url.Parse(decodeBody[map[string]any](t, rec)["redirect_to"].(string))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:33418", back.Host)
	require.Equal(t, "xyz", back.Query().Get("state"))
	return back.Query().Get("code")
}

func exchange(t *testing.T, h http.Handler, clientID, code, verifier string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code}, "redirect_uri": {oauthRedirect}, "code_verifier": {verifier}}
	return send(t, h, http.MethodPost, "/oauth/token", "application/x-www-form-urlencoded", form.Encode())
}

func refresh(t *testing.T, h http.Handler, clientID, token string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {token}}
	return send(t, h, http.MethodPost, "/oauth/token", "application/x-www-form-urlencoded", form.Encode())
}

// signIn runs the whole sign-in for userID and returns the client and the tokens it was issued.
func signIn(t *testing.T, h http.Handler, svc *coreServices, userID string) (clientID, access, refreshToken string) {
	t.Helper()
	session, err := svc.authSvc.CreateSession(t.Context(), userID)
	require.NoError(t, err)
	clientID = mustRegisterClient(t, h)
	rec := exchange(t, h, clientID, approve(t, h, session, clientID), oauthVerifier)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	tokens := decodeBody[map[string]any](t, rec)
	assert.Equal(t, "Bearer", tokens["token_type"])
	assert.InDelta(t, 3600, tokens["expires_in"], 0)
	return clientID, tokens["access_token"].(string), tokens["refresh_token"].(string)
}

// mcpStatus lists the tools with token, the call a client makes first.
func mcpStatus(t *testing.T, h http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
		`"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2026-07-28")
	req.Header.Set("Mcp-Method", "tools/list")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestMCPOAuth_Discovery_PointsAClientWithNoTokenAtSignIn(t *testing.T) {
	t.Parallel()
	h, _, _ := newRouter(t)

	rec := send(t, h, http.MethodPost, "/mcp", "application/json", `{}`)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, `Bearer realm="nexul", resource_metadata="http://example.com/.well-known/oauth-protected-resource/mcp"`, rec.Header().Get("WWW-Authenticate"))

	for _, path := range []string{"/.well-known/oauth-protected-resource/mcp", "/.well-known/oauth-protected-resource"} {
		rec = send(t, h, http.MethodGet, path, "", "")
		require.Equal(t, http.StatusOK, rec.Code, path)
		prm := decodeBody[map[string]any](t, rec)
		assert.Equal(t, "http://example.com/mcp", prm["resource"], path)
		assert.Equal(t, []any{"http://example.com"}, prm["authorization_servers"], path)
	}

	rec = send(t, h, http.MethodGet, "/.well-known/oauth-authorization-server", "", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	meta := decodeBody[map[string]any](t, rec)
	assert.Equal(t, "http://example.com", meta["issuer"])
	assert.Equal(t, "http://example.com/oauth/authorize", meta["authorization_endpoint"])
	assert.Equal(t, "http://example.com/oauth/token", meta["token_endpoint"])
	assert.Equal(t, "http://example.com/oauth/register", meta["registration_endpoint"])
	assert.Equal(t, []any{"S256"}, meta["code_challenge_methods_supported"])
	assert.Equal(t, []any{"authorization_code", "refresh_token"}, meta["grant_types_supported"])

	rec = send(t, h, http.MethodOptions, "/oauth/token", "", "")
	assert.Equal(t, http.StatusNoContent, rec.Code, "a browser client's preflight passes")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Content-Type")

	assert.Equal(t, http.StatusNotFound, send(t, h, http.MethodGet, "/.well-known/openid-configuration", "", "").Code)
	rec = callAPI(t, h, http.MethodPost, "/api/auth/me", "", "")
	assert.Equal(t, `Bearer realm="nexul"`, rec.Header().Get("WWW-Authenticate"), "the gateway's own 401 names no MCP metadata")
}

func TestMCPOAuth_Register_RefusesARedirectABrowserWouldRunOrSendInTheClear(t *testing.T) {
	t.Parallel()
	h, _, _ := newRouter(t)
	for _, redirect := range []string{"javascript:alert(1)", "data:text/html,hi", "http://attacker.example.com/cb", "https://example.com/cb#frag", "/relative"} {
		rec := registerClient(t, h, redirect)
		assert.Equal(t, http.StatusBadRequest, rec.Code, redirect)
		assert.Equal(t, "invalid_redirect_uri", decodeBody[map[string]any](t, rec)["error"], redirect)
	}
	for _, redirect := range []string{"https://agent.example.com/api/mcp/auth_callback", "http://localhost:6274/oauth/callback", "com.example.agent://oauth/callback"} {
		assert.Equal(t, http.StatusCreated, registerClient(t, h, redirect).Code, redirect)
	}
	rec := send(t, h, http.MethodPost, "/oauth/register", "application/json", `{"redirect_uris":[]}`)
	assert.Equal(t, "invalid_redirect_uri", decodeBody[map[string]any](t, rec)["error"])
}

func TestMCPOAuth_SignIn_ListsToolsUntilTheGrantIsRevoked(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	userID, pat := signedInOwner(t, store, svc)
	session, err := svc.authSvc.CreateSession(t.Context(), userID)
	require.NoError(t, err)
	clientID := mustRegisterClient(t, h)

	rec := callAPI(t, h, http.MethodGet, "/api/auth/oauth/authorize?"+authorizeQuery(clientID).Encode(), session, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, map[string]any{"client_name": "Globex Agent", "redirect_uri": oauthRedirect}, decodeBody[map[string]any](t, rec))

	rec = exchange(t, h, clientID, approve(t, h, session, clientID), oauthVerifier)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	tokens := decodeBody[map[string]any](t, rec)
	access := tokens["access_token"].(string)

	rec = mcpStatus(t, h, access)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"ticket_get"`)
	assert.Contains(t, callToolResult(t, h, access, "account_get", `{}`), `\"login\":\"alice\"`)
	require.Equal(t, http.StatusOK, mcpStatus(t, h, pat).Code, "a personal access token still connects")

	rec = refresh(t, h, clientID, tokens["refresh_token"].(string))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rotated := decodeBody[map[string]any](t, rec)
	assert.Equal(t, http.StatusUnauthorized, mcpStatus(t, h, access).Code, "a refresh retires the access token it replaced")
	require.Equal(t, http.StatusOK, mcpStatus(t, h, rotated["access_token"].(string)).Code)
	assert.Equal(t, "invalid_grant", decodeBody[map[string]any](t, refresh(t, h, clientID, tokens["refresh_token"].(string)))["error"], "a refresh token works once")

	grant := grantOf(t, store, userID, clientID)
	assert.Equal(t, "Globex Agent", grant.Name)
	rec = callAPI(t, h, http.MethodDelete, "/api/auth/tokens/"+grant.ID, session, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = mcpStatus(t, h, rotated["access_token"].(string))
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a revoked grant's next call fails")
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "resource_metadata=")
	assert.Equal(t, "invalid_grant", decodeBody[map[string]any](t, refresh(t, h, clientID, rotated["refresh_token"].(string)))["error"], "nor can it refresh")
	assert.Equal(t, http.StatusOK, mcpStatus(t, h, pat).Code)
}

func grantOf(t *testing.T, store *storage.Store, userID, clientID string) auth.PersonalAccessToken {
	t.Helper()
	tokens, err := store.PATs.ListByUser(t.Context(), userID)
	require.NoError(t, err)
	for _, tok := range tokens {
		if tok.OAuthClientID == clientID {
			return tok
		}
	}
	require.FailNow(t, "no grant for the client")
	return auth.PersonalAccessToken{}
}

func TestMCPOAuth_Exchange_RefusesAWrongVerifierAReusedCodeAndAnotherClient(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	userID, _ := signedInOwner(t, store, svc)
	session, err := svc.authSvc.CreateSession(t.Context(), userID)
	require.NoError(t, err)
	clientID := mustRegisterClient(t, h)

	code := approve(t, h, session, clientID)
	rec := exchange(t, h, clientID, code, strings.Repeat("w", 50))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_grant", decodeBody[map[string]any](t, rec)["error"], "a wrong PKCE verifier")
	assert.Equal(t, "invalid_grant", decodeBody[map[string]any](t, exchange(t, h, clientID, code, oauthVerifier))["error"], "the code was spent by the failed try")

	other := mustRegisterClient(t, h)
	assert.Equal(t, "invalid_grant", decodeBody[map[string]any](t, exchange(t, h, other, approve(t, h, session, clientID), oauthVerifier))["error"], "a code works only for its client")

	code = approve(t, h, session, clientID)
	require.Equal(t, http.StatusOK, exchange(t, h, clientID, code, oauthVerifier).Code)
	assert.Equal(t, "invalid_grant", decodeBody[map[string]any](t, exchange(t, h, clientID, code, oauthVerifier))["error"], "a code works once")

	q := authorizeQuery(clientID)
	q.Set("code_challenge_method", "plain")
	assert.Equal(t, http.StatusBadRequest, callAPI(t, h, http.MethodGet, "/api/auth/oauth/authorize?"+q.Encode(), session, "").Code, "plain PKCE is refused")
	q = authorizeQuery(clientID)
	q.Set("redirect_uri", "http://127.0.0.1:9/elsewhere")
	assert.Equal(t, http.StatusBadRequest, callAPI(t, h, http.MethodGet, "/api/auth/oauth/authorize?"+q.Encode(), session, "").Code, "an unregistered redirect is refused")
	assert.Equal(t, "unsupported_grant_type", decodeBody[map[string]any](t, send(t, h, http.MethodPost, "/oauth/token", "application/x-www-form-urlencoded", "grant_type=password&client_id="+clientID))["error"])
}

func TestMCPOAuth_Decide_NeedsASignedInBrowserAndADenialSendsAccessDenied(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	userID, pat := signedInOwner(t, store, svc)
	clientID := mustRegisterClient(t, h)
	body := `{"client_id":"` + clientID + `","redirect_uri":"` + oauthRedirect + `","response_type":"code","code_challenge":"` + oauthChallenge(oauthVerifier) + `","code_challenge_method":"S256","state":"xyz","approve":true}`
	assert.Equal(t, http.StatusForbidden, callAPI(t, h, http.MethodPost, "/api/auth/oauth/authorize", pat, body).Code, "a token cannot approve a grant")

	session, err := svc.authSvc.CreateSession(t.Context(), userID)
	require.NoError(t, err)
	rec := callAPI(t, h, http.MethodPost, "/api/auth/oauth/authorize", session, strings.Replace(body, `"approve":true`, `"approve":false`, 1))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, oauthRedirect+"?error=access_denied&state=xyz", decodeBody[map[string]any](t, rec)["redirect_to"])
}

func TestMCPOAuth_ExpiredAccessToken_Gets401AndTheRefreshStillWorks(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	h, store, svc := routerOver(t, db)
	userID, _ := signedInOwner(t, store, svc)
	clientID, access, refreshToken := signIn(t, h, svc, userID)

	_, err := db.ExecContext(t.Context(), `UPDATE personal_access_tokens SET expires_at = ? WHERE oauth_client_id = ?`, time.Now().Add(-time.Second).Unix(), clientID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, mcpStatus(t, h, access).Code)

	rec := refresh(t, h, clientID, refreshToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, http.StatusOK, mcpStatus(t, h, decodeBody[map[string]any](t, rec)["access_token"].(string)).Code)
}

func TestMCPOAuth_Grant_HoldsNoMoreThanItsPersonsPermissions(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	bobPAT := signedInReader(t, store, svc)
	_, access, _ := signIn(t, h, svc, "u-bob")

	viaPAT := callToolResult(t, h, bobPAT, "instance_get", `{}`)
	assert.Contains(t, viaPAT, "instance:read", "bob's role lacks instance:read")
	assert.Equal(t, viaPAT, callToolResult(t, h, access, "instance_get", `{}`), "the grant is refused exactly as his own token is")
}

func TestMCPOAuth_Grant_WorksOnlyOnMCPAndMintsNoCredentialThatOutlivesIt(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	userID, pat := signedInOwner(t, store, svc)
	_, access, _ := signIn(t, h, svc, userID)

	assert.Equal(t, http.StatusUnauthorized, callAPI(t, h, http.MethodPost, "/api/auth/tokens", access, `{"name":"escape"}`).Code, "the gateway refuses a grant")
	assert.Equal(t, http.StatusUnauthorized, callAPI(t, h, http.MethodGet, "/api/auth/me", access, "").Code)
	for _, tool := range []string{"computer_mcp_token_create", "computer_create", "automation_token_create", "automation_create"} {
		assert.Contains(t, callToolResult(t, h, access, tool, `{}`), "A signed-in app cannot create credentials", tool)
	}
	assert.NotContains(t, callToolResult(t, h, pat, "computer_mcp_token_create", `{}`), "signed-in app", "a pasted token still may")
	assert.Equal(t, http.StatusOK, mcpStatus(t, h, access).Code)
}
