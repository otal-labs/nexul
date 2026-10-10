package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/repository"
)

// TestMigration0088_EachWorkspaceKeepsTheAccountsItsProjectsUse upgrades a database at the previous schema: every
// workspace is assigned the GitHub accounts its projects attach repositories from, and nothing else.
func TestMigration0088_EachWorkspaceKeepsTheAccountsItsProjectsUse(t *testing.T) {
	db := migrateBefore(t, "0088")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES
    ('ws-acme', 'Acme', 'acme', 1, 1), ('ws-globex', 'Globex', 'globex', 1, 1), ('ws-empty', 'Empty', 'empty', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES
    ('p-api', 'API', 'API', 1, 'ws-acme', 1, 1), ('p-web', 'Web', 'WEB', 2, 'ws-acme', 1, 1),
    ('p-shop', 'Shop', 'SHOP', 3, 'ws-globex', 1, 1), ('p-lab', 'Lab', 'LAB', 4, 'ws-empty', 1, 1);
INSERT INTO project_repos (project_id, owner, name, full_name, added_at, connector_id) VALUES
    ('p-api', 'Acme', 'api', 'Acme/api', 1, 'github'), ('p-web', 'acme', 'web', 'acme/web', 1, 'github'),
    ('p-shop', 'globex', 'shop', 'globex/shop', 1, 'github'), ('p-shop', 'acme', 'shared', 'acme/shared', 1, 'github'),
    ('p-lab', 'initech', 'lab', 'initech/lab', 1, 'gitlab');
INSERT INTO connector_app_config (connector_id, client_id, client_secret, app_slug) VALUES ('github', 'Iv1.acme', '', 'nexul-acme');
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0088 and every later migration apply on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	assigned, err := s.GitHubInstallations.ListInstallationWorkspaces(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string][]repository.InstallationWorkspace{
		"acme":   {{ID: "ws-acme", Name: "Acme"}, {ID: "ws-globex", Name: "Globex"}},
		"globex": {{ID: "ws-globex", Name: "Globex"}},
	}, assigned, "one row per account and workspace, lowercase, and none from another git host")

	cfg, err := s.ConnectorAppConfig.GetAppConfig(ctx, "github")
	require.NoError(t, err)
	assert.Equal(t, connectors.AppConfig{ConnectorID: "github", ClientID: "Iv1.acme", AppSlug: "nexul-acme"}, cfg, "the App keeps reading as the connected account until a key is set")
}
