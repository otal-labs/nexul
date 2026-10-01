package chat

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func callTool(ctx context.Context, t *testing.T, s *Service, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

// ticking gives every write its own timestamp, so newest-first ordering is observable.
func ticking(s *Service) *Service {
	now := fixedNow
	s.now = func() time.Time {
		now = now.Add(time.Second)
		return now
	}
	return s
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(newTestService(newFakeRepo())) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
		if tool.Name == "message_post" {
			assert.ElementsMatch(t, []string{"conversation_id", "doc_id", "ticket_id", "project_id", "workspace_id", "body"}, keys(tool.InputSchema.Properties))
			assert.Equal(t, []string{"body"}, tool.InputSchema.Required)
		}
	}
	assert.Equal(t, []string{"conversation_list", "conversation_update", "conversation_delete", "message_list", "message_post"}, names)
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestMCPTools_Errors(t *testing.T) {
	repo := newFakeRepo()
	s := newTestServiceWithDocAccess(repo, newFakeDocAccess("owner:doc-1"))
	docThread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "owner")
	require.NoError(t, err)
	tests := []struct {
		name    string
		ctx     context.Context
		tool    string
		args    string
		wantErr error
	}{
		{"list conversations without a workspace", as("u-1"), "conversation_list", `{}`, apperrs.ErrInvalid},
		{"list conversations for another user", as("u-1"), "conversation_list", `{"workspace_id":"w-1","user_id":"u-2"}`, apperrs.ErrInvalid},
		{"list messages without a target", as("u-1"), "message_list", `{}`, apperrs.ErrInvalid},
		{"list messages with two targets", as("u-1"), "message_list", `{"doc_id":"doc-1","ticket_id":"t-1"}`, apperrs.ErrInvalid},
		{"list a ticket thread by the ticket's key", as("u-1"), "message_list", `{"ticket_id":"REF-102"}`, apperrs.ErrInvalid},
		{"post to a ticket thread by the ticket's key", as("u-1"), "message_post", `{"ticket_id":"REF-102","workspace_id":"w-1","body":"hi"}`, apperrs.ErrInvalid},
		{"a lowercase key is still a key", as("u-1"), "message_list", `{"ticket_id":"ref-102"}`, apperrs.ErrInvalid},
		{"a key whose prefix has a digit is still a key", as("u-1"), "message_list", `{"ticket_id":"P1-12"}`, apperrs.ErrInvalid},
		{"post without a body", as("u-1"), "message_post", `{"ticket_id":"t-1"}`, apperrs.ErrInvalid},
		{"post a blank body", as("owner"), "message_post", `{"conversation_id":"` + docThread.ID + `","body":"  "}`, apperrs.ErrInvalid},
		{"post to a new thread without a workspace", as("u-1"), "message_post", `{"ticket_id":"t-9","body":"hi"}`, apperrs.ErrInvalid},
		{"list a missing conversation", as("u-1"), "message_list", `{"conversation_id":"nope"}`, apperrs.ErrNotFound},
		{"post to a missing conversation", as("u-1"), "message_post", `{"conversation_id":"nope","body":"hi"}`, apperrs.ErrNotFound},
		{"list a doc thread without docs:thread", as("u-1"), "message_list", `{"doc_id":"doc-1"}`, apperrs.ErrForbidden},
		{"list a doc thread by id without docs:thread", as("u-1"), "message_list", `{"conversation_id":"` + docThread.ID + `"}`, apperrs.ErrForbidden},
		{"post to a doc thread without docs:thread", as("u-1"), "message_post", `{"doc_id":"doc-1","body":"hi"}`, apperrs.ErrForbidden},
		{"rename a missing conversation", as("u-1"), "conversation_update", `{"id":"nope","name":"eng"}`, apperrs.ErrNotFound},
		{"delete a doc thread", as("owner"), "conversation_delete", `{"id":"` + docThread.ID + `"}`, apperrs.ErrInvalid},
		{"list conversations without a caller", context.Background(), "conversation_list", `{"workspace_id":"w-1"}`, apperrs.ErrUnauthorized},
		{"list messages without a caller", context.Background(), "message_list", `{"ticket_id":"t-1"}`, apperrs.ErrUnauthorized},
		{"post without a caller", context.Background(), "message_post", `{"ticket_id":"t-1","body":"hi"}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(tt.ctx, t, s, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestMessageList_AThreadNobodyStartedIsEmptyAndStaysUnstarted(t *testing.T) {
	repo := newFakeRepo()
	s := newTestServiceWithDocAccess(repo, newFakeDocAccess("u-1:doc-1"))
	for _, args := range []string{`{"ticket_id":"t-1"}`, `{"doc_id":"doc-1"}`, `{"project_id":"p-1"}`} {
		out, err := callTool(as("u-1"), t, s, "message_list", args)
		require.NoError(t, err)
		assert.Equal(t, mcptool.Page[messageResult]{Items: []messageResult{}}, out, args)
	}
	assert.Empty(t, repo.eventsFor(TopicConversationCreated))
}

func TestMessagePost_StartsTheThreadOnce(t *testing.T) {
	repo := newFakeRepo()
	s := newTestServiceWithDocAccess(repo, newFakeDocAccess("u-1:doc-1"))
	for _, target := range []string{`"ticket_id":"t-1"`, `"doc_id":"doc-1"`, `"project_id":"p-1"`} {
		t.Run(target, func(t *testing.T) {
			first, err := callTool(as("u-1"), t, s, "message_post", `{`+target+`,"workspace_id":"w-1","body":"first"}`)
			require.NoError(t, err)
			again, err := callTool(as("u-1"), t, s, "message_post", `{`+target+`,"body":"again"}`)
			require.NoError(t, err)
			assert.Equal(t, first.(messageResult).ConversationID, again.(messageResult).ConversationID)
			assert.Equal(t, "again", again.(messageResult).Body)
			listed, err := callTool(as("u-1"), t, s, "message_list", `{`+target+`}`)
			require.NoError(t, err)
			assert.Equal(t, 2, listed.(mcptool.Page[messageResult]).Total, "message_list reaches the thread by the same target")
		})
	}
	assert.Len(t, repo.eventsFor(TopicConversationCreated), 3)
}

func TestMessageList_NewestFirstWithoutDeletedMessages(t *testing.T) {
	repo := newFakeRepo()
	s := ticking(newTestService(repo))
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)
	for _, body := range []string{"one", "two", "three"} {
		_, err := callTool(as("u-1"), t, s, "message_post", `{"conversation_id":"`+c.ID+`","body":"`+body+`"}`)
		require.NoError(t, err)
	}
	gone, err := s.PostMessage(context.Background(), c.ID, "u-1", "oops")
	require.NoError(t, err)
	require.NoError(t, s.DeleteMessage(context.Background(), gone.ID, "u-1"))

	out, err := callTool(as("u-2"), t, s, "message_list", `{"conversation_id":"`+c.ID+`","limit":2}`)
	require.NoError(t, err)
	page := out.(mcptool.Page[messageResult])
	assert.Equal(t, 3, page.Total)
	require.Len(t, page.Items, 2)
	assert.Equal(t, "three", page.Items[0].Body)
	assert.Equal(t, "two", page.Items[1].Body)
	assert.Equal(t, 2, page.NextOffset)

	out, err = callTool(as("u-2"), t, s, "message_list", `{"conversation_id":"`+c.ID+`","offset":2}`)
	require.NoError(t, err)
	assert.Equal(t, "one", out.(mcptool.Page[messageResult]).Items[0].Body)
}

func TestConversationList(t *testing.T) {
	repo := newFakeRepo()
	s := newTestServiceWithDocAccess(repo, newFakeDocAccess("u-2:doc-1"))
	general, err := s.CreateChannel(context.Background(), "w-1", "u-1", "general")
	require.NoError(t, err)
	dm, err := s.CreateDM(context.Background(), "w-1", "u-2", []string{"u-3"})
	require.NoError(t, err)
	thread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-2")
	require.NoError(t, err)

	ids := func(userID string) []string {
		out, err := callTool(as(userID), t, s, "conversation_list", `{"workspace_id":"w-1"}`)
		require.NoError(t, err)
		var got []string
		for _, c := range out.(mcptool.Page[conversationResult]).Items {
			got = append(got, c.ID)
		}
		return got
	}
	assert.ElementsMatch(t, []string{general.ID, dm.ID, thread.ID}, ids("u-2"))
	assert.ElementsMatch(t, []string{general.ID}, ids("u-1"), "no DM of others, no doc thread without docs:thread")
}

func TestConversationUpdateAndDelete(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)

	out, err := callTool(as("u-1"), t, s, "conversation_update", `{"id":"`+c.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, "eng", out.(conversationResult).Name, "an omitted name keeps the current one")

	out, err = callTool(as("u-1"), t, s, "conversation_update", `{"id":"`+c.ID+`","name":"platform"}`)
	require.NoError(t, err)
	assert.Equal(t, "platform", out.(conversationResult).Name)

	out, err = callTool(as("u-1"), t, s, "conversation_delete", `{"id":"`+c.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"id": c.ID, "deleted": true}, out)
	_, err = callTool(as("u-1"), t, s, "conversation_delete", `{"id":"`+c.ID+`"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestConversationList_PrivateChannels(t *testing.T) {
	f := newPrivateFixture(t)
	items := func(userID string) map[string]conversationResult {
		out, err := callTool(as(userID), t, f.s, "conversation_list", `{"workspace_id":"w-1"}`)
		require.NoError(t, err)
		byID := map[string]conversationResult{}
		for _, c := range out.(mcptool.Page[conversationResult]).Items {
			byID[c.ID] = c
		}
		return byID
	}
	member := items("u-2")
	assert.True(t, member[f.secret.ID].Private)
	assert.Equal(t, []string{"u-1", "u-2"}, member[f.secret.ID].MemberIDs)
	assert.Empty(t, member[f.secret.ID].ParticipantIDs)
	assert.False(t, member[f.eng.ID].Private)
	assert.Empty(t, member[f.eng.ID].MemberIDs)
	assert.NotContains(t, items("u-3"), f.secret.ID, "a private channel the caller is not in is absent")
	assert.Contains(t, items("u-owner"), f.secret.ID, "the Owner sees them all")
}

func TestConversationUpdate_PrivateAndMembers(t *testing.T) {
	t.Run("refusals", func(t *testing.T) {
		f := newPrivateFixture(t)
		tests := []struct {
			name    string
			actor   string
			args    string
			wantErr error
		}{
			{"member_ids without going private", "u-1", `{"id":"` + f.eng.ID + `","member_ids":["u-2"]}`, apperrs.ErrInvalid},
			{"member_ids while going public", "u-1", `{"id":"` + f.secret.ID + `","private":false,"member_ids":["u-2"]}`, apperrs.ErrInvalid},
			{"a private channel the caller is not in", "u-3", `{"id":"` + f.secret.ID + `","add_member_ids":["u-3"]}`, apperrs.ErrNotFound},
			{"removing someone else without channels:write", "u-2", `{"id":"` + f.secret.ID + `","remove_member_ids":["u-1"]}`, apperrs.ErrForbidden},
			{"switching without channels:write", "u-2", `{"id":"` + f.secret.ID + `","private":false}`, apperrs.ErrForbidden},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := callTool(as(tt.actor), t, f.s, "conversation_update", tt.args)
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})
	t.Run("an error after a change says what already applied", func(t *testing.T) {
		f := newPrivateFixture(t)
		_, err := callTool(as("u-1"), t, f.s, "conversation_update", `{"id":"`+f.secret.ID+`","name":"hush","remove_member_ids":["u-1","u-2"]}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.ErrorContains(t, err, "already applied: name")
	})
	t.Run("each field applies and omitted ones keep their value", func(t *testing.T) {
		f := newPrivateFixture(t)
		update := func(actor, fields string) conversationResult {
			out, err := callTool(as(actor), t, f.s, "conversation_update", `{"id":"`+f.eng.ID+`"`+fields+`}`)
			require.NoError(t, err)
			return out.(conversationResult)
		}
		got := update("u-1", `,"private":true,"member_ids":["u-2"]`)
		assert.True(t, got.Private)
		assert.Equal(t, []string{"u-1", "u-2"}, got.MemberIDs)

		got = update("u-1", `,"name":"platform"`)
		assert.Equal(t, "platform", got.Name)
		assert.True(t, got.Private, "a rename keeps the channel private")
		assert.Equal(t, []string{"u-1", "u-2"}, got.MemberIDs)

		got = update("u-2", `,"add_member_ids":["u-3"]`)
		assert.Equal(t, []string{"u-1", "u-2", "u-3"}, got.MemberIDs)
		got = update("u-3", `,"remove_member_ids":["u-3"]`)
		assert.Equal(t, []string{"u-1", "u-2"}, got.MemberIDs, "your own id leaves")

		got = update("u-1", ``)
		assert.Equal(t, "platform", got.Name)
		assert.True(t, got.Private)

		got = update("u-1", `,"private":false`)
		assert.False(t, got.Private)
		assert.Empty(t, got.MemberIDs)
	})
}
