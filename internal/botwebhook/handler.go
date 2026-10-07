package botwebhook

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the bot use-cases to the HTTP/JSON gateway (ADR 0019); the public execute route is not here.
type Handler struct {
	svc *Service
}

// NewHandler wires the bot management gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createRequest struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// Routes returns the bot management endpoints, mounted under /api/conversations and /api/botwebhooks.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/conversations/{id}/botwebhooks", h.list)
	mux.HandleFunc("POST /api/conversations/{id}/botwebhooks", h.create)
	mux.HandleFunc("PATCH /api/botwebhooks/{id}", h.update)
	mux.HandleFunc("DELETE /api/botwebhooks/{id}", h.delete)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	bots, err := h.svc.List(r.Context(), r.PathValue("id"), r.URL.Query().Get("deleted") == "true")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bots)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.Create(r.Context(), r.PathValue("id"), req.Name, req.Avatar)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, b)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req Changes
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.Update(r.Context(), r.PathValue("id"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, b)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
