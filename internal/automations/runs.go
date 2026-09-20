package automations

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// RunOutcome is the terminal state of one delivery attempt.
type RunOutcome string

const (
	RunOutcomeSuccess RunOutcome = "success"
	RunOutcomeFailure RunOutcome = "failure"
	RunOutcomeCrash   RunOutcome = "crash"
)

// maxRunLogBytes caps captured run logs at 1MB.
const maxRunLogBytes = 1 << 20

// Run is one delivery attempt's report; a redelivered crash makes a second Run row for the same EventID.
type Run struct {
	ID           string
	AutomationID string
	EventTopic   string
	EventID      string
	Outcome      RunOutcome
	Error        string
	StartedAt    time.Time
	FinishedAt   time.Time
	DurationMS   int64
	Logs         string
	CreatedAt    time.Time
}

// RunsRepo persists run reports; implementation lives in internal/platform/storage.
type RunsRepo interface {
	Create(ctx context.Context, r *Run) error
	Get(ctx context.Context, id string) (*Run, error)
	ListByAutomation(ctx context.Context, automationID string, limit int) ([]Run, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// appendCappedLog appends chunk to buf, discarding anything beyond maxRunLogBytes.
func appendCappedLog(buf, chunk string) string {
	if len(buf) >= maxRunLogBytes {
		return buf
	}
	room := maxRunLogBytes - len(buf)
	if len(chunk) > room {
		chunk = chunk[:room]
	}
	return buf + chunk
}

// RunsService is the read-side use-case for run history, kept separate from the CRUD gateway.
type RunsService struct {
	repo RunsRepo
	perm PermissionGate
}

// NewRunsService wires the run-history use-cases over the given repo and
// permission gate.
func NewRunsService(repo RunsRepo, perm PermissionGate) *RunsService {
	return &RunsService{repo: repo, perm: perm}
}

// ListByAutomation returns the automation's run history, newest first.
func (s *RunsService) ListByAutomation(ctx context.Context, actorID, automationID string, limit int) ([]Run, error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	runs, err := s.repo.ListByAutomation(ctx, automationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list runs for automation %s: %w", automationID, err)
	}
	return runs, nil
}

// Get returns one run's full report; it 404s on a mismatched automation, so run ids can't be probed by guessing.
func (s *RunsService) Get(ctx context.Context, actorID, automationID, runID string) (*Run, error) {
	if err := requirePermission(ctx, s.perm, actorID, permissions.AutomationsRead); err != nil {
		return nil, err
	}
	run, err := s.repo.Get(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get run %s: %w", runID, err)
	}
	if run.AutomationID != automationID {
		return nil, fmt.Errorf("get run %s: %w", runID, apperrs.ErrNotFound)
	}
	return run, nil
}

// Cleanup deletes rows older than retention; no permission check, it runs from a background loop, not a request.
func (s *RunsService) Cleanup(ctx context.Context, retention time.Duration) (int64, error) {
	n, err := s.repo.DeleteOlderThan(ctx, time.Now().Add(-retention))
	if err != nil {
		return 0, fmt.Errorf("cleanup automation runs: %w", err)
	}
	return n, nil
}

// RunsHandler adapts run history to the HTTP/JSON gateway (ADR 0019).
type RunsHandler struct {
	svc *RunsService
}

// NewRunsHandler wires the run-history REST gateway over the given service.
func NewRunsHandler(svc *RunsService) *RunsHandler {
	return &RunsHandler{svc: svc}
}

const defaultRunsLimit = 50

// Routes returns the run-history REST endpoints, both gated on automations:read.
func (h *RunsHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/automations/{id}/runs", h.list)
	mux.HandleFunc("GET /api/automations/{id}/runs/{runID}", h.get)
	return mux
}

func (h *RunsHandler) list(w http.ResponseWriter, r *http.Request) {
	limit := defaultRunsLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	runs, err := h.svc.ListByAutomation(r.Context(), actorID(r), r.PathValue("id"), limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, runs)
}

func (h *RunsHandler) get(w http.ResponseWriter, r *http.Request) {
	run, err := h.svc.Get(r.Context(), actorID(r), r.PathValue("id"), r.PathValue("runID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, run)
}

// RunCleanupLoop deletes run rows older than retention on interval until ctx is cancelled, the retention pass.
func RunCleanupLoop(ctx context.Context, svc *RunsService, retention, interval time.Duration, log *slog.Logger) {
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := svc.Cleanup(ctx, retention)
			if err != nil {
				log.Warn("automation run retention cleanup failed", "error", err)
				continue
			}
			if n > 0 {
				log.Info("automation run retention cleanup", "deleted", n)
			}
		}
	}
}
