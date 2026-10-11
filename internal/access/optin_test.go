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

var fixtureRoleSet = permissions.SetOf(permissions.TicketsRead, permissions.TicketsWrite, permissions.DocsRead, permissions.RunnersRead, permissions.ChatWrite)

// projectsFixture is one workspace with two projects: "client" is listed in p-open by name and "team" through its
// role, "nobody" holds no role, "owner" is the Owner, and nobody else is in p-hidden.
func projectsFixture(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	roles := newFakeRoles()
	roles.set("ws", "client", RolesHeld(HeldRole{ID: "role-client", Permissions: fixtureRoleSet}))
	roles.set("ws", "team", RolesHeld(HeldRole{ID: "role-team", Permissions: fixtureRoleSet}))
	roles.set("ws", "nobody", RolesHeld())
	roles.set("ws", "owner", RolesHeld(HeldRole{ID: "role-owner", IsOwner: true}))
	s := newService(repo, newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{projects: map[string]string{"p-open": "ws", "p-hidden": "ws"}, workspaces: map[string][]string{"team": {"ws"}, "client": {"ws"}, "nobody": {"ws"}, "owner": {"ws"}}})
	s.SetDocWorkspaces(&fakeDocWorkspace{byDoc: map[string]string{"d-open": "ws", "d-hidden": "ws"}, projectOf: map[string]string{"d-open": "p-open", "d-hidden": "p-hidden"}})
	setOverwrite(repo, "project", "p-open", "client", permissions.SetOf(permissions.ProjectsOpen))
	require.NoError(t, repo.SetRole(t.Context(), "project", "p-open", "role-team", permissions.SetOf(permissions.ProjectsOpen), nil))
	return s, repo
}

