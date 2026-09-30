package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestMigration0045_ChannelNamesFromBeforeItCollideCaseInsensitively(t *testing.T) {
	db := migrateBefore(t, "0045")
	_, err := db.Exec(`INSERT INTO users (id, login, created_at, updated_at) VALUES ('u1', 'onik97', 1, 1);
INSERT INTO conversations (id, workspace_id, kind, name, created_by, created_at, updated_at) VALUES ('c-old', 'workspace-default', 'channel', 'general', 'u1', 1, 1)`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0045 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)

	err = s.Chat.CreateConversation(t.Context(), newTestConversation("c-new", chat.KindChannel, "General", "", "u1"), []string{"u1"})
	require.ErrorIs(t, err, apperrs.ErrConflict, "a differently cased name is the same channel")

	got, err := s.Chat.GetChannelByName(t.Context(), "workspace-default", "GENERAL")
	require.NoError(t, err)
	assert.Equal(t, "c-old", got.ID, "lookup by name ignores case")

	require.NoError(t, s.Chat.CreateConversation(t.Context(), newTestConversation("c-voice", chat.KindVoiceChannel, "General", "", "u1"), []string{"u1"}))
}
