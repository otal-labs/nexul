package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0107_WhoPostedKeepsPosting: every member who reads a channel posts in it before the upgrade, so every
// role gains channels:post, and a token that posted with chat:write keeps posting with channels:post.
func TestMigration0107_WhoPostedKeepsPosting(t *testing.T) {
	db := migrateBefore(t, "0107")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-chatter', 'workspace-default', 'Chatter', 0, 1, 1, '["channels:delete","channels:write","chat:write"]'),
    ('role-reader', 'workspace-default', 'Reader', 0, 2, 2, '["docs:read"]'),
    ('role-empty', 'workspace-default', 'Empty', 0, 3, 3, '[]'),
    ('role-again', 'workspace-default', 'Again', 0, 4, 4, '["channels:post"]');
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at) VALUES
    ('install-chat', 'Bot', 'community', 'https://example.com', 's', '["chat:write"]', 'u-1', 1),
    ('install-docs', 'Docs', 'community', 'https://example.com', 's', '["docs:read"]', 'u-1', 1);
INSERT INTO automations (id, name, kind, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at) VALUES
    ('auto-chat', 'Chat', 'custom', x'', '{}', '{}', CAST('["chat:write"]' AS BLOB), 'hash-chat', 0, 0),
    ('auto-docs', 'Docs', 'custom', x'', '{}', '{}', CAST('["docs:read"]' AS BLOB), 'hash-docs', 0, 0),
    ('auto-empty', 'Empty', 'custom', x'', '{}', '{}', x'', 'hash-empty', 0, 0);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	for id, want := range map[string]string{
		"role-chatter": `["channels:delete","channels:post","channels:read","channels:write","chat:write"]`,
		"role-reader":  `["channels:post","channels:read","docs:read"]`,
		"role-empty":   `["channels:post","channels:read"]`,
		"role-again":   `["channels:post"]`,
	} {
		var got string
		require.NoError(t, db.QueryRow(`SELECT permissions FROM roles WHERE id = ?`, id).Scan(&got))
		assert.JSONEq(t, want, got, id)
	}

	for id, want := range map[string][]string{
		"install-chat": {"channels:post", "chat:write"},
		"install-docs": {"docs:read"},
	} {
		install, err := s.IntegrationInstalls.GetByID(ctx, id)
		require.NoError(t, err)
		assert.ElementsMatch(t, want, scopeStrings(install.Scopes), id)
	}
	for id, want := range map[string][]string{
		"auto-chat": {"channels:post", "chat:write"},
		"auto-docs": {"docs:read"},
	} {
		automation, err := s.Automations.Get(ctx, id)
		require.NoError(t, err)
		assert.ElementsMatch(t, want, scopeStrings(automation.Scopes), id)
	}
	automation, err := s.Automations.Get(ctx, "auto-empty")
	require.NoError(t, err)
	assert.Empty(t, automation.Scopes)
}
