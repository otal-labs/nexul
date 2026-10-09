package tenancy

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeProjects answers where a project lives (projectWorkspace) and what each person holds inside it; a project
// missing from held is hidden from that person.
type fakeProjects struct {
	held    map[string]map[string][]string // userID -> projectID -> actions
	listErr error
}

func (f fakeProjects) ProjectWorkspace(_ context.Context, projectID string) (string, error) {
	ws, ok := projectWorkspace[projectID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return ws, nil
}

func (f fakeProjects) WorkspaceProjects(_ context.Context, workspaceID string) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var ids []string
	for projectID, ws := range projectWorkspace {
		if ws == workspaceID {
			ids = append(ids, projectID)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

func (f fakeProjects) ProjectPermissions(_ context.Context, userID, projectID string) ([]string, bool) {
	actions, ok := f.held[userID][projectID]
	return actions, ok
}

// newAccessFixture has "actor" managing ws-1 and holding tickets:read and tickets:write on p-web, and "bob" a member
// of ws-1 on role-editor, which reads tickets.
func newAccessFixture(t *testing.T) *inviteTestFixture {
	t.Helper()
	f := newInviteFixture()
	f.grantManageMembers("actor")
	f.nameGate.perms["role-editor"] = permissions.SetOf(permissions.TicketsRead, permissions.ChatWrite)
	f.nameGate.isOwner["owner-role"] = true
	require.NoError(t, f.repo.AddMember(t.Context(), &Member{UserID: "bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
	require.NoError(t, f.repo.AddMember(t.Context(), &Member{UserID: "olga", WorkspaceID: "ws-1", RoleID: "owner-role"}))
	for _, id := range []string{"actor", "bob", "olga"} {
		f.accounts.accounts[id] = &TeamAccount{ID: id, Login: id, Status: "active"}
	}
	f.repo.teamWorkspaces = []*TeamWorkspace{{ID: "ws-1", Name: "Acme"}}
	f.svc.SetProjects(fakeProjects{held: map[string]map[string][]string{
		"actor": {"p-web": {"tickets:read", "tickets:write", "members:write"}, "p-other": {"tickets:read"}},
		"bob":   {"p-web": {"tickets:read"}},
	}})
	return f
}

func TestSetProjectAccess(t *testing.T) {
	ticketsWrite := permissions.SetOf(permissions.TicketsRead, permissions.TicketsWrite)
	tests := []struct {
		name              string
		actor, user, proj string
		allow             permissions.Set
		want              error
	}{
		{"a workspace area is not a project level", "actor", "bob", "p-web", permissions.SetOf(permissions.MembersWrite), apperrs.ErrInvalid},
		{"an unknown action is invalid", "actor", "bob", "p-web", permissions.SetOf("tickets:fly"), apperrs.ErrInvalid},
		{"a giver without members:write is refused", "bob", "bob", "p-web", ticketsWrite, apperrs.ErrForbidden},
		{"a project of another workspace is not found", "actor", "bob", "p-other", ticketsWrite, apperrs.ErrNotFound},
		{"an unknown project is not found", "actor", "bob", "p-gone", ticketsWrite, apperrs.ErrNotFound},
		{"a project hidden from the giver is not found", "actor", "bob", "p-api", ticketsWrite, apperrs.ErrNotFound},
		{"the Owner holds no Project access", "actor", "olga", "p-web", ticketsWrite, apperrs.ErrInvalid},
		{"a non-member is not found", "actor", "zed", "p-web", ticketsWrite, apperrs.ErrNotFound},
		{"a level the giver does not hold is refused", "actor", "bob", "p-web", permissions.SetOf(permissions.TicketsDelete), apperrs.ErrForbidden},
		{"a level the giver holds is set", "actor", "bob", "p-web", ticketsWrite, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAccessFixture(t)
			if tt.actor == "bob" {
				f.wsPerms.perms["bob"] = []string{}
			}
			err := f.svc.SetProjectAccess(t.Context(), tt.actor, "ws-1", tt.user, tt.proj, tt.allow)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Empty(t, f.repo.events)
				return
			}
			require.NoError(t, err)
			rows, err := f.repo.ProjectAccess(t.Context(), "ws-1", tt.user)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, tt.allow, rows[0].Allow)
			require.Len(t, f.repo.events, 1)
			assert.Equal(t, TopicProjectAccessChanged, f.repo.events[0].Topic)
			assert.Equal(t, ProjectAccessEvent{ResourceType: "project", ResourceID: tt.proj, UserID: tt.user, ActorID: tt.actor}, f.repo.events[0].Payload)
		})
	}

	t.Run("a level already held stays without the giver holding it, and an empty allow takes the project away", func(t *testing.T) {
		f := newAccessFixture(t)
		require.NoError(t, f.repo.SetProjectAccess(t.Context(), "p-web", "bob", permissions.SetOf(permissions.TicketsDelete)))
		require.NoError(t, f.svc.SetProjectAccess(t.Context(), "actor", "ws-1", "bob", "p-web", permissions.SetOf(permissions.TicketsDelete, permissions.TicketsRead)))
		require.NoError(t, f.svc.SetProjectAccess(t.Context(), "actor", "ws-1", "bob", "p-web", nil))
		rows, err := f.repo.ProjectAccess(t.Context(), "ws-1", "bob")
		require.NoError(t, err)
		assert.Empty(t, rows)
	})
}

func TestSetEveryProject(t *testing.T) {
	t.Run("an unknown value is invalid", func(t *testing.T) {
		f := newAccessFixture(t)
		require.ErrorIs(t, f.svc.SetEveryProject(t.Context(), "actor", "ws-1", "bob", "some"), apperrs.ErrInvalid)
	})
	t.Run("a giver without members:write is refused", func(t *testing.T) {
		f := newAccessFixture(t)
		require.ErrorIs(t, f.svc.SetEveryProject(t.Context(), "bob", "ws-1", "bob", EveryProjectNone), apperrs.ErrForbidden)
	})
	t.Run("the Owner can never be restricted", func(t *testing.T) {
		f := newAccessFixture(t)
		require.ErrorIs(t, f.svc.SetEveryProject(t.Context(), "actor", "ws-1", "olga", EveryProjectNone), apperrs.ErrInvalid)
	})
	t.Run("From role needs the role's project areas with no project", func(t *testing.T) {
		f := newAccessFixture(t)
		f.repo.restricted["ws-1/bob"] = true
		require.ErrorIs(t, f.svc.SetEveryProject(t.Context(), "actor", "ws-1", "bob", EveryProjectRole), apperrs.ErrForbidden,
			"tickets:read is a project area the giver holds only inside a project")
		assert.True(t, f.repo.restricted["ws-1/bob"])
	})
	t.Run("switching both ways keeps the stored levels and says so on the member event", func(t *testing.T) {
		f := newAccessFixture(t)
		f.wsPerms.perms["actor"] = []string{"members:write", "tickets:read"}
		require.NoError(t, f.repo.SetProjectAccess(t.Context(), "p-web", "bob", permissions.SetOf(permissions.TicketsRead)))
		require.NoError(t, f.svc.SetEveryProject(t.Context(), "actor", "ws-1", "bob", EveryProjectNone))
		assert.True(t, f.repo.restricted["ws-1/bob"])
		require.NoError(t, f.svc.SetEveryProject(t.Context(), "actor", "ws-1", "bob", EveryProjectRole))
		assert.False(t, f.repo.restricted["ws-1/bob"])
		rows, err := f.repo.ProjectAccess(t.Context(), "ws-1", "bob")
		require.NoError(t, err)
		assert.Len(t, rows, 1, "a round trip loses nothing")
		topics := []string{}
		for _, e := range f.repo.events {
			topics = append(topics, e.Topic)
		}
		assert.Equal(t, []string{TopicWorkspaceMemberUpdated, TopicWorkspaceMemberUpdated}, topics)
		assert.Equal(t, []string{"p-api", "p-web"}, f.repo.events[0].Payload.(MemberEvent).ProjectIDs, "the event names the projects the switch opens or closes")
	})
	t.Run("a failed project lookup changes nothing", func(t *testing.T) {
		f := newAccessFixture(t)
		f.wsPerms.perms["actor"] = []string{"members:write", "tickets:read"}
		f.svc.SetProjects(fakeProjects{listErr: errors.New("disk gone")})
		require.Error(t, f.svc.SetEveryProject(t.Context(), "actor", "ws-1", "bob", EveryProjectNone))
		assert.False(t, f.repo.restricted["ws-1/bob"])
		assert.Empty(t, f.repo.events)
	})
	t.Run("an unchanged row publishes nothing", func(t *testing.T) {
		f := newAccessFixture(t)
		require.NoError(t, f.svc.SetEveryProject(t.Context(), "actor", "ws-1", "bob", EveryProjectRole))
		assert.Empty(t, f.repo.events)
	})
}

func TestMemberProjectsAndTeam(t *testing.T) {
	f := newAccessFixture(t)
	f.repo.restricted["ws-1/bob"] = true
	require.NoError(t, f.repo.SetProjectAccess(t.Context(), "p-web", "bob", permissions.SetOf(permissions.TicketsRead)))
	require.NoError(t, f.repo.SetProjectAccess(t.Context(), "p-api", "bob", permissions.SetOf(permissions.DocsRead)))

	restricted, projects, err := f.svc.MemberProjects(t.Context(), "ws-1", "bob")
	require.NoError(t, err)
	assert.True(t, restricted)
	assert.ElementsMatch(t, []MeProject{{ProjectID: "p-web", Actions: permissions.SetOf(permissions.TicketsRead)}, {ProjectID: "p-api", Actions: permissions.SetOf(permissions.DocsRead)}}, projects)
	restricted, projects, err = f.svc.MemberProjects(t.Context(), "ws-1", "olga")
	require.NoError(t, err)
	assert.False(t, restricted)
	assert.Empty(t, projects)
	_, _, err = f.svc.MemberProjects(t.Context(), "ws-1", "zed")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	team, err := f.svc.ListTeam(t.Context(), "actor")
	require.NoError(t, err)
	var bob *TeamMembership
	for _, p := range team.People {
		for _, m := range p.Workspaces {
			if p.ID == "bob" {
				bob = m
			}
		}
	}
	require.NotNil(t, bob)
	assert.Equal(t, EveryProjectNone, bob.EveryProject)
	require.Len(t, bob.Projects, 1, "the viewer cannot open p-api, so the Team never names it")
	assert.Equal(t, "p-web", bob.Projects[0].ProjectID)
}

func TestListProjectPeople(t *testing.T) {
	f := newAccessFixture(t)
	_, err := f.svc.ListProjectPeople(t.Context(), "bob", "p-api")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a project hidden from the caller")
	_, err = f.svc.ListProjectPeople(t.Context(), "bob", "p-gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	f.repo.members["ws-1"] = append(f.repo.members["ws-1"], "actor")
	f.repo.roleIDs["ws-1"]["actor"] = "role-editor"
	people, err := f.svc.ListProjectPeople(t.Context(), "bob", "p-web")
	require.NoError(t, err)
	ids := []string{}
	for _, p := range people {
		ids = append(ids, p.UserID)
	}
	assert.ElementsMatch(t, []string{"bob", "actor"}, ids, "olga is in the workspace but the fake hides p-web from her")
}

func TestHandler_ProjectAccess(t *testing.T) {
	t.Run("a PATCH sets Every project, then each project's levels", func(t *testing.T) {
		f := newAccessFixture(t)
		rec := do(t, NewHandler(f.svc).Routes(), http.MethodPatch, "/api/workspaces/ws-1/members/bob",
			`{"every_project":"none","project_access":[{"project_id":"p-web","allow":["tickets:read","tickets:write"]}]}`, "actor")
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.True(t, f.repo.restricted["ws-1/bob"])
		assert.Len(t, f.repo.access["ws-1/bob"], 1)

		rec = do(t, NewHandler(f.svc).Routes(), http.MethodGet, "/api/workspaces/ws-1/me", "", "bob")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"role_name":"role-name-role-editor","permissions":[],"restricted":true,"projects":[{"project_id":"p-web","actions":["tickets:read","tickets:write"]}]}`, rec.Body.String())
	})
	t.Run("a PATCH stops at the first refusal", func(t *testing.T) {
		f := newAccessFixture(t)
		rec := do(t, NewHandler(f.svc).Routes(), http.MethodPatch, "/api/workspaces/ws-1/members/bob",
			`{"project_access":[{"project_id":"p-web","allow":["tickets:delete"]}]}`, "actor")
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Empty(t, f.repo.access["ws-1/bob"])
	})
	t.Run("project people answers only someone who may open the project", func(t *testing.T) {
		f := newAccessFixture(t)
		rec := do(t, NewHandler(f.svc).ProjectPeopleRoutes(), http.MethodGet, "/api/projects/p-api/people", "", "bob")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		rec = do(t, NewHandler(f.svc).ProjectPeopleRoutes(), http.MethodGet, "/api/projects/p-web/people", "", "bob")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"people":[{"user_id":"bob","login":"bob","display_name":"","avatar_url":""}]}`, rec.Body.String())
	})
}
