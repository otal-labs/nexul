package plays

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// RunHandler adapts the run pipeline to the HTTP/JSON gateway (ADR 0019); mounted under /api/plays.
type RunHandler struct {
	runner  *Runner
	targets ProjectTargets
}

// ProjectTargets lists the tickets or docs of a project the caller may read, so a board asks about its runs by project.
type ProjectTargets interface {
	TargetIDs(ctx context.Context, targetType TargetType, projectID string) ([]string, error)
}

// NewRunHandler wires the run gateway over the given runner and the project lookup the active-runs question takes.
func NewRunHandler(r *Runner, targets ProjectTargets) *RunHandler {
	return &RunHandler{runner: r, targets: targets}
}

type runRequest struct {
	TargetType         TargetType              `json:"target_type"`
	TargetID           string                  `json:"target_id"`
	MemoryIDs          []string                `json:"memory_ids"`
	CustomInstructions string                  `json:"custom_instructions"`
	ComputerID         string                  `json:"computer_id"`
	Provider           string                  `json:"provider"`
	Model              string                  `json:"model"`
	ModelOptions       []harness.OptionSetting `json:"model_options"`
}

// Routes returns the run endpoints. Latest choices take the play as a query parameter because a
// `{id}/latest-choices` pattern would conflict with `runs/{trailID}` in the mux.
func (h *RunHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/plays/{id}/run", h.run)
	mux.HandleFunc("GET /api/plays/runs", h.list)
	mux.HandleFunc("GET /api/plays/runs/active", h.active)
	mux.HandleFunc("GET /api/plays/runs/{trailID}", h.get)
	mux.HandleFunc("POST /api/plays/runs/{trailID}/stop", h.stop)
	mux.HandleFunc("POST /api/plays/runs/{trailID}/answer", h.answer)
	mux.HandleFunc("POST /api/plays/runs/{trailID}/continue", h.continueRun)
	mux.HandleFunc("GET /api/plays/latest-choices", h.latestChoices)
	mux.HandleFunc("POST /api/plays/decisions-check", h.retryDecisionsCheck)
	mux.HandleFunc("GET /api/plays/queue", h.queue)
	mux.HandleFunc("GET /api/plays/queued", h.queued)
	mux.HandleFunc("POST /api/plays/queue/resume", h.resumeQueue)
	mux.HandleFunc("POST /api/plays/queue/{itemID}/cancel", h.cancelQueued)
	return mux
}

// queue answers a ticket's or doc's auto runs: queued, started, skipped, didn't run, cancelled, and whether it is paused.
func (h *RunHandler) queue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	queue, err := h.runner.GetQueue(r.Context(), TargetType(q.Get("target_type")), q.Get("target_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, queue)
}

// queued answers what waits in the queue for one play, for its settings page.
func (h *RunHandler) queued(w http.ResponseWriter, r *http.Request) {
	items, err := h.runner.QueuedForPlay(r.Context(), r.URL.Query().Get("play_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, items)
}

type targetRequest struct {
	TargetType TargetType `json:"target_type"`
	TargetID   string     `json:"target_id"`
}

func (h *RunHandler) resumeQueue(w http.ResponseWriter, r *http.Request) {
	var req targetRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	queue, err := h.runner.ResumeAutoPlays(r.Context(), req.TargetType, req.TargetID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, queue)
}

func (h *RunHandler) cancelQueued(w http.ResponseWriter, r *http.Request) {
	item, err := h.runner.CancelQueued(r.Context(), r.PathValue("itemID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, item)
}

type decisionsCheckRequest struct {
	TicketID string `json:"ticket_id"`
}

func (h *RunHandler) retryDecisionsCheck(w http.ResponseWriter, r *http.Request) {
	var req decisionsCheckRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	trail, err := h.runner.RetryDecisionsCheck(r.Context(), req.TicketID, ViaWeb)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, trail)
}

func (h *RunHandler) stop(w http.ResponseWriter, r *http.Request) {
	trail, err := h.runner.Stop(r.Context(), r.PathValue("trailID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, trail)
}

type answerRequest struct {
	Answers map[string]harness.AnswerValue `json:"answers"`
}

func (h *RunHandler) answer(w http.ResponseWriter, r *http.Request) {
	var req answerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	trail, err := h.runner.Answer(r.Context(), r.PathValue("trailID"), harness.QuestionAnswer{Answers: req.Answers})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, trail)
}

type continueRequest struct {
	Message string `json:"message"`
}

// continueRun answers with the trail the message went to: the same one, or a new run when its thread was gone.
func (h *RunHandler) continueRun(w http.ResponseWriter, r *http.Request) {
	var req continueRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	trail, err := h.runner.Continue(r.Context(), r.PathValue("trailID"), req.Message, ViaWeb)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, trail)
}

func (h *RunHandler) run(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	trail, err := h.runner.Run(r.Context(), RunInput{
		PlayID: r.PathValue("id"), TargetType: req.TargetType, TargetID: req.TargetID, MemoryIDs: req.MemoryIDs,
		CustomInstructions: req.CustomInstructions,
		ComputerID:         req.ComputerID, Provider: req.Provider, Model: req.Model, ModelOptions: req.ModelOptions, Via: ViaWeb,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, trail)
}

func (h *RunHandler) get(w http.ResponseWriter, r *http.Request) {
	trail, err := h.runner.GetTrail(r.Context(), r.PathValue("trailID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, trail)
}

func (h *RunHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list, err := h.runner.ListTrails(r.Context(), TargetType(q.Get("target_type")), q.Get("target_id"), q.Get("play_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// active answers the one batched question per project: which of these targets has a run going, when it started, and
// which of those runs waits on an answer. project_id names every target in the project; target_ids, and ticket_ids
// alone as target_type=ticket, stay accepted (ADR 0082: the HTTP API only grows).
func (h *RunHandler) active(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	targetType, raw := TargetType(q.Get("target_type")), q.Get("target_ids")
	if targetType == "" && q.Has("ticket_ids") {
		targetType, raw = TargetTicket, q.Get("ticket_ids")
	}
	var ids []string
	if raw != "" {
		ids = strings.Split(raw, ",")
	}
	if projectID := q.Get("project_id"); projectID != "" {
		listed, err := h.targets.TargetIDs(r.Context(), targetType, projectID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		ids = append(ids, listed...)
	}
	trails, err := h.runner.ActiveTrails(r.Context(), targetType, ids)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	active, waiting, started := map[string]string{}, map[string]string{}, map[string]time.Time{}
	for target, t := range trails {
		active[target] = t.ID
		started[target] = t.StartedAt
		if t.State == TrailWaiting {
			waiting[target] = t.ID
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"active": active, "waiting": waiting, "started": started})
}

func (h *RunHandler) latestChoices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	choices, err := h.runner.LatestChoices(r.Context(), actorID(r.Context()), q.Get("play_id"), q.Get("project_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, choices)
}
