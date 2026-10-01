package automations

import (
	"context"
	"net/http"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

type ctxKey string

const automationCtxKey ctxKey = "automation_auth"

// AutomationAuth is the resolved credential for an automation-token request, attributed by the audit middleware.
type AutomationAuth struct {
	Automation *Automation
}

// AutomationFromCtx returns the automation auth for this request, or nil when
// the request was authenticated some other way (session, PAT, integration).
func AutomationFromCtx(ctx context.Context) *AutomationAuth {
	auth, _ := ctx.Value(automationCtxKey).(*AutomationAuth)
	return auth
}

// ScopeGate enforces the shared /api scope vocabulary; injected to avoid a cross-import (ADR 0017).
type ScopeGate func(method, path string, scopes []string) bool

// ScopeResolver expands writes to their implied reads, catching a typo'd scope at mint time, not the gate.
type ScopeResolver func(scopes []string) ([]string, error)

// SetGateway wires the gateway's scope enforcement once integrations exists.
func (s *Service) SetGateway(scopeAllows ScopeGate, resolveScopes ScopeResolver) {
	s.scopeAllows = scopeAllows
	s.resolveScopes = resolveScopes
}

// RequireAutomation is the third gateway auth mechanism: a dat_ token acts as its creator, gated by its scopes.
func (s *Service) RequireAutomation(userAuth func(http.Handler) http.Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		token = strings.TrimSpace(token)
		if !strings.HasPrefix(token, tokenPrefix) {
			userAuth(next).ServeHTTP(w, r)
			return
		}
		a, err := s.AuthenticateToken(r.Context(), token)
		if err != nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		if !isSelfRead(r, a.ID) && (s.scopeAllows == nil || !s.scopeAllows(r.Method, r.URL.Path, a.Scopes)) {
			httpx.WriteError(w, apperrs.ErrForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), automationCtxKey, &AutomationAuth{Automation: a})
		ctx = identity.WithActor(ctx, identity.Actor{
			ID:         a.CreatedBy,
			Automation: &identity.AutomationRef{ID: a.ID, Name: a.Name, WorkspaceID: a.WorkspaceID},
		})
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// isSelfRead reports a GET/HEAD of the automation's own subtree; reading yourself is identity, not a grant.
func isSelfRead(r *http.Request, automationID string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	self := "/api/automations/" + automationID
	return r.URL.Path == self || strings.HasPrefix(r.URL.Path, self+"/")
}
