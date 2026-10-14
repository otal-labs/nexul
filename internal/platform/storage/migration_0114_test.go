package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration_RolesDropTheInstanceAreas: only the instance Owner holds the instance by right (ADR 0151), so upgrading
// takes every instance-area bit but topology's and automations' from every role and edited Template role.
func TestMigration_RolesDropTheInstanceAreas(t *testing.T) {
	db := migrateBefore(t, "0114")
	_, err := db.Exec(`
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-admin', 'workspace-default', 'Administrator', 0, 1, 1, '["accounts:read","accounts:write","audit:read","automations:write","connectors:write","dns:write","docs:read","events:read","instance:write","integrations:read","machines:create","machines:deploy","templates:write","topology:write"]'),
    ('role-docs', 'workspace-default', 'Docs', 0, 2, 2, '["docs:read"]');
INSERT INTO instance_templates (kind, key, body, updated_at) VALUES
    ('role', 'devops', '{"name":"DevOps","permissions":["dns:read","dns:write","machines:write","topology:read"],"welcome_steps":["deploy_server"]}', 1),
    ('mention_chip', '', 'dns:write stays in prose', 1);
`)
	require.NoError(t, err)

	migrateUpTo(t, db, "0115")

	text := func(query string, args ...any) string {
		var got string
		require.NoError(t, db.QueryRow(query, args...).Scan(&got))
		return got
	}
	assert.JSONEq(t, `["automations:write","docs:read","machines:deploy","topology:write"]`, text(`SELECT permissions FROM roles WHERE id = 'role-admin'`))
	assert.JSONEq(t, `["docs:read"]`, text(`SELECT permissions FROM roles WHERE id = 'role-docs'`))
	assert.JSONEq(t, `{"name":"DevOps","permissions":["machines:write","topology:read"],"welcome_steps":["deploy_server"]}`,
		text(`SELECT body FROM instance_templates WHERE kind = 'role' AND key = 'devops'`))
	assert.Equal(t, "dns:write stays in prose", text(`SELECT body FROM instance_templates WHERE kind = 'mention_chip'`))
}
