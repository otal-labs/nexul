package composite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// WorkspaceService is the slice of the tenancy use-cases workspace_list and workspace_update call.
type WorkspaceService interface {
	ListForUser(ctx context.Context, userID string) ([]*tenancy.Workspace, error)
	MemberRoles(ctx context.Context, workspaceID, userID string) ([]*tenancy.TeamRole, error)
	ListPeople(ctx context.Context, actorID, workspaceID string) ([]tenancy.Person, error)
	Rename(ctx context.Context, userID, id string, name, slug *string) (*tenancy.Workspace, error)
	Archive(ctx context.Context, actorID, id string) (*tenancy.Workspace, error)
	Restore(ctx context.Context, actorID, id string) (*tenancy.Workspace, error)
	ListArchived(ctx context.Context, actorID string) ([]*tenancy.Workspace, error)
}

// AutoPlayLimits is the slice of the plays use-cases holding a workspace's bounds on automatic runs.
type AutoPlayLimits interface {
	AutoPlayLimits(ctx context.Context, workspaceID string) (plays.AutoPlayLimits, error)
	SetAutoPlayLimits(ctx context.Context, workspaceID string, patch plays.AutoPlayLimitsPatch) (plays.AutoPlayLimits, error)
}

// GitHubAccounts is the slice of the repository use-cases that detaches an installation's account from a workspace;
// attaching a repository is what links one (ADR 0147).
type GitHubAccounts interface {
	UnassignInstallation(ctx context.Context, account, workspaceID string) error
}

// AccessSetup is the slice of the access use-cases that saves a workspace's Access page, here only to finish its
// one-time setup.
type AccessSetup interface {
	SaveGrid(ctx context.Context, actorID, workspaceID string, in access.GridSave) error
}

// RoleReader is the slice of the roles use-cases workspace_list reads.
type RoleReader interface {
	List(ctx context.Context, workspaceID string) ([]*roles.Role, error)
}

type workspaceListIn struct {
	ID       string `json:"id,omitempty" jsonschema:"Only this workspace, with its people, its roles, and the permission catalog their permissions come from."`
	Archived bool   `json:"archived,omitempty" jsonschema:"true lists the archived workspaces instead of yours, each with archived_at, for the instance Owner who restores them with workspace_update archived false. Defaults to false."`
	mcptool.PageArgs
}

// workspaceResult carries roles and the catalog only when one workspace is asked for by id.
type workspaceResult struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Slug              string              `json:"slug"`
	YourRoles         []*tenancy.TeamRole `json:"your_roles"`
	People            []tenancy.Person    `json:"people,omitempty"`
	Roles             []roles.RoleResult  `json:"roles,omitempty"`
	PermissionCatalog []permissions.Info  `json:"permission_catalog,omitempty"`
	AutoPlayDailyCap  *int                `json:"auto_play_daily_cap,omitempty"`
	// AccessSetupFinished is false until an Owner finishes the one-time access setup.
	AccessSetupFinished bool       `json:"access_setup_finished"`
	ArchivedAt          *time.Time `json:"archived_at,omitempty"`
}

type workspaceUpdateIn struct {
	ID                   string   `json:"id" jsonschema:"The workspace to change, from workspace_list."`
	Name                 *string  `json:"name,omitempty" jsonschema:"The workspace's new display name, for example Norwood Labs. Omit to keep it."`
	Slug                 *string  `json:"slug,omitempty" jsonschema:"The workspace's new name in web links, for example norwood: lowercase letters and digits joined by single dashes, at most 48 characters, not a reserved path such as settings, and not taken by another workspace. Omit to keep it."`
	AutoPlayDailyCap     *int     `json:"auto_play_daily_cap,omitempty" jsonschema:"How many automatic runs auto plays may start on one ticket per rolling day, across every auto play, 1 to 50; a ticket past it pauses its auto plays. Omit to keep it."`
	RemoveGitHubAccounts []string `json:"remove_github_accounts,omitempty" jsonschema:"GitHub accounts or organisations, by login such as acme, to detach from this workspace: background work there stops reading their repositories as the GitHub App until someone attaches one again with project_update. Needs projects:create in this workspace."`
	Archived             *bool    `json:"archived,omitempty" jsonschema:"true archives the workspace: it leaves every list, search and live update for its members until the instance Owner restores it, and needs workspaces:delete there; false restores an archived workspace for everyone who was in it and needs instance:write, the instance Owner's. The instance's own workspace can't be archived. Omit to leave it as is."`
	FinishAccessSetup    bool     `json:"finish_access_setup,omitempty" jsonschema:"true ends the workspace's one-time access setup, after an upgrade wiped who is in each project: the Owners' Set up access prompt goes and members waiting on their projects stop waiting. List roles and people on projects first with permission_overwrite_update (projects:open allow); a project left listing nobody is seen only by Owners. Needs the Owner role."`
}

