package plays

import (
	"net/http"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// RunHandler adapts the run pipeline to the HTTP/JSON gateway (ADR 0019); mounted under /api/plays.
type RunHandler struct {
	runner *Runner
}

// NewRunHandler wires the run gateway over the given runner.
func NewRunHandler(r *Runner) *RunHandler {
	return &RunHandler{runner: r}
}

type runRequest struct {
	TargetType         TargetType `json:"target_type"`
	TargetID           string     `json:"target_id"`
	MemoryIDs          []string   `json:"memory_ids"`
	CustomInstructions string     `json:"custom_instructions"`
	MoveToStatusID     string     `json:"move_to_status_id"`
	ComputerID         string     `json:"computer_id"`
	Provider           string     `json:"provider"`
	Model              string     `json:"model"`
}

// Routes returns the run endpoints. Latest choices take the play as a query parameter because a
// `{id}/latest-choices` pattern would conflict with `runs/{trailID}` in the mux.
func (h *RunHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/plays/{id}/run", h.run)
	mux.HandleFunc("GET /api/plays/runs", h.list)
	mux.HandleFunc("GET /api/plays/runs/active", h.active)
	mux.HandleFunc("GET /api/plays/runs/{trailID}", h.get)
	mux.HandleFunc("POST /api/plays/runs/{trailID}/stop", h.stop)
	mux.HandleFunc("POST /api/plays/runs/{trailID}/answer", h.answer)
	mux.HandleFunc("GET /api/plays/latest-choices", h.latestChoices)
	return mux
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

func (h *RunHandler) run(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	trail, err := h.runner.Run(r.Context(), RunInput{
		PlayID: r.PathValue("id"), TargetType: req.TargetType, TargetID: req.TargetID, MemoryIDs: req.MemoryIDs,
		CustomInstructions: req.CustomInstructions, MoveToStatusID: req.MoveToStatusID,
		ComputerID: req.ComputerID, Provider: req.Provider, Model: req.Model, Via: ViaWeb,
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
	list, err := h.runner.ListTrails(r.Context(), TargetType(q.Get("target_type")), q.Get("target_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// active answers the board's one batched question per project: which of these tickets has a run going.
func (h *RunHandler) active(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if raw := r.URL.Query().Get("ticket_ids"); raw != "" {
		ids = strings.Split(raw, ",")
	}
	active, err := h.runner.ActiveTrails(r.Context(), TargetTicket, ids)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"active": active})
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