func TestOptIn_Resolver(t *testing.T) {
	s, repo := projectsFixture(t)
	setOverwrite(repo, "doc", "d-hidden", "client", permissions.SetOf(permissions.DocsRead))
	setOverwrite(repo, "doc", "d-open", "client", permissions.SetOf(permissions.DocsWrite))
	as := func(id string) context.Context {
		return identity.WithActor(context.Background(), identity.Actor{ID: id})
	}

	tests := []struct {
		name  string
		check func() error
		want  error
	}{
		{"a project nothing lists them in is not found", func() error {
			return s.RequireProject(as("client"), "p-hidden", permissions.Member)
		}, apperrs.ErrNotFound},
		{"a project action in a project they are not in is not found, whatever the role holds", func() error {
			return s.RequireProject(as("client"), "p-hidden", permissions.TicketsRead)
		}, apperrs.ErrNotFound},
		{"a project action with no project named is refused", func() error {
			return s.Require(as("client"), "ws", permissions.TicketsRead)
		}, apperrs.ErrForbidden},
		{"a rule naming them lets them in", func() error {
			return s.RequireProject(as("client"), "p-open", permissions.Member)
		}, nil},
		{"a rule for a role they hold lets them in", func() error {
			return s.RequireProject(as("team"), "p-open", permissions.Member)
		}, nil},
		{"the role's project actions hold inside a project they are in", func() error {
			return s.RequireProject(as("client"), "p-open", permissions.TicketsWrite)
		}, nil},
		{"a workspace action answers from the role", func() error {
			return s.Require(as("client"), "ws", permissions.ChatWrite)
		}, nil},
		{"an instance action answers from the role anywhere", func() error {
			return s.RequireAnywhere(as("client"), permissions.RunnersRead)
		}, nil},
		{"the Owner is in every project", func() error {
			return s.RequireProject(as("owner"), "p-hidden", permissions.ProjectsDelete)
		}, nil},
		{"no role is still a member", func() error {
			return s.Require(as("nobody"), "ws", permissions.Member)
		}, nil},
		{"no role holds no workspace action", func() error {
			return s.Require(as("nobody"), "ws", permissions.ChatWrite)
		}, apperrs.ErrForbidden},
		{"no role is in no project", func() error {
			return s.RequireProject(as("nobody"), "p-open", permissions.Member)
		}, apperrs.ErrNotFound},
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

	t.Run("a doc's rules sit on top of the project", func(t *testing.T) {
		ok, err := s.Can(context.Background(), "client", "d-open", permissions.DocsWrite)
		require.NoError(t, err)
		assert.True(t, ok)
	})
	t.Run("a doc rule in a project they are not in opens nothing", func(t *testing.T) {
		ok, err := s.Can(context.Background(), "client", "d-hidden", permissions.DocsRead)
		require.NoError(t, err)
		assert.False(t, ok)
	})
	t.Run("the workspace grid is what the roles grant", func(t *testing.T) {
		assert.ElementsMatch(t, fixtureRoleSet.Actions(), permissions.SetOfStrings(s.WorkspacePermissions(context.Background(), "client", "ws")).Actions())
		assert.Empty(t, s.WorkspacePermissions(context.Background(), "nobody", "ws"))
	})
	t.Run("the grid inside a project is the roles' there", func(t *testing.T) {
		actions, opens := s.ProjectPermissions(context.Background(), "team", "p-open")
		assert.True(t, opens)
		assert.ElementsMatch(t, fixtureRoleSet.Actions(), permissions.SetOfStrings(actions).Actions())
		_, opens = s.ProjectPermissions(context.Background(), "team", "p-hidden")
		assert.False(t, opens)
	})
}

func TestOptIn_ProjectRulesApplyInside(t *testing.T) {
	s, repo := projectsFixture(t)
	require.NoError(t, repo.SetRole(t.Context(), "project", "p-open", "role-team", permissions.SetOf(permissions.ProjectsOpen, permissions.ProjectsDelete), nil))
	require.NoError(t, repo.Set(t.Context(), "project", "p-open", "team", nil, permissions.SetOf(permissions.TicketsWrite)))

	held, from, err := s.Decide(t.Context(), "team", "project", "p-open", permissions.ProjectsDelete)
	require.NoError(t, err)
	assert.True(t, held, "a role rule on the project adds what the role lacks")
	assert.Equal(t, src(SourceRule, "project", "p-open", TargetRole, "role-team", StateAllow), from)
	held, from, err = s.Decide(t.Context(), "team", "project", "p-open", permissions.TicketsWrite)
	require.NoError(t, err)
	assert.False(t, held, "a rule naming the person beats the role")
	assert.Equal(t, src(SourceRule, "project", "p-open", TargetPerson, "team", StateDeny), from)
	assert.False(t, s.HasPermission(t.Context(), "team", "ws", permissions.ProjectsDelete, "", ""), "nothing a project's rules give holds outside it")
}

func TestOptIn_KeptOut(t *testing.T) {
	s, repo := projectsFixture(t)
	require.NoError(t, repo.Set(t.Context(), "project", "p-open", "team", nil, permissions.SetOf(permissions.ProjectsOpen)))

	assert.False(t, s.CanInProject(t.Context(), "team", "p-open", permissions.Member), "denied by name, kept out whatever the role")
	assert.True(t, s.CanInProject(t.Context(), "client", "p-open", permissions.Member))
	held, from, err := s.Decide(t.Context(), "team", "project", "p-open", permissions.ProjectsOpen)
	require.NoError(t, err)
	assert.False(t, held)
	assert.Equal(t, SourceNotInProject, from.Kind)
}

// A project's not found carries no id, so an error reached through a ticket or doc never confirms which project
// holds it (ADR 0087).
func TestOptIn_NotFoundNamesNoProject(t *testing.T) {
	s, _ := projectsFixture(t)
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
	s, repo := projectsFixture(t)
	docWorkspaces := &fakeDocWorkspace{
		byDoc:     map[string]string{"d-open": "ws", "d-open-2": "ws", "d-hidden": "ws"},
		projectOf: map[string]string{"d-open": "p-open", "d-open-2": "p-open", "d-hidden": "p-hidden"},
	}
	s.SetDocWorkspaces(docWorkspaces)
	setOverwrite(repo, "doc", "d-hidden", "client", permissions.SetOf(permissions.DocsRead))
	require.NoError(t, repo.Set(t.Context(), "doc", "d-open-2", "team", nil, permissions.SetOf(permissions.DocsRead)))
	setOverwrite(repo, "doc", "d-loose", "stranger", permissions.SetOf(permissions.DocsRead, permissions.DocsWrite))
	docIDs := []string{"d-open", "d-open-2", "d-hidden", "d-loose", "d-gone"}

	for _, user := range []string{"client", "team", "nobody", "owner", "stranger", ""} {
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
	s, repo := projectsFixture(t)
	roles := &countingRoles{fakeRoles: newFakeRoles()}
	roles.set("ws", "team", RolesHeld(HeldRole{ID: "role-team", Permissions: permissions.SetOf(permissions.DocsWrite)}))
	s.SetRoles(roles)
	docIDs := make([]string, 50)
	byDoc, projectOf := map[string]string{}, map[string]string{}
	for i := range docIDs {
		docIDs[i] = fmt.Sprintf("d-%d", i)
		byDoc[docIDs[i]], projectOf[docIDs[i]] = "ws", []string{"p-open", "p-hidden"}[i%2]
	}
	s.SetDocWorkspaces(&fakeDocWorkspace{byDoc: byDoc, projectOf: projectOf})
	setOverwrite(repo, "doc", "d-48", "team", permissions.SetOf(permissions.DocsRead))
	setOverwrite(repo, "doc", "d-49", "team", permissions.SetOf(permissions.DocsRead))
	repo.resetReads()

	got := s.CanDocs(t.Context(), "team", "", docIDs, permissions.DocsRead)

	assert.Len(t, got, 50)
	assert.True(t, got["d-48"])
	assert.False(t, got["d-49"], "a doc rule in a project they are not in opens nothing")
	assert.False(t, got["d-46"])
	assert.Equal(t, 1, roles.calls, "membership once for the lot")
	assert.Equal(t, 3, repo.readCounts(), "their rules on the workspace and both projects, the roles' there, then every doc's")
}

// fakeRoleWorkspaces resolves the fixture's roles to their workspace.
type fakeRoleWorkspaces map[string]string

func (f fakeRoleWorkspaces) WorkspaceIDForRole(_ context.Context, roleID string) (string, error) {
	ws, ok := f[roleID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

func TestSetRules_OnAProject(t *testing.T) {
	s, repo := projectsFixture(t)
	s.SetRoleWorkspaces(fakeRoleWorkspaces{"role-client": "ws", "role-team": "ws", "role-elsewhere": "ws-2"})
	open := []permissions.Action{permissions.ProjectsOpen}
	toClients := Targets{RoleIDs: []string{"role-client"}}

	require.ErrorIs(t, s.SetRules(t.Context(), "team", "project", []string{"p-hidden"}, toClients, open, StateAllow), apperrs.ErrNotFound,
		"a project they are not in")
	require.ErrorIs(t, s.SetRules(t.Context(), "team", "project", []string{"p-open"}, toClients, open, StateAllow), apperrs.ErrForbidden,
		"writing a project's rules takes projects:access on it")
	before := len(repo.events)
	require.NoError(t, s.SetRules(t.Context(), "owner", "project", []string{"p-open"}, Targets{UserIDs: []string{"nobody"}}, open, StateFallback))
	assert.Len(t, repo.events, before, "Fallback where no rule is changes nothing")
	require.NoError(t, s.SetRules(t.Context(), "owner", "project", []string{"p-hidden"}, toClients, open, StateAllow))
	assert.True(t, s.CanInProject(t.Context(), "client", "p-hidden", permissions.Member), "a rule for their role lets them in")
	assert.Equal(t, TargetRole, repo.events[len(repo.events)-1].Payload.(GrantEvent).Target)
	assert.Equal(t, "role-client", repo.events[len(repo.events)-1].Payload.(GrantEvent).RoleID)

	require.NoError(t, repo.SetRole(t.Context(), "project", "p-open", "role-team", permissions.SetOf(permissions.ProjectsOpen, permissions.ProjectsAccess), nil))
	require.NoError(t, s.SetRules(t.Context(), "team", "project", []string{"p-open"}, Targets{UserIDs: []string{"nobody"}}, open, StateAllow),
		"listing someone gives nothing the lister lacks")
	require.ErrorIs(t, s.SetRules(t.Context(), "team", "project", []string{"p-open"}, toClients, []permissions.Action{permissions.ProjectsDelete}, StateAllow),
		apperrs.ErrForbidden, "nobody allows what they don't hold there")
	require.NoError(t, s.SetRules(t.Context(), "team", "project", []string{"p-open"}, Targets{UserIDs: []string{"client"}}, open, StateDeny),
		"keeping someone out is a Deny, never limited")
	assert.False(t, s.CanInProject(t.Context(), "client", "p-open", permissions.Member))

	tests := []struct {
		name         string
		resourceType string
		to           Targets
		actions      []permissions.Action
		want         error
	}{
		{"a role of another workspace", "project", Targets{RoleIDs: []string{"role-elsewhere"}}, open, apperrs.ErrNotFound},
		{"an unknown role", "project", Targets{RoleIDs: []string{"role-gone"}}, open, apperrs.ErrNotFound},
		{"In this project on a doc", "doc", toClients, open, apperrs.ErrInvalid},
		{"an action a project's rules don't set", "project", toClients, []permissions.Action{permissions.PlaysRun}, apperrs.ErrInvalid},
		{"no target", "project", Targets{}, open, apperrs.ErrInvalid},
		{"an entity rules don't sit on", "ticket", toClients, open, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, s.SetRules(t.Context(), "owner", tt.resourceType, []string{"p-open"}, tt.to, tt.actions, StateAllow), tt.want)
		})
	}
}

// TestProjectActions_ListsOnlyTheProjectsSomeoneIsInWithWhatTheyHoldThere feeds /me: the rules on each project shape
// its actions, and a project nothing lists them in is left out.
func TestProjectActions_ListsOnlyTheProjectsSomeoneIsInWithWhatTheyHoldThere(t *testing.T) {
	s, repo := projectsFixture(t)
	require.NoError(t, repo.Set(t.Context(), "project", "p-open", "client", permissions.SetOf(permissions.ProjectsOpen), permissions.SetOf(permissions.TicketsWrite)))

	client, err := s.ProjectActions(t.Context(), "client", "ws")
	require.NoError(t, err)
	assert.Equal(t, []ProjectHeld{{ProjectID: "p-open", Actions: permissions.SetOf(permissions.DocsRead, permissions.TicketsRead)}}, client)

	nobody, err := s.ProjectActions(t.Context(), "nobody", "ws")
	require.NoError(t, err)
	assert.Empty(t, nobody)

	owner, err := s.ProjectActions(t.Context(), "owner", "ws")
	require.NoError(t, err)
	require.Len(t, owner, 2)
	assert.True(t, owner[0].Actions.Has(permissions.ProjectsDelete))
}
