package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// realUsers adapts the storage UsersRepo to access.Users for integration tests.
type realUsers struct {
	repo *storage.UsersRepo
}

func (u realUsers) ListUsers(ctx context.Context) ([]*access.User, error) {
	users, err := u.repo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*access.User, 0, len(users))
	for _, user := range users {
		out = append(out, &access.User{ID: user.ID, Login: user.Login, Name: user.Name})
	}
	return out, nil
}

func seedUser(t *testing.T, s *storage.Store, id, login string) {
	t.Helper()
	_, _, err := s.Users.UpsertUser(context.Background(), &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: login, Login: login})
	require.NoError(t, err)
}

func actor(id string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: id})
}

// testScopes mirrors server/cmd's accessScopes: project and membership lookups straight from storage.
type testScopes struct{ s *storage.Store }

func (sc testScopes) WorkspaceIDForProject(ctx context.Context, projectID string) (string, error) {
	p, err := sc.s.Projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

func (sc testScopes) WorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error) {
	ws, err := sc.s.Workspaces.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.ID
	}
	return out, nil
}

// joinDefaultWorkspace wires accessSvc's role and scope lookups over real storage and makes each user a member of
// the default workspace under a role holding docs:write, since only members create or list its docs.
func joinDefaultWorkspace(t *testing.T, s *storage.Store, accessSvc *access.Service, userIDs ...string) {
	t.Helper()
	ctx := context.Background()
	rolesSvc := roles.NewService(s.Roles, nil)
	tenancySvc := tenancy.NewService(s.Workspaces, s.WorkspaceMembers, s.WorkspaceInvites, testRoleGate{svc: rolesSvc}, testPermissionGate{}, testRoleNameGate{svc: rolesSvc}, testWorkspacePermissionGate{svc: accessSvc}, testAllowlistGate{}, testUserLookupGate{}, testChannelGate{}, testPlaysGate{}, testAccountGate{users: s.Users})
	rolesSvc.SetMemberGate(testMemberGate{svc: tenancySvc})
	accessSvc.SetRoles(accessRoleResolver{tenancy: tenancySvc, roles: rolesSvc})
	accessSvc.SetScopes(testScopes{s: s})
	role := &roles.Role{ID: "role-writer", WorkspaceID: "workspace-default", Name: "Writer", Permissions: permissions.SetOf(permissions.DocsWrite), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.Roles.Create(ctx, role))
	for _, id := range userIDs {
		require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: id, WorkspaceID: "workspace-default", RoleID: role.ID, CreatedAt: time.Now()}))
	}
}

