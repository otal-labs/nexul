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

func (f fakeProjects) ProjectActions(ctx context.Context, userID, workspaceID string) ([]MeProject, error) {
	ids, err := f.WorkspaceProjects(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	var out []MeProject
	for _, id := range ids {
		if actions, ok := f.held[userID][id]; ok {
			out = append(out, MeProject{ProjectID: id, Actions: permissions.SetOfStrings(actions)})
		}
	}
	return out, nil
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

func TestListProjectPeople(t *testing.T) {
	f := newAccessFixture(t)
	_, err := f.svc.ListProjectPeople(t.Context(), "bob", "p-api")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a project hidden from the caller")
	_, err = f.svc.ListProjectPeople(t.Context(), "bob", "p-gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	f.repo.members["ws-1"] = append(f.repo.members["ws-1"], "actor", "lena")
	f.repo.roleIDs["ws-1"]["actor"] = []string{"role-editor"}
	f.repo.roleIDs["ws-1"]["lena"] = []string{"role-editor"}
	f.accounts.accounts["lena"] = &TeamAccount{ID: "lena", Login: "lena", Status: "active"}
	f.repo.listed["p-web"] = []string{"bob", "olga", "actor"}
	people, err := f.svc.ListProjectPeople(t.Context(), "bob", "p-web")
	require.NoError(t, err)
	ids := []string{}
	for _, p := range people {
		ids = append(ids, p.UserID)
	}
	assert.Equal(t, []string{"bob", "olga", "actor"}, ids, "lena is not in p-web")

	f.repo.projectMembersErr = errors.New("database is locked")
	_, err = f.svc.ListProjectPeople(t.Context(), "bob", "p-web")
	require.ErrorIs(t, err, f.repo.projectMembersErr)
}

func TestHandler_ProjectPeople_AnswersOnlySomeoneInTheProject(t *testing.T) {
	f := newAccessFixture(t)
	f.repo.listed["p-web"] = []string{"bob", "olga"}
	rec := do(t, NewHandler(f.svc).ProjectPeopleRoutes(), http.MethodGet, "/api/projects/p-api/people", "", "bob")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	rec = do(t, NewHandler(f.svc).ProjectPeopleRoutes(), http.MethodGet, "/api/projects/p-web/people", "", "bob")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"people":[{"user_id":"bob","login":"bob","display_name":"","avatar_url":""},{"user_id":"olga","login":"olga","display_name":"","avatar_url":""}]}`, rec.Body.String())
}

// TestHandler_Me_ReadsEveryNonOwnerAsRestricted: a phone build from before projects were opt-in reads a non-Owner's
// project actions from projects, so permissions carries none and restricted is set; the Owner reads as before.
func TestHandler_Me_ReadsEveryNonOwnerAsRestricted(t *testing.T) {
	f := newAccessFixture(t)
	f.wsPerms.perms["bob"] = []string{"chat:write", "tickets:read"}
	f.repo.roleIDs["ws-1"]["bob"] = []string{"role-editor", "role-viewer"}
	rec := do(t, NewHandler(f.svc).Routes(), http.MethodGet, "/api/workspaces/ws-1/me", "", "bob")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"role_name":"role-name-role-editor, role-name-role-viewer",
		"roles":[{"id":"role-editor","name":"role-name-role-editor","is_owner":false},{"id":"role-viewer","name":"role-name-role-viewer","is_owner":false}],
		"permissions":["chat:write"],"restricted":true,"projects":[{"project_id":"p-web","actions":["tickets:read"]}]}`, rec.Body.String())

	f.wsPerms.perms["olga"] = []string{"chat:write", "tickets:read"}
	rec = do(t, NewHandler(f.svc).Routes(), http.MethodGet, "/api/workspaces/ws-1/me", "", "olga")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"role_name":"role-name-owner-role","roles":[{"id":"owner-role","name":"role-name-owner-role","is_owner":true}],
		"permissions":["chat:write","tickets:read"],"restricted":false,"projects":[]}`, rec.Body.String())
}

func TestListTeam_RemovedFieldsReadAsEmpty(t *testing.T) {
	f := newAccessFixture(t)
	team, err := f.svc.ListTeam(t.Context(), "actor")
	require.NoError(t, err)
	for _, p := range team.People {
		for _, m := range p.Workspaces {
			assert.Empty(t, m.Allow, p.ID)
			assert.Empty(t, m.Deny, p.ID)
			assert.Empty(t, m.EveryProject, p.ID)
			assert.Empty(t, m.Projects, p.ID)
		}
	}
}
