package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// testAccountGate mirrors server/cmd's accountGate over the real users and sessions tables; online stands in for the live sockets.
type testAccountGate struct {
	users    *storage.UsersRepo
	sessions *storage.SessionsRepo
	online   map[string]bool
}

func (g testAccountGate) Presence(ctx context.Context) (map[string]bool, map[string]time.Time, error) {
	if g.sessions == nil {
		return g.online, nil, nil
	}
	seen, err := g.sessions.LastActiveByUser(ctx)
	return g.online, seen, err
}

func (g testAccountGate) ListAccounts(ctx context.Context) ([]*tenancy.TeamAccount, error) {
	users, err := g.users.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*tenancy.TeamAccount, 0, len(users))
	for _, u := range users {
		out = append(out, testTeamAccount(u))
	}
	return out, nil
}

func (g testAccountGate) Account(ctx context.Context, userID string) (*tenancy.TeamAccount, error) {
	u, err := g.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return testTeamAccount(u), nil
}

func testTeamAccount(u *auth.User) *tenancy.TeamAccount {
	a := &tenancy.TeamAccount{ID: u.ID, Login: u.Login, Name: u.Name, AvatarURL: u.AvatarURL, Status: string(u.AccountStatus)}
	if u.DisplayName != nil {
		a.DisplayName = *u.DisplayName
	}
	if u.AvatarOverrideURL != nil {
		a.AvatarOverride = *u.AvatarOverrideURL
	}
	return a
}

// testCreatorGate lets anyone create a workspace, since the fixture starts with nobody holding anything, and asks
// access, like server/cmd does, for every other instance-level action.
type testCreatorGate struct{ access *access.Service }

func (g testCreatorGate) HoldsAnywhere(ctx context.Context, userID string, action permissions.Action) (bool, error) {
	if action == permissions.WorkspacesCreate {
		return true, nil
	}
	return g.access.HoldsAnywhere(ctx, userID, action)
}

type teamFixture struct {
	svc          *tenancy.Service
	roles        *roles.Service
	nexul, acme  *tenancy.Workspace
	editor       *roles.Role
	viewer       *roles.Role
	nexulOwnerID string
	users        *storage.UsersRepo
	sessions     *storage.SessionsRepo
	online       map[string]bool
}

// newTeamFixture: admin owns Nexul and dana owns Acme, so both hold every instance-level permission; admin is only
// a Viewer in dana's Acme.
func newTeamFixture(t *testing.T) teamFixture {
	t.Helper()
	ctx := t.Context()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, realUsers{s.Users})
	rolesSvc := roles.NewService(s.Roles, nil)
	online := map[string]bool{}
	svc := tenancy.NewService(s.Workspaces, s.WorkspaceMembers, s.WorkspaceInvites, testRoleGate{svc: rolesSvc}, testCreatorGate{access: accessSvc}, testRoleNameGate{svc: rolesSvc}, testWorkspacePermissionGate{svc: accessSvc}, testAllowlistGate{}, testUserLookupGate{}, testChannelGate{}, testPlaysGate{}, testAccountGate{users: s.Users, sessions: s.Sessions, online: online})
	rolesSvc.SetMemberGate(testMemberGate{svc: svc})
	accessSvc.SetRoles(accessRoleResolver{tenancy: svc, roles: rolesSvc})
	accessSvc.SetScopes(testScopes{s: s})
	rolesSvc.SetPermissionGate(testWorkspacePermissionGate{svc: accessSvc})

	seedUser(t, s, "admin", "admin")
	seedUser(t, s, "dana", "dana")
	seedUser(t, s, "bob", "bob")
	seedUser(t, s, "carol", "carol")

	nexul, err := svc.Create(ctx, "admin", "Nexul")
	require.NoError(t, err)
	acme, err := svc.Create(ctx, "dana", "Acme")
	require.NoError(t, err)
	editor, err := rolesSvc.Create(ctx, nexul.ID, "admin", "Editor", permissions.SetOf(permissions.DocsWrite))
	require.NoError(t, err)
	viewer, err := rolesSvc.Create(ctx, acme.ID, "dana", "Viewer", permissions.SetOf(permissions.DocsRead))
	require.NoError(t, err)
	ownerID, err := svc.MemberRoleID(ctx, nexul.ID, "admin")
	require.NoError(t, err)

	require.NoError(t, svc.AddMember(ctx, "dana", acme.ID, "admin", viewer.ID))
	require.NoError(t, svc.AddMember(ctx, "dana", acme.ID, "carol", viewer.ID))
	return teamFixture{svc: svc, roles: rolesSvc, nexul: nexul, acme: acme, editor: editor, viewer: viewer, nexulOwnerID: ownerID, users: s.Users, sessions: s.Sessions, online: online}
}

