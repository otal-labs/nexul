package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0086_DuplicateLabelsAreRenamedInCreationOrder(t *testing.T) {
	db := migrateBefore(t, "0086")
	_, err := db.Exec(`
DELETE FROM plays;
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-other', 'Other', 'other', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, created_at, updated_at) VALUES
    ('p-deploy-2', 'workspace-default', 'Deploy (2)', 'doc', 1, 1),
    ('p-deploy-a', 'workspace-default', 'Deploy', 'doc', 2, 2),
    ('p-deploy-b', 'workspace-default', 'deploy', 'doc', 3, 3),
    ('p-deploy-c', 'workspace-default', 'Deploy', 'doc', 3, 3),
    ('p-fix-b', 'workspace-default', 'Fix', 'doc', 5, 5),
    ('p-fix-a', 'workspace-default', 'Fix', 'doc', 5, 5),
    ('p-other', 'ws-other', 'Deploy', 'doc', 9, 9),
    ('p-alone', 'workspace-default', 'Alone', 'doc', 1, 1);
INSERT INTO integration_installs (id, name, trust_tier, webhook_url, webhook_secret, scopes, created_by, created_at) VALUES
    ('install-writes', 'Writes', 'community', 'https://example.com', 's', '["plays:read","plays:write"]', 'u-1', 1),
    ('install-reads', 'Reads', 'community', 'https://example.com', 's', '["plays:read"]', 'u-1', 1);
INSERT INTO automations (id, name, kind, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at) VALUES
    ('auto-writes', 'Writes', 'custom', x'', '{}', '{}', CAST('["plays:write"]' AS BLOB), 'hash-writes', 0, 0),
    ('auto-empty', 'Empty', 'custom', x'', '{}', '{}', x'', 'hash-empty', 0, 0);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	labels := map[string]string{}
	rows, err := db.Query(`SELECT id, label FROM plays`)
	require.NoError(t, err)
	for rows.Next() {
		var id, label string
		require.NoError(t, rows.Scan(&id, &label))
		labels[id] = label
	}
	require.NoError(t, rows.Close())
	assert.Equal(t, map[string]string{
		"p-deploy-2": "Deploy (2)",
		"p-deploy-a": "Deploy",
		"p-deploy-b": "deploy (2) (p-deploy)",
		"p-deploy-c": "Deploy (3)",
		"p-fix-a":    "Fix",
		"p-fix-b":    "Fix (2)",
		"p-other":    "Deploy",
		"p-alone":    "Alone",
	}, labels, "ties on created_at fall to the id; another workspace's label is its own")

	_, err = db.Exec(`INSERT INTO plays (id, workspace_id, label, type, created_at, updated_at) VALUES ('p-new', 'workspace-default', 'ALONE', 'doc', 10, 10)`)
	require.Error(t, err, "a label differing only in case is taken")

	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	install, err := s.IntegrationInstalls.GetByID(t.Context(), "install-writes")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"plays:read", "plays:run", "plays:write"}, scopeStrings(install.Scopes))
	install, err = s.IntegrationInstalls.GetByID(t.Context(), "install-reads")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"plays:read"}, scopeStrings(install.Scopes))
	automation, err := s.Automations.Get(t.Context(), "auto-writes")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"plays:run", "plays:write"}, scopeStrings(automation.Scopes))
	automation, err = s.Automations.Get(t.Context(), "auto-empty")
	require.NoError(t, err)
	assert.Empty(t, automation.Scopes)
}
