package chat

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// ctxKey namespaces this package's context keys (each domain defines its own consumer-side key, ADR 0017 seam rule).
type ctxKey string

const userIDKey ctxKey = "user_id"

// WithUserID lets chat attribute posts/edits/reads without importing auth (ADR 0017).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromCtx returns the user id injected by WithUserID, or "".
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// Handler adapts the chat use-cases to the HTTP/JSON gateway (ADR 0019); the browser never talks to MCP directly.
type Handler struct {
	svc *Service
}

// NewHandler wires the chat REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createChannelRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
}

type createDMRequest struct {
	WorkspaceID    string   `json:"workspace_id"`
	ParticipantIDs []string `json:"participant_ids"`
}

type ticketThreadRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type docThreadRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type postMessageRequest struct {
	Body string `json:"body"`
}

// Routes returns the chat REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/chat/conversations", h.listConversations)
	mux.HandleFunc("POST /api/chat/channels", h.createChannel)
	mux.HandleFunc("POST /api/chat/voice-channels", h.createVoiceChannel)
	mux.HandleFunc("POST /api/chat/dms", h.createDM)
	mux.HandleFunc("POST /api/chat/tickets/{ticketID}/thread", h.getOrCreateTicketThread)
	mux.HandleFunc("GET /api/chat/tickets/thread-status", h.hasTicketThreads)
	mux.HandleFunc("POST /api/chat/docs/{docID}/thread", h.getOrCreateDocThread)
	mux.HandleFunc("GET /api/chat/conversations/{id}/messages", h.listMessages)
	mux.HandleFunc("POST /api/chat/conversations/{id}/messages", h.postMessage)
	mux.HandleFunc("POST /api/chat/conversations/{id}/read", h.markRead)
	mux.HandleFunc("GET /api/chat/unread", h.unreadCounts)
	mux.HandleFunc("PATCH /api/chat/messages/{id}", h.editMessage)
	mux.HandleFunc("DELETE /api/chat/messages/{id}", h.deleteMessage)
	return mux
}

func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	cs, err := h.svc.ListConversations(r.Context(), r.URL.Query().Get("workspace_id"), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cs)
}

func (h *Handler) createChannel(w http.ResponseWriter, r *http.Request) {
	var req createChannelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.CreateChannel(r.Context(), req.WorkspaceID, UserIDFromCtx(r.Context()), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) createVoiceChannel(w http.ResponseWriter, r *http.Request) {
	var req createChannelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.CreateVoiceChannel(r.Context(), req.WorkspaceID, UserIDFromCtx(r.Context()), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) createDM(w http.ResponseWriter, r *http.Request) {
	var req createDMRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.CreateDM(r.Context(), req.WorkspaceID, UserIDFromCtx(r.Context()), req.ParticipantIDs)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) getOrCreateTicketThread(w http.ResponseWriter, r *http.Request) {
	var req ticketThreadRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.GetOrCreateTicketThread(r.Context(), req.WorkspaceID, r.PathValue("ticketID"), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) getOrCreateDocThread(w http.ResponseWriter, r *http.Request) {
	var req docThreadRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.GetOrCreateDocThread(r.Context(), req.WorkspaceID, r.PathValue("docID"), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) hasTicketThreads(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("ticket_ids")
	var ids []string
	if raw != "" {
		ids = strings.Split(raw, ",")
	}
	out, err := h.svc.HasTicketThreads(r.Context(), ids)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	ms, err := h.svc.ListMessages(r.Context(), r.PathValue("id"), UserIDFromCtx(r.Context()), limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ms)
}

func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {
	var req postMessageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.PostMessage(r.Context(), r.PathValue("id"), UserIDFromCtx(r.Context()), req.Body)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, m)
}

func (h *Handler) editMessage(w http.ResponseWriter, r *http.Request) {
	var req postMessageRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.EditMessage(r.Context(), r.PathValue("id"), UserIDFromCtx(r.Context()), req.Body)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteMessage(r.Context(), r.PathValue("id"), UserIDFromCtx(r.Context())); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkRead(r.Context(), r.PathValue("id"), UserIDFromCtx(r.Context())); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) unreadCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.svc.UnreadCounts(r.Context(), r.URL.Query().Get("workspace_id"), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, counts)
}