func personByLogin(t *testing.T, team *tenancy.Team, login string) *tenancy.TeamPerson {
	t.Helper()
	for _, p := range team.People {
		if p.Login == login {
			return p
		}
	}
	require.Failf(t, "person missing", "no %s in the team", login)
	return nil
}

func workspaceByID(t *testing.T, team *tenancy.Team, id string) *tenancy.TeamWorkspace {
	t.Helper()
	for _, w := range team.Workspaces {
		if w.ID == id {
			return w
		}
	}
	require.Failf(t, "workspace missing", "no %s in the team", id)
	return nil
}

func TestIntegration_ListTeam_ReturnsEachPersonsAccessAndTheViewersManageFlag(t *testing.T) {
	ctx := t.Context()
	f := newTeamFixture(t)
	require.NoError(t, f.svc.AddMember(ctx, "admin", f.nexul.ID, "bob", f.editor.ID))
	allow := permissions.SetOf(permissions.ProjectsWrite)
	require.NoError(t, f.svc.SetMemberOverrides(ctx, "admin", f.nexul.ID, "bob", &allow, nil))

	team, err := f.svc.ListTeam(ctx, "admin")
	require.NoError(t, err)

	assert.True(t, team.CanManageAccounts)
	assert.True(t, workspaceByID(t, team, f.nexul.ID).CanManageMembers, "the Owner holds members:write in Nexul")
	assert.False(t, workspaceByID(t, team, f.acme.ID).CanManageMembers, "every instance-level permission, through owning Nexul, does not manage Acme, where admin is a Viewer")
	acmeRoles := workspaceByID(t, team, f.acme.ID).Roles
	require.Len(t, acmeRoles, 2)
	assert.True(t, acmeRoles[0].IsOwner, "the Owner role is listed first")
	assert.Equal(t, "Viewer", acmeRoles[1].Name)

	bob := personByLogin(t, team, "bob")
	require.Len(t, bob.Workspaces, 1)
	assert.Equal(t, f.nexul.ID, bob.Workspaces[0].WorkspaceID)
	assert.Equal(t, "Nexul", bob.Workspaces[0].WorkspaceName)
	assert.Equal(t, "Editor", bob.Workspaces[0].RoleName)
	assert.False(t, bob.Workspaces[0].IsOwner)
	assert.Equal(t, allow, bob.Workspaces[0].Allow)
	assert.Empty(t, bob.Workspaces[0].Deny)

	admin := personByLogin(t, team, "admin")
	held := map[string]string{}
	for _, m := range admin.Workspaces {
		held[m.WorkspaceName] = m.RoleName
	}
	assert.Equal(t, map[string]string{"Nexul": "Owner", "Acme": "Viewer"}, held)
	assert.True(t, admin.Workspaces[0].IsOwner || admin.Workspaces[1].IsOwner)
}

