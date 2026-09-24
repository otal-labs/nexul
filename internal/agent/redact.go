package agent

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/redact"
)

// redactedConversations hides personal access tokens in every message a turn saves to chat, final replies included.
type redactedConversations struct {
	Conversations
}

func (r redactedConversations) PostAgentReply(ctx context.Context, conversationID, viaUserID, body string) (string, error) {
	return r.Conversations.PostAgentReply(ctx, conversationID, viaUserID, redact.Tokens(body))
}

func (r redactedConversations) PostSystemNote(ctx context.Context, conversationID, viaUserID, body string) error {
	return r.Conversations.PostSystemNote(ctx, conversationID, viaUserID, redact.Tokens(body))
}

func (r redactedConversations) PostUserMessage(ctx context.Context, conversationID, userID, body string) error {
	return r.Conversations.PostUserMessage(ctx, conversationID, userID, redact.Tokens(body))
}
