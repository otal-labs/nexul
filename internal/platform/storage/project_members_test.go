package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

func seedProject(t *testing.T, s *Store, id, workspaceID, prefix string) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, s.Projects.Create(t.Context(), &workspace.Project{ID: id, Name: "Project " + prefix, Prefix: prefix, WorkspaceID: workspaceID, CreatedAt: now, UpdatedAt: now}))
}

// ProjectMemberIDs answers who is in a project as the access check does: the Owner, and anyone its rules list by role
// or by name, unless a rule naming them keeps them out (ADR 0148).
func TestWorkspaceMembers_ProjectMemberIDs_AreWhoMayOpenTheProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-2", "Beta")))
	for _, r := range []*roles.Role{
		{ID: "role-own", WorkspaceID: "workspace-default", Name: "Owner", IsOwnerRole: true},
		{ID: "role-dev", WorkspaceID: "workspace-default", Name: "Developer"},
		{ID: "role-other", WorkspaceID: "workspace-default", Name: "Client"},
		{ID: "role-beta", WorkspaceID: "ws-2", Name: "Member"},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, s.Roles.Create(ctx, r))
	}
	members := []struct{ user, role string }{
		{"u-owner", "role-own"}, {"u-dev", "role-dev"}, {"u-named", "role-other"}, {"u-kept", "role-dev"}, {"u-out", "role-other"},
	}
	for i, m := range members {
		mustCreateUser(t, s, m.user, m.user)
		require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: m.user, WorkspaceID: "workspace-default", RoleID: m.role, CreatedAt: now.Add(time.Duration(i) * time.Second)}))
	}
	mustCreateUser(t, s, "u-beta", "beta")
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-beta", WorkspaceID: "ws-2", RoleID: "role-beta", CreatedAt: now}))
	seedProject(t, s, "p-web", "workspace-default", "WEB")
	seedProject(t, s, "p-api", "workspace-default", "API")
	open := permissions.SetOf(permissions.ProjectsOpen)
	require.NoError(t, s.Access.SetRole(ctx, "project", "p-web", "role-dev", open, nil))
	require.NoError(t, s.Access.Set(ctx, "project", "p-web", "u-named", permissions.SetOf(permissions.ProjectsOpen, permissions.TicketsRead), nil))
	require.NoError(t, s.Access.Set(ctx, "project", "p-web", "u-kept", nil, open))
	require.NoError(t, s.Access.Set(ctx, "project", "p-api", "u-out", open, nil))

	ids, err := s.WorkspaceMembers.ProjectMemberIDs(ctx, "workspace-default", "p-web")

	require.NoError(t, err)
	assert.Equal(t, []string{"u-owner", "u-dev", "u-named"}, ids, "the Owner, a listed role, a person by name; not one kept out")
}
