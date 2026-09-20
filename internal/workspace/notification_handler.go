package workspace

import (
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// NotificationHandler is mounted behind RequireAuth by the composition root.
type NotificationHandler struct {
	svc         *NotificationService
	currentUser func(*http.Request) string
}

// NewNotificationHandler's currentUser extracts the authenticated user's ID from the request context.
func NewNotificationHandler(svc *NotificationService, currentUser func(*http.Request) string) *NotificationHandler {
	return &NotificationHandler{svc: svc, currentUser: currentUser}
}

// Routes returns the notifications REST endpoints.
func (h *NotificationHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/notifications", h.list)
	mux.HandleFunc("GET /api/notifications/unread-count", h.unreadCount)
	mux.HandleFunc("POST /api/notifications/{id}/read", h.markRead)
	mux.HandleFunc("POST /api/notifications/read-all", h.markAllRead)
	return mux
}

func (h *NotificationHandler) list(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUser(r)
	if userID == "" {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	ns, err := h.svc.List(r.Context(), userID, limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ns)
}

func (h *NotificationHandler) unreadCount(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUser(r)
	if userID == "" {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
		return
	}
	n, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"count": n})
}

func (h *NotificationHandler) markRead(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUser(r)
	if userID == "" {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
		return
	}
	if err := h.svc.MarkRead(r.Context(), userID, r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NotificationHandler) markAllRead(w http.ResponseWriter, r *http.Request) {
	userID := h.currentUser(r)
	if userID == "" {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
		return
	}
	if err := h.svc.MarkAllRead(r.Context(), userID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
