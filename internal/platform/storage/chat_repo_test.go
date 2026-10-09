package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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

func TestChatRepo_InterviewThread_OnePerProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	_, err := s.Chat.GetInterviewThread(context.Background(), "p-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))
	thread := func(id string) *chat.Conversation {
		return &chat.Conversation{
			ID: id, WorkspaceID: "workspace-default", Kind: chat.KindInterviewThread, ProjectID: "p-1",
			CreatedBy: "u-1", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
		}
	}

	require.NoError(t, s.Chat.CreateConversation(context.Background(), thread("conv-1"), []string{"u-1"}))
	err = s.Chat.CreateConversation(context.Background(), thread("conv-2"), []string{"u-1"})
	require.ErrorIs(t, err, apperrs.ErrConflict)

	got, err := s.Chat.GetInterviewThread(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, "conv-1", got.ID)
	assert.Equal(t, chat.KindInterviewThread, got.Kind)
	assert.Equal(t, "p-1", got.ProjectID)
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

func TestChatRepo_ListMessages_NewestLimitedOldestFirst(t *testing.T) {
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
	assert.Equal(t, "msg-2", got[0].ID)
	assert.Equal(t, "msg-3", got[1].ID)
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

func TestChatRepo_CreateMessage_Via_RoundTripsAndASecondCopyConflicts(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(context.Background(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	m := &chat.Message{ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-1", Body: "typed in T3", Via: "T3", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}

	require.NoError(t, s.Chat.CreateMessage(context.Background(), m))
	require.ErrorIs(t, s.Chat.CreateMessage(context.Background(), m), apperrs.ErrConflict)

	got, err := s.Chat.GetMessage(context.Background(), "msg-1")
	require.NoError(t, err)
	assert.Equal(t, "T3", got.Via)
}

func TestChatRepo_AgentThread_LooksUpTheConversationAndKeepsWhatWasSeen(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(ctx, newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.CreateConversation(ctx, newTestConversation("conv-2", chat.KindChannel, "random", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.SetAgentThread(ctx, "conv-1", "th-1"))
	require.NoError(t, s.Chat.SetAgentSeen(ctx, "conv-1", "run-2"))

	got, err := s.Chat.GetConversationByAgentThread(ctx, "th-1")
	require.NoError(t, err)
	assert.Equal(t, "conv-1", got.ID)
	assert.Equal(t, "run-2", got.AgentSeen)
	_, err = s.Chat.GetConversationByAgentThread(ctx, "")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a conversation without a thread is never found by the empty id")
	require.ErrorIs(t, s.Chat.SetAgentSeen(ctx, "missing", "run-1"), apperrs.ErrNotFound)
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

// allowAll passes every permission check, for tests about what a change writes rather than who may make it.
type allowAll struct{}

func (allowAll) Require(context.Context, string, permissions.Action) error { return nil }

func (allowAll) RequireProject(context.Context, string, permissions.Action) error { return nil }

func outboxPayload(t *testing.T, s *Store, topic string) []map[string]any {
	t.Helper()
	rows, err := s.db.QueryContext(t.Context(), `SELECT payload FROM outbox WHERE topic = ? ORDER BY created_at`, topic)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var out []map[string]any
	for rows.Next() {
		var raw []byte
		require.NoError(t, rows.Scan(&raw))
		var p map[string]any
		require.NoError(t, json.Unmarshal(raw, &p))
		out = append(out, p)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestChatChannels_EveryChangeIsWrittenToTheOutboxWithIt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	svc := chat.NewService(s.Chat)
	svc.SetGate(allowAll{})
	ctx := identity.WithActor(t.Context(), identity.Actor{ID: "u-1"})

	created, err := svc.CreateChannel(ctx, "workspace-default", "u-1", "eng")
	require.NoError(t, err)
	_, err = svc.CreateChannel(ctx, "workspace-default", "u-1", "ops")
	require.NoError(t, err)
	_, err = svc.PostMessage(ctx, created.ID, "u-1", "before the rename")
	require.NoError(t, err)

	createdEvents := outboxPayload(t, s, chat.TopicConversationCreated)
	require.Len(t, createdEvents, 2)
	assert.Equal(t, created.ID, createdEvents[0]["conversation"].(map[string]any)["id"])

	_, err = svc.RenameChannel(ctx, created.ID, "OPS")
	require.ErrorIs(t, err, apperrs.ErrConflict, "a name another channel has, in any case, conflicts as it does on create")
	assert.Empty(t, outboxPayload(t, s, chat.TopicConversationUpdated), "a refused rename writes no event")

	_, err = svc.RenameChannel(ctx, created.ID, "platform")
	require.NoError(t, err)
	assert.Equal(t, []map[string]any{{
		"conversation_id": created.ID, "workspace_id": "workspace-default", "kind": "channel",
		"name": "platform", "previous_name": "eng", "actor_id": "u-1",
	}}, outboxPayload(t, s, chat.TopicConversationUpdated))

	_, err = svc.DeleteChannel(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []map[string]any{{
		"conversation_id": created.ID, "workspace_id": "workspace-default", "kind": "channel",
		"name": "platform", "actor_id": "u-1",
	}}, outboxPayload(t, s, chat.TopicConversationDeleted))

	_, err = s.Chat.GetConversation(t.Context(), created.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	var messages int
	require.NoError(t, s.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, created.ID).Scan(&messages))
	assert.Zero(t, messages, "its messages are deleted with it")
}

func TestChatRepo_SetReaction_GroupsByFirstUseAndWritesOnlyRealChanges(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	for _, u := range []string{"u-1", "u-2"} {
		seedChatUser(t, s, u)
	}
	require.NoError(t, s.Chat.CreateConversation(t.Context(), newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))
	require.NoError(t, s.Chat.CreateMessage(t.Context(), &chat.Message{ID: "msg-1", ConversationID: "conv-1", AuthorID: "u-1", Body: "hi", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}))
	evt := func() eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: ids.New(), Topic: chat.TopicMessageReactionsChanged, Payload: map[string]string{}}
	}
	for i, r := range []struct{ user, emoji string }{{"u-2", "👍"}, {"u-1", "🎉"}, {"u-1", "👍"}, {"u-1", "👍"}} {
		require.NoError(t, s.Chat.SetReaction(t.Context(), "msg-1", r.user, r.emoji, true, chatFixedNow.Add(time.Duration(i)*time.Second), evt()))
	}
	require.NoError(t, s.Chat.SetReaction(t.Context(), "msg-1", "u-2", "🎉", false, chatFixedNow, evt()))

	got, err := s.Chat.ListMessages(t.Context(), "conv-1", 10)
	require.NoError(t, err)
	assert.Equal(t, []chat.Reaction{{Emoji: "👍", UserIDs: []string{"u-2", "u-1"}}, {Emoji: "🎉", UserIDs: []string{"u-1"}}}, got[0].Reactions)
	assert.Len(t, outboxPayload(t, s, chat.TopicMessageReactionsChanged), 3, "a repeated add and a remove of nothing write no event")

	require.NoError(t, s.Chat.DeleteMessage(t.Context(), "msg-1", chatFixedNow))
	deleted, err := s.Chat.GetMessage(t.Context(), "msg-1")
	require.NoError(t, err)
	assert.Empty(t, deleted.Reactions)
}

// With deferred transactions any writer outside the serializer breaks a read-then-write transaction, which exposes one.
func TestChatRepo_AgentCursorWrites_QueueBehindTheSerializer(t *testing.T) {
	data, err := os.ReadFile(migratedTemplate(t))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	db, err := sql.Open("sqlite", path+"?"+strings.Replace(pragmas, "&_txlock=immediate", "", 1))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	s := New(db, testEncKey)
	ctx := t.Context()
	seedChatUser(t, s, "u-1")
	require.NoError(t, s.Chat.CreateConversation(ctx, newTestConversation("conv-1", chat.KindChannel, "general", "", "u-1"), []string{"u-1"}))

	var wg sync.WaitGroup
	errs := make(chan error, 1800)
	for g := range 3 {
		wg.Go(func() {
			for i := range 200 {
				errs <- s.w.WithTx(ctx, db, func(tx *sql.Tx) error {
					var n int
					if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM docs`).Scan(&n); err != nil {
						return err
					}
					_, err := tx.ExecContext(ctx, `INSERT INTO docs (id, title, body, version, created_at, updated_at) VALUES (?, 't', 'b', 1, 1, 1)`, fmt.Sprintf("d-%d-%d", g, i))
					return err
				})
			}
		})
		wg.Go(func() {
			for i := range 200 {
				errs <- s.Chat.SetAgentSeen(ctx, "conv-1", fmt.Sprintf("run-%d-%d", g, i))
				errs <- s.Chat.SetAgentSyncedAt(ctx, "conv-1", chatFixedNow)
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
