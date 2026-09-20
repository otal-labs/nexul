package tenancy

import (
	"context"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

// WithUserID lets tenancy attribute workspace actions without importing auth (ADR 0017).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromCtx returns the user id injected by WithUserID, or "".
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// Handler adapts tenancy use-cases to HTTP/JSON, since the browser never talks to MCP.
type Handler struct {
	svc *Service
}

// NewHandler wires the tenancy REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

// Routes are wrapped with the auth-user-id injection adapter before mounting behind RequireAuth.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workspaces", h.list)
	mux.HandleFunc("POST /api/workspaces", h.create)
	mux.HandleFunc("PATCH /api/workspaces/{workspaceID}", h.rename)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/me", h.me)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/members", h.listMembers)
	mux.HandleFunc("POST /api/workspaces/{workspaceID}/members", h.inviteMember)
	mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/members/{userID}", h.removeMember)
	mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/members/{userID}", h.changeMemberRole)
	mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/invites/{login}", h.cancelInvite)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	workspaces, err := h.svc.ListForUser(r.Context(), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, workspaces)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	ws, err := h.svc.Create(r.Context(), UserIDFromCtx(r.Context()), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, ws)
}

// rename is instance-admin only: the owner wizard's finish step and workspace settings land here.
func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	var req createWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	ws, err := h.svc.Rename(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ws)
}

// meResponse is the /me wire shape the frontend's hasPermission(action) helper reads directly.
type meResponse struct {
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
}

// me is re-fetched whenever useWorkspaceStore's selectedWorkspaceId changes.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspaceID")
	userID := UserIDFromCtx(r.Context())
	roleName, err := h.svc.MemberRoleName(r.Context(), workspaceID, userID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	permissions := h.svc.MemberPermissions(r.Context(), workspaceID, userID)
	httpx.WriteJSON(w, http.StatusOK, meResponse{RoleName: roleName, Permissions: permissions})
}

type inviteMemberRequest struct {
	Login  string `json:"login"`
	RoleID string `json:"role_id"`
}

type changeMemberRoleRequest struct {
	RoleID string `json:"role_id"`
}

// listMembers requires members:write.
func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListWorkspaceMembers(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// inviteMember requires an already-allowlisted login; the Owner role can't be assigned this way.
func (h *Handler) inviteMember(w http.ResponseWriter, r *http.Request) {
	var req inviteMemberRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.InviteMember(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), req.Login, req.RoleID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	err := h.svc.RemoveMember(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), r.PathValue("userID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) cancelInvite(w http.ResponseWriter, r *http.Request) {
	err := h.svc.CancelInvite(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), r.PathValue("login"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changeMemberRole(w http.ResponseWriter, r *http.Request) {
	var req changeMemberRoleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	err := h.svc.ChangeMemberRole(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), r.PathValue("userID"), req.RoleID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
