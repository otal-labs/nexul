package voice

import (
	"context"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// ctxKey namespaces this package's context keys (each domain defines its own consumer-side key, ADR 0017 seam rule).
type ctxKey string

const userIDKey ctxKey = "user_id"

// WithUserID attaches the authenticated user id so the voice gateway can attribute a join without importing auth (ADR 0017).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromCtx returns the user id injected by WithUserID, or "".
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

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
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/voice/{conversationID}/token", h.join)
	mux.HandleFunc("POST /api/voice/{conversationID}/leave", h.leave)
	mux.HandleFunc("GET /api/voice/occupancy", h.occupancy)
	return mux
}

func (h *Handler) leave(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Leave(r.Context(), r.PathValue("conversationID"), UserIDFromCtx(r.Context())); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	tok, err := h.svc.Join(r.Context(), r.PathValue("conversationID"), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tok)
}

func (h *Handler) occupancy(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.svc.Occupancy())
}
