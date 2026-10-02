package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func TestMigration0057_WhateverCouldLockADocKeepsLockingIt(t *testing.T) {
	db := migrateBefore(t, "0057")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-writer', 'workspace-default', 'Writer', 0, 1, 1, '["docs:read","docs:write"]'),
    ('role-reader', 'workspace-default', 'Reader', 0, 1, 1, '["docs:read"]'),
    ('role-empty', 'workspace-default', 'Empty', 0, 1, 1, '[]');
INSERT INTO permission_overwrites (resource_type, resource_id, user_id, allow, deny, created_at, updated_at) VALUES
    ('doc', 'd-creator', 'u-1', '["docs:delete","docs:read","docs:write","permissions:write"]', '[]', 1, 1),
    ('doc', 'd-denied', 'u-1', '[]', '["docs:write"]', 1, 1),
    ('doc', 'd-reader', 'u-1', '["docs:read"]', '["docs:delete"]', 1, 1);
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at) VALUES
    ('install-docs', 'Docs', 'community', 'https://example.com', 's', '["docs:write"]', 'u-1', 1),
    ('install-read', 'Read', 'community', 'https://example.com', 's', '["docs:read"]', 'u-1', 1);
INSERT INTO automations (id, name, kind, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at) VALUES
    ('auto-docs', 'Docs', 'custom', x'', '{}', '{}', CAST('["docs:write"]' AS BLOB), 'hash-docs', 0, 0),
    ('auto-empty', 'Empty', 'custom', x'', '{}', '{}', x'', 'hash-empty', 0, 0);
INSERT INTO invitations (id, token_hash, invited_by, created_at, expires_at) VALUES ('inv-1', 'h', 'u-1', 1, 2);
INSERT INTO invitation_grants (invitation_id, workspace_id, role_id, allow_json, deny_json, restricted, project_access_json) VALUES
    ('inv-1', 'workspace-default', 'role-reader', '["docs:write"]', '[]', 1,
     '[{"project_id":"p-1","allow":["docs:read","docs:write"]},{"project_id":"p-2","allow":["tickets:read"]}]');
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0057_docs_lock_permission.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0057_docs_lock_permission", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	for id, want := range map[string]permissions.Set{
		"role-writer": permissions.SetOf(permissions.DocsRead, permissions.DocsWrite, permissions.DocsLock),
		"role-reader": permissions.SetOf(permissions.DocsRead),
		"role-empty":  nil,
	} {
		role, err := s.Roles.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, want, role.Permissions, id)
	}

	for doc, want := range map[string][2]permissions.Set{
		"d-creator": {permissions.CreatorGrant.With(permissions.DocsLock), nil},
		"d-denied":  {nil, permissions.SetOf(permissions.DocsWrite, permissions.DocsLock)},
		"d-reader":  {permissions.SetOf(permissions.DocsRead), permissions.SetOf(permissions.DocsDelete)},
	} {
		ow, err := s.Access.Get(ctx, "doc", doc, "u-1")
		require.NoError(t, err)
		assert.Equal(t, want[0], ow.Allow, "%s allow", doc)
		assert.Equal(t, want[1], ow.Deny, "%s deny: whoever could not lock it before still cannot", doc)
	}

	install, err := s.IntegrationInstalls.GetByID(ctx, "install-docs")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"docs:lock", "docs:write"}, scopeStrings(install.Scopes))
	install, err = s.IntegrationInstalls.GetByID(ctx, "install-read")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"docs:read"}, scopeStrings(install.Scopes))

	automation, err := s.Automations.Get(ctx, "auto-docs")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"docs:lock", "docs:write"}, scopeStrings(automation.Scopes))
	automation, err = s.Automations.Get(ctx, "auto-empty")
	require.NoError(t, err)
	assert.Empty(t, automation.Scopes)

	var allow, projects string
	require.NoError(t, db.QueryRow(`SELECT allow_json, project_access_json FROM invitation_grants WHERE invitation_id = 'inv-1'`).Scan(&allow, &projects))
	assert.JSONEq(t, `["docs:lock","docs:write"]`, allow, "a pending invitation lands with what it promised")
	assert.JSONEq(t, `[{"project_id":"p-1","allow":["docs:lock","docs:read","docs:write"]},{"project_id":"p-2","allow":["tickets:read"]}]`, projects,
		"and so do its per-project levels")
}
