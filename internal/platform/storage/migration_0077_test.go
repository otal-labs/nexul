package storage

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
)

const messagesBefore0077 = `SELECT id, conversation_id, author_id, body, mentions, attachment_id, edited_at, deleted_at, created_at, updated_at,
author_kind, handoffs, via FROM messages ORDER BY id`

func messageRows(t *testing.T, db *sql.DB) [][]any {
	t.Helper()
	rows, err := db.Query(messagesBefore0077)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	var out [][]any
	for rows.Next() {
		row := make([]any, 13)
		ptrs := make([]any, len(row))
		for i := range row {
			ptrs[i] = &row[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		out = append(out, row)
	}
	require.NoError(t, rows.Err())
	return out
}

// TestMigration0077_EveryMessageSurvivesTheRebuild upgrades a database holding every kind of message: each row comes
// through column for column, its reactions stay, a note still reaches search, and a bot may now author a message.
func TestMigration0077_EveryMessageSurvivesTheRebuild(t *testing.T) {
	db := migrateBefore(t, "0077")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1), ('u-bob', 'bob', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO tickets (id, project_id, title, status, created_at, updated_at) VALUES ('t-1', 'p-web', 'Login times out', 'open', 1, 1);
INSERT INTO conversations (id, workspace_id, kind, name, created_by, created_at, updated_at) VALUES ('c-eng', 'workspace-default', 'channel', 'eng', 'u-alice', 1, 1);
INSERT INTO conversations (id, workspace_id, kind, ticket_id, created_by, created_at, updated_at) VALUES ('c-t1', 'workspace-default', 'ticket_thread', 't-1', 'u-alice', 1, 1);
INSERT INTO attachments (id, conversation_id, name, content_type, size, uploaded_by, created_at, data) VALUES ('a-note', 'c-t1', 'plan.md', 'text/markdown', 5, 'u-alice', 1, 'steps');
INSERT INTO messages (id, conversation_id, author_id, body, mentions, attachment_id, edited_at, deleted_at, created_at, updated_at, author_kind, handoffs, via) VALUES
    ('m-1', 'c-eng', 'u-alice', 'hi @bob', '[{"kind":"user","handle":"bob"}]', NULL, 5, NULL, 2, 5, 'user', NULL, ''),
    ('m-2', 'c-eng', 'u-bob', 'done', '[]', NULL, NULL, NULL, 3, 3, 'agent', '[{"id":"task-1"}]', ''),
    ('m-3', 'c-eng', 'u-bob', '', '[]', NULL, NULL, 9, 4, 9, 'user', NULL, ''),
    ('m-4', 'c-eng', 'u-alice', 'from the harness', '[]', NULL, NULL, NULL, 5, 5, 'user', NULL, 'T3'),
    ('m-5', 'c-t1', 'u-alice', 'the plan', '[]', 'a-note', NULL, NULL, 6, 6, 'agent', NULL, ''),
    ('m-6', 'c-t1', 'u-alice', 'connect T3', '[]', NULL, NULL, NULL, 7, 7, 'system', NULL, '');
INSERT INTO message_reactions (message_id, emoji, user_id, created_at) VALUES ('m-1', '👍', 'u-bob', 8);`)
	require.NoError(t, err)
	before := messageRows(t, db)

	require.NoError(t, Migrate(db), "0077 and every later migration apply on top, as an upgrade would")

	assert.Equal(t, before, messageRows(t, db), "every message keeps every column it had")
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM message_reactions WHERE message_id = 'm-1'`), "rebuilding messages does not cascade its reactions away")
	assert.Equal(t, 6, count(t, db, `SELECT COUNT(*) FROM messages WHERE author_name IS NULL AND author_avatar_url IS NULL AND embeds IS NULL`))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_messages_conversation'`))

	_, err = db.Exec(`
INSERT INTO attachments (id, conversation_id, name, content_type, size, uploaded_by, created_at, data) VALUES ('a-later', 'c-t1', 'later.md', 'text/markdown', 9, 'u-alice', 10, 'rollback');
INSERT INTO messages (id, conversation_id, author_id, body, attachment_id, created_at, updated_at, author_kind) VALUES ('m-7', 'c-t1', 'u-alice', 'later', 'a-later', 10, 10, 'agent');`)
	require.NoError(t, err)
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM ticket_notes_fts WHERE ticket_notes_fts MATCH 'rollback'`), "a note posted after the upgrade still reaches search")

	s := New(db, testEncKey)
	ctx := t.Context()
	at := time.Unix(20, 0).UTC()
	bot := &botwebhook.Bot{ID: "b-ci", ConversationID: "c-eng", Name: "CI", Token: "tok", CreatedBy: "u-alice", CreatedAt: at, UpdatedAt: at}
	require.NoError(t, s.Botwebhooks.Create(ctx, bot))
	require.NoError(t, s.Chat.CreateMessage(ctx, &chat.Message{
		ID: "m-bot", ConversationID: "c-eng", AuthorID: bot.ID, AuthorKind: chat.AuthorBot, AuthorName: "GitHub", AuthorAvatarURL: "https://example.com/gh.png",
		Body: "build passed", CreatedAt: at, UpdatedAt: at,
	}), "a bot's id is a message author now that the users foreign key is gone")

	bot.Name, bot.UpdatedAt = "Builds", at.Add(time.Minute)
	require.NoError(t, s.Botwebhooks.Update(ctx, bot))
	deleted := at.Add(time.Hour)
	bot.DeletedAt = &deleted
	require.NoError(t, s.Botwebhooks.Update(ctx, bot))
	got, err := s.Chat.GetMessage(ctx, "m-bot")
	require.NoError(t, err)
	assert.Equal(t, "GitHub", got.AuthorName, "a bot message keeps the name it showed through its bot's rename and delete")
	assert.Equal(t, "https://example.com/gh.png", got.AuthorAvatarURL)
	assert.Equal(t, chat.AuthorBot, got.AuthorKind)
}
