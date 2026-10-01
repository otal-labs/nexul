package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestMigration0051_ChannelsFromBeforeItStayPublicAndCanGoPrivate(t *testing.T) {
	db := migrateBefore(t, "0051")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-1', 'one', 1, 1), ('u-2', 'two', 1, 1);
INSERT INTO conversations (id, workspace_id, kind, name, created_by, created_at, updated_at) VALUES
    ('conv-eng', 'workspace-default', 'channel', 'eng', 'u-1', 1, 1);
INSERT INTO conversation_participants (conversation_id, user_id, created_at) VALUES ('conv-eng', 'u-1', 1);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0051 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	got, err := s.Chat.GetConversation(ctx, "conv-eng")
	require.NoError(t, err)
	assert.False(t, got.Private, "an existing channel stays public")
	assert.Empty(t, got.ParticipantIDs, "its creator's row makes no member list")
	listed, err := s.Chat.ListConversationsForUser(ctx, "workspace-default", "u-2")
	require.NoError(t, err)
	require.Len(t, listed, 1, "every member still lists it")

	require.NoError(t, s.Chat.SetChannelPrivate(ctx, "conv-eng", true, []string{"u-2"}, time.Unix(2, 0)))
	got, err = s.Chat.GetConversation(ctx, "conv-eng")
	require.NoError(t, err)
	assert.True(t, got.Private)
	assert.Equal(t, []string{"u-2"}, got.ParticipantIDs, "going private replaces the leftover creator row")

	require.NoError(t, s.Chat.AddParticipants(ctx, "conv-eng", []string{"u-1", "u-2"}, time.Unix(3, 0)))
	require.NoError(t, s.Chat.RemoveParticipants(ctx, "conv-eng", []string{"u-2"}))
	got, err = s.Chat.GetConversation(ctx, "conv-eng")
	require.NoError(t, err)
	assert.Equal(t, []string{"u-1"}, got.ParticipantIDs)

	require.NoError(t, s.Chat.SetChannelPrivate(ctx, "conv-eng", false, nil, time.Unix(4, 0)))
	got, err = s.Chat.GetConversation(ctx, "conv-eng")
	require.NoError(t, err)
	assert.False(t, got.Private)
	assert.Empty(t, got.ParticipantIDs)
	require.ErrorIs(t, s.Chat.SetChannelPrivate(ctx, "conv-gone", true, nil, time.Unix(5, 0)), apperrs.ErrNotFound)
}