type workspaceUpdateResult struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Slug                  string   `json:"slug"`
	AutoPlayDailyCap      *int     `json:"auto_play_daily_cap,omitempty"`
	GitHubAccountsChanged []string `json:"github_accounts_changed,omitempty"`
	AccessSetupFinished   bool     `json:"access_setup_finished,omitempty"`
	Archived              *bool    `json:"archived,omitempty"`
}

// WorkspaceTools lists the caller's workspaces, the ids the project, play, memory, role, and invitation tools are
// scoped by, and renames one or changes its auto play cap.
func WorkspaceTools(w WorkspaceService, r RoleReader, limits AutoPlayLimits, accounts GitHubAccounts, setup AccessSetup) []mcptool.Tool {
	return []mcptool.Tool{workspaceListTool(w, r, limits), workspaceUpdateTool(w, limits, accounts, setup)}
}

func workspaceUpdateTool(w WorkspaceService, limits AutoPlayLimits, accounts GitHubAccounts, setup AccessSetup) mcptool.Tool {
	return mcptool.New("workspace_update", "Rename workspace or change its settings",
		"Changes a workspace's display name, its slug, its daily cap on automatic runs per ticket, how many runs a person may have going before their next auto run there waits, detaches GitHub accounts from it, finishes its one-time access setup, archives or restores it, or several; a field you omit keeps its value. "+
			"Changing the slug moves every web link into the workspace, so links using the old slug stop working and are not redirected; "+
			"a rename that keeps the slug leaves every link working. "+
			"The name and slug need workspaces:write in that workspace and apply first; auto_play_daily_cap and auto_play_runs_at_once need autoplays:write; remove_github_accounts needs projects:create; finish_access_setup needs the Owner role and applies next to last. "+
			"archived true needs workspaces:delete there and applies last; it hides the workspace from every list, search and live update of its members until archived false, which needs instance:write (the instance Owner's), restores it and applies first; the instance's own workspace can't be archived, and workspace_list with archived true finds the archived ones. "+
			"Returns the workspace's id, name, and slug as they now stand, with the auto play limits and the GitHub accounts when they changed, and archived when you sent it; workspace_list finds the id.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in workspaceUpdateIn) (any, error) {
			a, ok := identity.ActorFromCtx(ctx)
			if !ok || a.ID == "" {
				return nil, fmt.Errorf("%w: changing a workspace needs a signed-in user", apperrs.ErrUnauthorized)
			}
			if !in.changesSettings() && in.Archived == nil {
				return nil, fmt.Errorf("%w: send name, slug, auto_play_daily_cap, auto_play_runs_at_once, remove_github_accounts, finish_access_setup, archived, or several", apperrs.ErrInvalid)
			}
			if in.Archived != nil && !*in.Archived {
				return restoreThenChange(ctx, w, limits, accounts, setup, a.ID, in)
			}
			res, err := changeSettings(ctx, w, limits, accounts, setup, a.ID, in)
			if err != nil {
				return nil, err
			}
			if in.Archived == nil {
				return res, nil
			}
			return archiveLast(ctx, w, a.ID, in, res, in.changesSettings())
		})
}

func (in workspaceUpdateIn) changesSettings() bool {
	return in.Name != nil || in.Slug != nil || in.changesLimits() || len(in.RemoveGitHubAccounts) > 0 || in.FinishAccessSetup
}

func (in workspaceUpdateIn) changesLimits() bool {
	return in.AutoPlayDailyCap != nil
}

