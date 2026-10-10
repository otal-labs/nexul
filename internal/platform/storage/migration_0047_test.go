package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/crypto"
)

// TestMigration0047_AutomationsMoveIntoWorkspaces upgrades a database holding today's rows: both defaults, a custom
// automation placed on a host with its token, a run, a version, a cursor, a secret, and a second workspace.
func TestMigration0047_AutomationsMoveIntoWorkspaces(t *testing.T) {
	db := migrateBefore(t, "0047")
	key := []byte("0123456789abcdef0123456789abcdef")
	secret, err := crypto.Encrypt(key, []byte("sk-live"))
	require.NoError(t, err)
	_, err = db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-b', 'Beta', 'beta', 2, 2);
INSERT INTO automation_hosts (id, name, last_seen, created_at) VALUES ('host-1', 'worker-1', 1, 1);
INSERT INTO automations (id, name, kind, enabled, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at) VALUES
    ('default-ticket-finished', 'Ticket finished', 'default', 0, CAST('["ticket.finished"]' AS BLOB), '{}', '{"completedStatusId":"st-done"}', CAST('["tickets:write"]' AS BLOB), 'hash-tf', 1, 1),
    ('default-pr-opened', 'PR opened', 'default', 1, CAST('["git.pr_opened"]' AS BLOB), '{}', '{}', CAST('["tickets:write"]' AS BLOB), 'hash-po', 1, 1);
INSERT INTO automations (id, name, kind, enabled, subscriptions, config_schema, config_values, scopes, created_by, token_hash, token_prefix, host_id, created_at, updated_at) VALUES
    ('custom-1', 'Notifier', 'custom', 1, CAST('["ticket.created"]' AS BLOB), '{}', '{"channel":"#ops"}', CAST('["tickets:read"]' AS BLOB), 'u-1', 'hash-custom', 'abc123', 'host-1', 1, 1);
INSERT INTO automation_runs (id, automation_id, event_topic, event_id, outcome, started_at, finished_at, created_at) VALUES
    ('run-1', 'custom-1', 'ticket.created', 'ev-1', 'success', 1, 1, 1);
INSERT INTO automation_versions (id, automation_id, sequence, code, pusher_id, status, created_at) VALUES
    ('ver-1', 'default-ticket-finished', 1, CAST('code' AS BLOB), 'seed', 'active', 1);
INSERT INTO automation_cursors (automation_id, last_created_at, last_event_id, updated_at) VALUES ('custom-1', 5, 'ev-1', 5);
`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO automation_secrets (name, value, created_at, updated_at) VALUES ('API_KEY', ?, 1, 1)`, string(secret))
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0047_automations_workspace.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0047_automations_workspace", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, key)
	ctx := t.Context()

	custom, err := s.Automations.Get(ctx, "custom-1")
	require.NoError(t, err)
	assert.Equal(t, "workspace-default", custom.WorkspaceID, "a custom automation lands in the default workspace")
	assert.Equal(t, "hash-custom", custom.TokenHash, "its token keeps working")
	require.NotNil(t, custom.HostID)
	assert.Equal(t, "host-1", *custom.HostID)
	assert.True(t, custom.Enabled)
	assert.JSONEq(t, `{"channel":"#ops"}`, string(custom.ConfigValues))

	home, err := s.Automations.Get(ctx, "default-ticket-finished")
	require.NoError(t, err)
	assert.Equal(t, "workspace-default", home.WorkspaceID)
	assert.False(t, home.Enabled)
	assert.JSONEq(t, `{"completedStatusId":"st-done"}`, string(home.ConfigValues))

	copied, err := s.Automations.Get(ctx, "default-ticket-finished-ws-b")
	require.NoError(t, err)
	assert.Equal(t, "ws-b", copied.WorkspaceID)
	assert.False(t, copied.Enabled, "the copy keeps today's switch")
	assert.JSONEq(t, `{}`, string(copied.ConfigValues), "a status id from another workspace's project is not carried over")
	assert.NotEqual(t, home.TokenHash, copied.TokenHash)
	opened, err := s.Automations.Get(ctx, "default-pr-opened-ws-b")
	require.NoError(t, err)
	assert.True(t, opened.Enabled)
	_, err = s.Automations.Get(ctx, "custom-1-ws-b")
	require.Error(t, err, "custom automations are not copied")

	runs, err := s.AutomationRuns.ListByAutomation(ctx, "custom-1", 10)
	require.NoError(t, err)
	assert.Len(t, runs, 1, "run history stays with its automation")
	active, err := s.AutomationVersions.Active(ctx, "default-ticket-finished")
	require.NoError(t, err)
	assert.Equal(t, "ver-1", active.ID)
	_, ok, err := s.AutomationCursors.Get(ctx, "custom-1")
	require.NoError(t, err)
	assert.True(t, ok, "the delivery cursor survives, so nothing is replayed")

	secrets, err := s.AutomationSecrets.All(ctx, "workspace-default")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"API_KEY": "sk-live"}, secrets, "the existing pool is the default workspace's")
	other, err := s.AutomationSecrets.All(ctx, "ws-b")
	require.NoError(t, err)
	assert.Empty(t, other)

	for _, ws := range []string{"workspace-default", "ws-b"} {
		var on bool
		require.NoError(t, db.QueryRow(`SELECT decisions_check_enabled FROM workspaces WHERE id = ?`, ws).Scan(&on))
		assert.False(t, on, "the decisions check starts off in %s", ws)
	}
}
