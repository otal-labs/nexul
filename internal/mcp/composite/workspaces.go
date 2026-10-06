package composite

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// WorkspaceService is the slice of the tenancy use-cases workspace_list and workspace_update call.
type WorkspaceService interface {
	ListForUser(ctx context.Context, userID string) ([]*tenancy.Workspace, error)
	MemberRoleName(ctx context.Context, workspaceID, userID string) (string, error)
	ListPeople(ctx context.Context, actorID, workspaceID string) ([]tenancy.Person, error)
	Rename(ctx context.Context, userID, id string, name, slug *string) (*tenancy.Workspace, error)
}

// RoleReader is the slice of the roles use-cases workspace_list reads.
type RoleReader interface {
	List(ctx context.Context, workspaceID string) ([]*roles.Role, error)
}

type workspaceListIn struct {
	ID string `json:"id,omitempty" jsonschema:"Only this workspace, with its people, its roles, and the permission catalog their permissions come from."`
	mcptool.PageArgs
}

// workspaceResult carries roles and the catalog only when one workspace is asked for by id.
type workspaceResult struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Slug              string             `json:"slug"`
	Role              string             `json:"role"`
	People            []tenancy.Person   `json:"people,omitempty"`
	Roles             []roles.RoleResult `json:"roles,omitempty"`
	PermissionCatalog []permissions.Info `json:"permission_catalog,omitempty"`
}

type workspaceUpdateIn struct {
	ID   string  `json:"id" jsonschema:"The workspace to change, from workspace_list."`
	Name *string `json:"name,omitempty" jsonschema:"The workspace's new display name, for example Norwood Labs. Omit to keep it."`
	Slug *string `json:"slug,omitempty" jsonschema:"The workspace's new name in web links, for example norwood: lowercase letters and digits joined by single dashes, at most 48 characters, not a reserved path such as settings, and not taken by another workspace. Omit to keep it."`
}

type workspaceUpdateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// WorkspaceTools lists the caller's workspaces, the ids the project, play, memory, role, and invitation tools are
// scoped by, and renames one.
func WorkspaceTools(w WorkspaceService, r RoleReader) []mcptool.Tool {
	return []mcptool.Tool{workspaceListTool(w, r), workspaceUpdateTool(w)}
}

func workspaceUpdateTool(w WorkspaceService) mcptool.Tool {
	return mcptool.New("workspace_update", "Rename workspace or change its slug",
		"Changes a workspace's display name, its slug, or both; a field you omit keeps its value. "+
			"Changing the slug moves every web link into the workspace, so links using the old slug stop working and are not redirected; "+
			"a rename that keeps the slug leaves every link working. "+
			"Needs workspaces:write in that workspace. Returns the workspace's id, name, and slug as they now stand; workspace_list finds the id.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in workspaceUpdateIn) (any, error) {
			a, ok := identity.ActorFromCtx(ctx)
			if !ok || a.ID == "" {
				return nil, fmt.Errorf("%w: changing a workspace needs a signed-in user", apperrs.ErrUnauthorized)
			}
			if in.Name == nil && in.Slug == nil {
				return nil, fmt.Errorf("%w: send name, slug, or both", apperrs.ErrInvalid)
			}
			updated, err := w.Rename(ctx, a.ID, in.ID, in.Name, in.Slug)
			if err != nil {
				return nil, err
			}
			return workspaceUpdateResult{ID: updated.ID, Name: updated.Name, Slug: updated.Slug}, nil
		})
}

func workspaceListTool(w WorkspaceService, r RoleReader) mcptool.Tool {
	return mcptool.New("workspace_list", "List workspaces",
		"Lists the workspaces you belong to, with each one's id, name, slug (its name in web links and the workspace "+
			"a ticket tool takes to tell apart keys two workspaces share), and your role in it. Start here when a tool "+
			"needs a workspace_id: project_list, play_list, and the memory, role, and invitation tools are scoped by "+
			"workspace. With an id it returns only that workspace, with its people (user_id, login, display_name, avatar_url: "+
			"how to name a chat message's author_id, and the login an @mention takes), its roles (id, name, whether it is "+
			"the Owner role, and permissions) and the permission_catalog of every valid permission, which role_update takes. "+
			"It returns only workspaces you are a member of.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in workspaceListIn) (any, error) {
			a, ok := identity.ActorFromCtx(ctx)
			if !ok || a.ID == "" {
				return nil, fmt.Errorf("%w: listing workspaces needs a signed-in user", apperrs.ErrUnauthorized)
			}
			ws, err := w.ListForUser(ctx, a.ID)
			if err != nil {
				return nil, err
			}
			if in.ID != "" {
				return workspaceWithRoles(ctx, w, r, ws, in, a.ID)
			}
			out := make([]workspaceResult, 0, len(ws))
			for _, one := range ws {
				role, err := w.MemberRoleName(ctx, one.ID, a.ID)
				if err != nil {
					return nil, err
				}
				out = append(out, workspaceResult{ID: one.ID, Name: one.Name, Slug: one.Slug, Role: role})
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func workspaceWithRoles(ctx context.Context, w WorkspaceService, r RoleReader, ws []*tenancy.Workspace, in workspaceListIn, userID string) (any, error) {
	for _, one := range ws {
		if one.ID != in.ID {
			continue
		}
		role, err := w.MemberRoleName(ctx, one.ID, userID)
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
		res := workspaceResult{ID: one.ID, Name: one.Name, Slug: one.Slug, Role: role, People: people, Roles: shaped, PermissionCatalog: permissions.Catalog()}
		return mcptool.Paginate([]workspaceResult{res}, in.PageArgs), nil
	}
	return nil, fmt.Errorf("%w: workspace %s is not one you belong to; workspace_list without an id lists yours", apperrs.ErrNotFound, in.ID)
}
