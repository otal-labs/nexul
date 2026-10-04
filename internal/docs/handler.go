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

// saveDocRequest's FolderID places a new doc; empty is the project's default folder.
type saveDocRequest struct {
	ProjectID string `json:"project_id"`
	FolderID  string `json:"folder_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}

// cloneDocRequest's empty ProjectID duplicates the doc in its own project.
type cloneDocRequest struct {
	ProjectID string `json:"project_id"`
}

type namedVersionRequest struct {
	Name string `json:"name"`
}

// Routes returns the docs REST endpoints; literals outrank the {id} wildcard so they don't collide with a lookup.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/docs/import", h.importDoc)
	mux.HandleFunc("POST /api/docs", h.create)
	mux.HandleFunc("GET /api/docs", h.list)
	mux.HandleFunc("GET /api/docs/search", h.search)
	mux.HandleFunc("GET /api/docs/folders", h.listFolders)
	mux.HandleFunc("POST /api/docs/folders", h.createFolder)
	mux.HandleFunc("PUT /api/docs/folders/{id}", h.renameFolder)
	mux.HandleFunc("DELETE /api/docs/folders/{id}", h.deleteFolder)
	mux.HandleFunc("POST /api/docs/{id}/move", h.move)
	mux.HandleFunc("POST /api/docs/{id}/archive", h.archive)
	mux.HandleFunc("POST /api/docs/{id}/restore", h.restore)
	mux.HandleFunc("POST /api/docs/{id}/lock", h.lock)
	mux.HandleFunc("POST /api/docs/{id}/unlock", h.unlock)
	mux.HandleFunc("POST /api/docs/{id}/clone", h.clone)
	mux.HandleFunc("GET /api/docs/{id}/export", h.exportDoc)
	mux.HandleFunc("GET /api/docs/{id}/versions/{version}", h.getVersion)
	mux.HandleFunc("GET /api/docs/{id}/versions", h.listVersions)
	mux.HandleFunc("POST /api/docs/{id}/versions", h.createNamedVersion)
	mux.HandleFunc("GET /api/docs/{id}/watchers", h.listWatchers)
	mux.HandleFunc("PUT /api/docs/{id}/watchers/me", h.watch)
	mux.HandleFunc("DELETE /api/docs/{id}/watchers/me", h.unwatch)
	mux.HandleFunc("GET /api/docs/{id}/clarification", h.getClarification)
	mux.HandleFunc("PUT /api/docs/{id}/clarification/questions/{questionId}", h.answerQuestion)
	mux.HandleFunc("DELETE /api/docs/{id}/clarification/questions/{questionId}", h.clearAnswer)
	mux.HandleFunc("PUT /api/docs/{id}/clarification/rounds/{round}/anything-else", h.saveAnythingElse)
	mux.HandleFunc("POST /api/docs/{id}/clarification/close", h.closeClarification)
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
	d, err := h.svc.CreateInFolder(r.Context(), req.ProjectID, req.FolderID, req.Title, req.Body)
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

func (h *Handler) lock(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Lock(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) unlock(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Unlock(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) clone(w http.ResponseWriter, r *http.Request) {
	var req cloneDocRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.Clone(r.Context(), r.PathValue("id"), req.ProjectID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
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

func (h *Handler) listWatchers(w http.ResponseWriter, r *http.Request) {
	ws, err := h.svc.Watchers(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ws)
}

func (h *Handler) watch(w http.ResponseWriter, r *http.Request) {
	h.setWatching(w, r, true)
}

func (h *Handler) unwatch(w http.ResponseWriter, r *http.Request) {
	h.setWatching(w, r, false)
}

// setWatching answers with the doc's watchers as they now stand, so the caller can redraw the count.
func (h *Handler) setWatching(w http.ResponseWriter, r *http.Request, watching bool) {
	ws, err := h.svc.SetWatching(r.Context(), r.PathValue("id"), watching)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ws)
}

type anythingElseRequest struct {
	Text string `json:"text"`
}

func (h *Handler) getClarification(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Clarification(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) answerQuestion(w http.ResponseWriter, r *http.Request) {
	var req Answer
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	q, err := h.svc.AnswerQuestion(r.Context(), r.PathValue("id"), r.PathValue("questionId"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, q)
}

func (h *Handler) clearAnswer(w http.ResponseWriter, r *http.Request) {
	q, err := h.svc.ClearAnswer(r.Context(), r.PathValue("id"), r.PathValue("questionId"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, q)
}

func (h *Handler) saveAnythingElse(w http.ResponseWriter, r *http.Request) {
	round, err := strconv.Atoi(r.PathValue("round"))
	if err != nil || round < 1 {
		httpx.WriteError(w, fmt.Errorf("%w: round must be a positive integer", apperrs.ErrInvalid))
		return
	}
	var req anythingElseRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	saved, err := h.svc.SaveAnythingElse(r.Context(), r.PathValue("id"), round, req.Text)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, saved)
}

func (h *Handler) closeClarification(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.CloseClarification(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}
