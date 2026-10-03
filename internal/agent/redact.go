package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

// redactedConversations hides personal access tokens in every message a turn saves to chat, final replies included.
type redactedConversations struct {
	Conversations
}

func (r redactedConversations) PostAgentReply(ctx context.Context, conversationID, viaUserID, body string, handoffs []harness.Handoff) (string, error) {
	raw, found, err := redact.JSON(handoffs)
	if err != nil {
		return "", err
	}
	if found {
		handoffs = nil
		if err := json.Unmarshal(raw, &handoffs); err != nil {
			return "", fmt.Errorf("unmarshal redacted hand-offs: %w", err)
		}
	}
	return r.Conversations.PostAgentReply(ctx, conversationID, viaUserID, redact.Tokens(body), handoffs)
}

func (r redactedConversations) PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) error {
	return r.Conversations.PostSystemMessage(ctx, conversationID, viaUserID, redact.Tokens(body))
}

func (r redactedConversations) PostUserMessage(ctx context.Context, conversationID, userID, body string) error {
	return r.Conversations.PostUserMessage(ctx, conversationID, userID, redact.Tokens(body))
}
