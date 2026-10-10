package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func TestMigration0082_AutoPlaysFollowThePlaysBits(t *testing.T) {
	db := migrateBefore(t, "0082")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-editor', 'workspace-default', 'Editor', 0, 1, 1, '["plays:read","plays:write"]'),
    ('role-deleter', 'workspace-default', 'Deleter', 0, 1, 1, '["plays:delete"]'),
    ('role-reader', 'workspace-default', 'Reader', 0, 1, 1, '["plays:read"]'),
    ('role-automations', 'workspace-default', 'Automations', 0, 1, 1, '["automations:read","automations:write"]'),
    ('role-empty', 'workspace-default', 'Empty', 0, 1, 1, '[]');
INSERT INTO permission_overwrites (resource_type, resource_id, user_id, allow, deny, created_at, updated_at) VALUES
    ('project', 'p-1', 'u-1', '["plays:write"]', '["plays:delete"]', 1, 1),
    ('project', 'p-2', 'u-1', '[]', '["plays:write"]', 1, 1);
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at) VALUES
    ('install-plays', 'Plays', 'community', 'https://example.com', 's', '["plays:read"]', 'u-1', 1),
    ('install-docs', 'Docs', 'community', 'https://example.com', 's', '["docs:read"]', 'u-1', 1);
INSERT INTO automations (id, name, kind, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at) VALUES
    ('auto-plays', 'Plays', 'custom', x'', '{}', '{}', CAST('["plays:write"]' AS BLOB), 'hash-plays', 0, 0),
    ('auto-automations', 'Automations', 'custom', x'', '{}', '{}', CAST('["automations:write"]' AS BLOB), 'hash-automations', 0, 0),
    ('auto-empty', 'Empty', 'custom', x'', '{}', '{}', x'', 'hash-empty', 0, 0);
INSERT INTO invitations (id, token_hash, invited_by, created_at, expires_at) VALUES ('inv-1', 'h', 'u-1', 1, 2);
INSERT INTO invitation_grants (invitation_id, workspace_id, role_id, allow_json, deny_json, restricted, project_access_json) VALUES
    ('inv-1', 'workspace-default', 'role-reader', '["plays:delete"]', '["plays:read"]', 0, '[]');
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0082_auto_plays.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0082_auto_plays", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	for id, want := range map[string]permissions.Set{
		"role-editor":      permissions.SetOf(permissions.PlaysRead, permissions.PlaysWrite, permissions.AutoplaysRead, permissions.AutoplaysWrite),
		"role-deleter":     permissions.SetOf(permissions.PlaysDelete, permissions.AutoplaysDelete),
		"role-reader":      permissions.SetOf(permissions.PlaysRead, permissions.AutoplaysRead),
		"role-automations": permissions.SetOf(permissions.AutomationsRead, permissions.AutomationsWrite),
		"role-empty":       nil,
	} {
		role, err := s.Roles.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, want, role.Permissions, id)
	}

	ow, err := s.Access.Get(ctx, "project", "p-1", "u-1")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.PlaysWrite, permissions.AutoplaysRead, permissions.AutoplaysWrite), ow.Allow)
	assert.Equal(t, permissions.SetOf(permissions.PlaysDelete, permissions.AutoplaysDelete), ow.Deny)
	ow, err = s.Access.Get(ctx, "project", "p-2", "u-1")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.PlaysWrite, permissions.AutoplaysWrite), ow.Deny, "denying writes never denied reading")

	install, err := s.IntegrationInstalls.GetByID(ctx, "install-plays")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"autoplays:read", "plays:read"}, scopeStrings(install.Scopes))
	install, err = s.IntegrationInstalls.GetByID(ctx, "install-docs")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"docs:read"}, scopeStrings(install.Scopes))

	for id, want := range map[string][]string{
		"auto-plays":       {"autoplays:read", "autoplays:write", "plays:write"},
		"auto-automations": {"automations:write"},
		"auto-empty":       nil,
	} {
		automation, err := s.Automations.Get(ctx, id)
		require.NoError(t, err)
		assert.ElementsMatch(t, want, scopeStrings(automation.Scopes), id)
	}

	var allow, deny string
	require.NoError(t, db.QueryRow(`SELECT allow_json, deny_json FROM invitation_grants WHERE invitation_id = 'inv-1'`).Scan(&allow, &deny))
	assert.JSONEq(t, `["autoplays:delete","plays:delete"]`, allow, "a pending invitation lands with what it promised")
	assert.JSONEq(t, `["autoplays:read","plays:read"]`, deny)

	var limit int
	require.NoError(t, db.QueryRow(`SELECT auto_play_daily_cap FROM workspaces WHERE id = 'workspace-default'`).Scan(&limit))
	assert.Equal(t, 5, limit, "an existing workspace starts at the default cap")
}
