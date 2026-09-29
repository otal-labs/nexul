package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration0036_BackfillsNotificationWorkspaces_DropsUnresolvable(t *testing.T) {
	db := migrateBefore(t, "0036")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u1', 'owner', 0, 0);
INSERT INTO workspaces (id, name, created_at, updated_at) VALUES ('ws-2', 'Second', 0, 0);
INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at) VALUES ('p-2', 'Two', 0, 'ws-2', 0, 0);
INSERT INTO tickets (id, title, status, project_id, created_at, updated_at) VALUES ('t-1', 'Ticket', 'open', 'p-2', 0, 0), ('t-loose', 'Loose', 'open', NULL, 0, 0);
INSERT INTO docs (id, title, body, project_id, created_at, updated_at) VALUES ('d-1', 'Doc', '', 'p-2', 0, 0);
INSERT INTO memories (id, workspace_id, title, body, created_at, updated_at) VALUES ('m-1', 'ws-2', 'Memory', '', 0, 0);
INSERT INTO notifications (id, user_id, workspace_id, kind, subject_type, subject_id, read, created_at) VALUES
    ('n-ticket', 'u1', '', 'ticket.assigned', 'ticket', 't-1', 0, 0),
    ('n-doc', 'u1', '', 'doc.created', 'doc', 'd-1', 0, 0),
    ('n-memory', 'u1', '', 'memory.updated', 'memory', 'm-1', 0, 0),
    ('n-scoped', 'u1', 'workspace-default', 'doc.created', 'doc', 'd-1', 0, 0),
    ('n-gone', 'u1', '', 'ticket.assigned', 'ticket', 't-deleted', 0, 0),
    ('n-loose', 'u1', '', 'ticket.assigned', 'ticket', 't-loose', 0, 0);
`)
	require.NoError(t, err)

	script, err := migrationFS.ReadFile("migrations/0036_notification_workspace_backfill.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0036_notification_workspace_backfill", string(script)))

	for _, id := range []string{"n-ticket", "n-doc", "n-memory"} {
		assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM notifications WHERE id = ? AND workspace_id = 'ws-2'`, id), id)
	}
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM notifications WHERE id = 'n-scoped' AND workspace_id = 'workspace-default'`),
		"a row that already has a workspace keeps it")
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM notifications WHERE workspace_id = ''`),
		"a subject that no longer resolves to a workspace leaves no row no inbox shows")
}
