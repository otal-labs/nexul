package auditlog

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the audit log settings to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the audit log gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes are the Owner's settings: read them, save them, remove them.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/instance/audit", h.get)
	mux.HandleFunc("PUT /api/instance/audit", h.update)
	mux.HandleFunc("DELETE /api/instance/audit", h.remove)
	mux.HandleFunc("POST /api/instance/audit/stream/test", h.test)
	mux.HandleFunc("POST /api/instance/audit/stream/retry", h.retry)
	mux.HandleFunc("POST /api/instance/audit/stream/rotate-secret", h.rotate)
	return mux
}

// test sends one signed sample to the form's destination, or the saved one, and stores nothing.
func (h *Handler) test(w http.ResponseWriter, r *http.Request) {
	var p Patch
	if err := httpx.DecodeJSON(r, &p); err != nil {
		httpx.WriteError(w, err)
		return
	}
	res, err := h.svc.Test(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.RetryFailed(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) rotate(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.RotateSecret(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.View(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var p Patch
	if err := httpx.DecodeJSON(r, &p); err != nil {
		httpx.WriteError(w, err)
		return
	}
	v, err := h.svc.Update(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Remove(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}
