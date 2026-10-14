package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration_RemovesTheGoneMachineBits: seeing a machine takes its rules (ADR 0151), so upgrading takes
// runners:read, runners:write, runners:delete and machines:read from every role, shipped-role edit, token scope and
// instance-area rule that held them, and the rules on the "runners" area move to "machines" holding machines:create alone.
func TestMigration_RemovesTheGoneMachineBits(t *testing.T) {
	db := migrateBefore(t, "0110")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-ops', 'workspace-default', 'Ops', 0, 1, 1, '["docs:read","machines:read","machines:write","runners:delete","runners:read","runners:write"]'),
    ('role-docs', 'workspace-default', 'Docs', 0, 2, 2, '["docs:read","machines:write"]'),
    ('role-gone', 'workspace-default', 'Gone', 0, 3, 3, '["runners:read"]'),
    ('role-empty', 'workspace-default', 'Empty', 0, 4, 4, '[]');
INSERT INTO instance_templates (kind, key, body, updated_at) VALUES
    ('role', 'devops', '{"name":"DevOps","permissions":["dns:read","machines:read","machines:write","runners:write"],"welcome_steps":["deploy_server"]}', 1),
    ('role', 'client', '{"name":"Client","permissions":["docs:read"],"welcome_steps":[]}', 1),
    ('mention_chip', '', 'machines:read stays in prose', 1);
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at) VALUES
    ('install-fleet', 'Fleet', 'community', 'https://example.com', 's', '["machines:read","runners:read","stacks:read"]', 'u-1', 1),
    ('install-docs', 'Docs', 'community', 'https://example.com', 's', '["docs:read"]', 'u-1', 1);
INSERT INTO automations (id, name, kind, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at) VALUES
    ('auto-fleet', 'Fleet', 'custom', x'', '{}', '{}', CAST('["runners:write","machines:write"]' AS BLOB), 'hash-fleet', 0, 0),
    ('auto-empty', 'Empty', 'custom', x'', '{}', '{}', x'', 'hash-empty', 0, 0);
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at) VALUES
    ('instance_area', 'runners', 'person', 'u-1', NULL, '["machines:create","machines:write","runners:read"]', '["runners:delete"]', 1, 1),
    ('instance_area', 'runners', 'role', NULL, 'role-ops', '["runners:write"]', '["machines:create"]', 1, 1),
    ('instance_area', 'runners', 'role', NULL, 'role-docs', '["runners:read","machines:read"]', '[]', 1, 1),
    ('instance_area', 'dns', 'person', 'u-1', NULL, '["dns:read"]', '[]', 1, 1),
    ('machine', 'm-1', 'person', 'u-1', NULL, '["machines:see","machines:write"]', '[]', 1, 1);
`)
	require.NoError(t, err)

	migrateUpTo(t, db, "0111") // 0111 adds the show_locked_actions field to the shipped developer Template roles, which its own test covers.
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	text := func(query string, args ...any) string {
		var got string
		require.NoError(t, db.QueryRow(query, args...).Scan(&got))
		return got
	}
	for id, want := range map[string]string{
		"role-ops":   `["docs:read","machines:write"]`,
		"role-docs":  `["docs:read","machines:write"]`,
		"role-gone":  `[]`,
		"role-empty": `[]`,
	} {
		assert.JSONEq(t, want, text(`SELECT permissions FROM roles WHERE id = ?`, id), id)
	}
	assert.JSONEq(t, `{"name":"DevOps","permissions":["dns:read","machines:write"],"welcome_steps":["deploy_server"]}`,
		text(`SELECT body FROM instance_templates WHERE kind = 'role' AND key = 'devops'`), "an edited Template role")
	assert.JSONEq(t, `{"name":"Client","permissions":["docs:read"],"welcome_steps":[]}`, text(`SELECT body FROM instance_templates WHERE key = 'client'`))
	assert.Equal(t, "machines:read stays in prose", text(`SELECT body FROM instance_templates WHERE kind = 'mention_chip'`))

	install, err := s.IntegrationInstalls.GetByID(ctx, "install-fleet")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"stacks:read"}, scopeStrings(install.Scopes))
	automation, err := s.Automations.Get(ctx, "auto-fleet")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"machines:write"}, scopeStrings(automation.Scopes))
	automation, err = s.Automations.Get(ctx, "auto-empty")
	require.NoError(t, err)
	assert.Empty(t, automation.Scopes)

	var areas int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM permission_overwrites WHERE resource_type = 'instance_area' AND resource_id = 'runners'`).Scan(&areas))
	assert.Zero(t, areas, "no rule is left on the old area key")
	assert.JSONEq(t, `["machines:create"]`, text(`SELECT allow FROM permission_overwrites WHERE resource_id = 'machines' AND user_id = 'u-1'`))
	assert.JSONEq(t, `[]`, text(`SELECT deny FROM permission_overwrites WHERE resource_id = 'machines' AND user_id = 'u-1'`))
	assert.JSONEq(t, `["machines:create"]`, text(`SELECT deny FROM permission_overwrites WHERE resource_id = 'machines' AND role_id = 'role-ops'`))
	var droppedForNothing int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM permission_overwrites WHERE resource_id = 'machines' AND role_id = 'role-docs'`).Scan(&droppedForNothing))
	assert.Zero(t, droppedForNothing, "a rule holding only gone actions is dropped")
	assert.JSONEq(t, `["dns:read"]`, text(`SELECT allow FROM permission_overwrites WHERE resource_type = 'instance_area' AND resource_id = 'dns'`), "another area is untouched")
	assert.JSONEq(t, `["machines:see","machines:write"]`, text(`SELECT allow FROM permission_overwrites WHERE resource_type = 'machine' AND resource_id = 'm-1'`), "a machine's own rule is untouched")
}
