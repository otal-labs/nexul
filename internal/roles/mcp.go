package roles

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type roleUpdateIn struct {
	ID                string    `json:"id,omitempty" jsonschema:"The role to change, from workspace_list with the workspace's id. Omit to create or clone."`
	WorkspaceID       string    `json:"workspace_id" jsonschema:"The workspace the role is in; with clone_from_id, the workspace to copy it into."`
	Name              *string   `json:"name,omitempty" jsonschema:"The role's name. Required to create."`
	Description       *string   `json:"description,omitempty" jsonschema:"What someone with the role does, in one line of at most 140 characters. It shows on invitations and wherever someone picks roles. An empty string clears it."`
	Permissions       *[]string `json:"permissions,omitempty" jsonschema:"The role's whole permission set as values from permission_catalog, such as docs:read; it replaces the current set and [] clears it."`
	WelcomeSteps      *[]string `json:"welcome_steps,omitempty" jsonschema:"The Welcome wizard steps the role gives its holders, from github, computer, projects, deploy_server and phone; it replaces the current list and [] clears it. A step added here is given again to everyone holding the role."`
	ShowLockedActions *bool     `json:"show_locked_actions,omitempty" jsonschema:"Whether holders see an action they can't take dimmed with the reason, instead of not seeing it; off for a new role."`
	CloneFromID       string    `json:"clone_from_id,omitempty" jsonschema:"A custom role in another workspace to copy into workspace_id instead of writing a new one."`
}

type roleDeleteIn struct {
	ID          string `json:"id" jsonschema:"The role id, from workspace_list with the workspace's id."`
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace, from workspace_list."`
}

// sendsFields is whether the call carries any field of the role itself.
func (in roleUpdateIn) sendsFields() bool {
	return in.Name != nil || in.Description != nil || in.Permissions != nil || in.WelcomeSteps != nil || in.ShowLockedActions != nil
}

// RoleResult is a role as MCP tools return it: what an agent needs to assign, edit, or clone it.
type RoleResult struct {
	ID                string   `json:"id"`
	WorkspaceID       string   `json:"workspace_id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	IsOwnerRole       bool     `json:"is_owner_role"`
	Permissions       []string `json:"permissions"`
	WelcomeSteps      []string `json:"welcome_steps"`
	ShowLockedActions bool     `json:"show_locked_actions"`
}

// ToRoleResult shapes a role for an MCP result.
func ToRoleResult(r *Role) RoleResult {
	perms := make([]string, 0, len(r.Permissions))
	for _, a := range r.Permissions {
		perms = append(perms, string(a))
	}
	return RoleResult{ID: r.ID, WorkspaceID: r.WorkspaceID, Name: r.Name, Description: r.Description, IsOwnerRole: r.IsOwnerRole, Permissions: perms, WelcomeSteps: r.WelcomeSteps, ShowLockedActions: r.ShowLockedActions}
}

type roleDeleted struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// MCPTools returns the role write tools; roles are read through workspace_list with a workspace's id.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{roleUpdateTool(s), roleDeleteTool(s)}
}

func roleUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("role_update", "Create, clone, or update role",
		"Creates, clones, or changes a workspace role. Without id it creates a role in workspace_id; "+
			"with clone_from_id it copies that custom role from its own workspace into workspace_id, renamed to \"<name> (copy)\" when the name is taken; "+
			"with id it changes only the fields you send. "+
			"workspace_list with the workspace's id lists its roles and the permission_catalog of valid permissions; the Owner role can't be edited or cloned. "+
			"A permission whose catalog area is project counts only inside the projects a holder is in, which each project's rules decide (permission_overwrite_update); a workspace or instance one counts everywhere. "+
			"Returns the role as it now stands; remove one with role_delete.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in roleUpdateIn) (any, error) {
			r, err := updateRole(ctx, s, in)
			if err != nil {
				return nil, err
			}
			return ToRoleResult(r), nil
		})
}

func updateRole(ctx context.Context, s *Service, in roleUpdateIn) (*Role, error) {
	actorID, err := mcpActorID(ctx)
	if err != nil {
		return nil, err
	}
	if in.CloneFromID != "" {
		if in.sendsFields() || in.ID != "" {
			return nil, fmt.Errorf("%w: clone_from_id copies the source's name, description, permissions, welcome steps and show_locked_actions into workspace_id as a new role; omit id, name, description, permissions, welcome_steps and show_locked_actions, then change the copy with role_update and its id", apperrs.ErrInvalid)
		}
		return s.Clone(ctx, "", in.CloneFromID, in.WorkspaceID, actorID)
	}
	perms, err := parsePermissions(in.Permissions)
	if err != nil {
		return nil, err
	}
	steps := deref(in.WelcomeSteps, nil)
	if in.ID == "" {
		return s.Create(ctx, in.WorkspaceID, actorID, deref(in.Name, ""), deref(in.Description, ""), perms, steps, deref(in.ShowLockedActions, false))
	}
	r, err := s.Get(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, fmt.Errorf("%w (workspace_list with the workspace's id lists its roles)", err)
	}
	if !in.sendsFields() {
		return r, nil
	}
	if in.Permissions == nil {
		perms = r.Permissions
	}
	return s.Update(ctx, in.WorkspaceID, in.ID, actorID, deref(in.Name, r.Name), in.Description, perms, steps, in.ShowLockedActions)
}

func roleDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("role_delete", "Delete role",
		"Deletes a custom role from its workspace for good. "+
			"Members holding it lose it and the permissions it granted, and someone left with no role stays a member who sees no project; "+
			"rules naming it go with it, and so do pending invites and invitation links giving it. The Owner role can't be deleted. "+
			"Returns the deleted id; change a role instead with role_update.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in roleDeleteIn) (any, error) {
			actorID, err := mcpActorID(ctx)
			if err != nil {
				return nil, err
			}
			if err := s.Delete(ctx, in.WorkspaceID, in.ID, actorID); err != nil {
				return nil, err
			}
			return roleDeleted{ID: in.ID, Deleted: true}, nil
		})
}

func mcpActorID(ctx context.Context) (string, error) {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return "", fmt.Errorf("%w: changing roles needs a signed-in user", apperrs.ErrUnauthorized)
	}
	return a.ID, nil
}

// parsePermissions rejects an unknown value instead of dropping it, so an agent learns the value was never granted.
func parsePermissions(values *[]string) (permissions.Set, error) {
	if values == nil {
		return nil, nil
	}
	out := make([]permissions.Action, 0, len(*values))
	for _, v := range *values {
		a, ok := permissions.ParseAction(v)
		if !ok {
			return nil, fmt.Errorf("%w: %q is not a permission; workspace_list with the workspace's id returns permission_catalog", apperrs.ErrInvalid, v)
		}
		out = append(out, a)
	}
	return permissions.SetOf(out...), nil
}

func deref[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
