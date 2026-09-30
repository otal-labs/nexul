package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func TestMigration0040_RolesThatWriteStacksReadTheirLogs(t *testing.T) {
	db := migrateBefore(t, "0040")
	_, err := db.Exec(`
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-ops', 'workspace-default', 'Ops', 0, 1, 1, '["deploys:write","stacks:read","stacks:write"]'),
    ('role-viewer', 'workspace-default', 'Viewer', 0, 1, 1, '["stacks:read"]'),
    ('role-granted', 'workspace-default', 'Granted', 0, 1, 1, '["stacks:logs","stacks:write"]'),
    ('role-empty', 'workspace-default', 'Empty', 0, 1, 1, '[]');
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0040_stacks_logs_permission.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0040_stacks_logs_permission", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))

	for id, want := range map[string]permissions.Set{
		"role-ops":     permissions.SetOf(permissions.DeploysWrite, permissions.StacksRead, permissions.StacksWrite, permissions.StacksLogs),
		"role-viewer":  permissions.SetOf(permissions.StacksRead),
		"role-granted": permissions.SetOf(permissions.StacksLogs, permissions.StacksWrite),
		"role-empty":   nil,
	} {
		role, err := s.Roles.Get(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, want, role.Permissions, id)
	}
}
