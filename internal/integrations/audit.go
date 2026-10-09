package integrations

import (
	"context"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// ActorResolver returns the audit identity: actorType, actorID, tokenID (ADR 0017).
type ActorResolver func(ctx context.Context) (actorType, actorID, tokenID string)

// readOnlyRoutes are the non-GET routes that only read; every other non-GET route changes state (ADR 0138).
var readOnlyRoutes = []string{
	"POST /api/auth/bootstrap/verify",
	"POST /api/connectors/{id}/manual/verify",
	"POST /api/dns/tunnels/{tunnelID}/verify",
	"POST /api/dns/verify",
	"POST /api/invitations/preview",
	"POST /api/logs",
	"POST /api/machines/{id}/discover",
	"POST /api/mentions/resolve",
	"POST /api/repositories/scan",
	"POST /api/tickets/dev-status",
	"POST /api/tickets/labels/colors",
}

// readOnlyMux matches readOnlyRoutes with the router's own pattern rules.
var readOnlyMux = func() *http.ServeMux {
	mux := http.NewServeMux()
	for _, route := range readOnlyRoutes {
		mux.Handle(route, http.NotFoundHandler())
	}
	return mux
}()

// Audited reports whether a request changes state and so is recorded: GET, HEAD, OPTIONS and readOnlyRoutes are not.
func Audited(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	_, pattern := readOnlyMux.Handler(r)
	return pattern == ""
}

// AuditLog wraps /api, recording one row per request that changes state, attributed to the user, automation, or
// integration that made it (best-effort).
func (s *Service) AuditLog(resolve ActorResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if Audited(r) {
			s.record(r.Context(), resolve, r.Method+" "+r.URL.Path)
		}
	})
}

// AuditTool is the MCP adapter's hook: one row per call to a tool that changes state, attributed as AuditLog does.
func (s *Service) AuditTool(resolve ActorResolver) func(ctx context.Context, tool string) {
	return func(ctx context.Context, tool string) {
		s.record(ctx, resolve, "mcp "+tool)
	}
}

func (s *Service) record(ctx context.Context, resolve ActorResolver, action string) {
	actorType, actorID, tokenID := resolve(ctx)
	if actorID == "" {
		return
	}
	entry := AuditEntry{
		ID:        ids.New(),
		ActorType: actorType,
		ActorID:   actorID,
		TokenID:   tokenID,
		Action:    action,
		CreatedAt: s.cfg.Now().UTC(),
	}
	if err := s.cfg.Audit.Append(ctx, entry); err != nil {
		logging.FromCtx(ctx).Error("audit append failed", "action", entry.Action, "err", err)
	}
}
