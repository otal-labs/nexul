package composite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// AccountStatusSetter is the slice of the auth use-cases account_update calls.
type AccountStatusSetter interface {
	UpdateAccountStatus(ctx context.Context, actorID, targetID string, status auth.AccountStatus) error
}

// Team is the slice of the tenancy use-cases the account tools read and change workspace access through.
type Team interface {
	ListTeam(ctx context.Context, actorID string) (*tenancy.Team, error)
	MemberRoleID(ctx context.Context, workspaceID, userID string) (string, error)
	AddMember(ctx context.Context, actorID, workspaceID, userID, roleID string) error
	ChangeMemberRole(ctx context.Context, actorID, workspaceID, userID, roleID string) error
	SetMemberOverrides(ctx context.Context, actorID, workspaceID, userID string, allow, deny *permissions.Set) error
	RemoveMember(ctx context.Context, actorID, workspaceID, userID string) error
	SetEveryProject(ctx context.Context, actorID, workspaceID, userID, every string) error
	SetProjectAccess(ctx context.Context, actorID, workspaceID, userID, projectID string, allow permissions.Set) error
}

// AccountTools are the account tools that carry workspace access, which the tenancy domain owns.
func AccountTools(a AccountStatusSetter, t Team) []mcptool.Tool {
	return []mcptool.Tool{accountListTool(t), accountUpdateTool(a, t)}
}

