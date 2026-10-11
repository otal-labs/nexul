package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0101_APendingInviteGoesWithItsRole upgrades an open login invite as it was, then deletes its role,
// which the old foreign key refused: the invite goes with it, and an invite for another role stays.
func TestMigration0101_APendingInviteGoesWithItsRole(t *testing.T) {
	db := migrateBefore(t, "0101")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('alice', 'alice', 1, 1);
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-dev', 'ws-acme', 'Developer', 0, 1, 1, '["docs:read"]'),
    ('role-client', 'ws-acme', 'Client', 0, 2, 2, '["tickets:read"]');
INSERT INTO workspace_invites (workspace_id, login, role_id, invited_by, created_at) VALUES
    ('ws-acme', 'lena', 'role-dev', 'alice', 5),
    ('ws-acme', 'sam', 'role-client', 'alice', 6);
`)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM roles WHERE id = 'role-dev'`)
	require.Error(t, err, "before the upgrade the invite blocks deleting its role")

	require.NoError(t, Migrate(db))

	var kept int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM workspace_invites`).Scan(&kept))
	assert.Equal(t, 2, kept, "the upgrade keeps every invite as it was")
	_, err = db.Exec(`DELETE FROM roles WHERE id = 'role-dev'`)
	require.NoError(t, err)
	rows, err := db.Query(`SELECT login, role_id, invited_by, created_at FROM workspace_invites`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var got []string
	for rows.Next() {
		var login, roleID, invitedBy string
		var createdAt int64
		require.NoError(t, rows.Scan(&login, &roleID, &invitedBy, &createdAt))
		got = append(got, login+" "+roleID+" "+invitedBy)
		assert.Equal(t, int64(6), createdAt)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"sam role-client alice"}, got)
}
