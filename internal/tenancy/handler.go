package tenancy

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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

// ProjectPeopleRoutes serves who may open a project; registered as an exact pattern under /api/projects.
func (h *Handler) ProjectPeopleRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/projects/{projectID}/people", h.listProjectPeople)
	return mux
}

// listProjectPeople answers anyone who may open the project: the pickers a ticket's developer and tester come from,
// and the roles a doc's permissions panel may list.
func (h *Handler) listProjectPeople(w http.ResponseWriter, r *http.Request) {
	people, err := h.svc.ListProjectPeople(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("projectID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	roleIDs, err := h.svc.ListProjectRoles(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("projectID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ProjectPeople{People: people, RoleIDs: roleIDs})
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

// meResponse is the /me wire shape the frontend's hasPermission(action) helper reads directly; Restricted is true for
// every non-Owner, whose Permissions hold nothing project-area and whose Projects say what they hold in each.
type meResponse struct {
	// RoleName is Roles' names joined, for a client from before a member held several.
	RoleName    string      `json:"role_name"`
	Roles       []*TeamRole `json:"roles"`
	Permissions []string    `json:"permissions"`
	Restricted  bool        `json:"restricted"`
	Projects    []MeProject `json:"projects"`
	// AccessSetupFinished is false until an Owner finishes the workspace's one-time access setup.
	AccessSetupFinished bool `json:"access_setup_finished"`
	// ShowLockedActions is true when any held role has the switch on: an action they can't take is shown dimmed with the reason.
	ShowLockedActions bool `json:"show_locked_actions"`
}

// me is re-fetched whenever useWorkspaceStore's selectedWorkspaceId changes.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	access, err := h.svc.MyAccess(r.Context(), r.PathValue("workspaceID"), UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, meResponse{
		RoleName: RoleNames(access.Roles), Roles: access.Roles, Permissions: access.Permissions,
		Restricted: access.Restricted, Projects: access.Projects, AccessSetupFinished: access.AccessSetupFinished, ShowLockedActions: access.ShowLockedActions,
	})
}

// rolesRequest names roles as role_ids, or as the one role_id a client from before several roles sends.
type rolesRequest struct {
	RoleID  string    `json:"role_id"`
	RoleIDs *[]string `json:"role_ids"`
}

// roleIDs is RoleIDs when sent, else RoleID as a set of one; sent is false when neither was.
func (r rolesRequest) roleIDs() (ids []string, sent bool) {
	if r.RoleIDs != nil {
		return *r.RoleIDs, true
	}
	if r.RoleID != "" {
		return []string{r.RoleID}, true
	}
	return nil, false
}

type inviteMemberRequest struct {
	Login string `json:"login"`
	rolesRequest
}

type addMemberRequest struct {
	rolesRequest
}

// changeMemberRoleRequest replaces a member's roles; allow, deny, every_project and project_access are removed fields
// that still parse, refused unless empty (ADR 0148).
type changeMemberRoleRequest struct {
	rolesRequest
	Allow         *permissions.Set     `json:"allow"`
	Deny          *permissions.Set     `json:"deny"`
	EveryProject  *string              `json:"every_project"`
	ProjectAccess []projectAccessPatch `json:"project_access"`
}

type projectAccessPatch struct {
	ProjectID string          `json:"project_id"`
	Allow     permissions.Set `json:"allow"`
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
	roleIDs, _ := req.roleIDs()
	if err := h.svc.InviteMember(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), req.Login, roleIDs...); err != nil {
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
	roleIDs, _ := req.roleIDs()
	err := h.svc.AddMember(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), r.PathValue("userID"), roleIDs...)
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
	if err := h.patchMember(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("workspaceID"), r.PathValue("userID"), req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// patchMember replaces the roles; a body carrying only empty removed fields changes nothing, as it did before.
func (h *Handler) patchMember(ctx context.Context, actorID, workspaceID, userID string, req changeMemberRoleRequest) error {
	if err := RefusePerPersonAccess(deref(req.Allow), deref(req.Deny), deref(req.EveryProject), len(req.ProjectAccess)); err != nil {
		return err
	}
	roleIDs, sent := req.roleIDs()
	removedOnly := req.Allow != nil || req.Deny != nil || req.EveryProject != nil || req.ProjectAccess != nil
	if !sent && removedOnly {
		return nil
	}
	if !sent {
		return fmt.Errorf("%w: role_ids is required", apperrs.ErrInvalid)
	}
	return h.svc.SetMemberRoles(ctx, actorID, workspaceID, userID, roleIDs...)
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
