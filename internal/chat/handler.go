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
	WorkspaceID string   `json:"workspace_id"`
	Name        string   `json:"name"`
	Private     bool     `json:"private"`
	MemberIDs   []string `json:"member_ids"`
}

type setPrivateRequest struct {
	Private   bool     `json:"private"`
	MemberIDs []string `json:"member_ids"`
}

type addMembersRequest struct {
	UserIDs []string `json:"user_ids"`
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

type renameConversationRequest struct {
	Name string `json:"name"`
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
	mux.HandleFunc("POST /api/chat/projects/{projectID}/interview-thread", h.getOrCreateInterviewThread)
	mux.HandleFunc("PATCH /api/chat/conversations/{id}", h.renameConversation)
	mux.HandleFunc("DELETE /api/chat/conversations/{id}", h.deleteConversation)
	mux.HandleFunc("PUT /api/chat/conversations/{id}/private", h.setPrivate)
	mux.HandleFunc("POST /api/chat/conversations/{id}/members", h.addMembers)
	mux.HandleFunc("DELETE /api/chat/conversations/{id}/members/{userID}", h.removeMember)
	mux.HandleFunc("POST /api/chat/conversations/{id}/leave", h.leave)
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
	h.create(w, r, KindChannel, h.svc.CreateChannel)
}

func (h *Handler) createVoiceChannel(w http.ResponseWriter, r *http.Request) {
	h.create(w, r, KindVoiceChannel, h.svc.CreateVoiceChannel)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, kind Kind, public func(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error)) {
	var req createChannelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.createAs(r.Context(), req, kind, public)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) createAs(ctx context.Context, req createChannelRequest, kind Kind, public func(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error)) (*Conversation, error) {
	creator := UserIDFromCtx(ctx)
	if req.Private {
		return h.svc.CreatePrivateChannel(ctx, req.WorkspaceID, creator, req.Name, kind, req.MemberIDs)
	}
	return public(ctx, req.WorkspaceID, creator, req.Name)
}

func (h *Handler) setPrivate(w http.ResponseWriter, r *http.Request) {
	var req setPrivateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.writeConversation(w)(h.svc.SetChannelPrivate(r.Context(), r.PathValue("id"), req.Private, req.MemberIDs))
}

func (h *Handler) addMembers(w http.ResponseWriter, r *http.Request) {
	var req addMembersRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	h.writeConversation(w)(h.svc.AddChannelMembers(r.Context(), r.PathValue("id"), req.UserIDs))
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	h.writeConversation(w)(h.svc.RemoveChannelMembers(r.Context(), r.PathValue("id"), []string{r.PathValue("userID")}))
}

func (h *Handler) leave(w http.ResponseWriter, r *http.Request) {
	h.writeConversation(w)(h.svc.RemoveChannelMembers(r.Context(), r.PathValue("id"), []string{UserIDFromCtx(r.Context())}))
}

func (h *Handler) writeConversation(w http.ResponseWriter) func(*Conversation, error) {
	return func(c *Conversation, err error) {
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, c)
	}
}

func (h *Handler) renameConversation(w http.ResponseWriter, r *http.Request) {
	var req renameConversationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.RenameChannel(r.Context(), r.PathValue("id"), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteConversation(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.DeleteChannel(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) getOrCreateInterviewThread(w http.ResponseWriter, r *http.Request) {
	var req ticketThreadRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.GetOrCreateInterviewThread(r.Context(), req.WorkspaceID, r.PathValue("projectID"), UserIDFromCtx(r.Context()))
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