// changeSettings applies every field but archived, in the order the tool's description gives.
func changeSettings(ctx context.Context, w WorkspaceService, limits AutoPlayLimits, accounts GitHubAccounts, setup AccessSetup, userID string, in workspaceUpdateIn) (workspaceUpdateResult, error) {
	out, applied, err := updateWorkspace(ctx, w, limits, userID, in)
	if err != nil {
		return out, err
	}
	res, err := changeGitHubAccounts(ctx, w, accounts, userID, in, out, applied)
	if err != nil {
		return out, err
	}
	if !in.FinishAccessSetup {
		return res.(workspaceUpdateResult), nil
	}
	res, err = finishAccessSetup(ctx, w, setup, userID, in, res.(workspaceUpdateResult))
	if err != nil {
		return out, err
	}
	return res.(workspaceUpdateResult), nil
}

// finishAccessSetup ends the one-time access setup last, saying which earlier parts applied when it fails.
func finishAccessSetup(ctx context.Context, w WorkspaceService, setup AccessSetup, userID string, in workspaceUpdateIn, out workspaceUpdateResult) (any, error) {
	if out.ID == "" {
		ws, err := memberWorkspace(ctx, w, userID, in.ID)
		if err != nil {
			return nil, err
		}
		out = workspaceUpdateResult{ID: ws.ID, Name: ws.Name, Slug: ws.Slug, AutoPlayDailyCap: out.AutoPlayDailyCap, GitHubAccountsChanged: out.GitHubAccountsChanged}
	}
	if err := setup.SaveGrid(ctx, userID, in.ID, access.GridSave{Finish: true}); err != nil {
		if in.Name != nil || in.Slug != nil || in.AutoPlayDailyCap != nil || len(in.RemoveGitHubAccounts) > 0 {
			return nil, &mcptool.PartialError{Applied: []string{"everything but finish_access_setup"}, Err: fmt.Errorf("finish_access_setup: %w", err)}
		}
		return nil, err
	}
	out.AccessSetupFinished = true
	return out, nil
}

// changeGitHubAccounts detaches accounts last, saying which earlier parts applied when one fails.
func changeGitHubAccounts(ctx context.Context, w WorkspaceService, accounts GitHubAccounts, userID string, in workspaceUpdateIn, out workspaceUpdateResult, applied []string) (any, error) {
	if len(in.RemoveGitHubAccounts) == 0 {
		return out, nil
	}
	fail := func(err error) (any, error) {
		if len(applied) > 0 {
			return nil, &mcptool.PartialError{Applied: applied, Err: err}
		}
		return nil, err
	}
	if out.ID == "" {
		ws, err := memberWorkspace(ctx, w, userID, in.ID)
		if err != nil {
			return nil, err
		}
		out = workspaceUpdateResult{ID: ws.ID, Name: ws.Name, Slug: ws.Slug}
	}
	for _, account := range in.RemoveGitHubAccounts {
		if err := accounts.UnassignInstallation(ctx, account, in.ID); err != nil {
			return fail(fmt.Errorf("remove_github_accounts %s: %w", account, err))
		}
		applied = append(applied, "remove "+account)
	}
	out.GitHubAccountsChanged = in.RemoveGitHubAccounts
	return out, nil
}

func updateWorkspace(ctx context.Context, w WorkspaceService, limits AutoPlayLimits, userID string, in workspaceUpdateIn) (workspaceUpdateResult, []string, error) {
	var out workspaceUpdateResult
	var applied []string
	if in.Name != nil || in.Slug != nil {
		updated, err := w.Rename(ctx, userID, in.ID, in.Name, in.Slug)
		if err != nil {
			return out, nil, err
		}
		out = workspaceUpdateResult{ID: updated.ID, Name: updated.Name, Slug: updated.Slug}
		applied = append(applied, "name and slug")
	}
	if !in.changesLimits() {
		return out, applied, nil
	}
	if out.ID == "" {
		ws, err := memberWorkspace(ctx, w, userID, in.ID)
		if err != nil {
			return out, nil, err
		}
		out = workspaceUpdateResult{ID: ws.ID, Name: ws.Name, Slug: ws.Slug}
	}
	set, err := limits.SetAutoPlayLimits(ctx, in.ID, plays.AutoPlayLimitsPatch{DailyCapPerTicket: in.AutoPlayDailyCap})
	if err != nil && len(applied) > 0 {
		return out, nil, &mcptool.PartialError{Applied: applied, Err: fmt.Errorf("auto play limits: %w", err)}
	}
	if err != nil {
		return out, nil, err
	}
	out.AutoPlayDailyCap = &set.DailyCapPerTicket
	return out, append(applied, "auto play limits"), nil
}

