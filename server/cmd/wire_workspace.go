package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/workspace"
)

// workspaceUserStore adapts auth users to workspace's UserStore seam (ADR 0017): workspace cannot import auth.
type workspaceUserStore struct {
	users *storage.UsersRepo
}

func (a workspaceUserStore) GetUserByLogin(ctx context.Context, login string) (*workspace.User, error) {
	u, err := a.users.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	return &workspace.User{ID: u.ID, Login: u.Login}, nil
}

func (a workspaceUserStore) LoginForUserID(ctx context.Context, userID string) (string, error) {
	u, err := a.users.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Login, nil
}

// workspaceMembersStore adapts tenancy's raw membership store to workspace's WorkspaceMemberStore seam
// (ADR 0017); it bypasses tenancy.Service on purpose, since the memory.updated fan-out has no acting user to
// check MembersWrite with (a bus handler reacting to an event, not a request).
type workspaceMembersStore struct {
	members *storage.WorkspaceMembersRepo
}

func (a workspaceMembersStore) ListMemberUserIDs(ctx context.Context, workspaceID string) ([]string, error) {
	ms, err := a.members.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.UserID
	}
	return out, nil
}

// pushWorkspaceNamer adapts the tenancy store to push's WorkspaceNamer seam (ADR 0017): the body names the workspace.
type pushWorkspaceNamer struct {
	workspaces *storage.WorkspacesRepo
}

func (a pushWorkspaceNamer) WorkspaceName(ctx context.Context, workspaceID string) (string, error) {
	w, err := a.workspaces.Get(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	return w.Name, nil
}
