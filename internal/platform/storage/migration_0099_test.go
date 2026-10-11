package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0099_WhoMadeOrManagedProjectsKeepsDoingSo: creating a project follows projects:write, and managing a
// project's access follows members:write, so nobody gains or loses either on upgrade.
func TestMigration0099_WhoMadeOrManagedProjectsKeepsDoingSo(t *testing.T) {
	db := migrateBefore(t, "0099")
	_, err := db.Exec(`
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-admin', 'workspace-default', 'Admin', 0, 1, 1, '["members:write","projects:write"]'),
    ('role-dev', 'workspace-default', 'Developer', 0, 2, 2, '["projects:write","tickets:read"]'),
    ('role-lead', 'workspace-default', 'Lead', 0, 3, 3, '["members:write"]'),
    ('role-client', 'workspace-default', 'Client', 0, 4, 4, '["tickets:read"]'),
    ('role-again', 'workspace-default', 'Again', 0, 5, 5, '["projects:create","projects:write"]');
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	want := map[string]string{
		"role-admin":  `["members:write","projects:access","projects:create","projects:write"]`,
		"role-dev":    `["projects:create","projects:write","tickets:read"]`,
		"role-lead":   `["members:write","projects:access"]`,
		"role-client": `["tickets:read"]`,
		"role-again":  `["projects:create","projects:write"]`,
	}
	for id, perms := range want {
		var got string
		require.NoError(t, db.QueryRow(`SELECT permissions FROM roles WHERE id = ?`, id).Scan(&got))
		assert.JSONEq(t, perms, got, id)
	}
}
