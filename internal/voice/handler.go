package voice

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts voice's use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the voice REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns voice's authed REST endpoints (mounted under /api/voice).
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/voice/{conversationID}/token", h.join)
	mux.HandleFunc("POST /api/voice/{conversationID}/listen", h.listen)
	mux.HandleFunc("POST /api/voice/{conversationID}/leave", h.leave)
	mux.HandleFunc("GET /api/voice/occupancy", h.occupancy)
	return mux
}

func (h *Handler) leave(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Leave(r.Context(), r.PathValue("conversationID"), identity.UserID(r.Context())); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	tok, err := h.svc.Join(r.Context(), r.PathValue("conversationID"), identity.UserID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tok)
}

func (h *Handler) listen(w http.ResponseWriter, r *http.Request) {
	tok, err := h.svc.Listen(r.Context(), r.PathValue("conversationID"), identity.UserID(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tok)
}

func (h *Handler) occupancy(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.svc.Occupancy(r.Context()))
}
