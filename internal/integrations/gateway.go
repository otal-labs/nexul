package integrations

import (
	"context"
	"net/http"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

type ctxKey string

const integrationCtxKey ctxKey = "integration_auth"

// IntegrationAuth is the resolved credential for an integration-token request, attributed by the audit middleware.
type IntegrationAuth struct {
	Install *Install
	Token   *IntegrationToken
}

// IntegrationFromCtx returns the integration auth for this request, or nil when
// the request was authenticated as a user (session/PAT).
func IntegrationFromCtx(ctx context.Context) *IntegrationAuth {
	auth, _ := ctx.Value(integrationCtxKey).(*IntegrationAuth)
	return auth
}

// RequireIntegration is the second gateway auth mechanism (ADR 0043): a token acts as its installer, gated to its scopes.
func (s *Service) RequireIntegration(userAuth func(http.Handler) http.Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		token = strings.TrimSpace(token)
		if !strings.HasPrefix(token, tokenPrefix) {
			userAuth(next).ServeHTTP(w, r)
			return
		}
		install, integrationToken, err := s.AuthenticateToken(r.Context(), token)
		if err != nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		if !scopeAllows(r.Method, r.URL.Path, install.Scopes) {
			httpx.WriteError(w, apperrs.ErrForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), integrationCtxKey, &IntegrationAuth{Install: install, Token: integrationToken})
		ctx = identity.WithActor(ctx, identity.Actor{ID: install.CreatedBy, CanCreateWorkspace: true})
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// pathAliases routes a mounted path prefix to the scope domain that actually governs it: one-release
// naming migrations (services), sibling handlers served by the same domain's settings surface, and
// sub-surfaces (agent, automation-secrets) whose parent domain owns the grant.
var pathAliases = map[string]string{
	"services":           "stacks",
	"categories":         "projects",
	"ticket-types":       "projects",
	"statuses":           "projects",
	"agent":              "chat",
	"automation-secrets": "automations",
}

// exactDenylist blocks a path regardless of scopes; runners/install returns the shared runner secret,
// which a scoped token must never read no matter what it's been granted.
var exactDenylist = map[string]bool{
	http.MethodGet + " /api/runners/install": true,
}

// requiredAction maps an HTTP method to the action it needs: GET/HEAD read, DELETE delete, everything else write.
func requiredAction(method string) action {
	switch method {
	case http.MethodGet, http.MethodHead:
		return actionRead
	case http.MethodDelete:
		return actionDelete
	default:
		return actionWrite
	}
}

// scopeAllows enforces the domain x action scope grid (ADR 0043) against a request method + path. It is
// generic: no per-domain switch, since the required scope is just "<domain>:<action>".
func scopeAllows(method, path string, scopes []Scope) bool {
	path = strings.TrimSuffix(path, "/")
	if exactDenylist[method+" "+path] {
		return false
	}
	domain := gatewayPrefix(path)
	if domain == "" {
		return false
	}
	required := Scope(domain + ":" + string(requiredAction(method)))
	if !allScopes[required] {
		return false
	}
	return HasScopes(scopes, required)
}

// ScopeAllows exports the method+path -> scope vocabulary so other domains can gate /api without importing this (ADR 0017).
func ScopeAllows(method, path string, scopes []string) bool {
	scoped := make([]Scope, len(scopes))
	for i, sc := range scopes {
		scoped[i] = Scope(sc)
	}
	return scopeAllows(method, path, scoped)
}

// gatewayPrefix maps an /api path to its scope domain, following pathAliases; an unknown or
// identity-plane domain (auth, pairing) yields a domain string that never appears in allScopes,
// so scopeAllows denies it.
func gatewayPrefix(path string) string {
	// Defense in depth in case a future router doesn't canonicalize paths the way net/http.ServeMux does.
	if strings.Contains(path, "..") || strings.Contains(path, "//") {
		return ""
	}
	if !strings.HasPrefix(path, "/api/") {
		return ""
	}
	rest := strings.TrimPrefix(path, "/api/")
	first, _, _ := strings.Cut(rest, "/")
	if alias, ok := pathAliases[first]; ok {
		return alias
	}
	return first
}