// TestIntegration_AccessEndToEnd covers the ws-22 acceptance flow over real
// SQLite: creator gets full permissions, a stranger is denied, the owner can
// open anything, bulk grants unlock a doc, and search only shows readable docs.
func TestIntegration_AccessEndToEnd(t *testing.T) {
	ctx := context.Background()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, realUsers{s.Users})
	docsSvc := docs.NewService(s.Docs, accessSvc, nil)

	seedUser(t, s, "owner", "owner")
	seedUser(t, s, "alice", "alice")
	seedUser(t, s, "bob", "bob")
	joinDefaultWorkspace(t, s, accessSvc, "owner", "alice", "bob")

	// Creator receives full permissions automatically.
	doc, err := docsSvc.Create(actor("alice"), "project-general", "Shared Spec", "SQLite migrations")
	require.NoError(t, err)

	// Creator can read and edit their own doc.
	got, err := docsSvc.Get(actor("alice"), doc.ID)
	require.NoError(t, err)
	assert.Equal(t, "Shared Spec", got.Title)

	// A stranger cannot open the target.
	_, err = docsSvc.Get(actor("bob"), doc.ID)
	require.Error(t, err)

	// The only bypass is the doc's workspace Owner role, covered by TestIntegration_HasPermission_WorkspacePrecedence;
	// here "owner" is no member and proves access the way any other user would, through an explicit full grant.
	require.NoError(t, s.Access.Set(ctx, "doc", doc.ID, "owner", permissions.CreatorGrant, nil))
	_, err = docsSvc.Get(actor("owner"), doc.ID)
	require.NoError(t, err)

	// Search returns only docs the requester can open.
	results, err := docsSvc.Search(actor("bob"), "sqlite", 10)
	require.NoError(t, err)
	assert.Empty(t, results, "stranger sees no results")
	results, err = docsSvc.Search(actor("owner"), "sqlite", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Owner bulk-grants read to bob.
	require.NoError(t, accessSvc.SetGrants(ctx, "owner", []string{doc.ID}, []string{"bob"}, []permissions.Action{permissions.DocsRead}, true))
	_, err = docsSvc.Get(actor("bob"), doc.ID)
	require.NoError(t, err)
	results, err = docsSvc.Search(actor("bob"), "sqlite", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Revoking read locks bob out again.
	require.NoError(t, accessSvc.SetGrants(ctx, "owner", []string{doc.ID}, []string{"bob"}, []permissions.Action{permissions.DocsRead}, false))
	_, err = docsSvc.Get(actor("bob"), doc.ID)
	require.Error(t, err)

	// permission_overwrites has no FK to docs, so cleanup is explicit in docs.Service.Delete, not an automatic DB cascade.
	require.NoError(t, accessSvc.SetGrants(ctx, "owner", []string{doc.ID}, []string{"bob"}, []permissions.Action{permissions.DocsRead}, true))
	require.NoError(t, docsSvc.Delete(actor("owner"), doc.ID))
	_, err = s.Access.Get(ctx, "doc", doc.ID, "bob")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound), "overwrites must be removed with the doc")
}

// TestIntegration_ArchivedHiddenFromSearch covers that archived docs never
// appear in search.
func TestIntegration_ArchivedHiddenFromSearch(t *testing.T) {

	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, realUsers{s.Users})
	docsSvc := docs.NewService(s.Docs, accessSvc, nil)
	seedUser(t, s, "owner", "owner")
	joinDefaultWorkspace(t, s, accessSvc, "owner")

	doc, err := docsSvc.Create(actor("owner"), "project-general", "Archivable", "FTS content")
	require.NoError(t, err)

	_, err = docsSvc.Archive(actor("owner"), doc.ID)
	require.NoError(t, err)

	results, err := docsSvc.Search(actor("owner"), "fts", 10)
	require.NoError(t, err)
	assert.Empty(t, results, "archived doc must not appear in search")

	_, err = docsSvc.Restore(actor("owner"), doc.ID)
	require.NoError(t, err)
	results, err = docsSvc.Search(actor("owner"), "fts", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

// TestIntegration_ListDisclosure covers the disclosure rule: a user who cannot
// open a doc sees its title but cannot open the target.
func TestIntegration_ListDisclosure(t *testing.T) {

	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, realUsers{s.Users})
	docsSvc := docs.NewService(s.Docs, accessSvc, nil)
	seedUser(t, s, "owner", "owner")
	seedUser(t, s, "alice", "alice")
	seedUser(t, s, "bob", "bob")
	joinDefaultWorkspace(t, s, accessSvc, "owner", "alice", "bob")

	_, err := docsSvc.Create(actor("alice"), "project-general", "Private Notes", "nobody else should open this")
	require.NoError(t, err)

	items, err := docsSvc.List(actor("bob"))
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.False(t, items[0].CanOpen)
	assert.Equal(t, "Private Notes", items[0].Title)

	// Bob still cannot open the target.
	_, err = docsSvc.Get(actor("bob"), items[0].ID)
	require.Error(t, err)
}

// testRoleGate/testMemberGate mirror server/cmd/main.go's roleGate/
// roleMemberGate adapters, needed to construct real tenancy.Service +
// roles.Service instances for the HasPermission precedence integration test
// below.
type testRoleGate struct{ svc *roles.Service }

