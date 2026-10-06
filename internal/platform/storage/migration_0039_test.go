package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0039_BackfillsSlugsAndScopesPrefixesToTheirWorkspace(t *testing.T) {
	db := migrateBefore(t, "0039")
	_, err := db.Exec(`
UPDATE workspaces SET name = 'Norwood Labs!' WHERE id = 'workspace-default';
INSERT INTO workspaces (id, name, created_at, updated_at) VALUES
    ('ws-otal', 'Otal', 10, 10),
    ('ws-otal-twin', '  OTAL  ', 20, 20),
    ('0199c0de-7a1b-7c2d-8e3f-000000000001', 'Otal 2', 15, 15),
    ('ws-settings', 'Settings', 30, 30),
    ('ws-accents', 'Café Crème', 40, 40),
    ('ws-symbols', '!!!', 50, 50);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES
    ('p-1', 'Web', 'WEB', 0, 'ws-otal', 0, 0);
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0039_workspace_slugs.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0039_workspace_slugs", string(script)))

	slugs := map[string]string{}
	rows, err := db.Query(`SELECT id, slug FROM workspaces`)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rows.Close()) })
	for rows.Next() {
		var id, slug string
		require.NoError(t, rows.Scan(&id, &slug))
		slugs[id] = slug
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, map[string]string{
		"workspace-default":                    "norwood-labs",
		"ws-otal":                              "otal",
		"ws-otal-twin":                         "otal-2-ws-otal",
		"0199c0de-7a1b-7c2d-8e3f-000000000001": "otal-2",
		"ws-settings":                          "settings-2",
		"ws-accents":                           "caf-cr-me",
		"ws-symbols":                           "workspace",
	}, slugs)

	_, err = db.Exec(`INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-2', 'Web', 'WEB', 0, 'ws-settings', 0, 0)`)
	require.NoError(t, err, "another workspace may reuse a prefix")
	_, err = db.Exec(`INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-3', 'Web again', 'WEB', 1, 'ws-otal', 0, 0)`)
	require.Error(t, err, "a prefix stays unique inside its workspace")
	_, err = db.Exec(`UPDATE workspaces SET slug = 'otal' WHERE id = 'ws-symbols'`)
	require.Error(t, err, "slugs are unique across the instance")
}
