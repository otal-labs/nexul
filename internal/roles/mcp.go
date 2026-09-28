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
	ID          string    `json:"id,omitempty" jsonschema:"The role to change, from workspace_list with the workspace's id. Omit to create a role, or to clone one with clone_from_id."`
	WorkspaceID string    `json:"workspace_id" jsonschema:"The workspace the role is in; with clone_from_id, the workspace to copy it into."`
	Name        *string   `json:"name,omitempty" jsonschema:"The role's name, for example Reviewers. Required to create; omit to keep the current one."`
	Permissions *[]string `json:"permissions,omitempty" jsonschema:"The role's whole permission set as values from permission_catalog, for example [\"docs:read\",\"docs:write\"]; it replaces the current set and [] clears it. Omit to keep the current set."`
	CloneFromID string    `json:"clone_from_id,omitempty" jsonschema:"The id of a custom role in another workspace to copy, name and permissions, into workspace_id instead of writing a new one."`
}

type roleDeleteIn struct {
	ID          string `json:"id" jsonschema:"The role's id, from workspace_list with the workspace's id."`
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace the role is in."`
}

// RoleResult is a role as MCP tools return it: what an agent needs to assign, edit, or clone it.
type RoleResult struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspace_id"`
	Name        string   `json:"name"`
	IsOwnerRole bool     `json:"is_owner_role"`
	Permissions []string `json:"permissions"`
}

// ToRoleResult shapes a role for an MCP result.
func ToRoleResult(r *Role) RoleResult {
	perms := make([]string, 0, len(r.Permissions))
	for _, a := range r.Permissions {
		perms = append(perms, string(a))
	}
	return RoleResult{ID: r.ID, WorkspaceID: r.WorkspaceID, Name: r.Name, IsOwnerRole: r.IsOwnerRole, Permissions: perms}
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
		"Creates, clones, or changes a workspace role. Without id it creates a role in workspace_id from name and permissions; "+
			"with clone_from_id it copies that custom role from its own workspace into workspace_id, renamed to \"<name> (copy)\" when the name is taken; "+
			"with id it changes only the fields you send. "+
			"workspace_list with the workspace's id lists its roles and the permission_catalog of valid permissions, and the Owner role can't be edited or cloned. "+
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
		if in.ID != "" || in.Name != nil || in.Permissions != nil {
			return nil, fmt.Errorf("%w: clone_from_id copies the source's name and permissions into workspace_id as a new role; omit id, name, and permissions, then change the copy with role_update and its id", apperrs.ErrInvalid)
		}
		return s.Clone(ctx, "", in.CloneFromID, in.WorkspaceID, actorID)
	}
	perms, err := parsePermissions(in.Permissions)
	if err != nil {
		return nil, err
	}
	if in.ID == "" {
		return s.Create(ctx, in.WorkspaceID, actorID, deref(in.Name, ""), perms)
	}
	r, err := s.Get(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, fmt.Errorf("%w (workspace_list with the workspace's id lists its roles)", err)
	}
	if in.Name == nil && in.Permissions == nil {
		return r, nil
	}
	if in.Permissions == nil {
		perms = r.Permissions
	}
	return s.Update(ctx, in.WorkspaceID, in.ID, actorID, deref(in.Name, r.Name), perms)
}

func roleDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("role_delete", "Delete role",
		"Deletes a custom role from its workspace for good. "+
			"Members holding it lose the permissions it granted, so reassign them first; the Owner role can't be deleted. "+
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
