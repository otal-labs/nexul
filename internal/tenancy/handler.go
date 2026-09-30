package tenancy

import (
	"context"
	"net/http"
	"strconv"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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

type renameWorkspaceRequest struct {
	// Name and Slug are left alone when omitted.
	Name *string `json:"name,omitempty"`
	Slug *string `json:"slug,omitempty"`
}

// Routes are wrapped with the auth-user-id injection adapter before mounting behind RequireAuth.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/workspaces", h.list)
	mux.HandleFunc("POST /api/workspaces", h.create)
	mux.HandleFunc("PATCH /api/workspaces/{workspaceID}", h.rename)
	mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/mention-chip-template", h.setMentionChipTemplate)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/me", h.me)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/members", h.listMembers)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/people", h.listPeople)
	mux.HandleFunc("POST /api/workspaces/{workspaceID}/members", h.inviteMember)
	mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/members/{userID}", h.removeMember)
	mux.HandleFunc("PUT /api/workspaces/{workspaceID}/members/{userID}", h.addMember)
	mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/members/{userID}", h.changeMemberRole)
	mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/invites/{login}", h.cancelInvite)
	return mux
}

// TeamRoutes serves the instance-wide Team read; mounted at /api/team behind the same auth-user-id adapter.
func (h *Handler) TeamRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/team", h.team)
	return mux
}

// PeopleRoutes serves uploaded pictures; mounted at /api/people behind the same auth-user-id adapter.
func (h *Handler) PeopleRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/people/{userID}/avatar", h.avatar)
	return mux
}

// listPeople answers any member of the workspace, since seeing who you work with needs no permission.
func (h *Handler) listPeople(w http.ResponseWriter, r *http.Request) {
	people, err := h.svc.ListPeople(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, People{People: people})
}

// avatar sets nosniff so an uploaded picture can never be re-interpreted as HTML on this origin.
func (h *Handler) avatar(w http.ResponseWriter, r *http.Request) {
	contentType, data, err := h.svc.Avatar(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("userID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// The URL carries a content hash, so a changed picture is a new URL and this copy never goes stale.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // the status is already sent; a failed write only means the client went away
}

// team is scoped by the use-case: everything for a holder of accounts:read, else only the workspaces the caller manages.
func (h *Handler) team(w http.ResponseWriter, r *http.Request) {
	team, err := h.svc.ListTeam(r.Context(), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, team)
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

// rename needs workspaces:write in the workspace: the owner wizard's finish step and workspace settings land here.
func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	var req renameWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	ws, err := h.svc.Rename(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), req.Name, req.Slug)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ws)
}

type setMentionChipTemplateRequest struct {
	MentionChipTemplate string `json:"mention_chip_template"`
}

// setMentionChipTemplate requires workspaces:write in that workspace; any member reads it off the workspace list.
func (h *Handler) setMentionChipTemplate(w http.ResponseWriter, r *http.Request) {
	var req setMentionChipTemplateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	ws, err := h.svc.SetMentionChipTemplate(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), req.MentionChipTemplate)
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

type addMemberRequest struct {
	RoleID string `json:"role_id"`
}

// changeMemberRoleRequest is a patch: an omitted field keeps its value, and allow or deny replace that set.
type changeMemberRoleRequest struct {
	RoleID string           `json:"role_id"`
	Allow  *permissions.Set `json:"allow"`
	Deny   *permissions.Set `json:"deny"`
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

// addMember puts an existing account into the workspace; members:write there, never the Owner role.
func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	var req addMemberRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	err := h.svc.AddMember(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), r.PathValue("userID"), req.RoleID)
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
	ctx, actorID, workspaceID, userID := r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), r.PathValue("userID")
	overrides := req.Allow != nil || req.Deny != nil
	if req.RoleID != "" || !overrides {
		if err := h.svc.ChangeMemberRole(ctx, actorID, workspaceID, userID, req.RoleID); err != nil {
			httpx.WriteError(w, err)
			return
		}
	}
	if overrides {
		if err := h.svc.SetMemberOverrides(ctx, actorID, workspaceID, userID, req.Allow, req.Deny); err != nil {
			httpx.WriteError(w, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
