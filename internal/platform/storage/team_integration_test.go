package storage_test

import (
	"context"
	"testing"

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

// testAccountGate mirrors server/cmd's accountGate over the real users table.
type testAccountGate struct{ users *storage.UsersRepo }

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

// testInstanceAdminGate reads can_create_workspace from the real users table, like server/cmd's instanceAdminGate.
type testInstanceAdminGate struct{ users *storage.UsersRepo }

func (g testInstanceAdminGate) CanCreateWorkspace(ctx context.Context, userID string) (bool, error) {
	u, err := g.users.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return u.CanCreateWorkspace, nil
}

type teamFixture struct {
	svc          *tenancy.Service
	roles        *roles.Service
	nexul, acme  *tenancy.Workspace
	editor       *roles.Role
	viewer       *roles.Role
	nexulOwnerID string
	users        *storage.UsersRepo
}

// newTeamFixture: admin and dana administer the instance; admin owns Nexul and is only a Viewer in dana's Acme.
func newTeamFixture(t *testing.T) teamFixture {
	t.Helper()
	ctx := t.Context()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, realUsers{s.Users})
	rolesSvc := roles.NewService(s.Roles, nil)
	svc := tenancy.NewService(s.Workspaces, s.WorkspaceMembers, s.WorkspaceInvites, testRoleGate{svc: rolesSvc}, testInstanceAdminGate{users: s.Users}, testRoleNameGate{svc: rolesSvc}, testWorkspacePermissionGate{svc: accessSvc}, testAllowlistGate{}, testUserLookupGate{}, testChannelGate{}, testPlaysGate{}, testAccountGate{users: s.Users})
	rolesSvc.SetMemberGate(testMemberGate{svc: svc})
	accessSvc.SetRoles(accessRoleResolver{tenancy: svc, roles: rolesSvc})

	seedUser(t, s, "admin", "admin", true)
	seedUser(t, s, "dana", "dana", true)
	seedUser(t, s, "bob", "bob", false)
	seedUser(t, s, "carol", "carol", false)

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
	return teamFixture{svc: svc, roles: rolesSvc, nexul: nexul, acme: acme, editor: editor, viewer: viewer, nexulOwnerID: ownerID, users: s.Users}
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
	assert.False(t, workspaceByID(t, team, f.acme.ID).CanManageMembers, "an instance admin who is only a Viewer in Acme cannot manage its members")
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

	team, err := f.svc.ListTeam(ctx, "bob")
	require.NoError(t, err)

	assert.False(t, team.CanManageAccounts, "account status stays with instance administrators")
	require.Len(t, team.Workspaces, 1)
	assert.Equal(t, f.nexul.ID, team.Workspaces[0].ID)
	logins := []string{}
	for _, p := range team.People {
		logins = append(logins, p.Login)
		for _, m := range p.Workspaces {
			assert.Equal(t, f.nexul.ID, m.WorkspaceID, "%s's access elsewhere is left out", p.Login)
		}
	}
	assert.ElementsMatch(t, []string{"admin", "bob"}, logins, "carol and dana hold nothing in Nexul")
}

// Instance admin is not workspace admin (ADR 0024): every membership edit in Acme needs members:write in Acme.
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
