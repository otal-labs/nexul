package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/memories"
)

// TestMigration0061_AnswersOutliveTheMemoryButNotTheProject upgrades a database holding a project with an interview
// memory, then checks the answers stay when the memory goes and go with the project.
func TestMigration0061_AnswersOutliveTheMemoryButNotTheProject(t *testing.T) {
	db := migrateBefore(t, "0061")
	_, err := db.Exec(`
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO memories (id, workspace_id, project_id, kind, title, body, always_included, created_at, updated_at) VALUES
    ('m-interview', 'workspace-default', 'p-web', 'interview', 'Interview', '', 1, 0, 0);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0061 applies on top, as an upgrade would")

	ctx := t.Context()
	s := New(db, testEncKey)
	at := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	saved, err := s.Memories.UpsertAnswer(ctx, &memories.InterviewAnswer{
		ID: "a-1", WorkspaceID: "workspace-default", ProjectID: "p-web", Question: "Testing", Selected: []string{"Unit"}, AnsweredBy: "u-1", AnsweredAt: at,
	})
	require.NoError(t, err)
	assert.Nil(t, saved.Options, "a template question has no options stored")

	_, err = db.Exec(`DELETE FROM memories WHERE id = 'm-interview'`)
	require.NoError(t, err)
	as, err := s.Memories.ListAnswers(ctx, "p-web")
	require.NoError(t, err)
	assert.Len(t, as, 1, "deleting the interview memory keeps the answers")

	_, err = db.Exec(`DELETE FROM projects WHERE id = 'p-web'`)
	require.NoError(t, err)
	as, err = s.Memories.ListAnswers(ctx, "p-web")
	require.NoError(t, err)
	assert.Empty(t, as, "deleting the project removes them")
}