func (g testRoleGate) CreateOwnerRole(ctx context.Context, workspaceID string) (string, error) {
	r, err := g.svc.CreateOwnerRole(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	return r.ID, nil
}

type testMemberGate struct{ svc *tenancy.Service }

func (g testMemberGate) MemberRoleID(ctx context.Context, workspaceID, userID string) (string, error) {
	return g.svc.MemberRoleID(ctx, workspaceID, userID)
}

// testRoleNameGate mirrors main.go's roleNameGate adapter, needed to construct a real tenancy.Service for this test.
type testRoleNameGate struct{ svc *roles.Service }

func (g testRoleNameGate) RoleName(ctx context.Context, workspaceID, roleID string) (string, error) {
	r, err := g.svc.Get(ctx, workspaceID, roleID)
	if err != nil {
		return "", err
	}
	return r.Name, nil
}

func (g testRoleNameGate) IsOwnerRole(ctx context.Context, workspaceID, roleID string) (bool, error) {
	r, err := g.svc.Get(ctx, workspaceID, roleID)
	if err != nil {
		return false, err
	}
	return r.IsOwnerRole, nil
}

func (g testRoleNameGate) RolePermissions(ctx context.Context, workspaceID, roleID string) (permissions.Set, error) {
	r, err := g.svc.Get(ctx, workspaceID, roleID)
	if err != nil {
		return nil, err
	}
	return r.Permissions, nil
}

// testAllowlistGate mirrors server/cmd/main.go's allowlistGate adapter
// (Membership invites), needed to construct a real tenancy.Service for this
// test. Every login is allowed — this integration test predates Membership
// invites and doesn't exercise the not-allowlisted rejection.
type testAllowlistGate struct{}

func (testAllowlistGate) IsAllowlisted(context.Context, string) (bool, error) {
	return true, nil
}

// testUserLookupGate mirrors server/cmd/main.go's userLookupGate adapter
// (Membership invites), needed to construct a real tenancy.Service for this
// test. No login ever resolves — this integration test predates Membership
// invites and doesn't exercise InviteMember/ListWorkspaceMembers.
type testUserLookupGate struct{}

func (testUserLookupGate) UserIDForLogin(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (testUserLookupGate) LoginForUserID(context.Context, string) (string, error) {
	return "", apperrs.ErrNotFound
}

// testPermissionGate is an always-allow stand-in for tenancy's instance-level check: these tests exercise
// HasPermission's precedence chain, not workspace-creation gating.
type testPermissionGate struct{}

func (testPermissionGate) HoldsAnywhere(context.Context, string, permissions.Action) (bool, error) {
	return true, nil
}

// testWorkspacePermissionGate mirrors server/cmd/main.go's
// workspacePermissionGate adapter, needed to construct a real tenancy.Service
// for this test.
type testWorkspacePermissionGate struct{ svc *access.Service }

func (g testWorkspacePermissionGate) WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string {
	return g.svc.WorkspacePermissions(ctx, userID, workspaceID)
}

// testChannelGate mirrors server/cmd/main.go's channelGate adapter
// (live-chat ticket 07), needed to construct a real tenancy.Service for this
// test. This test exercises HasPermission's precedence chain, not chat, so
// it's a no-op.
type testChannelGate struct{}

func (testChannelGate) CreateGeneralChannel(context.Context, string, string) error {
	return nil
}

// testPlaysGate mirrors server/cmd/wire_gates.go's playsGate adapter (ticket 20). This test exercises
// HasPermission's precedence chain, not plays, so it's a no-op.
type testPlaysGate struct{}

func (testPlaysGate) SeedDefaultPlays(context.Context, string) error {
	return nil
}

// accessRoleResolver adapts tenancy.Service + roles.Service to
// access.RoleResolver, mirroring the real composition-root wiring
// (server/cmd/main.go).
type accessRoleResolver struct {
	tenancy *tenancy.Service
	roles   *roles.Service
}

func (a accessRoleResolver) MemberRole(ctx context.Context, workspaceID, userID string) (access.RoleInfo, error) {
	roleID, err := a.tenancy.MemberRoleID(ctx, workspaceID, userID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	r, err := a.roles.Get(ctx, workspaceID, roleID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	return access.RoleInfo{IsOwnerRole: r.IsOwnerRole, Permissions: r.Permissions}, nil
}

// TestIntegration_HasPermission_WorkspacePrecedence covers the Discord-order
// precedence chain (ADR 0042) end to end over real SQLite-backed
// roles + tenancy stores: the workspace Owner role bypasses everything, a
// workspace-wide overwrite can deny what a role allows, and a
// resource-instance overwrite can allow what a workspace-wide overwrite
// denies.
func TestIntegration_HasPermission_WorkspacePrecedence(t *testing.T) {
	ctx := context.Background()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, realUsers{s.Users})

	rolesSvc := roles.NewService(s.Roles, nil)
	tenancySvc := tenancy.NewService(s.Workspaces, s.WorkspaceMembers, s.WorkspaceInvites, testRoleGate{svc: rolesSvc}, testPermissionGate{}, testRoleNameGate{svc: rolesSvc}, testWorkspacePermissionGate{svc: accessSvc}, testAllowlistGate{}, testUserLookupGate{}, testChannelGate{}, testPlaysGate{}, testAccountGate{users: s.Users})
	rolesSvc.SetMemberGate(testMemberGate{svc: tenancySvc})
	rolesSvc.SetPermissionGate(testWorkspacePermissionGate{svc: accessSvc})
	accessSvc.SetRoles(accessRoleResolver{tenancy: tenancySvc, roles: rolesSvc})

	seedUser(t, s, "owner", "owner")
	seedUser(t, s, "alice", "alice")
	seedUser(t, s, "stranger", "stranger")

	ws, err := tenancySvc.Create(ctx, "owner", "Acme")
	require.NoError(t, err)

	// The workspace creator is bound to the Owner role and bypasses
	// everything, with no permission set bit or overwrite needed.
	assert.True(t, accessSvc.HasPermission(ctx, "owner", ws.ID, permissions.ProjectsWrite, "", ""))

	// A custom role's permission set is the base layer.
	editorRole, err := rolesSvc.Create(ctx, ws.ID, "owner", "Editor", permissions.SetOf(permissions.ProjectsWrite))
	require.NoError(t, err)
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{
		UserID: "alice", WorkspaceID: ws.ID, RoleID: editorRole.ID, CreatedAt: time.Now(),
	}))
	assert.True(t, accessSvc.HasPermission(ctx, "alice", ws.ID, permissions.ProjectsWrite, "", ""),
		"role mask alone grants the action")

	// A workspace-wide deny overrides the role's allow.
	require.NoError(t, s.Access.Set(ctx, "workspace", ws.ID, "alice", nil, permissions.SetOf(permissions.ProjectsWrite)))
	assert.False(t, accessSvc.HasPermission(ctx, "alice", ws.ID, permissions.ProjectsWrite, "", ""),
		"workspace-wide deny beats role allow")

	// A resource-instance allow overrides the workspace-wide deny —
	// most-specific overwrite wins overall.
	require.NoError(t, s.Access.Set(ctx, "project", "proj-1", "alice", permissions.SetOf(permissions.ProjectsWrite), nil))
	assert.True(t, accessSvc.HasPermission(ctx, "alice", ws.ID, permissions.ProjectsWrite, "project", "proj-1"),
		"resource-instance allow beats workspace-wide deny")
	assert.False(t, accessSvc.HasPermission(ctx, "alice", ws.ID, permissions.ProjectsWrite, "", ""),
		"the resource-instance overwrite doesn't leak into a check with no resource")

	// A user with no role and no overwrite in the workspace is denied.
	assert.False(t, accessSvc.HasPermission(ctx, "stranger", ws.ID, permissions.ProjectsWrite, "", ""))
}