// accountResult is one person on the Team: the account without its sign-in identities, plus its workspace access.
type accountResult struct {
	ID          string             `json:"id"`
	Login       string             `json:"login"`
	Name        string             `json:"name"`
	DisplayName string             `json:"display_name,omitempty"`
	AvatarURL   string             `json:"avatar_url,omitempty"`
	Status      string             `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	Online      bool               `json:"online"`
	LastSeenAt  *time.Time         `json:"last_seen_at"`
	Workspaces  []membershipResult `json:"workspaces"`
}

type membershipResult struct {
	WorkspaceID      string                   `json:"workspace_id"`
	WorkspaceName    string                   `json:"workspace_name"`
	RoleID           string                   `json:"role_id"`
	RoleName         string                   `json:"role_name"`
	IsOwner          bool                     `json:"is_owner"`
	Allow            permissions.Set          `json:"allow"`
	Deny             permissions.Set          `json:"deny"`
	CanManageMembers bool                     `json:"can_manage_members"`
	EveryProject     string                   `json:"every_project"`
	Projects         []*tenancy.ProjectAccess `json:"projects,omitzero"`
}

type accountListIn struct {
	mcptool.PageArgs
}

type accountWorkspaceIn struct {
	WorkspaceID   string            `json:"workspace_id" jsonschema:"The workspace's id, from workspace_list."`
	RoleID        string            `json:"role_id,omitempty" jsonschema:"A role of that workspace, never its Owner role; workspace_list with the workspace's id lists them. Required when the account is not a member yet."`
	Allow         *permissions.Set  `json:"allow,omitempty" jsonschema:"Replaces the account's workspace-wide allow overrides, each <domain>:<action>, for example [\"docs:write\"]. An empty list clears them; omit it to keep them."`
	Deny          *permissions.Set  `json:"deny,omitempty" jsonschema:"Replaces the account's workspace-wide deny overrides, each <domain>:<action>, for example [\"members:write\"]. An empty list clears them; omit it to keep them."`
	EveryProject  string            `json:"every_project,omitempty" jsonschema:"role (the role's project areas apply on every project) or none (a Restricted member, who sees only the projects in project_access). Omit it to keep the current one."`
	ProjectAccess []projectAccessIn `json:"project_access,omitempty" jsonschema:"Project access for a Restricted member, one entry per project to change; projects not listed keep their levels."`
}

type projectAccessIn struct {
	ProjectID string          `json:"project_id" jsonschema:"A project of that workspace, from project_list."`
	Allow     permissions.Set `json:"allow" jsonschema:"Replaces the levels on that project, each a project-area <domain>:<action> from permission_catalog, for example [\"tickets:read\",\"tickets:write\"]. An empty list takes the project away."`
}

type accountUpdateIn struct {
	ID                 string               `json:"id" jsonschema:"The account's id, from account_list."`
	Status             auth.AccountStatus   `json:"status,omitempty" jsonschema:"active or disabled. active reactivates a disabled account and restores a removed one. Needs accounts:write in any workspace."`
	Workspaces         []accountWorkspaceIn `json:"workspaces,omitempty" jsonschema:"Workspace access to set: adds the account to a workspace it is not in with role_id, changes its role where it is a member, and replaces its overrides where allow or deny is sent."`
	RemoveWorkspaceIDs []string             `json:"remove_workspace_ids,omitempty" jsonschema:"Workspaces to take the account out of; its account and authored content stay."`
}

type accountUpdateResult struct {
	Applied []string       `json:"applied"`
	Account *accountResult `json:"account,omitempty"`
}

func accountListTool(t Team) mcptool.Tool {
	return mcptool.New("account_list", "List accounts",
		"Lists the Team: accounts with their status (active, disabled, or removed), whether they are online now, "+
			"when they were last seen (to the hour, null once signed out everywhere), and their workspace access, one "+
			"entry per workspace with the role, whether that is the Owner role, the workspace-wide allow and deny "+
			"overrides, can_manage_members, whether you hold members:write there and so may change that access, and "+
			"every_project: role, where the role's project areas reach every project, or none, a Restricted member "+
			"whose projects list the project access they hold on projects you can open yourself. "+
			"A holder of accounts:read in any workspace sees every account and workspace; anyone else sees only the "+
			"workspaces where they hold members:write and the people in them, and callers with neither are refused. Use account_get "+
			"for your own account, account_update to change an account's status or workspace access, and "+
			"account_delete to remove it. People online come first, then the most recently seen.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in accountListIn) (any, error) {
			team, err := t.ListTeam(ctx, accountActor(ctx))
			if err != nil {
				return nil, err
			}
			out := make([]accountResult, 0, len(team.People))
			for _, p := range team.People {
				out = append(out, toAccountResult(p, manageable(team)))
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func accountUpdateTool(a AccountStatusSetter, t Team) mcptool.Tool {
	return mcptool.New("account_update", "Update account",
		"Changes an account's status and its workspace access. status disabled blocks sign-in while keeping "+
			"memberships and credentials; active reactivates a disabled account, or restores a removed one without "+
			"the access account_delete took away; status needs accounts:write in any workspace. workspaces adds the "+
			"account to a workspace, changes its role, or replaces its overrides there, and remove_workspace_ids "+
			"takes it out of one; each needs members:write in that workspace, a role or allow override may only "+
			"carry permissions you hold there yourself, and the Owner role can never be given, changed, or removed. "+
			"Inside a workspaces entry, every_project none restricts the person to the projects they hold access to and "+
			"role lifts it, and project_access replaces their levels on each project named; nobody grants a level they "+
			"don't hold on that project, and role needs the role's project levels on every project, which a Restricted "+
			"caller never has. Only the fields you send change. The changes apply in the order listed "+
			"here and stop at the first failure, whose message names the field and the ones already applied. "+
			"Returns the fields applied and the account as account_list shows it to you, when it shows it at all.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in accountUpdateIn) (any, error) {
			if in.Status == "" && len(in.Workspaces) == 0 && len(in.RemoveWorkspaceIDs) == 0 {
				return nil, fmt.Errorf("%w: send status, workspaces, or remove_workspace_ids", apperrs.ErrInvalid)
			}
			actor := accountActor(ctx)
			applied, err := runSteps(ctx, accountUpdateSteps(a, t, actor, in))
			if err != nil {
				return nil, err
			}
			return accountUpdateResult{Applied: applied, Account: readAccount(ctx, t, actor, in.ID)}, nil
		})
}

func accountUpdateSteps(a AccountStatusSetter, t Team, actor string, in accountUpdateIn) []step {
	var steps []step
	if in.Status != "" {
		steps = append(steps, step{field: "status", run: func(ctx context.Context) error {
			return a.UpdateAccountStatus(ctx, actor, in.ID, in.Status)
		}})
	}
	for _, w := range in.Workspaces {
		steps = append(steps, workspaceSteps(t, actor, in.ID, w)...)
	}
	for _, id := range in.RemoveWorkspaceIDs {
		steps = append(steps, step{field: "remove_workspace_ids[" + id + "]", run: func(ctx context.Context) error {
			return t.RemoveMember(ctx, actor, id, in.ID)
		}})
	}
	return steps
}

// workspaceSteps sets one workspace entry: membership and overrides, then Every project, then each project's levels.
func workspaceSteps(t Team, actor, userID string, w accountWorkspaceIn) []step {
	field := "workspaces[" + w.WorkspaceID + "]"
	var steps []step
	if w.RoleID != "" || w.Allow != nil || w.Deny != nil || (w.EveryProject == "" && w.ProjectAccess == nil) {
		steps = append(steps, step{field: field, hint: "workspace_list with the workspace's id lists its roles", run: func(ctx context.Context) error {
			return setWorkspaceAccess(ctx, t, actor, userID, w)
		}})
	}
	if w.EveryProject != "" {
		steps = append(steps, step{field: field + ".every_project", run: func(ctx context.Context) error {
			return t.SetEveryProject(ctx, actor, w.WorkspaceID, userID, w.EveryProject)
		}})
	}
	for _, p := range w.ProjectAccess {
		steps = append(steps, step{field: field + ".project_access[" + p.ProjectID + "]", hint: "project_list lists the workspace's projects", run: func(ctx context.Context) error {
			return t.SetProjectAccess(ctx, actor, w.WorkspaceID, userID, p.ProjectID, p.Allow)
		}})
	}
	return steps
}

// setWorkspaceAccess adds the account where it is not a member yet, else changes its role; then sets any overrides.
func setWorkspaceAccess(ctx context.Context, t Team, actor, userID string, w accountWorkspaceIn) error {
	_, err := t.MemberRoleID(ctx, w.WorkspaceID, userID)
	member := err == nil
	if !member && !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	if !member {
		if err := t.AddMember(ctx, actor, w.WorkspaceID, userID, w.RoleID); err != nil {
			return err
		}
	}
	if member && w.RoleID != "" {
		if err := t.ChangeMemberRole(ctx, actor, w.WorkspaceID, userID, w.RoleID); err != nil {
			return err
		}
	}
	if w.Allow == nil && w.Deny == nil {
		return nil
	}
	return t.SetMemberOverrides(ctx, actor, w.WorkspaceID, userID, w.Allow, w.Deny)
}

// readAccount is nil when the account falls outside the caller's view of the Team.
func readAccount(ctx context.Context, t Team, actor, id string) *accountResult {
	team, err := t.ListTeam(ctx, actor)
	if err != nil {
		return nil
	}
	for _, p := range team.People {
		if p.ID == id {
			r := toAccountResult(p, manageable(team))
			return &r
		}
	}
	return nil
}

func manageable(team *tenancy.Team) map[string]bool {
	out := make(map[string]bool, len(team.Workspaces))
	for _, w := range team.Workspaces {
		out[w.ID] = w.CanManageMembers
	}
	return out
}

func toAccountResult(p *tenancy.TeamPerson, canManage map[string]bool) accountResult {
	r := accountResult{
		ID: p.ID, Login: p.Login, Name: p.Name, DisplayName: p.DisplayName, AvatarURL: p.AvatarURL,
		Status: p.Status, CreatedAt: p.CreatedAt,
		Online: p.Online, LastSeenAt: p.LastSeenAt,
		Workspaces: make([]membershipResult, 0, len(p.Workspaces)),
	}
	for _, m := range p.Workspaces {
		r.Workspaces = append(r.Workspaces, membershipResult{
			WorkspaceID: m.WorkspaceID, WorkspaceName: m.WorkspaceName, RoleID: m.RoleID, RoleName: m.RoleName,
			IsOwner: m.IsOwner, Allow: m.Allow, Deny: m.Deny, CanManageMembers: canManage[m.WorkspaceID],
			EveryProject: m.EveryProject, Projects: restrictedProjects(m),
		})
	}
	return r
}

func accountActor(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

// restrictedProjects hides the rows a From role member keeps for a later switch back, since they grant nothing now.
func restrictedProjects(m *tenancy.TeamMembership) []*tenancy.ProjectAccess {
	if m.EveryProject != tenancy.EveryProjectNone {
		return nil
	}
	return m.Projects
}
