package integrations

import (
	"context"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// ActorResolver returns the audit identity: actorType, actorID, tokenID (ADR 0017).
type ActorResolver func(ctx context.Context) (actorType, actorID, tokenID string)

// AuditLog wraps /api, recording one row per request so user and integration calls are both attributed (best-effort).
func (s *Service) AuditLog(resolve ActorResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		actorType, actorID, tokenID := resolve(r.Context())
		if actorID == "" {
			return
		}
		entry := AuditEntry{
			ID:        ids.New(),
			ActorType: actorType,
			ActorID:   actorID,
			TokenID:   tokenID,
			Action:    r.Method + " " + r.URL.Path,
			CreatedAt: s.cfg.Now().UTC(),
		}
		if err := s.cfg.Audit.Append(r.Context(), entry); err != nil {
			logging.FromCtx(r.Context()).Error("audit append failed", "action", entry.Action, "err", err)
		}
	})
}
