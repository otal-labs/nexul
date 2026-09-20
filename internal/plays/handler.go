package plays

import (
	"fmt"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts the plays use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the plays REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type playRequest struct {
	Label              string   `json:"label"`
	Type               Type     `json:"type"`
	Description        string   `json:"description"`
	Instructions       string   `json:"instructions"`
	Enabled            bool     `json:"enabled"`
	ShowWhenStage      *Stage   `json:"show_when_stage"`
	ExcludedProjectIDs []string `json:"excluded_project_ids"`
}

// Routes are mounted under the tenancy domain's /api/workspaces prefix.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/plays", h.list)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/plays/applicable", h.listApplicable)
	mux.HandleFunc("POST /api/workspaces/{workspaceID}/plays", h.create)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/plays/{playID}", h.get)
	mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/plays/{playID}", h.update)
	mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/plays/{playID}", h.delete)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), r.PathValue("workspaceID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// listApplicable serves a ticket or doc's Plays section: the run buttons the caller may see and fire.
func (h *Handler) listApplicable(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	typeStr := q.Get("type")
	if typeStr == "" {
		httpx.WriteError(w, fmt.Errorf("%w: type is required", apperrs.ErrInvalid))
		return
	}
	var stage *Stage
	if raw := q.Get("stage"); raw != "" {
		s := Stage(raw)
		stage = &s
	}
	list, err := h.svc.ListApplicable(r.Context(), r.PathValue("workspaceID"), actorIDFromRequest(r), q.Get("project_id"), Type(typeStr), stage)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req playRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.Create(r.Context(), r.PathValue("workspaceID"), createInputFrom(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("workspaceID"), r.PathValue("playID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req playRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.Update(r.Context(), r.PathValue("workspaceID"), r.PathValue("playID"), updateInputFrom(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("workspaceID"), r.PathValue("playID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func actorIDFromRequest(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}

func createInputFrom(req playRequest) CreateInput {
	return CreateInput(req)
}

func updateInputFrom(req playRequest) UpdateInput {
	return UpdateInput{
		Label: req.Label, Description: req.Description, Instructions: req.Instructions,
		Enabled: req.Enabled, ShowWhenStage: req.ShowWhenStage, ExcludedProjectIDs: req.ExcludedProjectIDs,
	}
}
