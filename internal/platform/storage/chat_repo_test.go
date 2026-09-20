package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var chatFixedNow = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func newTestConversation(id string, kind chat.Kind, name, ticketID, createdBy string) *chat.Conversation {
	return &chat.Conversation{
		ID: id, WorkspaceID: "workspace-default", Kind: kind, Name: name, TicketID: ticketID,
		CreatedBy: createdBy, CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
	}
}

func newTestDocThreadConversation(id, docID, createdBy string) *chat.Conversation {
	return &chat.Conversation{
		ID: id, WorkspaceID: "workspace-default", Kind: chat.KindDocThread, DocID: docID,
		CreatedBy: createdBy, CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
	}
}

func seedChatUser(t *testing.T, s *Store, id string) {
	t.Helper()
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser(id, id, id))
	require.NoError(t, err)
}

func TestChatRepo_CreateConversation_GetConversation_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	c := newTestConversation("conv-1", chat.KindChannel, "engineering", "", "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), c, []string{"u-1"}))

	got, err := s.Chat.GetConversation(context.Background(), "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "engineering", got.Name)
	assert.Equal(t, chat.KindChannel, got.Kind)
	assert.Equal(t, "workspace-default", got.WorkspaceID)
}

func TestChatRepo_GetConversation_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Chat.GetConversation(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_CreateConversation_DuplicateChannelName_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	err := s.Chat.CreateConversation(context.Background(), newTestConversation("conv-2", chat.KindChannel, "general", "", "u-1"), []string{"u-1"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestChatRepo_GetChannelByName_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))

	got, err := s.Chat.GetChannelByName(context.Background(), "workspace-default", "general")
	require.NoError(t, err)
	assert.Equal(t, "conv-1", got.ID)

	_, err = s.Chat.GetChannelByName(context.Background(), "workspace-default", "does-not-exist")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_GetTicketThread_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Chat.GetTicketThread(context.Background(), "t-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_GetDocThread_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Chat.GetDocThread(context.Background(), "doc-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_CreateConversation_DocThread_DuplicateDoc_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestDocThreadConversation("conv-1", "doc-1", "u-1"), []string{"u-1"}))
	err := s.Chat.CreateConversation(context.Background(), newTestDocThreadConversation("conv-2", "doc-1", "u-1"), []string{"u-1"})
	require.ErrorIs(t, err, apperrs.ErrConflict)

	got, err := s.Chat.GetDocThread(context.Background(), "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "conv-1", got.ID)
	assert.Equal(t, chat.KindDocThread, got.Kind)
	assert.Equal(t, "doc-1", got.DocID)
}

func TestChatRepo_CreateConversation_TicketThread_DuplicateTicket_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "")))

	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindTicketThread, "", "t-1", "u-1"), []string{"u-1"}))
	err := s.Chat.CreateConversation(context.Background(), newTestConversation("conv-2", chat.KindTicketThread, "", "t-1", "u-1"), []string{"u-1"})
	require.ErrorIs(t, err, apperrs.ErrConflict)

	got, err := s.Chat.GetTicketThread(context.Background(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, "conv-1", got.ID)
}

func TestChatRepo_HasTicketThreads_BatchesAndOmitsMissing(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-2", "")))
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindTicketThread, "", "t-1", "u-1"), []string{"u-1"}))

	got, err := s.Chat.HasTicketThreads(context.Background(), []string{"t-1", "t-2"})
	require.NoError(t, err)
	assert.True(t, got["t-1"])
	_, ok := got["t-2"]
	assert.False(t, ok, "ticket with no thread is absent, not false")
}

func TestChatRepo_HasTicketThreads_Empty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	got, err := s.Chat.HasTicketThreads(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestChatRepo_ListConversationsForUser_ChannelsPlusOwnParticipation(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	seedChatUser(t, s, "u-2")
	seedChatUser(t, s, "u-3")

	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("channel-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("dm-mine", chat.KindDM, "", "", "u-1"), []string{"u-1", "u-2"}))
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("dm-other", chat.KindDM, "", "", "u-2"), []string{"u-2", "u-3"}))

	got, err := s.Chat.ListConversationsForUser(context.Background(), "workspace-default", "u-1")
	require.NoError(t, err)
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	assert.Contains(t, ids, "channel-1")
	assert.Contains(t, ids, "dm-mine")
	assert.NotContains(t, ids, "dm-other")
}

func TestChatRepo_ListConversationsForUser_VoiceChannelsArePublicLikeChannels(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	seedChatUser(t, s, "u-2")

	// u-1 creates the voice channel (its only explicit participant row); u-2 never joins it.
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("voice-1", chat.KindVoiceChannel, "hangout", "", "u-1"), []string{"u-1"}))

	got, err := s.Chat.ListConversationsForUser(context.Background(), "workspace-default", "u-2")
	require.NoError(t, err)
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	assert.Contains(t, ids, "voice-1", "voice channels are public to the whole workspace like regular channels")
}