func TestIntegration_ListTeam_WithoutMembersWriteAnywhere_IsForbidden(t *testing.T) {
	f := newTeamFixture(t)
	_, err := f.svc.ListTeam(t.Context(), "carol")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestIntegration_ListTeam_WorkspaceManager_SeesOnlyTheWorkspacesTheyManage(t *testing.T) {
	ctx := t.Context()
	f := newTeamFixture(t)
	manager, err := f.roles.Create(ctx, f.nexul.ID, "admin", "Manager", permissions.SetOf(permissions.MembersWrite))
	require.NoError(t, err)
	require.NoError(t, f.svc.AddMember(ctx, "admin", f.nexul.ID, "bob", manager.ID))
	f.online["carol"], f.online["dana"] = true, true

	team, err := f.svc.ListTeam(ctx, "bob")
	require.NoError(t, err)

	assert.False(t, team.CanManageAccounts, "account status needs accounts:write, which a Manager does not hold")
	require.Len(t, team.Workspaces, 1)
	assert.Equal(t, f.nexul.ID, team.Workspaces[0].ID)
	logins := []string{}
	for _, p := range team.People {
		logins = append(logins, p.Login)
		for _, m := range p.Workspaces {
			assert.Equal(t, f.nexul.ID, m.WorkspaceID, "%s's access elsewhere is left out", p.Login)
		}
	}
	assert.ElementsMatch(t, []string{"admin", "bob"}, logins, "carol and dana hold nothing in Nexul, online or not")
}

func seedSession(t *testing.T, f teamFixture, id, userID string, lastActive time.Time) {
	t.Helper()
	require.NoError(t, f.sessions.CreateSession(t.Context(), &auth.Session{
		ID: id, UserID: userID, TokenHash: "hash-" + id, Client: auth.ClientBrowser,
		CreatedAt: lastActive, LastActiveAt: lastActive, ExpiresAt: lastActive.Add(24 * time.Hour),
	}))
}

func TestIntegration_ListTeam_OnlineFirstThenLatestSessionActivity(t *testing.T) {
	ctx := t.Context()
	f := newTeamFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	seedSession(t, f, "bob-laptop", "bob", now.Add(-2*time.Hour))
	seedSession(t, f, "bob-phone", "bob", now.Add(-5*time.Hour))
	seedSession(t, f, "carol-laptop", "carol", now.Add(-time.Hour))
	seedSession(t, f, "dana-laptop", "dana", now.Add(-72*time.Hour))
	f.online["dana"] = true

	team, err := f.svc.ListTeam(ctx, "admin")
	require.NoError(t, err)

	logins := []string{}
	for _, p := range team.People {
		logins = append(logins, p.Login)
	}
	assert.Equal(t, []string{"dana", "carol", "bob", "admin"}, logins, "online first, then most recently seen, never-seen last")
	assert.True(t, personByLogin(t, team, "dana").Online)
	assert.False(t, personByLogin(t, team, "carol").Online)
	bob := personByLogin(t, team, "bob")
	require.NotNil(t, bob.LastSeenAt)
	assert.Equal(t, now.Add(-2*time.Hour), *bob.LastSeenAt, "last seen is the latest of bob's sessions")
	assert.Nil(t, personByLogin(t, team, "admin").LastSeenAt, "no session row, no last seen")
}

// Instance-level permission is not workspace management (ADR 0024): every membership edit in Acme needs members:write in Acme.
func TestIntegration_TeamEdits_WithoutMembersWriteInTheWorkspace_AreForbidden(t *testing.T) {
	ctx := t.Context()
	f := newTeamFixture(t)
	allow := permissions.SetOf(permissions.DocsWrite)

	require.ErrorIs(t, f.svc.ChangeMemberRole(ctx, "admin", f.acme.ID, "carol", f.viewer.ID), apperrs.ErrForbidden)
	require.ErrorIs(t, f.svc.AddMember(ctx, "admin", f.acme.ID, "bob", f.viewer.ID), apperrs.ErrForbidden)
	require.ErrorIs(t, f.svc.SetMemberOverrides(ctx, "admin", f.acme.ID, "carol", &allow, nil), apperrs.ErrForbidden)
	require.ErrorIs(t, f.svc.RemoveMember(ctx, "admin", f.acme.ID, "carol"), apperrs.ErrForbidden)
}

func TestIntegration_TeamEdits_OwnerStaysProtected(t *testing.T) {
	ctx := t.Context()
	f := newTeamFixture(t)
	deny := permissions.SetOf(permissions.DocsRead)

	require.ErrorIs(t, f.svc.AddMember(ctx, "admin", f.nexul.ID, "carol", f.nexulOwnerID), apperrs.ErrInvalid)
	require.ErrorIs(t, f.svc.SetMemberOverrides(ctx, "dana", f.acme.ID, "dana", nil, &deny), apperrs.ErrInvalid)
}

// Carol holds only docs:read in Acme, so the members list refuses her; the people list must not.
func TestIntegration_ListPeople_ABasicMemberSeesEveryMembersNameAndPicture(t *testing.T) {
	ctx := t.Context()
	f := newTeamFixture(t)
	name, picture := "Lewis", "data:image/png;base64,aGVsbG8="
	require.NoError(t, f.users.SetProfileOverride(ctx, "admin", &name, &picture))

	_, err := f.svc.ListWorkspaceMembers(ctx, "carol", f.acme.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "the precondition: a Viewer cannot read the managed roster")

	people, err := f.svc.ListPeople(ctx, "carol", f.acme.ID)
	require.NoError(t, err)
	byLogin := map[string]tenancy.Person{}
	for _, p := range people {
		byLogin[p.Login] = p
	}
	require.Len(t, byLogin, 3, "dana, admin, and carol are Acme's members")
	assert.Equal(t, "Lewis", byLogin["admin"].DisplayName)
	assert.Equal(t, "admin", byLogin["admin"].UserID)
	assert.Regexp(t, `^/api/people/admin/avatar\?v=[0-9a-f]+$`, byLogin["admin"].AvatarURL)
	assert.Empty(t, byLogin["dana"].DisplayName, "no name set anywhere leaves the client to fall back to the login")

	contentType, data, err := f.svc.Avatar(ctx, "carol", "admin")
	require.NoError(t, err)
	assert.Equal(t, "image/png", contentType)
	assert.Equal(t, []byte("hello"), data)
}

func TestIntegration_ListPeople_ANonMemberIsRefused(t *testing.T) {
	ctx := t.Context()
	f := newTeamFixture(t)

	_, err := f.svc.ListPeople(ctx, "bob", f.acme.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	_, _, err = f.svc.Avatar(ctx, "bob", "carol")
	require.ErrorIs(t, err, apperrs.ErrForbidden, "bob shares no workspace with carol")
}
