package repository

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the repository use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the repository REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the repository REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/repositories/scan", h.scan)
	mux.HandleFunc("GET /api/repositories", h.list)
	mux.HandleFunc("GET /api/repositories/installations", h.installations)
	mux.HandleFunc("GET /api/repositories/install-url", h.installURL)
	mux.HandleFunc("PUT /api/repositories/installations/{account}/workspaces/{workspaceID}", h.assign)
	mux.HandleFunc("DELETE /api/repositories/installations/{account}/workspaces/{workspaceID}", h.unassign)
	return mux
}

type scanRequest struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Ref   string `json:"ref"`
}

func (h *Handler) scan(w http.ResponseWriter, r *http.Request) {
	var req scanRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	result, err := h.svc.Scan(r.Context(), req.Owner, req.Name, req.Ref)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	repos, err := h.svc.ListRepos(r.Context(), q.Get("workspace_id"), q.Get("q"), q.Get("refresh") == "1")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"repositories": repos})
}

func (h *Handler) installations(w http.ResponseWriter, r *http.Request) {
	installs, err := h.svc.ListInstallations(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"installations": installs})
}

func (h *Handler) installURL(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.InstallURL(r.Context(), r.URL.Query().Get("workspace_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"url": u})
}

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.AssignInstallation(r.Context(), r.PathValue("account"), r.PathValue("workspaceID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) unassign(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.UnassignInstallation(r.Context(), r.PathValue("account"), r.PathValue("workspaceID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