func TestChatRepo_ListConversationsForUser_DocThreadsAreRowVisibleToTheWholeWorkspace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	seedChatUser(t, s, "u-2")
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	// u-1 creates the doc thread (its only explicit participant row); u-2 never joins it. The row is
	// still returned here — chat.Service.ListConversations is what filters it by docs:thread (ticket 19).
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestDocThreadConversation("doc-thread-1", "doc-1", "u-1"), []string{"u-1"}))

	got, err := s.Chat.ListConversationsForUser(context.Background(), "workspace-default", "u-2")
	require.NoError(t, err)
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	assert.Contains(t, ids, "doc-thread-1")
}

func TestChatRepo_ListConversationsForUser_PopulatesDMParticipantIDs(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	seedChatUser(t, s, "u-2")
	seedChatUser(t, s, "u-3")

	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("channel-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("dm-1", chat.KindDM, "", "", "u-1"), []string{"u-1", "u-2", "u-3"}))

	got, err := s.Chat.ListConversationsForUser(context.Background(), "workspace-default", "u-1")
	require.NoError(t, err)

	byID := make(map[string]*chat.Conversation, len(got))
	for _, c := range got {
		byID[c.ID] = c
	}
	assert.ElementsMatch(t, []string{"u-1", "u-2", "u-3"}, byID["dm-1"].ParticipantIDs)
	assert.Empty(t, byID["channel-1"].ParticipantIDs, "channels don't carry an explicit member list")
}

func TestChatRepo_CreateMessage_GetMessage_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))

	m := &chat.Message{
		ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-1", Body: "hey @onik97",
		Mentions:  []chat.Mention{{Kind: chat.MentionUser, Handle: "onik97"}},
		CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
	}
	require.NoError(t, s.Chat.CreateMessage(context.Background(), m))

	got, err := s.Chat.GetMessage(context.Background(), "msg-1")
	require.NoError(t, err)
	assert.Equal(t, "hey @onik97", got.Body)
	assert.Equal(t, []chat.Mention{{Kind: chat.MentionUser, Handle: "onik97"}}, got.Mentions)
	assert.Nil(t, got.EditedAt)
	assert.Nil(t, got.DeletedAt)
}

func TestChatRepo_GetMessage_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Chat.GetMessage(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_CreateMessage_UnknownConversation_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	m := &chat.Message{ID: "msg-1", ConversationID: "does-not-exist", AuthorID: "u-1", Body: "hi", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}
	err := s.Chat.CreateMessage(context.Background(), m)
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestChatRepo_ListMessages_OrderedAndLimited(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	for i, id := range []string{"msg-1", "msg-2", "msg-3"} {
		m := &chat.Message{
			ID: id, ConversationID: "conv-1", AuthorID: "u-1", Body: id,
			CreatedAt: chatFixedNow.Add(time.Duration(i) * time.Minute), UpdatedAt: chatFixedNow,
		}
		require.NoError(t, s.Chat.CreateMessage(context.Background(), m))
	}

	got, err := s.Chat.ListMessages(context.Background(), "conv-1", 2)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "msg-1", got[0].ID)
	assert.Equal(t, "msg-2", got[1].ID)
}

func TestChatRepo_UpdateMessage_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-1", Body: "hi", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}))

	editedAt := chatFixedNow.Add(time.Minute)
	mentions := []chat.Mention{{Kind: chat.MentionAgent, Handle: chat.AgentHandle}}
	require.NoError(t, s.Chat.UpdateMessage(context.Background(), "msg-1", "hi @Agent", mentions, editedAt))

	got, err := s.Chat.GetMessage(context.Background(), "msg-1")
	require.NoError(t, err)
	assert.Equal(t, "hi @Agent", got.Body)
	assert.Equal(t, mentions, got.Mentions)
	require.NotNil(t, got.EditedAt)
	assert.Equal(t, editedAt.Unix(), got.EditedAt.Unix())
}

func TestChatRepo_UpdateMessage_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Chat.UpdateMessage(context.Background(), "missing", "hi", nil, chatFixedNow)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_DeleteMessage_SoftDeletes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-1", Body: "hi", Mentions: []chat.Mention{{Kind: chat.MentionUser, Handle: "onik97"}}, CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}))

	deletedAt := chatFixedNow.Add(time.Minute)
	require.NoError(t, s.Chat.DeleteMessage(context.Background(), "msg-1", deletedAt))

	got, err := s.Chat.GetMessage(context.Background(), "msg-1")
	require.NoError(t, err)
	assert.Empty(t, got.Body)
	assert.Empty(t, got.Mentions)
	require.NotNil(t, got.DeletedAt)
	assert.Equal(t, deletedAt.Unix(), got.DeletedAt.Unix())
}

