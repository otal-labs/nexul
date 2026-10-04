package storage

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
)

// TestMigration0064_RepliesCarryTheirHandoffsUntilDeleted upgrades over a reply from before hand-offs, which reads with none.
func TestMigration0064_RepliesCarryTheirHandoffsUntilDeleted(t *testing.T) {
	db := migrateBefore(t, "0064")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1);
INSERT INTO conversations (id, workspace_id, kind, name, created_by, created_at, updated_at) VALUES ('c-1', 'workspace-default', 'channel', 'eng', 'u-1', 1, 1);
INSERT INTO messages (id, conversation_id, author_id, author_kind, body, created_at, updated_at) VALUES ('m-old', 'c-1', 'u-1', 'agent', 'Done.', 1, 1);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0064 applies on top, as an upgrade would")

	ctx := t.Context()
	repo := New(db, testEncKey).Chat
	old, err := repo.GetMessage(ctx, "m-old")
	require.NoError(t, err)
	assert.Nil(t, old.Handoffs, "a reply from before reads with no hand-offs")

	at := time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC)
	handoffs := []chat.Handoff{{ID: "native-8", Driver: "claudeAgent", Model: "claude-opus-5-5", Title: "Find callers",
		Prompt: "Find every caller of mapItem.", State: "done", Reply: "Two callers.",
		Steps: []chat.HandoffStep{{Kind: "tool_result", CallID: "c-1", Tool: "Shell", Summary: "rg -n mapItem internal", At: at}}}}
	require.NoError(t, repo.CreateMessage(ctx, &chat.Message{ID: "m-new", ConversationID: "c-1", AuthorID: "u-1", AuthorKind: chat.AuthorAgent,
		Body: "Found them.", CreatedAt: at, UpdatedAt: at, Handoffs: handoffs}))
	got, err := repo.GetMessage(ctx, "m-new")
	require.NoError(t, err)
	assert.Equal(t, handoffs, got.Handoffs)

	require.NoError(t, repo.DeleteMessage(ctx, "m-new", at.Add(time.Minute)))
	var stored sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, `SELECT handoffs FROM messages WHERE id = 'm-new'`).Scan(&stored))
	assert.False(t, stored.Valid, "deleting the reply removes its hand-offs")
}
