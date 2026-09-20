package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

// accessPlayWorkspaceResolver reads from the repo directly to avoid recursing into a permission check;
// unlike a doc, a play already carries its own workspace id (ticket 21).
type accessPlayWorkspaceResolver struct {
	plays *storage.PlaysRepo
}

func (a accessPlayWorkspaceResolver) WorkspaceIDForPlay(ctx context.Context, playID string) (string, error) {
	p, err := a.plays.Get(ctx, playID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

// accessRoleResolver adapts tenancy+roles for HasPermission's role-mask layer.
type accessRoleResolver struct {
	tenancy *tenancy.Service
	roles   *roles.Service
}

func (a accessRoleResolver) MemberRole(ctx context.Context, workspaceID, userID string) (access.RoleInfo, error) {
	roleID, err := a.tenancy.MemberRoleID(ctx, workspaceID, userID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	r, err := a.roles.Get(ctx, workspaceID, roleID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	return access.RoleInfo{IsOwnerRole: r.IsOwnerRole, Permissions: r.Permissions}, nil
}

// accessDocWorkspaceResolver reads from the repo directly to avoid recursing into a permission check.
type accessDocWorkspaceResolver struct {
	docs     *storage.DocsRepo
	projects *workspace.Service
}

func (a accessDocWorkspaceResolver) WorkspaceIDForDoc(ctx context.Context, docID string) (string, error) {
	d, err := a.docs.GetByID(ctx, docID)
	if err != nil {
		return "", err
	}
	if d.ProjectID == "" {
		return "", nil
	}
	p, err := a.projects.Get(ctx, d.ProjectID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

// accessUsers adapts UsersRepo to access's Users interface, mapping auth.User to access.User (ADR 0017).
type accessUsers struct {
	users *storage.UsersRepo
}

func (a accessUsers) GetUserByID(ctx context.Context, id string) (*access.User, error) {
	u, err := a.users.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapAccessUser(u), nil
}

func (a accessUsers) ListUsers(ctx context.Context) ([]*access.User, error) {
	us, err := a.users.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*access.User, 0, len(us))
	for _, u := range us {
		out = append(out, mapAccessUser(u))
	}
	return out, nil
}

func mapAccessUser(u *auth.User) *access.User {
	return &access.User{ID: u.ID, Login: u.Login, Name: u.Name, CanCreateWorkspace: u.CanCreateWorkspace}
}
