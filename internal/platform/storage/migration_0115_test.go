package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration_AutomationsLeaveTheInstanceAreas: automations are workspace actions (ADR 0154), so upgrading deletes the
// rules on the "automations" instance area and leaves every other rule and every role's automations:* bits alone, and
// an enrollment code waiting for a host is the server's own until it is used.
func TestMigration_AutomationsLeaveTheInstanceAreas(t *testing.T) {
	db := migrateBefore(t, "0115")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-editor', 'workspace-default', 'Editor', 0, 1, 1, '["automations:read","automations:write"]');
INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, user_id, allow, deny, created_at, updated_at) VALUES
    ('instance_area', 'automations', 'role', 'role-editor', NULL, '["automations:delete"]', '[]', 1, 1),
    ('instance_area', 'automations', 'person', NULL, 'u-alice', '[]', '["automations:write"]', 1, 1),
    ('instance_area', 'dns', 'person', NULL, 'u-alice', '["dns:read"]', '[]', 1, 1),
    ('machine', 'm-edge', 'person', NULL, 'u-alice', '["machines:see"]', '[]', 1, 1);
INSERT INTO automation_host_enrollment_codes (code_hash, name, machine, created_at, expires_at) VALUES ('hash-1', 'jobs', 'edge-box', 1, 2);
`)
	require.NoError(t, err)

	migrateUpTo(t, db, "0116")

	var left string
	require.NoError(t, db.QueryRow(`SELECT group_concat(key) FROM (SELECT resource_type || ':' || resource_id AS key FROM permission_overwrites ORDER BY key)`).Scan(&left))
	assert.Equal(t, "instance_area:dns,machine:m-edge", left)

	var permissions string
	require.NoError(t, db.QueryRow(`SELECT permissions FROM roles WHERE id = 'role-editor'`).Scan(&permissions))
	assert.JSONEq(t, `["automations:read","automations:write"]`, permissions, "the role keeps its automations bits")

	var enrolledBy string
	require.NoError(t, db.QueryRow(`SELECT enrolled_by FROM automation_host_enrollment_codes WHERE code_hash = 'hash-1'`).Scan(&enrolledBy))
	assert.Empty(t, enrolledBy)
}
