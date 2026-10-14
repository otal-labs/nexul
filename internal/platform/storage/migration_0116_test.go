package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// TestMigration0116_ExistingRolesKeepNoDescription upgrades two workspaces with Owner roles, copies of template roles
// and members holding them: every role reads with an empty description and keeps everything else, and a description
// written afterwards is read back.
func TestMigration0116_ExistingRolesKeepNoDescription(t *testing.T) {
	db := migrateBefore(t, "0116")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('alice', 'alice', 1, 1), ('bob', 'bob', 1, 1);
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1), ('ws-globex', 'Globex', 'globex', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions, welcome_steps) VALUES
    ('role-acme-owner', 'ws-acme', 'Owner', 1, 1, 1, '[]', '[]'),
    ('role-acme-client', 'ws-acme', 'Client', 0, 2, 2, '["docs:read","tickets:write"]', '["projects"]'),
    ('role-globex-owner', 'ws-globex', 'Owner', 1, 1, 1, '[]', '[]'),
    ('role-globex-devops', 'ws-globex', 'DevOps', 0, 3, 3, '["deploys:write"]', '["deploy_server"]');
INSERT INTO workspace_members (user_id, workspace_id, created_at) VALUES ('alice', 'ws-acme', 10), ('bob', 'ws-acme', 11), ('bob', 'ws-globex', 12);
INSERT INTO member_roles (user_id, workspace_id, role_id, created_at) VALUES
    ('alice', 'ws-acme', 'role-acme-owner', 10), ('bob', 'ws-acme', 'role-acme-client', 11), ('bob', 'ws-globex', 'role-globex-devops', 12);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	for _, ws := range []string{"ws-acme", "ws-globex"} {
		rs, err := s.Roles.List(ctx, ws)
		require.NoError(t, err)
		require.Len(t, rs, 2)
		for _, r := range rs {
			assert.Empty(t, r.Description, r.ID)
		}
	}
	client, err := s.Roles.Get(ctx, "role-acme-client")
	require.NoError(t, err)
	assert.Equal(t, "Client", client.Name)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.TicketsWrite), client.Permissions)
	assert.Equal(t, []string{"projects"}, client.WelcomeSteps)

	client.Description = "Reads and writes the tickets and docs of the projects you're added to."
	client.UpdatedAt = time.Unix(20, 0).UTC()
	require.NoError(t, s.Roles.Update(ctx, client))
	again, err := s.Roles.Get(ctx, "role-acme-client")
	require.NoError(t, err)
	assert.Equal(t, client.Description, again.Description)
}
