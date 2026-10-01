package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// restrictedFixture is one workspace with two projects: "client" is a Restricted member holding Project access to
// p-open only, "team" is an unrestricted member with the same role, and "owner" is the Owner.
func restrictedFixture(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	roleSet := permissions.SetOf(permissions.TicketsRead, permissions.TicketsWrite, permissions.DocsRead, permissions.RunnersRead, permissions.ChatWrite)
	roles := newFakeRoles()
	roles.set("ws", "client", RoleInfo{Permissions: roleSet, Restricted: true})
	roles.set("ws", "team", RoleInfo{Permissions: roleSet})
	roles.set("ws", "owner", RoleInfo{IsOwnerRole: true, Restricted: true})
	s := newService(repo, newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{projects: map[string]string{"p-open": "ws", "p-hidden": "ws"}, workspaces: map[string][]string{"team": {"ws"}}})
	s.SetDocWorkspaces(&fakeDocWorkspace{byDoc: map[string]string{"d-open": "ws", "d-hidden": "ws"}, projectOf: map[string]string{"d-open": "p-open", "d-hidden": "p-hidden"}})
	setOverwrite(repo, "project", "p-open", "client", permissions.SetOf(permissions.TicketsRead, permissions.DocsRead))
	return s, repo
}

func TestRestrictedMember_Resolver(t *testing.T) {
	s, repo := restrictedFixture(t)
	setOverwrite(repo, "doc", "d-hidden", "client", permissions.SetOf(permissions.DocsRead))
	setOverwrite(repo, "doc", "d-open", "client", permissions.SetOf(permissions.DocsWrite))
	setOverwrite(repo, "project", "p-hidden", "team", permissions.SetOf(permissions.ProjectsDelete))
	client := identity.WithActor(context.Background(), identity.Actor{ID: "client"})
	team := identity.WithActor(context.Background(), identity.Actor{ID: "team"})
	owner := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	tests := []struct {
		name  string
		check func() error
		want  error
	}{
		{"a project they hold no access to is not found", func() error {
			return s.RequireProject(client, "p-hidden", permissions.Member)
		}, apperrs.ErrNotFound},
		{"a project action in a hidden project is not found, whatever the role holds", func() error {
			return s.RequireProject(client, "p-hidden", permissions.TicketsRead)
		}, apperrs.ErrNotFound},
		{"a project action with no project is refused", func() error {
			return s.Require(client, "ws", permissions.TicketsRead)
		}, apperrs.ErrForbidden},
		{"an instance action is refused in the workspace", func() error {
			return s.Require(client, "ws", permissions.RunnersRead)
		}, apperrs.ErrForbidden},
		{"an instance action is refused inside an open project", func() error {
			return s.RequireProject(client, "p-open", permissions.RunnersRead)
		}, apperrs.ErrForbidden},
		{"the role adds nothing inside an open project", func() error {
			return s.RequireProject(client, "p-open", permissions.TicketsWrite)
		}, apperrs.ErrForbidden},
		{"Project access answers inside an open project", func() error {
			return s.RequireProject(client, "p-open", permissions.TicketsRead)
		}, nil},
		{"they may open a project they hold access to", func() error {
			return s.RequireProject(client, "p-open", permissions.Member)
		}, nil},
		{"a workspace action answers from the role", func() error {
			return s.Require(client, "ws", permissions.ChatWrite)
		}, nil},
		{"the Owner is never held back", func() error {
			return s.RequireProject(owner, "p-hidden", permissions.ProjectsDelete)
		}, nil},
		{"an unrestricted member answers from the role in every project", func() error {
			return s.RequireProject(team, "p-hidden", permissions.TicketsWrite)
		}, nil},
		{"an unrestricted member's stored Project access adds nothing", func() error {
			return s.RequireProject(team, "p-hidden", permissions.ProjectsDelete)
		}, apperrs.ErrForbidden},
		{"an unrestricted member keeps instance actions anywhere", func() error {
			return s.RequireAnywhere(team, permissions.RunnersRead)
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.check()
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}

	t.Run("a doc's overwrite sits on top of the project layer", func(t *testing.T) {
		ok, err := s.Can(context.Background(), "client", "d-open", permissions.DocsWrite)
		require.NoError(t, err)
		assert.True(t, ok)
		ok, err = s.Can(context.Background(), "client", "d-open", permissions.DocsRead)
		require.NoError(t, err)
		assert.True(t, ok, "the project layer grants what the doc overwrite does not name")
	})
	t.Run("a doc allow in a hidden project opens nothing", func(t *testing.T) {
		ok, err := s.Can(context.Background(), "client", "d-hidden", permissions.DocsRead)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.False(t, s.HasPermission(context.Background(), "client", "ws", permissions.DocsRead, "doc", "d-hidden"))
	})
	t.Run("the grid with no project holds no project or instance area", func(t *testing.T) {
		assert.Equal(t, []string{string(permissions.ChatWrite)}, s.WorkspacePermissions(context.Background(), "client", "ws"))
	})
	t.Run("the grid inside a project is Project access plus workspace areas", func(t *testing.T) {
		actions, opens := s.ProjectPermissions(context.Background(), "client", "p-open")
		assert.True(t, opens)
		assert.ElementsMatch(t, []string{"docs:read", "tickets:read", "chat:write"}, actions)
		_, opens = s.ProjectPermissions(context.Background(), "client", "p-hidden")
		assert.False(t, opens)
	})
}

func TestRestrictedMember_NoRowsSeesNoProject(t *testing.T) {
	s, repo := restrictedFixture(t)
	require.NoError(t, repo.Set(context.Background(), "project", "p-open", "client", nil, nil))
	for _, projectID := range []string{"p-open", "p-hidden"} {
		assert.False(t, s.CanInProject(context.Background(), "client", projectID, permissions.Member), projectID)
	}
	assert.True(t, s.CanInProject(context.Background(), "team", "p-hidden", permissions.Member))
	assert.False(t, s.CanInProject(context.Background(), "client", "p-unknown", permissions.Member))
	assert.False(t, s.CanInProject(context.Background(), "", "p-open", permissions.Member))
}
