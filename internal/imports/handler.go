package imports

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the import use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the import gateway.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type startRequest struct {
	Source     string `json:"source"`
	ProjectID  string `json:"project_id"`
	From       string `json:"from"`
	Credential string `json:"credential"`
}

// Routes returns the import endpoints, mounted under /api/imports.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/imports", h.list)
	mux.HandleFunc("POST /api/imports", h.start)
	mux.HandleFunc("GET /api/imports/{id}", h.get)
	mux.HandleFunc("POST /api/imports/{id}/confirm", h.confirm)
	mux.HandleFunc("POST /api/imports/{id}/cancel", h.cancel)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	imps, err := h.svc.List(r.Context(), r.URL.Query().Get("workspace_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, imps)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	imp, err := h.svc.Start(r.Context(), Start(req))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, imp)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	imp, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, imp)
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	imp, err := h.svc.Confirm(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, imp)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	imp, err := h.svc.Cancel(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, imp)
}
