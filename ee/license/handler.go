package license

import (
	"errors"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the license use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the license REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the license endpoints: read it, paste a key, remove it.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/instance/license", h.get)
	mux.HandleFunc("PUT /api/instance/license", h.set)
	mux.HandleFunc("DELETE /api/instance/license", h.remove)
	return mux
}

type setRequest struct {
	Key string `json:"key"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Status(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	var req setRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.Set(r.Context(), req.Key)
	if errors.Is(err, apperrs.ErrInvalid) {
		httpx.WriteFieldError(w, err, "key")
		return
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Remove(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}
