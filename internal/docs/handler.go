package docs

import (
	"fmt"
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the docs use-cases to the HTTP/JSON gateway (ADR 0019); the browser never talks to MCP directly.
type Handler struct {
	svc *Service
}

// NewHandler wires the docs REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type saveDocRequest struct {
	ProjectID string `json:"project_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}

type namedVersionRequest struct {
	Name string `json:"name"`
}

// Routes returns the docs REST endpoints; literals outrank the {id} wildcard so they don't collide with a lookup.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/docs/import", h.importDoc)
	mux.HandleFunc("POST /api/docs", h.create)
	mux.HandleFunc("GET /api/docs", h.list)
	mux.HandleFunc("GET /api/docs/search", h.search)
	mux.HandleFunc("POST /api/docs/{id}/archive", h.archive)
	mux.HandleFunc("POST /api/docs/{id}/restore", h.restore)
	mux.HandleFunc("GET /api/docs/{id}/export", h.exportDoc)
	mux.HandleFunc("GET /api/docs/{id}/versions/{version}", h.getVersion)
	mux.HandleFunc("GET /api/docs/{id}/versions", h.listVersions)
	mux.HandleFunc("POST /api/docs/{id}/versions", h.createNamedVersion)
	mux.HandleFunc("GET /api/docs/{id}", h.get)
	mux.HandleFunc("PUT /api/docs/{id}", h.update)
	mux.HandleFunc("DELETE /api/docs/{id}", h.delete)
	return mux
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req saveDocRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.Create(r.Context(), req.ProjectID, req.Title, req.Body)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
}

// importDoc creates a doc from a markdown body.
func (h *Handler) importDoc(w http.ResponseWriter, r *http.Request) {
	var req saveDocRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.ImportMarkdown(r.Context(), req.ProjectID, req.Title, req.Body)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		ds, err := h.svc.ListByProject(r.Context(), projectID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ds)
		return
	}
	ds, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ds)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := h.svc.Search(r.Context(), q, limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, results)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

// exportDoc returns the doc's body as markdown.
func (h *Handler) exportDoc(w http.ResponseWriter, r *http.Request) {
	md, err := h.svc.ExportMarkdown(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"markdown": md})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req saveDocRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.Update(r.Context(), r.PathValue("id"), req.Title, req.Body)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Archive(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) restore(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Restore(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	vs, err := h.svc.ListVersions(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, vs)
}

// createNamedVersion snapshots the doc as a milestone version, named by the caller and attributed to them.
func (h *Handler) createNamedVersion(w http.ResponseWriter, r *http.Request) {
	var req namedVersionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	v, err := h.svc.CreateNamedVersion(r.Context(), r.PathValue("id"), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, v)
}

func (h *Handler) getVersion(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 {
		httpx.WriteError(w, fmt.Errorf("%w: version must be a positive integer", apperrs.ErrInvalid))
		return
	}
	v, err := h.svc.GetVersion(r.Context(), r.PathValue("id"), version)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}
