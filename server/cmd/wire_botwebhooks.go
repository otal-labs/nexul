package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
)

// Chat posts a bot's message through the seam botwebhook declares, so chat never imports it.
var _ botwebhook.Poster = (*chat.Service)(nil)

// botwebhookConversations adapts chat's own read rule, as the caller, to botwebhook's Conversations seam.
type botwebhookConversations struct {
	svc *chat.Service
}

func (b botwebhookConversations) Conversation(ctx context.Context, id string) (*botwebhook.Conversation, error) {
	c, err := b.svc.GetConversation(ctx, id)
	if err != nil {
		return nil, err
	}
	return &botwebhook.Conversation{ID: c.ID, WorkspaceID: c.WorkspaceID, MembersOnly: c.MembersOnly()}, nil
}
