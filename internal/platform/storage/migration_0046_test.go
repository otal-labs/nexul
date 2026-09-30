package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func TestMigration0046_WhateverCreatedChannelsKeepsCreatingThem(t *testing.T) {
	db := migrateBefore(t, "0046")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-chatter', 'workspace-default', 'Chatter', 0, 1, 1, '["chat:write","docs:read"]'),
    ('role-reader', 'workspace-default', 'Reader', 0, 1, 1, '["docs:read"]'),
    ('role-empty', 'workspace-default', 'Empty', 0, 1, 1, '[]');
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at) VALUES
    ('install-chat', 'Bot', 'community', 'https://example.com', 's', '["chat:write"]', 'u-1', 1),
    ('install-docs', 'Docs', 'community', 'https://example.com', 's', '["docs:read"]', 'u-1', 1);
INSERT INTO automations (id, name, kind, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at) VALUES
    ('auto-chat', 'Chat', 'custom', x'', '{}', '{}', CAST('["chat:write"]' AS BLOB), 'hash-chat', 0, 0),
    ('auto-empty', 'Empty', 'custom', x'', '{}', '{}', x'', 'hash-empty', 0, 0);
INSERT INTO conversations (id, workspace_id, kind, name, created_by, created_at, updated_at) VALUES
    ('conv-general', 'workspace-default', 'channel', 'general', 'u-1', 1, 1),
    ('conv-eng', 'workspace-default', 'channel', 'eng', 'u-1', 1, 1),
    ('conv-voice', 'workspace-default', 'voice_channel', 'general', 'u-1', 1, 1);
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0046_channels_permission.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0046_channels_permission", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	channels := []permissions.Action{permissions.ChannelsWrite, permissions.ChannelsDelete}
	for id, want := range map[string]permissions.Set{
		"role-chatter": permissions.SetOf(append(channels, permissions.ChatWrite, permissions.DocsRead)...),
		"role-reader":  permissions.SetOf(permissions.DocsRead),
		"role-empty":   nil,
	} {
		role, err := s.Roles.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, want, role.Permissions, id)
	}

	install, err := s.IntegrationInstalls.GetByID(ctx, "install-chat")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"channels:delete", "channels:write", "chat:write"}, scopeStrings(install.Scopes))
	install, err = s.IntegrationInstalls.GetByID(ctx, "install-docs")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"docs:read"}, scopeStrings(install.Scopes))

	automation, err := s.Automations.Get(ctx, "auto-chat")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"channels:delete", "channels:write", "chat:write"}, scopeStrings(automation.Scopes))
	automation, err = s.Automations.Get(ctx, "auto-empty")
	require.NoError(t, err)
	assert.Empty(t, automation.Scopes)

	for id, general := range map[string]bool{"conv-general": true, "conv-eng": false, "conv-voice": false} {
		c, err := s.Chat.GetConversation(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, general, c.General, id)
	}
}

func scopeStrings[S ~string](scopes []S) []string {
	out := make([]string, len(scopes))
	for i, sc := range scopes {
		out[i] = string(sc)
	}
	return out
}
