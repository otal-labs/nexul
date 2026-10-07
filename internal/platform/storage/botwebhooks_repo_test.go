package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func newBotStore(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	seedChatUser(t, s, "u-alice")
	require.NoError(t, s.Chat.CreateConversation(t.Context(), newTestConversation("c-eng", chat.KindChannel, "eng", "", "u-alice"), []string{"u-alice"}))
	return s
}

func newTestBot(id, name string) *botwebhook.Bot {
	return &botwebhook.Bot{ID: id, ConversationID: "c-eng", Name: name, Token: "token-" + id, CreatedBy: "u-alice", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}
}

// TestBotwebhooksRepo_Token_IsSealedAtRest: the column never holds the token a URL carries, yet a read gives it back,
// and the create's event lands in the outbox with the row.
func TestBotwebhooksRepo_Token_IsSealedAtRest(t *testing.T) {
	t.Parallel()
	s := newBotStore(t)
	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: botwebhook.TopicCreated, Payload: botwebhook.Event{BotwebhookID: "b-ci", ConversationID: "c-eng", Name: "CI"}}
	require.NoError(t, s.Botwebhooks.Create(t.Context(), newTestBot("b-ci", "CI"), evt))
	assert.Equal(t, 1, count(t, s.db, `SELECT COUNT(*) FROM outbox WHERE id = 'evt-1' AND topic = 'botwebhook.created'`))

	var stored string
	require.NoError(t, s.db.QueryRow(`SELECT token FROM botwebhooks WHERE id = 'b-ci'`).Scan(&stored))
	assert.NotContains(t, stored, "token-b-ci")
	got, err := s.Botwebhooks.Get(t.Context(), "b-ci")
	require.NoError(t, err)
	assert.Equal(t, "token-b-ci", got.Token)
	assert.Equal(t, "CI", got.Name)
	assert.Nil(t, got.DeletedAt)
}

// TestBotwebhooksRepo_LiveName_UniqueIgnoringCaseUntilDeleted: the index refuses a second live "ci", and deleting the
// first frees its name and moves it to the deleted list.
func TestBotwebhooksRepo_LiveName_UniqueIgnoringCaseUntilDeleted(t *testing.T) {
	t.Parallel()
	s := newBotStore(t)
	ctx := t.Context()
	first := newTestBot("b-1", "CI")
	require.NoError(t, s.Botwebhooks.Create(ctx, first))
	require.ErrorIs(t, s.Botwebhooks.Create(ctx, newTestBot("b-2", "ci")), apperrs.ErrConflict)

	deleted := chatFixedNow.Add(time.Hour)
	first.DeletedAt, first.DeletedBy = &deleted, "u-alice"
	require.NoError(t, s.Botwebhooks.Update(ctx, first))
	require.NoError(t, s.Botwebhooks.Create(ctx, newTestBot("b-2", "ci")))

	live, err := s.Botwebhooks.List(ctx, "c-eng", false)
	require.NoError(t, err)
	gone, err := s.Botwebhooks.List(ctx, "c-eng", true)
	require.NoError(t, err)
	require.Len(t, live, 1)
	require.Len(t, gone, 1)
	assert.Equal(t, "b-2", live[0].ID)
	assert.Equal(t, "b-1", gone[0].ID)
	assert.Equal(t, "u-alice", gone[0].DeletedBy)
	assert.True(t, deleted.Equal(*gone[0].DeletedAt))
}

func TestBotwebhooksRepo_Update_MissingBot_IsNotFound(t *testing.T) {
	t.Parallel()
	s := newBotStore(t)
	require.ErrorIs(t, s.Botwebhooks.Update(t.Context(), newTestBot("b-gone", "CI")), apperrs.ErrNotFound)
	_, err := s.Botwebhooks.Get(t.Context(), "b-gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

// TestBotwebhooksRepo_GoesWithItsConversation: deleting a channel takes its bots with it.
func TestBotwebhooksRepo_GoesWithItsConversation(t *testing.T) {
	t.Parallel()
	s := newBotStore(t)
	require.NoError(t, s.Botwebhooks.Create(t.Context(), newTestBot("b-ci", "CI")))
	require.NoError(t, s.Chat.DeleteConversation(t.Context(), "c-eng"))
	_, err := s.Botwebhooks.Get(t.Context(), "b-ci")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

// TestChatRepo_BotMessage_KeepsItsSnapshotAndLosesItsEmbedsOnDelete: what a bot message showed round-trips, and a
// delete clears its embeds with its body.
func TestChatRepo_BotMessage_KeepsItsSnapshotAndLosesItsEmbedsOnDelete(t *testing.T) {
	t.Parallel()
	s := newBotStore(t)
	ctx := t.Context()
	require.NoError(t, s.Chat.CreateMessage(ctx, &chat.Message{
		ID: "m-bot", ConversationID: "c-eng", AuthorID: "b-ci", AuthorKind: chat.AuthorBot, AuthorName: "Grafana",
		AuthorAvatarURL: "https://example.com/g.png", Embeds: []byte(`[{"title":"CPU high"}]`), CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow,
	}))
	got, err := s.Chat.GetMessage(ctx, "m-bot")
	require.NoError(t, err)
	assert.Equal(t, "Grafana", got.AuthorName)
	assert.Equal(t, "https://example.com/g.png", got.AuthorAvatarURL)
	assert.JSONEq(t, `[{"title":"CPU high"}]`, string(got.Embeds))

	require.NoError(t, s.Chat.DeleteMessage(ctx, "m-bot", chatFixedNow.Add(time.Minute)))
	got, err = s.Chat.GetMessage(ctx, "m-bot")
	require.NoError(t, err)
	assert.Nil(t, got.Embeds)
	assert.Equal(t, "Grafana", got.AuthorName, "a deleted message still says who wrote it")
}
