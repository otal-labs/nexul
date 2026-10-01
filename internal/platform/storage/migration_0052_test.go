package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0052_WorkspaceMemoriesGoWithTheirVersionsAndAttachments upgrades a database holding a workspace
// memory and a project memory, each with a version and an attachment; migrations run with foreign keys off, so
// nothing cascades for free.
func TestMigration0052_WorkspaceMemoriesGoWithTheirVersionsAndAttachments(t *testing.T) {
	db := migrateBefore(t, "0052")
	_, err := db.Exec(`
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO memories (id, workspace_id, project_id, title, body, version, created_at, updated_at) VALUES
    ('m-ws', 'workspace-default', NULL, 'Team tone', '', 2, 0, 0),
    ('m-web', 'workspace-default', 'p-web', 'Deploy quirks', '', 1, 0, 0);
INSERT INTO memory_versions (id, memory_id, version, title, body, created_at) VALUES
    ('v-ws-1', 'm-ws', 1, 'Team tone', '', 0),
    ('v-ws-2', 'm-ws', 2, 'Team tone', '', 0),
    ('v-web-1', 'm-web', 1, 'Deploy quirks', '', 0);
INSERT INTO attachments (id, memory_id, name, content_type, size, data, created_at) VALUES
    ('a-ws', 'm-ws', 'tone.png', 'image/png', 1, x'00', 0),
    ('a-web', 'm-web', 'deploy.png', 'image/png', 1, x'00', 0);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0052 and every later migration apply on top, as an upgrade would")

	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM memories WHERE project_id IS NULL`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM memory_versions WHERE memory_id = 'm-ws'`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM attachments WHERE memory_id = 'm-ws'`))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM memories WHERE id = 'm-web'`), "project memories are untouched")
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM memory_versions WHERE memory_id = 'm-web'`))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM attachments WHERE memory_id = 'm-web'`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE name = 'idx_memories_workspace_scoped'`))
}
