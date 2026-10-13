package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0104_ProjectsMadeBeforeWhosInItCountItDone: a project in setup keeps its other marks and resumes past
// the new step.
func TestMigration0104_ProjectsMadeBeforeWhosInItCountItDone(t *testing.T) {
	db := migrateBefore(t, "0104")
	_, err := db.Exec(`
INSERT INTO projects (id, workspace_id, name, prefix, position, created_at, updated_at, setup_finished, setup_steps) VALUES
    ('p-setup', 'workspace-default', 'Setup', 'SET', 0, 1, 1, 0, '{"project":"done","repository":"skipped"}'),
    ('p-done', 'workspace-default', 'Done', 'DON', 1, 2, 2, 1, '{}');
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	want := map[string]string{
		"p-setup": `{"project":"done","repository":"skipped","people":"done"}`,
		"p-done":  `{"people":"done"}`,
	}
	for id, steps := range want {
		var got string
		require.NoError(t, db.QueryRow(`SELECT setup_steps FROM projects WHERE id = ?`, id).Scan(&got))
		assert.JSONEq(t, steps, got, id)
	}
}
