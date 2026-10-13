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
	AddMember(ctx context.Context, actorID, workspaceID, userID string, roleIDs ...string) error
	UpdateMemberRoles(ctx context.Context, actorID, workspaceID, userID string, add, remove []string) error
	RemoveMember(ctx context.Context, actorID, workspaceID, userID string) error
}

// AccountTools are the account tools that carry workspace access, which the tenancy domain owns.
func AccountTools(a AccountStatusSetter, t Team) []mcptool.Tool {
	return []mcptool.Tool{accountListTool(t), accountUpdateTool(a, t)}
}

// accountResult is one person on the Team: the account with the providers it signs in through but none of their identities, plus its workspace access.
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
	Providers   []string           `json:"providers"`
	Workspaces  []membershipResult `json:"workspaces"`
}

type membershipResult struct {
	WorkspaceID      string              `json:"workspace_id"`
	WorkspaceName    string              `json:"workspace_name"`
	Roles            []*tenancy.TeamRole `json:"roles"`
	IsOwner          bool                `json:"is_owner"`
	CanManageMembers bool                `json:"can_manage_members"`
}

type accountListIn struct {
	mcptool.PageArgs
}

type accountWorkspaceIn struct {
	WorkspaceID string   `json:"workspace_id" jsonschema:"The workspace's id, from workspace_list."`
	AddRoles    []string `json:"add_roles,omitempty" jsonschema:"Role ids to give the account there, never the Owner role; workspace_list with the workspace's id lists them. At least one when the account is not a member yet."`
	RemoveRoles []string `json:"remove_roles,omitempty" jsonschema:"Role ids to take away there, never the Owner role; taking away every role leaves a member who sees no project."`
}

type accountUpdateIn struct {
	ID                 string               `json:"id" jsonschema:"The account's id, from account_list."`
	Status             auth.AccountStatus   `json:"status,omitempty" jsonschema:"active or disabled. active reactivates a disabled account and restores a removed one. Needs accounts:write in any workspace."`
	Workspaces         []accountWorkspaceIn `json:"workspaces,omitempty" jsonschema:"Roles to change per workspace: adds the account to a workspace it is not in with add_roles, and gives or takes away roles where it is a member."`
	RemoveWorkspaceIDs []string             `json:"remove_workspace_ids,omitempty" jsonschema:"Workspaces to take the account out of; its account and authored content stay."`
}

type accountUpdateResult struct {
	Applied []string       `json:"applied"`
	Account *accountResult `json:"account,omitempty"`
}

func accountListTool(t Team) mcptool.Tool {
	return mcptool.New("account_list", "List accounts",
		"Lists the Team: accounts with their status (active, disabled, or removed), whether they are online now, "+
			"when they were last seen (to the hour, null once signed out everywhere), providers (the sign-in accounts "+
			"linked, github, google or discord, in the order they were linked), and their workspace access, one "+
			"entry per workspace with roles, every role they hold there in the order given (id, name, whether it is "+
			"the Owner role), is_owner, whether the Owner role is among them, and can_manage_members, whether you hold "+
			"members:write there and so may change their roles. Which projects someone is in is set on each project, "+
			"through permission_overwrite_list and permission_overwrite_update. "+
			"A holder of accounts:read in any workspace sees every account and workspace; anyone else sees only the "+
			"workspaces where they hold members:write and the people in them, and callers with neither are refused. Use account_get "+
			"for your own account, account_update to change an account's status or roles, and "+
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
		"Changes an account's status and the roles it holds in each workspace. status disabled blocks sign-in and disconnects "+
			"their computers while keeping memberships and credentials; active reactivates a disabled account, or restores a removed one without "+
			"the access account_delete took away; status needs accounts:write in any workspace. A workspaces entry adds the "+
			"account to a workspace it is not in with add_roles, or gives it add_roles and takes away remove_roles where "+
			"it is a member; remove_workspace_ids takes it out of one. Each needs members:write in that workspace, you "+
			"may give or take away only a role whose permissions you hold there yourself, and the Owner role is never "+
			"given or taken. Someone left with no role stays a member and sees no project. Which projects a person is "+
			"in is set on each project with permission_overwrite_update. Only the fields you send change. The changes apply in the order listed "+
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

// workspaceSteps sets one workspace entry's membership and roles.
func workspaceSteps(t Team, actor, userID string, w accountWorkspaceIn) []step {
	field := "workspaces[" + w.WorkspaceID + "]"
	return []step{{field: field, hint: "workspace_list with the workspace's id lists its roles", run: func(ctx context.Context) error {
		return setWorkspaceRoles(ctx, t, actor, userID, w)
	}}}
}

// setWorkspaceRoles adds the account where it is not a member yet, else gives and takes away the roles named.
func setWorkspaceRoles(ctx context.Context, t Team, actor, userID string, w accountWorkspaceIn) error {
	_, err := t.MemberRoleID(ctx, w.WorkspaceID, userID)
	member := err == nil
	if !member && !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	if member {
		return t.UpdateMemberRoles(ctx, actor, w.WorkspaceID, userID, w.AddRoles, w.RemoveRoles)
	}
	if len(w.RemoveRoles) > 0 {
		return fmt.Errorf("%w: the account is not a member of workspace %s, so it holds no role to take away", apperrs.ErrInvalid, w.WorkspaceID)
	}
	return t.AddMember(ctx, actor, w.WorkspaceID, userID, w.AddRoles...)
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
		Online: p.Online, LastSeenAt: p.LastSeenAt, Providers: p.Providers,
		Workspaces: make([]membershipResult, 0, len(p.Workspaces)),
	}
	for _, m := range p.Workspaces {
		r.Workspaces = append(r.Workspaces, membershipResult{
			WorkspaceID: m.WorkspaceID, WorkspaceName: m.WorkspaceName, Roles: m.Roles,
			IsOwner: m.IsOwner, CanManageMembers: canManage[m.WorkspaceID],
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
