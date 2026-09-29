package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0038_BackfillsDocAuthorFromTheCreatorGrant(t *testing.T) {
	db := migrateBefore(t, "0038")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-author', 'author', 0, 0), ('u-shared', 'shared', 0, 0);
INSERT INTO workspaces (id, name, created_at, updated_at) VALUES ('ws-2', 'Second', 0, 0);
INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at) VALUES ('p-2', 'Two', 0, 'ws-2', 0, 0);
INSERT INTO docs (id, title, body, project_id, created_at, updated_at) VALUES ('d-1', 'Doc', '', 'p-2', 0, 0), ('d-orphan', 'Orphan', '', 'p-2', 0, 0);
INSERT INTO permission_overwrites (resource_type, resource_id, user_id, allow, created_at, updated_at) VALUES
    ('doc', 'd-1', 'u-shared', '["docs:read"]', 20, 20),
    ('doc', 'd-1', 'u-author', '["docs:delete","docs:read","docs:write","permissions:write"]', 10, 30);
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0038_doc_created_by.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0038_doc_created_by", string(script)))

	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM docs WHERE id = 'd-1' AND created_by = 'u-author'`),
		"the earliest overwrite, the creator grant, names the author even after someone else was shared in")
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM docs WHERE id = 'd-orphan' AND created_by = ''`),
		"a doc with no overwrite keeps an empty author")
}
