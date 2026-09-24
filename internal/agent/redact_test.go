package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/redact"
)

func TestRedactedConversations_EverySavedMessageHidesTheToken(t *testing.T) {
	t.Parallel()
	token := "dep_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO-_"
	inner := newFakeConversations(Conversation{})
	convs := redactedConversations{inner}
	ctx := t.Context()

	_, err := convs.PostAgentReply(ctx, "conv-1", "u1", "Connected with "+token)
	require.NoError(t, err)
	require.NoError(t, convs.PostSystemNote(ctx, "conv-1", "u1", "Agent turn failed: "+token))
	require.NoError(t, convs.PostUserMessage(ctx, "conv-1", "u1", "use "+token))

	replies, notes := inner.snapshot()
	assert.Equal(t, "Connected with "+redact.Placeholder, replies[0].body)
	assert.Equal(t, "Agent turn failed: "+redact.Placeholder, notes[0].body)
	assert.Equal(t, "use "+redact.Placeholder, inner.userPosts[0].body)
}

func TestNewService_LiveFramesLeaveRedacted(t *testing.T) {
	t.Parallel()
	token := "dep_abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO-_"
	live := &fakeLive{}
	svc := NewService(Config{Conversations: newFakeConversations(Conversation{}), Live: live})

	require.NoError(t, svc.live.Publish(t.Context(), TopicAgentStream, StreamFrame{ConversationID: "conv-1", Text: "Connected with " + token, Streaming: true}))
	require.NoError(t, svc.live.Publish(t.Context(), TopicAgentStream, StreamFrame{ConversationID: "conv-1", Text: "plain"}))

	frames := live.snapshot()
	require.Len(t, frames, 2)
	assert.Equal(t, "Connected with "+redact.Placeholder, frames[0].Text)
	assert.True(t, frames[0].Streaming)
	assert.Equal(t, "plain", frames[1].Text)
}
