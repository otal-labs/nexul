package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0111_OnlyTheShippedDeveloperTemplatesTurnTheSwitchOn upgrades an instance with a workspace's roles and
// edited instance templates: every existing role stays off, and of the template rows only the four shipped keys that
// carry the switch gain it, with the rest of their text kept.
func TestMigration0111_OnlyTheShippedDeveloperTemplatesTurnTheSwitchOn(t *testing.T) {
	db := migrateBefore(t, "0111")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-owner', 'ws-acme', 'Owner', 1, 1, 1, '[]'),
    ('role-devops', 'ws-acme', 'DevOps', 0, 2, 2, '["stacks:read"]');
INSERT INTO instance_templates (kind, key, body, updated_by, updated_at) VALUES
    ('role', 'devops', '{"name":"Platform","permissions":["stacks:read"],"welcome_steps":["deploy_server"]}', 'alice', 5),
    ('role', 'junior_developer', '{"name":"Junior Developer","permissions":[],"welcome_steps":[]}', 'alice', 5),
    ('role', 'client', '{"name":"Client","permissions":[],"welcome_steps":["projects"]}', 'alice', 5),
    ('role', 'k-support', '{"name":"Support","permissions":[],"welcome_steps":[]}', 'alice', 5),
    ('mention_chip', 'devops', '{ticket.Ticket}', 'alice', 5);`)
	require.NoError(t, err)

	migrateUpTo(t, db, "0112")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))

	for _, id := range []string{"role-owner", "role-devops"} {
		role, err := s.Roles.Get(t.Context(), id)
		require.NoError(t, err)
		assert.False(t, role.ShowLockedActions, id)
	}
	body := func(kind, key string) string {
		var b string
		require.NoError(t, db.QueryRow(`SELECT body FROM instance_templates WHERE kind = ? AND key = ?`, kind, key).Scan(&b))
		return b
	}
	assert.JSONEq(t, `{"name":"Platform","permissions":["stacks:read"],"welcome_steps":["deploy_server"],"show_locked_actions":true}`, body("role", "devops"))
	assert.JSONEq(t, `{"name":"Junior Developer","permissions":[],"welcome_steps":[],"show_locked_actions":true}`, body("role", "junior_developer"))
	assert.JSONEq(t, `{"name":"Client","permissions":[],"welcome_steps":["projects"]}`, body("role", "client"))
	assert.JSONEq(t, `{"name":"Support","permissions":[],"welcome_steps":[]}`, body("role", "k-support"), "a template the instance added keeps its switch off")
	assert.Equal(t, `{ticket.Ticket}`, body("mention_chip", "devops"), "another kind that shares a key is untouched")
}
