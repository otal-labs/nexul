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

// WorkspaceReader is the slice of the tenancy use-cases workspace_list reads.
type WorkspaceReader interface {
	ListForUser(ctx context.Context, userID string) ([]*tenancy.Workspace, error)
	MemberRoleName(ctx context.Context, workspaceID, userID string) (string, error)
}

// RoleReader is the slice of the roles use-cases workspace_list reads.
type RoleReader interface {
	List(ctx context.Context, workspaceID string) ([]*roles.Role, error)
}

type workspaceListIn struct {
	ID string `json:"id,omitempty" jsonschema:"Only this workspace, with its roles and the permission catalog their permissions come from."`
	mcptool.PageArgs
}

// workspaceResult carries roles and the catalog only when one workspace is asked for by id.
type workspaceResult struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Role              string             `json:"role"`
	Roles             []roles.RoleResult `json:"roles,omitempty"`
	PermissionCatalog []permissions.Info `json:"permission_catalog,omitempty"`
}

// WorkspaceTools lists the caller's workspaces, the ids the project, play, memory, role, and invitation tools are scoped by.
func WorkspaceTools(w WorkspaceReader, r RoleReader) []mcptool.Tool {
	return []mcptool.Tool{mcptool.New("workspace_list", "List workspaces",
		"Lists the workspaces you belong to, with each one's id, name, and your role in it. Start here when a tool "+
			"needs a workspace_id: project_list, play_list, and the memory, role, and invitation tools are scoped by "+
			"workspace. With an id it returns only that workspace, with its roles (id, name, whether it is the Owner role, "+
			"and permissions) and the permission_catalog of every valid permission, which role_update takes. "+
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
				out = append(out, workspaceResult{ID: one.ID, Name: one.Name, Role: role})
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})}
}

func workspaceWithRoles(ctx context.Context, w WorkspaceReader, r RoleReader, ws []*tenancy.Workspace, in workspaceListIn, userID string) (any, error) {
	for _, one := range ws {
		if one.ID != in.ID {
			continue
		}
		role, err := w.MemberRoleName(ctx, one.ID, userID)
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
		res := workspaceResult{ID: one.ID, Name: one.Name, Role: role, Roles: shaped, PermissionCatalog: permissions.Catalog()}
		return mcptool.Paginate([]workspaceResult{res}, in.PageArgs), nil
	}
	return nil, fmt.Errorf("%w: workspace %s is not one you belong to; workspace_list without an id lists yours", apperrs.ErrNotFound, in.ID)
}
