package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

func TestMigration0049_ExistingMembersAndGrantsStayFromRole(t *testing.T) {
	db := migrateBefore(t, "0049")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-owner', 'owner', 0, 0), ('u-dev', 'dev', 0, 0);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-owner', 'workspace-default', 'Owner', 1, 0, 0, '[]'),
    ('role-dev', 'workspace-default', 'Developer', 0, 0, 0, '["tickets:read","tickets:write"]');
INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at) VALUES
    ('u-owner', 'workspace-default', 'role-owner', 0), ('u-dev', 'workspace-default', 'role-dev', 0);
INSERT INTO invitations (id, token_hash, invited_by, created_at, expires_at) VALUES
    ('inv-1', '` + testTokenHash + `', 'u-owner', 4102444800, 4103049600);
INSERT INTO invitation_grants (invitation_id, workspace_id, role_id, allow_json, deny_json) VALUES
    ('inv-1', 'workspace-default', 'role-dev', '["docs:read"]', '[]');
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0049 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	dev, err := s.WorkspaceMembers.Member(ctx, "workspace-default", "u-dev")
	require.NoError(t, err)
	assert.False(t, dev.Restricted, "an upgraded member is From role")
	assert.Equal(t, "role-dev", dev.RoleID)
	ws, err := s.WorkspaceMembers.UnrestrictedWorkspaceIDs(ctx, "u-dev")
	require.NoError(t, err)
	assert.Equal(t, []string{"workspace-default"}, ws, "an upgraded membership still counts at instance level")

	invitation, err := s.Invitations.GetByTokenHash(ctx, testTokenHash, time.Unix(4102444900, 0))
	require.NoError(t, err)
	require.Len(t, invitation.Grants, 1)
	assert.False(t, invitation.Grants[0].Restricted(), "an upgraded invitation still admits From role")
	assert.Empty(t, invitation.Grants[0].ProjectAccess)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead), invitation.Grants[0].Allow, "nothing taken away")
}

// testTokenHash is a well-formed SHA-256 hex digest for invitation rows seeded in SQL.
const testTokenHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func seedProject(t *testing.T, s *Store, id, workspaceID, prefix string) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, s.Projects.Create(t.Context(), &workspace.Project{ID: id, Name: "Project " + prefix, Prefix: prefix, WorkspaceID: workspaceID, CreatedAt: now, UpdatedAt: now}))
}

// Removing a member, deleting a project, and switching Every project both ways must leave no stale access and lose
// no stored levels (ADR 0097).
func TestProjectAccess_LifecycleLeavesNoStaleRows(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-2", "Beta")))
	mustCreateUser(t, s, "u-client", "client")
	mustCreateUser(t, s, "u-other", "other")
	for _, ws := range []string{"workspace-default", "ws-2"} {
		require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: "role-" + ws, WorkspaceID: ws, Name: "Member", CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-client", WorkspaceID: ws, RoleID: "role-" + ws, CreatedAt: now}))
	}
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-other", WorkspaceID: "workspace-default", RoleID: "role-workspace-default", CreatedAt: now}))
	seedProject(t, s, "p-web", "workspace-default", "WEB")
	seedProject(t, s, "p-api", "workspace-default", "API")
	seedProject(t, s, "p-beta", "ws-2", "BET")
	read := permissions.SetOf(permissions.TicketsRead)
	for _, p := range []string{"p-web", "p-api"} {
		require.NoError(t, s.WorkspaceMembers.SetProjectAccess(ctx, p, "u-client", read))
	}
	require.NoError(t, s.WorkspaceMembers.SetProjectAccess(ctx, "p-beta", "u-client", read))
	require.NoError(t, s.WorkspaceMembers.SetProjectAccess(ctx, "p-web", "u-other", read))

	t.Run("switching Every project both ways keeps the stored levels", func(t *testing.T) {
		require.NoError(t, s.WorkspaceMembers.SetRestricted(ctx, "workspace-default", "u-client", true))
		restricted, err := s.WorkspaceMembers.UnrestrictedWorkspaceIDs(ctx, "u-client")
		require.NoError(t, err)
		assert.Equal(t, []string{"ws-2"}, restricted, "a restricted membership no longer counts at instance level")
		require.NoError(t, s.WorkspaceMembers.SetRestricted(ctx, "workspace-default", "u-client", false))
		rows, err := s.WorkspaceMembers.ProjectAccess(ctx, "workspace-default", "u-client")
		require.NoError(t, err)
		assert.Len(t, rows, 2)
	})

	t.Run("an empty allow takes one project away", func(t *testing.T) {
		require.NoError(t, s.WorkspaceMembers.SetProjectAccess(ctx, "p-api", "u-client", nil))
		rows, err := s.WorkspaceMembers.ProjectAccess(ctx, "workspace-default", "u-client")
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "p-web", rows[0].ProjectID)
		assert.Equal(t, "Project WEB", rows[0].ProjectName)
	})

	t.Run("deleting a project deletes everyone's access to it", func(t *testing.T) {
		require.NoError(t, s.Projects.Delete(ctx, "p-web"))
		_, err := s.Access.Get(ctx, "project", "p-web", "u-other")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
		all, err := s.WorkspaceMembers.ListAllProjectAccess(ctx)
		require.NoError(t, err)
		require.Len(t, all, 1)
		assert.Equal(t, "p-beta", all[0].ProjectID)
	})

	t.Run("removing a member deletes their access in that workspace only", func(t *testing.T) {
		seedProject(t, s, "p-docs", "workspace-default", "DOC")
		require.NoError(t, s.WorkspaceMembers.SetProjectAccess(ctx, "p-docs", "u-client", read))
		require.NoError(t, s.WorkspaceMembers.RemoveMember(ctx, "workspace-default", "u-client"))
		_, err := s.Access.Get(ctx, "project", "p-docs", "u-client")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
		_, err = s.Access.Get(ctx, "project", "p-beta", "u-client")
		assert.NoError(t, err, "another workspace's access stays")
	})
}

