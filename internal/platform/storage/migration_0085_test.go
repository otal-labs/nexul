package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/workspace"
)

// TestMigration0085_ExistingProjectsCountAsSetUp: a project from before the setup record upgrades to finished.
func TestMigration0085_ExistingProjectsCountAsSetUp(t *testing.T) {
	db := migrateBefore(t, "0085")
	_, err := db.Exec(`INSERT INTO projects (id, name, prefix, position, workspace_id, icon, created_at, updated_at)
VALUES ('p-old', 'Backend', 'BE', 0, 'workspace-default', '', 1, 1);`)
	require.NoError(t, err)

	for _, version := range []string{"0086_unique_play_labels", "0087_decisions_check_play"} {
		script, err := migrationFS.ReadFile("migrations/" + version + ".sql")
		require.NoError(t, err)
		require.NoError(t, applyMigration(db, version, string(script)))
	}
	pending, err := Pending(db)
	require.NoError(t, err)
	assert.Contains(t, pending, "0085_project_setup")
	require.NoError(t, MigrateWithBackup(db, t.TempDir(), "test"))
	require.NoError(t, MigrateWithBackup(db, t.TempDir(), "test"))

	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	got, err := s.Projects.Get(t.Context(), "p-old")
	require.NoError(t, err)
	assert.Equal(t, workspace.ProjectSetup{Finished: true, Steps: map[workspace.SetupStep]workspace.SetupMark{}}, got.Setup)
}
