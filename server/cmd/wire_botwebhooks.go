package main

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

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

// botwebhookPoster posts through chat's bot use-case, so chat never imports botwebhook.
type botwebhookPoster struct {
	svc *chat.Service
}

func (b botwebhookPoster) PostBotMessage(ctx context.Context, p botwebhook.Post) (string, time.Time, error) {
	m, err := b.svc.PostBotMessage(ctx, chat.BotPost{
		ConversationID: p.ConversationID, BotID: p.BotID, Name: p.Name, AvatarURL: p.AvatarURL, Body: p.Body,
		Embeds: p.Embeds, Mentions: p.Mentionable, Via: p.Via, Audit: p.Audit,
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return m.ID, m.CreatedAt, nil
}

// botwebhookPeople lists a conversation's workspace members by id and login; a post has no caller to gate it on.
type botwebhookPeople struct {
	conversations *storage.ChatRepo
	members       *storage.WorkspaceMembersRepo
	users         *auth.Service
}

func (b botwebhookPeople) Members(ctx context.Context, conversationID string) ([]botwebhook.Person, error) {
	c, err := b.conversations.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	members, err := b.members.ListByWorkspace(ctx, c.WorkspaceID)
	if err != nil {
		return nil, err
	}
	users, err := b.users.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	logins := make(map[string]string, len(users))
	for _, u := range users {
		logins[u.ID] = u.Login
	}
	out := make([]botwebhook.Person, 0, len(members))
	for _, m := range members {
		if login := logins[m.UserID]; login != "" {
			out = append(out, botwebhook.Person{ID: m.UserID, Login: login})
		}
	}
	return out, nil
}
