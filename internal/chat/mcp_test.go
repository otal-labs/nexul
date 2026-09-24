package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func toolByName(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

func withActor(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo()))
	require.Len(t, tools, 5)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{"chat_list_conversations", "chat_list_messages", "chat_post_message", "doc_thread_get", "interview_thread_get"}, names)
}

func TestMCPTools_ListConversations(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.CreateChannel(context.Background(), "w-1", "u-1", "general")
	require.NoError(t, err)
	dm, err := s.CreateDM(context.Background(), "w-1", "u-2", []string{"u-3"})
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "chat_list_conversations").Call

	t.Run("lists channels plus the authenticated user's own conversations", func(t *testing.T) {
		got, err := call(withActor("u-2"), map[string]any{"workspace_id": "w-1"})
		require.NoError(t, err)
		cs, ok := got.([]*Conversation)
		require.True(t, ok)
		require.Len(t, cs, 2)
		var ids []string
		for _, c := range cs {
			ids = append(ids, c.ID)
		}
		assert.Contains(t, ids, dm.ID)
	})
	t.Run("missing workspace id is invalid", func(t *testing.T) {
		_, err := call(withActor("u-2"), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("no actor in context is an empty user id, which is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"workspace_id": "w-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ListMessages(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)
	_, err = s.PostMessage(context.Background(), c.ID, "u-1", "hi")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "chat_list_messages").Call

	t.Run("returns the conversation's messages", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"conversation_id": c.ID})
		require.NoError(t, err)
		ms, ok := got.([]*Message)
		require.True(t, ok)
		require.Len(t, ms, 1)
		assert.Equal(t, "hi", ms[0].Body)
	})
	t.Run("missing conversation id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_PostMessage(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "chat_post_message").Call

	t.Run("posts as the authenticated user", func(t *testing.T) {
		got, err := call(withActor("u-1"), map[string]any{"conversation_id": c.ID, "body": "hey @onik97"})
		require.NoError(t, err)
		m, ok := got.(*Message)
		require.True(t, ok)
		assert.Equal(t, "u-1", m.AuthorID)
		assert.Len(t, m.Mentions, 1)
	})
	t.Run("missing body is invalid", func(t *testing.T) {
		_, err := call(withActor("u-1"), map[string]any{"conversation_id": c.ID})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("no actor in context is an empty author id, which is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"conversation_id": c.ID, "body": "hi"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_InterviewThreadGet(t *testing.T) {
	call := toolByName(t, MCPTools(newTestService(newFakeRepo())), "interview_thread_get").Call
	got, err := call(withActor("owner"), map[string]any{"workspace_id": "w-1", "project_id": "p-1"})
	require.NoError(t, err)
	c, ok := got.(*Conversation)
	require.True(t, ok)
	assert.Equal(t, KindInterviewThread, c.Kind)
	assert.Equal(t, "p-1", c.ProjectID)

	_, err = call(withActor("owner"), map[string]any{"workspace_id": "w-1"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestMCPTools_DocThreadGet(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	s.SetDocAccess(newFakeDocAccess("owner:doc-1"))
	call := toolByName(t, MCPTools(s), "doc_thread_get").Call

	t.Run("gets or creates the thread for a caller with docs:thread", func(t *testing.T) {
		got, err := call(withActor("owner"), map[string]any{"workspace_id": "w-1", "doc_id": "doc-1"})
		require.NoError(t, err)
		c, ok := got.(*Conversation)
		require.True(t, ok)
		assert.Equal(t, KindDocThread, c.Kind)
		assert.Equal(t, "doc-1", c.DocID)
	})
	t.Run("a caller without docs:thread is forbidden, matching the gateway", func(t *testing.T) {
		_, err := call(withActor("other-user"), map[string]any{"workspace_id": "w-1", "doc_id": "doc-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("missing doc id is invalid", func(t *testing.T) {
		_, err := call(withActor("owner"), map[string]any{"workspace_id": "w-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}