func TestProjectsRepo_ListRestrictedAccess_NamesOnlyRestrictedMembers(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: "role-member", WorkspaceID: "workspace-default", Name: "Member", CreatedAt: now, UpdatedAt: now}))
	for _, id := range []string{"u-client", "u-team"} {
		mustCreateUser(t, s, id, id[2:])
		require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: id, WorkspaceID: "workspace-default", RoleID: "role-member", CreatedAt: now}))
		require.NoError(t, s.WorkspaceMembers.SetProjectAccess(ctx, "project-general", id, permissions.SetOf(permissions.TicketsRead)))
	}
	require.NoError(t, s.WorkspaceMembers.SetRestricted(ctx, "workspace-default", "u-client", true))

	access, err := s.Projects.ListRestrictedAccess(ctx, "project-general")
	require.NoError(t, err)
	assert.Equal(t, []workspace.ProjectAccessEntry{{RestrictedMember: workspace.RestrictedMember{UserID: "u-client", Name: "client"}, Actions: permissions.SetOf(permissions.TicketsRead)}}, access,
		"a From role member's stored row is unused, so they are not listed")
}

func TestInvitationsRepo_RestrictedGrant(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	setup := func(t *testing.T) *Store {
		s := newTestStore(t)
		seedInvitationWorkspace(t, s, "ws-1", "Acme", "role-editor", false, "actor")
		seedInvitationWorkspace(t, s, "ws-2", "Beta", "role-beta", false, "actor")
		seedProject(t, s, "p-web", "ws-1", "WEB")
		seedProject(t, s, "p-api", "ws-1", "API")
		seedProject(t, s, "p-beta", "ws-2", "BET")
		require.NoError(t, s.Roles.Create(t.Context(), &roles.Role{ID: "role-plain", WorkspaceID: "ws-1", Name: "Plain", CreatedAt: now, UpdatedAt: now}))
		return s
	}
	grant := func(projects ...*tenancy.ProjectAccess) *tenancy.Invitation {
		invitation := newInvitation("inv-1", "actor", "ws-1", "role-editor", now, 7*24*time.Hour)
		invitation.Grants[0].EveryProject = tenancy.EveryProjectNone
		invitation.Grants[0].ProjectAccess = projects
		return invitation
	}
	docsRead := permissions.SetOf(permissions.DocsRead)

	errorCases := []struct {
		name       string
		invitation func() *tenancy.Invitation
		restrict   bool
		want       error
	}{
		{"project access needs every project none", func() *tenancy.Invitation {
			inv := grant(&tenancy.ProjectAccess{ProjectID: "p-web", Allow: docsRead})
			inv.Grants[0].EveryProject = tenancy.EveryProjectRole
			return inv
		}, false, apperrs.ErrInvalid},
		{"an unknown every project value", func() *tenancy.Invitation {
			inv := grant()
			inv.Grants[0].EveryProject = "some"
			return inv
		}, false, apperrs.ErrInvalid},
		{"a workspace area is not a project level", func() *tenancy.Invitation {
			return grant(&tenancy.ProjectAccess{ProjectID: "p-web", Allow: permissions.SetOf(permissions.MembersWrite)})
		}, false, apperrs.ErrInvalid},
		{"a project of another workspace", func() *tenancy.Invitation {
			return grant(&tenancy.ProjectAccess{ProjectID: "p-beta", Allow: docsRead})
		}, false, apperrs.ErrInvalid},
		{"a project that does not exist", func() *tenancy.Invitation {
			return grant(&tenancy.ProjectAccess{ProjectID: "p-gone", Allow: docsRead})
		}, false, apperrs.ErrInvalid},
		{"a level the giver does not hold", func() *tenancy.Invitation {
			return grant(&tenancy.ProjectAccess{ProjectID: "p-web", Allow: permissions.SetOf(permissions.TicketsWrite)})
		}, false, apperrs.ErrForbidden},
		{"a restricted giver on a project they cannot open", func() *tenancy.Invitation {
			inv := grant(&tenancy.ProjectAccess{ProjectID: "p-api", Allow: docsRead})
			inv.Grants[0].RoleID = "role-plain"
			return inv
		}, true, apperrs.ErrForbidden},
		{"a restricted giver on a project they hold the level on", func() *tenancy.Invitation {
			inv := grant(&tenancy.ProjectAccess{ProjectID: "p-web", Allow: docsRead})
			inv.Grants[0].RoleID = "role-plain"
			return inv
		}, true, nil},
	}
	for _, tt := range errorCases {
		t.Run(tt.name, func(t *testing.T) {
			s := setup(t)
			if tt.restrict {
				require.NoError(t, s.WorkspaceMembers.SetRestricted(t.Context(), "ws-1", "actor", true))
				require.NoError(t, s.WorkspaceMembers.SetProjectAccess(t.Context(), "p-web", "actor", docsRead))
			}
			_, tokenHash := invitationToken(t, "31")
			err := s.Invitations.Create(t.Context(), tt.invitation(), tokenHash)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}

	t.Run("redeeming lands the person restricted, skipping a project deleted since", func(t *testing.T) {
		s := setup(t)
		ctx := t.Context()
		_, tokenHash := invitationToken(t, "32")
		require.NoError(t, s.Invitations.Create(ctx, grant(&tenancy.ProjectAccess{ProjectID: "p-web", Allow: docsRead}, &tenancy.ProjectAccess{ProjectID: "p-api", Allow: docsRead}), tokenHash))
		require.NoError(t, s.Projects.Delete(ctx, "p-api"))

		preview, err := s.Invitations.GetByTokenHash(ctx, tokenHash, now)
		require.NoError(t, err)
		require.Len(t, preview.Grants[0].ProjectAccess, 1)
		assert.Equal(t, "Project WEB", preview.Grants[0].ProjectAccess[0].ProjectName, "the preview names the projects")

		identity := tenancy.InvitationIdentity{ID: "user-client", Provider: "github", ProviderUserID: "provider-client", Login: "client"}
		acceptanceHash := createAcceptanceHandoff(t, s, "inv-1", now, identity, "33")
		_, err = s.Invitations.Redeem(ctx, acceptanceHash, identity, now)
		require.NoError(t, err)
		m, err := s.WorkspaceMembers.Member(ctx, "ws-1", "user-client")
		require.NoError(t, err)
		assert.True(t, m.Restricted, "no window of seeing everything")
		rows, err := s.WorkspaceMembers.ProjectAccess(ctx, "ws-1", "user-client")
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "p-web", rows[0].ProjectID)
		assert.Equal(t, docsRead, rows[0].Allow)
	})

	t.Run("an invitation whose giver lost a level admits nobody", func(t *testing.T) {
		s := setup(t)
		ctx := t.Context()
		_, tokenHash := invitationToken(t, "34")
		invitation := grant(&tenancy.ProjectAccess{ProjectID: "p-web", Allow: docsRead})
		invitation.Grants[0].RoleID = "role-plain"
		require.NoError(t, s.Invitations.Create(ctx, invitation, tokenHash))
		require.NoError(t, s.Access.Set(ctx, "workspace", "ws-1", "actor", nil, docsRead))

		identity := tenancy.InvitationIdentity{ID: "user-client", Provider: "github", ProviderUserID: "provider-client", Login: "client"}
		acceptanceHash := createAcceptanceHandoff(t, s, "inv-1", now, identity, "35")
		_, err := s.Invitations.Redeem(ctx, acceptanceHash, identity, now)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		_, err = s.WorkspaceMembers.Member(ctx, "ws-1", "user-client")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}
