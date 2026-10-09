package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// storeRoles resolves a member's role straight from the store, the way server/cmd wires access.
type storeRoles struct{ s *Store }

func (r storeRoles) MemberRole(ctx context.Context, workspaceID, userID string) (access.RoleInfo, error) {
	roleID, err := r.s.WorkspaceMembers.RoleIDFor(ctx, workspaceID, userID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	role, err := r.s.Roles.Get(ctx, roleID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	return access.RoleInfo{IsOwnerRole: role.IsOwnerRole, Permissions: role.Permissions}, nil
}

type storeScopes struct{ s *Store }

func (sc storeScopes) WorkspaceIDForProject(ctx context.Context, projectID string) (string, error) {
	p, err := sc.s.Projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

func (sc storeScopes) ProjectIDs(ctx context.Context, workspaceID string) ([]string, error) {
	ps, err := sc.s.Projects.List(ctx, workspaceID)
	ids := make([]string, len(ps))
	for i, p := range ps {
		ids[i] = p.ID
	}
	return ids, err
}

func (sc storeScopes) UnrestrictedWorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error) {
	return sc.s.WorkspaceMembers.UnrestrictedWorkspaceIDs(ctx, userID)
}

func (sc storeScopes) WorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error) {
	return sc.s.WorkspaceMembers.WorkspaceIDs(ctx, userID)
}

// formerAdminActions is everything the instance-admin flag used to open (ADR 0088).
var formerAdminActions = []permissions.Action{
	permissions.InstanceRead, permissions.InstanceWrite, permissions.AccountsRead, permissions.AccountsWrite,
	permissions.AccountsDelete, permissions.WorkspacesCreate, permissions.RunnersWrite, permissions.RunnersDelete,
	permissions.AutomationsWrite, permissions.AutomationsDelete, permissions.ConnectorsWrite,
	permissions.IntegrationsRead, permissions.IntegrationsWrite, permissions.IntegrationsDelete, permissions.AuditRead,
}

// upgradeTo0037 applies 0037 to a database built at 0036 and boots it the way the server does.
func upgradeTo0037(t *testing.T, db *sql.DB) (*Store, *access.Service) {
	t.Helper()
	script, err := migrationFS.ReadFile("migrations/0037_instance_admin_to_owner.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0037_instance_admin_to_owner", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	require.NoError(t, checkNotNewer(db, t.TempDir(), "test"))
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, nil)
	accessSvc.SetRoles(storeRoles{s: s})
	accessSvc.SetScopes(storeScopes{s: s})
	return s, accessSvc
}

func TestMigration0037_OwnerKeepsEveryFormerAdminPower_OtherAdminsLoseTheirs(t *testing.T) {
	db := migrateBefore(t, "0037")
	_, err := db.Exec(`
INSERT INTO users (id, login, can_create_workspace, created_at, updated_at) VALUES
    ('u-owner', 'owner', 1, 1, 1), ('u-admin', 'admin', 1, 2, 2), ('u-member', 'member', 0, 3, 3);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-owner', 'workspace-default', 'Owner', 1, 1, 1, '[]'),
    ('role-editor', 'workspace-default', 'Editor', 0, 1, 1, '["docs:read","docs:write"]');
INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at) VALUES
    ('u-owner', 'workspace-default', 'role-owner', 1),
    ('u-admin', 'workspace-default', 'role-editor', 1),
    ('u-member', 'workspace-default', 'role-editor', 1);
`)
	require.NoError(t, err)

	s, accessSvc := upgradeTo0037(t, db)
	ctx := t.Context()

	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'can_create_workspace'`))
	owners, err := s.Users.ListActiveOwnerIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"u-owner"}, owners, "an instance that already has an active Owner promotes nobody")
	for _, action := range formerAdminActions {
		held, err := accessSvc.HoldsAnywhere(ctx, "u-owner", action)
		require.NoError(t, err)
		assert.True(t, held, "the Owner holds %s", action)
		held, err = accessSvc.HoldsAnywhere(ctx, "u-admin", action)
		require.NoError(t, err)
		assert.False(t, held, "a former administrator who is not an Owner holds %s only through a role", action)
	}
	held, err := accessSvc.HoldsAnywhere(ctx, "u-admin", permissions.DocsWrite)
	require.NoError(t, err)
	assert.True(t, held, "the custom role keeps what it granted")
}

func TestMigration0037_InstanceWithNoActiveOwner_PromotesItsEarliestActiveAdmin(t *testing.T) {
	db := migrateBefore(t, "0037")
	_, err := db.Exec(`
INSERT INTO users (id, login, can_create_workspace, account_status, created_at, updated_at) VALUES
    ('u-gone', 'gone', 1, 'disabled', 1, 1), ('u-admin', 'admin', 1, 'active', 2, 2),
    ('u-later', 'later', 1, 'active', 3, 3), ('u-member', 'member', 0, 'active', 0, 0);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-owner', 'workspace-default', 'Owner', 1, 1, 1, '[]'),
    ('role-viewer', 'workspace-default', 'Viewer', 0, 1, 1, '["docs:read"]');
INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at) VALUES
    ('u-gone', 'workspace-default', 'role-owner', 1),
    ('u-admin', 'workspace-default', 'role-viewer', 1);
`)
	require.NoError(t, err)

	s, accessSvc := upgradeTo0037(t, db)
	ctx := t.Context()

	owners, err := s.Users.ListActiveOwnerIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"u-admin"}, owners, "the earliest active administrator takes the existing Owner role")
	held, err := accessSvc.HoldsAnywhere(ctx, "u-admin", permissions.AccountsWrite)
	require.NoError(t, err)
	assert.True(t, held, "and with it the power to reactivate the disabled Owner")
}

func TestMigration0037_FreshInstance_ChangesNothingButTheColumn(t *testing.T) {
	db := migrateBefore(t, "0037")
	s, _ := upgradeTo0037(t, db)

	owners, err := s.Users.ListActiveOwnerIDs(t.Context())
	require.NoError(t, err)
	assert.Empty(t, owners, "the owner wizard still runs")
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM roles WHERE workspace_id = 'workspace-default'`))
}
