package access

import (
	"context"
	"fmt"
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
	s.SetScopes(fakeScopes{projects: map[string]string{"p-open": "ws", "p-hidden": "ws"}, workspaces: map[string][]string{"team": {"ws"}}, restricted: map[string][]string{"client": {"ws"}, "owner": {"ws"}}})
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

// A hidden project's not found carries no id, so an error reached through a ticket or doc never confirms which
// hidden project holds it (ADR 0087).
func TestRestrictedMember_HiddenNotFoundNamesNoProject(t *testing.T) {
	s, _ := restrictedFixture(t)
	client := identity.WithActor(context.Background(), identity.Actor{ID: "client"})

	err := s.RequireProject(client, "p-hidden", permissions.TicketsRead)

	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.NotContains(t, err.Error(), "p-hidden")
}

// countingRoles counts membership lookups, the per-check cost CanDocs reads once for a whole project.
type countingRoles struct {
	*fakeRoles
	calls int
}

func (c *countingRoles) MemberRole(ctx context.Context, workspaceID, userID string) (RoleInfo, error) {
	c.calls++
	return c.fakeRoles.MemberRole(ctx, workspaceID, userID)
}

func TestCanDocs_AnswersAsASingleCheckDoes(t *testing.T) {
	s, repo := restrictedFixture(t)
	docWorkspaces := &fakeDocWorkspace{
		byDoc:     map[string]string{"d-open": "ws", "d-open-2": "ws", "d-hidden": "ws"},
		projectOf: map[string]string{"d-open": "p-open", "d-open-2": "p-open", "d-hidden": "p-hidden"},
	}
	s.SetDocWorkspaces(docWorkspaces)
	setOverwrite(repo, "doc", "d-hidden", "client", permissions.SetOf(permissions.DocsRead))
	require.NoError(t, repo.Set(t.Context(), "doc", "d-open-2", "team", nil, permissions.SetOf(permissions.DocsRead)))
	setOverwrite(repo, "doc", "d-loose", "stranger", permissions.SetOf(permissions.DocsRead, permissions.DocsWrite))
	docIDs := []string{"d-open", "d-open-2", "d-hidden", "d-loose", "d-gone"}

	for _, user := range []string{"client", "team", "owner", "stranger", ""} {
		for _, action := range []permissions.Action{permissions.DocsRead, permissions.DocsWrite} {
			anywhere := s.CanDocs(t.Context(), user, "", docIDs, action)
			for _, id := range docIDs {
				want := s.HasPermission(t.Context(), user, docWorkspaces.byDoc[id], action, "doc", id)
				assert.Equal(t, want, anywhere[id], "%s %s on %s", user, action, id)
				named := s.CanDocs(t.Context(), user, docWorkspaces.projectOf[id], []string{id}, action)
				assert.Equal(t, want, named[id], "%s %s on %s in its project", user, action, id)
			}
		}
	}
}

func TestCanDocs_ReadsEachLayerOnceForTheLot(t *testing.T) {
	s, repo := restrictedFixture(t)
	roles := &countingRoles{fakeRoles: newFakeRoles()}
	roles.set("ws", "team", RoleInfo{Permissions: permissions.SetOf(permissions.DocsWrite)})
	s.SetRoles(roles)
	docIDs := make([]string, 50)
	byDoc, projectOf := map[string]string{}, map[string]string{}
	for i := range docIDs {
		docIDs[i] = fmt.Sprintf("d-%d", i)
		byDoc[docIDs[i]], projectOf[docIDs[i]] = "ws", []string{"p-open", "p-hidden"}[i%2]
	}
	s.SetDocWorkspaces(&fakeDocWorkspace{byDoc: byDoc, projectOf: projectOf})
	setOverwrite(repo, "doc", "d-49", "team", permissions.SetOf(permissions.DocsRead))
	repo.resetReads()

	got := s.CanDocs(t.Context(), "team", "", docIDs, permissions.DocsRead)

	assert.Len(t, got, 50)
	assert.True(t, got["d-49"])
	assert.False(t, got["d-48"])
	assert.Equal(t, 1, roles.calls, "membership once for the lot")
	assert.Equal(t, readCounts{getMany: 1, get: 1}, repo.readCounts(), "the workspace overwrite, then every doc's in one read")
}
