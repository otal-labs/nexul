package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0058_BackfillsNotesAlreadyPosted upgrades a database holding a live note, a deleted one, and a
// conversation file no message links, the three states a ticket thread's files can be in at 0057.
func TestMigration0058_BackfillsNotesAlreadyPosted(t *testing.T) {
	db := migrateBefore(t, "0058")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'onik', 0, 0);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO tickets (id, title, status, project_id, created_at, updated_at) VALUES ('t-1', 'Slow board', 'open', 'p-web', 0, 0);
INSERT INTO conversations (id, workspace_id, kind, ticket_id, created_by, created_at, updated_at) VALUES ('c-1', 'workspace-default', 'ticket_thread', 't-1', 'u-1', 0, 0);
INSERT INTO attachments (id, conversation_id, name, content_type, size, data, created_at) VALUES
    ('a-live', 'c-1', 'context.md', 'text/markdown; charset=utf-8', 15, CAST('reconnect storm' AS BLOB), 0),
    ('a-image', 'c-1', 'virtualized.png', 'image/png', 11, CAST('virtualized' AS BLOB), 0);
INSERT INTO messages (id, conversation_id, author_id, author_kind, body, attachment_id, deleted_at, created_at, updated_at) VALUES
    ('m-live', 'c-1', 'u-1', 'agent', '', 'a-live', NULL, 0, 0),
    ('m-gone', 'c-1', 'u-1', 'agent', '', NULL, 1, 0, 0);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0058 and every later migration apply on top, as an upgrade would")

	s := New(db, testEncKey)
	hits, err := s.Tickets.Search(context.Background(), "reconnect", 10)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "t-1", hits[0].ID)
	hits, err = s.Tickets.Search(context.Background(), "virtualized", 10)
	require.NoError(t, err)
	assert.Empty(t, hits, "a conversation file no message links is not a note")
}
