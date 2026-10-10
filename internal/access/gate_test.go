package access

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type fakeScopes struct {
	projects   map[string]string
	workspaces map[string][]string
	// restricted lists each person's Restricted memberships, which only WorkspaceIDsForUser counts.
	restricted map[string][]string
}

func (f fakeScopes) WorkspaceIDForProject(_ context.Context, projectID string) (string, error) {
	ws, ok := f.projects[projectID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

func (f fakeScopes) ProjectIDs(_ context.Context, workspaceID string) ([]string, error) {
	var out []string
	for projectID, ws := range f.projects {
		if ws == workspaceID {
			out = append(out, projectID)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (f fakeScopes) UnrestrictedWorkspaceIDsForUser(_ context.Context, userID string) ([]string, error) {
	return f.workspaces[userID], nil
}

func (f fakeScopes) WorkspaceIDsForUser(_ context.Context, userID string) ([]string, error) {
	return append(f.workspaces[userID], f.restricted[userID]...), nil
}

// TestRequire covers what the gate adds over HasPermission: who is let through without a person to resolve, the
// not-found answer a stranger gets, and the instance-level check across a caller's workspaces.
func TestRequire(t *testing.T) {
	roles := newFakeRoles()
	roles.set("ws-a", "alice", RoleInfo{})
	roles.set("ws-b", "alice", RoleInfo{Permissions: permissions.SetOf(permissions.RunnersRead)})
	s := newService(newFakeRepo(), newFakeUsers())
	s.SetRoles(roles)
	roles.set("ws-b", "olga", RoleInfo{IsOwnerRole: true})
	s.SetScopes(fakeScopes{projects: map[string]string{"p-1": "ws-a"}, workspaces: map[string][]string{"alice": {"ws-a", "ws-b"}, "olga": {"ws-b"}}})
	alice := identity.WithActor(context.Background(), identity.Actor{ID: "alice"})
	defaultAutomation := identity.WithActor(context.Background(), identity.Actor{Automation: &identity.AutomationRef{ID: "a-1", WorkspaceID: "ws-a"}})

	tests := []struct {
		name  string
		check func() error
		want  error
	}{
		{"the server's own call passes", func() error {
			return s.Require(context.Background(), "ws-a", permissions.TicketsRead)
		}, nil},
		{"a shipped default automation passes on its gateway-checked scopes", func() error {
			return s.RequireProject(defaultAutomation, "p-1", permissions.TicketsWrite)
		}, nil},
		{"a shipped default automation is held to its own workspace", func() error {
			return s.Require(defaultAutomation, "ws-b", permissions.TicketsRead)
		}, apperrs.ErrNotFound},
		{"a shipped default automation cannot reach another workspace's project", func() error {
			s.SetScopes(fakeScopes{projects: map[string]string{"p-1": "ws-a", "p-2": "ws-b"}, workspaces: map[string][]string{"alice": {"ws-a", "ws-b"}, "olga": {"ws-b"}}})
			return s.RequireProject(defaultAutomation, "p-2", permissions.TicketsWrite)
		}, apperrs.ErrNotFound},
		{"an actor with no id is not a member", func() error {
			return s.Require(identity.WithActor(context.Background(), identity.Actor{}), "ws-a", permissions.Member)
		}, apperrs.ErrNotFound},
		{"membership alone passes the member check", func() error {
			return s.Require(alice, "ws-a", permissions.Member)
		}, nil},
		{"an unknown project is not found", func() error {
			return s.RequireProject(alice, "p-gone", permissions.Member)
		}, apperrs.ErrNotFound},
		{"an instance-level read held in any of the caller's workspaces passes", func() error {
			return s.RequireAnywhere(alice, permissions.RunnersRead)
		}, nil},
		{"an instance-level action held nowhere is forbidden", func() error {
			return s.RequireAnywhere(alice, permissions.TopologyRead)
		}, apperrs.ErrForbidden},
		{"an Owner of any workspace holds every instance-level action", func() error {
			return s.RequireAnywhere(identity.WithActor(context.Background(), identity.Actor{ID: "olga"}), permissions.InstanceWrite)
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.check()
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

// TestPermissionsAnywhere pins what /me reports as the caller's instance-level grid: the union across every
// workspace they belong to, so a client shows the same instance areas the server lets through.
func TestPermissionsAnywhere(t *testing.T) {
	roles := newFakeRoles()
	roles.set("ws-a", "alice", RoleInfo{Permissions: permissions.SetOf(permissions.RunnersRead)})
	roles.set("ws-b", "alice", RoleInfo{Permissions: permissions.SetOf(permissions.AccountsRead, permissions.DNSRead)})
	roles.set("ws-b", "olga", RoleInfo{IsOwnerRole: true})
	s := newService(newFakeRepo(), newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{workspaces: map[string][]string{"alice": {"ws-a", "ws-b"}, "olga": {"ws-b"}}})

	got, err := s.PermissionsAnywhere(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, []string{"runners:read", "dns:read", "accounts:read"}, got)

	got, err = s.PermissionsAnywhere(context.Background(), "olga")
	require.NoError(t, err)
	assert.Len(t, got, len(permissions.AllActions()))

	got, err = s.PermissionsAnywhere(context.Background(), "stranger")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestWorkspacesWith: the workspaces holding the action, never a restricted one, and none for the server's own call.
func TestWorkspacesWith(t *testing.T) {
	roles := newFakeRoles()
	roles.set("ws-a", "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite)})
	roles.set("ws-b", "alice", RoleInfo{Permissions: permissions.SetOf(permissions.TicketsRead)})
	roles.set("ws-c", "alice", RoleInfo{Permissions: permissions.SetOf(permissions.ProjectsWrite)})
	s := newService(newFakeRepo(), newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{workspaces: map[string][]string{"alice": {"ws-a", "ws-b"}}, restricted: map[string][]string{"alice": {"ws-c"}}})

	got, err := s.WorkspacesWith(identity.WithActor(context.Background(), identity.Actor{ID: "alice"}), permissions.ProjectsWrite)
	require.NoError(t, err)
	assert.Equal(t, []string{"ws-a"}, got)

	got, err = s.WorkspacesWith(context.Background(), permissions.ProjectsWrite)
	require.NoError(t, err)
	assert.Empty(t, got)
}
