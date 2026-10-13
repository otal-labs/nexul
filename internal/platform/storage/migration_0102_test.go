package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0102_WhoEditedTheInterviewTemplateKeepsDoingSo: editing it follows memories:write, so nobody gains or
// loses it on upgrade.
func TestMigration0102_WhoEditedTheInterviewTemplateKeepsDoingSo(t *testing.T) {
	db := migrateBefore(t, "0102")
	_, err := db.Exec(`
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-writer', 'workspace-default', 'Writer', 0, 1, 1, '["memories:read","memories:write"]'),
    ('role-reader', 'workspace-default', 'Reader', 0, 2, 2, '["memories:read"]'),
    ('role-again', 'workspace-default', 'Again', 0, 3, 3, '["memories:template","memories:write"]');
`)
	require.NoError(t, err)

	migrateUpTo(t, db, "0107")

	want := map[string]string{
		"role-writer": `["memories:read","memories:template","memories:write"]`,
		"role-reader": `["memories:read"]`,
		"role-again":  `["memories:template","memories:write"]`,
	}
	for id, perms := range want {
		var got string
		require.NoError(t, db.QueryRow(`SELECT permissions FROM roles WHERE id = ?`, id).Scan(&got))
		assert.JSONEq(t, perms, got, id)
	}
}