func memberWorkspace(ctx context.Context, w WorkspaceService, userID, id string) (*tenancy.Workspace, error) {
	ws, err := w.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, one := range ws {
		if one.ID == id {
			return one, nil
		}
	}
	return nil, fmt.Errorf("%w: workspace %s is not one you belong to; workspace_list without an id lists yours", apperrs.ErrNotFound, id)
}

func workspaceListTool(w WorkspaceService, r RoleReader, limits AutoPlayLimits) mcptool.Tool {
	return mcptool.New("workspace_list", "List workspaces",
		"Lists the workspaces you belong to, with each one's id, name, slug (its name in web links and the workspace "+
			"a ticket tool takes to tell apart keys two workspaces share), and your_roles, every role you hold there in "+
			"the order given (id, name, whether it is the Owner role), none for a member who sees no project. Start here when a tool "+
			"needs a workspace_id: project_list, play_list, and the memory, role, and invitation tools are scoped by "+
			"workspace. With an id it returns only that workspace, with its people (user_id, login, display_name, avatar_url: "+
			"how to name a chat message's author_id, and the login an @mention takes), its roles (id, name, whether it is "+
			"the Owner role, and permissions), the permission_catalog of every valid permission, which role_update takes, and, "+
			"with autoplays:read, its auto_play_daily_cap on automatic runs per ticket per day and its auto_play_runs_at_once, how many runs a person may have going before their next auto run there waits. "+
			"It returns only workspaces you are a member of, never an archived one; archived true lists the archived workspaces instead for the instance Owner, who needs instance:write.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in workspaceListIn) (any, error) {
			a, ok := identity.ActorFromCtx(ctx)
			if !ok || a.ID == "" {
				return nil, fmt.Errorf("%w: listing workspaces needs a signed-in user", apperrs.ErrUnauthorized)
			}
			if in.Archived {
				return archivedWorkspaces(ctx, w, a.ID, in.PageArgs)
			}
			ws, err := w.ListForUser(ctx, a.ID)
			if err != nil {
				return nil, err
			}
			if in.ID != "" {
				return workspaceWithRoles(ctx, w, r, limits, ws, in, a.ID)
			}
			out := make([]workspaceResult, 0, len(ws))
			for _, one := range ws {
				held, err := w.MemberRoles(ctx, one.ID, a.ID)
				if err != nil {
					return nil, err
				}
				out = append(out, workspaceResult{ID: one.ID, Name: one.Name, Slug: one.Slug, YourRoles: held, AccessSetupFinished: one.AccessSetupFinished})
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func workspaceWithRoles(ctx context.Context, w WorkspaceService, r RoleReader, limits AutoPlayLimits, ws []*tenancy.Workspace, in workspaceListIn, userID string) (any, error) {
	for _, one := range ws {
		if one.ID != in.ID {
			continue
		}
		held, err := w.MemberRoles(ctx, one.ID, userID)
		if err != nil {
			return nil, err
		}
		people, err := w.ListPeople(ctx, userID, one.ID)
		if err != nil {
			return nil, err
		}
		rs, err := r.List(ctx, one.ID)
		if err != nil {
			return nil, err
		}
		shaped := make([]roles.RoleResult, 0, len(rs))
		for _, x := range rs {
			shaped = append(shaped, roles.ToRoleResult(x))
		}
		res := workspaceResult{ID: one.ID, Name: one.Name, Slug: one.Slug, YourRoles: held, People: people, Roles: shaped, PermissionCatalog: permissions.Catalog(), AccessSetupFinished: one.AccessSetupFinished}
		limit, err := limits.AutoPlayLimits(ctx, one.ID)
		if err != nil && !errors.Is(err, apperrs.ErrForbidden) {
			return nil, err
		}
		if err == nil {
			res.AutoPlayDailyCap = &limit.DailyCapPerTicket
		}
		return mcptool.Paginate([]workspaceResult{res}, in.PageArgs), nil
	}
	return nil, fmt.Errorf("%w: workspace %s is not one you belong to; workspace_list without an id lists yours", apperrs.ErrNotFound, in.ID)
}
