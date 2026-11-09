package integrations

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// AuditRetention is how long an audit row, a dedupe record, a published outbox row and a deploy log line are kept (ADR 0138); a constant because the install takes no settings.
const AuditRetention = 45 * 24 * time.Hour

// auditPurgeBatch bounds one retention delete, so the purge never holds the single writer for long.
const auditPurgeBatch = 500

const auditRetentionInterval = 24 * time.Hour

// ActorResolver returns the audit identity: actorType, actorID, tokenID (ADR 0017).
type ActorResolver func(ctx context.Context) (actorType, actorID, tokenID string)

// readOnlyRoutes are the non-GET routes that only read; every other non-GET route changes state (ADR 0138).
var readOnlyRoutes = []string{
	"POST /api/auth/bootstrap/verify",
	"POST /api/auth/passkeys/login/start",
	"POST /api/auth/passkeys/start",
	"POST /api/automation-hosts/self/logs",
	"POST /api/connectors/{id}/manual/verify",
	"POST /api/connectors/{id}/private-key/verify",
	"POST /api/dns/tunnels/{tunnelID}/verify",
	"POST /api/dns/verify",
	"POST /api/invitations/preview",
	"POST /api/logs",
	"POST /api/machines/{id}/discover",
	"POST /api/mentions/resolve",
	"POST /api/repositories/scan",
	"POST /api/tickets/dev-status",
	"POST /api/tickets/labels/colors",
	"POST /api/voice/{conversationID}/listen",
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

// PurgeAudit deletes the rows older than AuditRetention, one bounded batch per write; unchecked, since it runs from
// a background loop, not a request.
func (s *Service) PurgeAudit(ctx context.Context) (int64, error) {
	return purgeInBatches(ctx, "audit rows", s.cfg.Now().Add(-AuditRetention), s.cfg.Audit.DeleteBefore)
}

// PurgeEvents deletes the dedupe records, published outbox rows and deploy log lines older than AuditRetention, in
// the same bounded batches; the outbox keeps what an enabled automation has not read.
func (s *Service) PurgeEvents(ctx context.Context) (int64, error) {
	if s.cfg.Retention == nil {
		return 0, nil
	}
	before := s.cfg.Now().Add(-AuditRetention)
	var total int64
	for _, step := range []struct {
		what string
		del  func(context.Context, time.Time, int) (int64, error)
	}{
		{"processed events", s.cfg.Retention.DeleteProcessedBefore},
		{"outbox rows", s.cfg.Retention.DeleteOutboxBefore},
		{"deploy log lines", s.cfg.Retention.DeleteDeployLogBefore},
	} {
		n, err := purgeInBatches(ctx, step.what, before, step.del)
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func purgeInBatches(ctx context.Context, what string, before time.Time, del func(context.Context, time.Time, int) (int64, error)) (int64, error) {
	var total int64
	for {
		n, err := del(ctx, before, auditPurgeBatch)
		total += n
		if err != nil {
			return total, fmt.Errorf("purge %s before %s: %w", what, before, err)
		}
		if n < auditPurgeBatch {
			return total, nil
		}
	}
}

// RunAuditRetention purges at start and daily after that, until ctx is cancelled.
func (s *Service) RunAuditRetention(ctx context.Context, log *slog.Logger) {
	for {
		n, err := s.PurgeAudit(ctx)
		events, eventsErr := s.PurgeEvents(ctx)
		n, err = n+events, errors.Join(err, eventsErr)
		if err != nil {
			log.Warn("audit retention purge failed", "deleted", n, "error", err)
		}
		if err == nil && n > 0 {
			log.Info("audit retention purge", "deleted", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(auditRetentionInterval):
		}
	}
}
