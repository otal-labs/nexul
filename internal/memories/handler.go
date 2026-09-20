package memories

import (
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the memories use-cases to the HTTP/JSON gateway (ADR 0019); the browser never talks to MCP directly.
type Handler struct {
	svc *Service
}

// NewHandler wires the memories REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type saveMemoryRequest struct {
	// ProjectID empty makes the memory workspace-scoped (ADR 0059); WorkspaceID is then required and ignored otherwise.
	ProjectID      string `json:"project_id"`
	WorkspaceID    string `json:"workspace_id"`
	Title          string `json:"title"`
	WhenToUse      string `json:"when_to_use"`
	Body           string `json:"body"`
	AlwaysIncluded bool   `json:"always_included"`
}

type cloneMemoryRequest struct {
	// ProjectID empty clones to the workspace (ADR 0059); WorkspaceID is then required and ignored otherwise.
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id"`
}

// Routes returns the memories REST endpoints. Browser calls never carry MCP provenance, so every use-case
// call here passes an empty via.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/memories", h.create)
	mux.HandleFunc("GET /api/memories", h.list)
	mux.HandleFunc("GET /api/memories/{id}", h.get)
	mux.HandleFunc("PUT /api/memories/{id}", h.update)
	mux.HandleFunc("DELETE /api/memories/{id}", h.delete)
	mux.HandleFunc("GET /api/memories/{id}/versions", h.listVersions)
	mux.HandleFunc("GET /api/memories/{id}/versions/{version}", h.getVersion)
	mux.HandleFunc("POST /api/memories/{id}/revert", h.revert)
	mux.HandleFunc("POST /api/memories/{id}/clone", h.clone)
	return mux
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req saveMemoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.Create(r.Context(), req.ProjectID, req.WorkspaceID, req.Title, req.WhenToUse, req.Body, req.AlwaysIncluded, "")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, m)
}

// list branches on project_id (one project) or workspace_id (the whole page, grouped by project client-side).
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		ms, err := h.svc.ListForProject(r.Context(), projectID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ms)
		return
	}
	ms, err := h.svc.List(r.Context(), r.URL.Query().Get("workspace_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ms)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req saveMemoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.Update(r.Context(), r.PathValue("id"), req.Title, req.WhenToUse, req.Body, req.AlwaysIncluded, "")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	vs, err := h.svc.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, vs)
}

func (h *Handler) getVersion(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil {
		httpx.WriteError(w, apperrs.ErrInvalid)
		return
	}
	v, err := h.svc.GetVersion(r.Context(), r.PathValue("id"), version)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

type revertMemoryRequest struct {
	Version int `json:"version"`
}

func (h *Handler) revert(w http.ResponseWriter, r *http.Request) {
	var req revertMemoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.Revert(r.Context(), r.PathValue("id"), req.Version, "")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

func (h *Handler) clone(w http.ResponseWriter, r *http.Request) {
	var req cloneMemoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.Clone(r.Context(), r.PathValue("id"), req.ProjectID, req.WorkspaceID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, m)
}
