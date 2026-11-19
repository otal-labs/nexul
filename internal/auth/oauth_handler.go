package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

const oauthBodyMax = 64 << 10

// OAuthPaths are where OAuthRoutes answers, each mounted outside /api because MCP clients look for them there.
var OAuthPaths = []string{
	"/.well-known/oauth-authorization-server",
	"/.well-known/oauth-protected-resource",
	"/.well-known/oauth-protected-resource/mcp",
	"/oauth/register",
	"/oauth/token",
}

// OAuthRoutes serves MCP sign-in's public half: the discovery documents, client registration and the token endpoint.
func (h *Handler) OAuthRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", h.authorizationServerMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", h.protectedResourceMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource/mcp", h.protectedResourceMetadata)
	mux.HandleFunc("POST /oauth/register", h.registerOAuthClient)
	mux.HandleFunc("POST /oauth/token", h.oauthToken)
	return openCORS(mux)
}

// openCORS lets a browser-based MCP client discover, register and fetch tokens; none of these read a cookie.
func openCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, MCP-Protocol-Version")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) authorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	if !h.svc.OAuthEnabled() {
		http.NotFound(w, r)
		return
	}
	issuer := h.apiOrigin(r)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                h.spaOrigin(r) + "/oauth/authorize",
		"token_endpoint":                        issuer + "/oauth/token",
		"registration_endpoint":                 issuer + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

func (h *Handler) protectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if !h.svc.OAuthEnabled() {
		http.NotFound(w, r)
		return
	}
	origin := h.apiOrigin(r)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"resource":                 origin + "/mcp",
		"authorization_servers":    []string{origin},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Nexul",
	})
}

type registerClientRequest struct {
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

func (h *Handler) registerOAuthClient(w http.ResponseWriter, r *http.Request) {
	var req registerClientRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, oauthBodyMax)).Decode(&req); err != nil {
		writeOAuthError(w, oauthErr("invalid_client_metadata", "the body is not a JSON client registration"))
		return
	}
	client, err := h.svc.RegisterOAuthClient(r.Context(), httpx.ClientAddr(r), req.ClientName, req.RedirectURIs)
	if err != nil {
		writeOAuthError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ID,
		"client_id_issued_at":        client.CreatedAt.Unix(),
		"client_name":                client.Name,
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
}

func (h *Handler) oauthToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, oauthBodyMax)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, oauthErr("invalid_request", "the body is not a form"))
		return
	}
	clientID := r.PostForm.Get("client_id")
	if user, _, ok := r.BasicAuth(); ok && clientID == "" {
		clientID = user
	}
	if clientID == "" {
		writeOAuthError(w, oauthErr("invalid_request", "client_id is required"))
		return
	}
	tokens, err := h.tokenGrant(r, clientID)
	if err != nil {
		writeOAuthError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tokens)
}

func (h *Handler) tokenGrant(r *http.Request, clientID string) (*OAuthTokens, error) {
	form := r.PostForm
	switch form.Get("grant_type") {
	case "authorization_code":
		return h.svc.ExchangeOAuthCode(r.Context(), clientID, form.Get("code"), form.Get("redirect_uri"), form.Get("code_verifier"))
	case "refresh_token":
		return h.svc.RefreshOAuth(r.Context(), clientID, form.Get("refresh_token"))
	}
	return nil, oauthErr("unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
}

// writeOAuthError answers in RFC 6749's shape; anything that is not an OAuth error keeps the gateway's own mapping.
func writeOAuthError(w http.ResponseWriter, err error) {
	var oe *OAuthError
	if !errors.As(err, &oe) {
		httpx.WriteError(w, err)
		return
	}
	status := http.StatusBadRequest
	if oe.Code == "invalid_client" {
		status = http.StatusUnauthorized
	}
	httpx.WriteJSON(w, status, map[string]string{"error": oe.Code, "error_description": oe.Description})
}

func oauthRequestFromQuery(r *http.Request) OAuthRequest {
	q := r.URL.Query()
	return OAuthRequest{
		ClientID: q.Get("client_id"), RedirectURI: q.Get("redirect_uri"), ResponseType: q.Get("response_type"),
		CodeChallenge: q.Get("code_challenge"), CodeChallengeMethod: q.Get("code_challenge_method"), State: q.Get("state"),
	}
}

// oauthConsent tells the consent page who is asking; it is a GET so the page can show it before anyone decides.
func (h *Handler) oauthConsent(w http.ResponseWriter, r *http.Request) {
	consent, err := h.svc.OAuthConsent(r.Context(), oauthRequestFromQuery(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, consent)
}

type decideOAuthRequest struct {
	OAuthRequest
	Approve bool `json:"approve"`
}

// decideOAuth takes the consent page's answer; only a signed-in browser may give it, never a token.
func (h *Handler) decideOAuth(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req decideOAuthRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	redirect, err := h.svc.DecideOAuth(r.Context(), currentUserID(r), req.OAuthRequest, req.Approve)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"redirect_to": redirect})
}
