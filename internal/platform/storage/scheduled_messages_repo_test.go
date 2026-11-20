package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

func seedScheduledMessage(t *testing.T, s *Store, repeats chat.Repeat) *chat.ScheduledMessage {
	t.Helper()
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(t.Context(), newTestConversation("conv-1", chat.KindChannel, "eng", "", "u-1"), []string{"u-1"}))
	m := &chat.ScheduledMessage{
		ID: "sm-1", ConversationID: "conv-1", AuthorID: "u-1", Body: "Heads up for tomorrow", StartsAt: chatFixedNow,
		SendAt: chatFixedNow, Repeats: repeats, Timezone: "UTC", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
	}
	require.NoError(t, s.Chat.CreateScheduledMessage(t.Context(), m))
	return m
}

func sentMessage(id string) *chat.Message {
	return &chat.Message{ID: id, ConversationID: "conv-1", AuthorID: "u-1", AuthorKind: chat.AuthorUser, Body: "Heads up for tomorrow",
		CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}
}

func TestChatRepo_SendScheduledMessage_PostsOnceAndMovesOrRemovesIt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := seedScheduledMessage(t, s, chat.RepeatDaily)
	next := chatFixedNow.AddDate(0, 0, 1)

	require.NoError(t, s.Chat.SendScheduledMessage(t.Context(), m, sentMessage("msg-1"), next))
	err := s.Chat.SendScheduledMessage(t.Context(), m, sentMessage("msg-2"), next)

	require.ErrorIs(t, err, apperrs.ErrConflict, "a second pass over what it read before must not post again")
	got, err := s.Chat.GetScheduledMessage(t.Context(), "sm-1")
	require.NoError(t, err)
	assert.True(t, next.Equal(got.SendAt))
	_, err = s.Chat.GetMessage(t.Context(), "msg-2")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	moved := *got
	require.NoError(t, s.Chat.SendScheduledMessage(t.Context(), &moved, sentMessage("msg-3"), time.Time{}))
	_, err = s.Chat.GetScheduledMessage(t.Context(), "sm-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_SendScheduledMessage_AnEditMeanwhilePostsNothing(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := seedScheduledMessage(t, s, chat.RepeatNone)
	edited := *m
	edited.Body = "Heads up: moved to Friday"
	require.NoError(t, s.Chat.UpdateScheduledMessage(t.Context(), &edited))

	err := s.Chat.SendScheduledMessage(t.Context(), m, sentMessage("msg-1"), time.Time{})

	require.ErrorIs(t, err, apperrs.ErrConflict)
	_, err = s.Chat.GetMessage(t.Context(), "msg-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_ScheduledMessages_GoWithTheirConversation(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedScheduledMessage(t, s, chat.RepeatWeekly)

	require.NoError(t, s.Chat.DeleteConversation(t.Context(), "conv-1"))

	_, total, err := s.Chat.PageScheduledMessages(t.Context(), "conv-1", "u-1", paging.Window{})
	require.NoError(t, err)
	assert.Zero(t, total)
	_, ok, err := s.Chat.NextScheduledMessage(t.Context())
	require.NoError(t, err)
	assert.False(t, ok)
}