func TestChatRepo_DeleteMessage_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Chat.DeleteMessage(context.Background(), "missing", chatFixedNow)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestChatRepo_MarkRead_UnreadCounts(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	seedChatUser(t, s, "u-2")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))

	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-2", Body: "first", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}))
	require.NoError(t, s.Chat.MarkRead(context.Background(), "conv-1", "u-1", chatFixedNow.Add(30*time.Second)))
	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-2", ConversationID: "conv-1", AuthorID: "u-2", Body: "second", CreatedAt: chatFixedNow.Add(time.Minute), UpdatedAt: chatFixedNow}))
	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-3", ConversationID: "conv-1", AuthorID: "u-1", Body: "own, excluded", CreatedAt: chatFixedNow.Add(time.Minute), UpdatedAt: chatFixedNow}))

	counts, err := s.Chat.UnreadCounts(context.Background(), "workspace-default", "u-1")
	require.NoError(t, err)
	assert.Equal(t, 1, unreadCountsMap(counts)["conv-1"])

	// u-2 never marked read; from u-2's perspective, only u-1's one message counts (u-2's own two are excluded as author).
	counts2, err := s.Chat.UnreadCounts(context.Background(), "workspace-default", "u-2")
	require.NoError(t, err)
	assert.Equal(t, 1, unreadCountsMap(counts2)["conv-1"])
}

func unreadCountsMap(rows []chat.UnreadCount) map[string]int {
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.ConversationID] = r.Count
	}
	return out
}

func TestChatRepo_MarkRead_UpsertsOnRepeatedCalls(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.MarkRead(context.Background(), "conv-1", "u-1", chatFixedNow))
	require.NoError(t, s.Chat.MarkRead(context.Background(), "conv-1", "u-1", chatFixedNow.Add(time.Hour)))
}

// TestChatRepo_CreateMessage_AuthorKind_RoundTrip: it round-trips, defaulting to "user" for a zero value.
func TestChatRepo_CreateMessage_AuthorKind_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))

	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{
		ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-1", AuthorKind: chat.AuthorAgent, Body: "reply",
		CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
	}))
	got, err := s.Chat.GetMessage(context.Background(), "msg-1")
	require.NoError(t, err)
	assert.Equal(t, chat.AuthorAgent, got.AuthorKind)

	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{
		ID: "msg-2", ConversationID: "conv-1", AuthorID: "u-1", Body: "hi",
		CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
	}))
	got2, err := s.Chat.GetMessage(context.Background(), "msg-2")
	require.NoError(t, err)
	assert.Equal(t, chat.AuthorUser, got2.AuthorKind)
}

func TestChatRepo_ListMessagesSince_ExcludesOlderAndDeleted(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))

	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-1", Body: "before", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}))
	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-2", ConversationID: "conv-1", AuthorID: "u-1", Body: "after", CreatedAt: chatFixedNow.Add(time.Minute), UpdatedAt: chatFixedNow}))
	require.NoError(t, s.Chat.CreateMessage(context.Background(), &chat.Message{ID: "msg-3", ConversationID: "conv-1", AuthorID: "u-1", Body: "after-deleted", CreatedAt: chatFixedNow.Add(2 * time.Minute), UpdatedAt: chatFixedNow}))
	require.NoError(t, s.Chat.DeleteMessage(context.Background(), "msg-3", chatFixedNow.Add(3*time.Minute)))

	got, err := s.Chat.ListMessagesSince(context.Background(), "conv-1", chatFixedNow)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "msg-2", got[0].ID)
}

func TestChatRepo_SetAgentThread_And_SetAgentSyncedAt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))

	got, err := s.Chat.GetConversation(context.Background(), "conv-1")
	require.NoError(t, err)
	assert.Empty(t, got.AgentThreadID)
	assert.True(t, got.AgentSyncedAt.IsZero() || got.AgentSyncedAt.Unix() == 0)

	require.NoError(t, s.Chat.SetAgentThread(context.Background(), "conv-1", "thread-abc"))
	syncedAt := chatFixedNow.Add(time.Hour)
	require.NoError(t, s.Chat.SetAgentSyncedAt(context.Background(), "conv-1", syncedAt))

	got, err = s.Chat.GetConversation(context.Background(), "conv-1")
	require.NoError(t, err)
	assert.Equal(t, "thread-abc", got.AgentThreadID)
	assert.Equal(t, syncedAt.Unix(), got.AgentSyncedAt.Unix())

	require.ErrorIs(t, s.Chat.SetAgentThread(context.Background(), "missing", "x"), apperrs.ErrNotFound)
}
