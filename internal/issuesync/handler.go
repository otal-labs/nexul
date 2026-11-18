package issuesync

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the sync to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the sync's routes over svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// setRequest is a PUT's body; an omitted field keeps its value.
type setRequest struct {
	Owner   string  `json:"owner"`
	Repo    string  `json:"repo"`
	Label   *string `json:"label,omitempty"`
	Mode    *Mode   `json:"mode,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// Routes returns the project's issue sync endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/projects/{id}/issue-sync", h.list)
	mux.HandleFunc("PUT /api/projects/{id}/issue-sync", h.set)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	syncs, err := h.svc.Syncs(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, syncs)
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	var req setRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	sy, err := h.svc.Set(r.Context(), r.PathValue("id"), Change(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sy)
}
