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

	require.NoError(t, Migrate(db), "0085 and every later migration apply on top, as an upgrade would")

	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	got, err := s.Projects.Get(t.Context(), "p-old")
	require.NoError(t, err)
	assert.Equal(t, workspace.ProjectSetup{Finished: true, Steps: map[workspace.SetupStep]workspace.SetupMark{}}, got.Setup)
}
