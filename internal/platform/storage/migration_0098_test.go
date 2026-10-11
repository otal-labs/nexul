package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// TestMigration0098_EveryMemberKeepsTheirOneRoleAsASetOfOne upgrades members under every kind of role, Owner included.
func TestMigration0098_EveryMemberKeepsTheirOneRoleAsASetOfOne(t *testing.T) {
	db := migrateBefore(t, "0098")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES
    ('alice', 'alice', 1, 1), ('bob', 'bob', 1, 1), ('sam', 'sam', 1, 1), ('lena', 'lena', 1, 1);
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-owner', 'workspace-default', 'Owner', 1, 1, 1, '[]'),
    ('role-dev', 'workspace-default', 'Developer', 0, 2, 2, '["tickets:read","tickets:write","members:write"]'),
    ('role-empty', 'workspace-default', 'Member', 0, 3, 3, '[]'),
    ('role-acme-owner', 'ws-acme', 'Owner', 1, 1, 1, '[]');
INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at, restricted) VALUES
    ('alice', 'workspace-default', 'role-owner', 10, 0),
    ('bob', 'workspace-default', 'role-dev', 11, 0),
    ('sam', 'workspace-default', 'role-empty', 12, 1),
    ('lena', 'workspace-default', 'role-dev', 13, 0),
    ('lena', 'ws-acme', 'role-acme-owner', 14, 0);
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at) VALUES
    ('workspace', 'workspace-default', 'person', 'bob', '[]', '["tickets:write"]', 1, 1);
`)
	require.NoError(t, err)
	rows := func(query string) []string {
		r, err := db.Query(query)
		require.NoError(t, err)
		defer func() { require.NoError(t, r.Close()) }()
		var out []string
		for r.Next() {
			var line string
			require.NoError(t, r.Scan(&line))
			out = append(out, line)
		}
		require.NoError(t, r.Err())
		return out
	}
	before := rows(`SELECT user_id || '|' || workspace_id || '|' || role_id || '|' || created_at FROM workspace_members ORDER BY 1`)
	memberships := rows(`SELECT user_id || '|' || workspace_id || '|' || created_at || '|' || restricted FROM workspace_members ORDER BY 1`)

	migrateUpTo(t, db, "0100")

	assert.Equal(t, before, rows(`SELECT user_id || '|' || workspace_id || '|' || role_id || '|' || created_at FROM member_roles ORDER BY 1`))
	assert.Equal(t, memberships, rows(`SELECT user_id || '|' || workspace_id || '|' || created_at || '|' || restricted FROM workspace_members ORDER BY 1`))
	assert.Empty(t, rows(`SELECT name FROM pragma_table_info('workspace_members') WHERE name = 'role_id'`))

	s := New(db, testEncKey)
	ctx := t.Context()
	owner, err := s.WorkspaceMembers.MemberRole(ctx, "workspace-default", "alice")
	require.NoError(t, err)
	assert.True(t, owner.IsOwnerRole, "the Owner")
	assert.Equal(t, "role-owner", owner.RoleID)
	dev, err := s.WorkspaceMembers.MemberRole(ctx, "workspace-default", "bob")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.TicketsRead, permissions.TicketsWrite, permissions.MembersWrite, permissions.ProjectsAccess), dev.Permissions,
		"a role's set, with what 0099 gives a members:write holder")
	restricted, err := s.WorkspaceMembers.Member(ctx, "workspace-default", "sam")
	require.NoError(t, err)
	assert.Equal(t, []string{"role-empty"}, restricted.RoleIDs, "a Restricted member")
	owners, err := s.Users.ListActiveOwnerIDs(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "lena"}, owners, "an Owner of any workspace")
	override, err := s.Access.Get(ctx, "workspace", "workspace-default", "bob")
	require.NoError(t, err)
	assert.Empty(t, override.Allow)
	assert.Equal(t, permissions.SetOf(permissions.TicketsWrite), override.Deny, "overrides are untouched")

	require.NoError(t, s.WorkspaceMembers.RemoveMember(ctx, "ws-acme", "lena"))
	assert.Equal(t, []string{"lena|role-dev"}, rows(`SELECT user_id || '|' || role_id FROM member_roles WHERE user_id = 'lena'`), "roles go with their membership")
	_, err = db.Exec(`DELETE FROM roles WHERE id = 'role-empty'`)
	require.NoError(t, err)
	_, err = s.WorkspaceMembers.Member(ctx, "workspace-default", "sam")
	require.NoError(t, err, "a membership outlives its last role")
}
