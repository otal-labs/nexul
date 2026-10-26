package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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

// botwebhookMessages reads a message ungated; botwebhook checks its conversation as the caller before using it.
type botwebhookMessages struct {
	chat *storage.ChatRepo
}

func (b botwebhookMessages) BotMessage(ctx context.Context, id string) (*botwebhook.PostedMessage, error) {
	m, err := b.chat.GetMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	return &botwebhook.PostedMessage{ConversationID: m.ConversationID, AvatarURL: m.AuthorAvatarURL, Embeds: m.Embeds, Deleted: m.DeletedAt != nil}, nil
}

// botwebhookPoster posts through chat's bot use-case, so chat never imports botwebhook.
type botwebhookPoster struct {
	svc   *chat.Service
	users *storage.UsersRepo
}

func (b botwebhookPoster) PostBotMessage(ctx context.Context, p botwebhook.Post) (string, time.Time, error) {
	line, err := b.Stopped(ctx, p.ConversationID, p.CreatorID)
	if err != nil {
		return "", time.Time{}, err
	}
	if line != "" {
		return "", time.Time{}, fmt.Errorf("%w: %s", apperrs.ErrForbidden, line)
	}
	m, err := b.svc.PostBotMessage(ctx, chat.BotPost{
		ID: p.ID, ConversationID: p.ConversationID, BotID: p.BotID, CreatorID: p.CreatorID, Name: p.Name, AvatarURL: p.AvatarURL, Body: p.Body,
		Embeds: p.Embeds, Mentions: p.Mentionable, Via: p.Via, Audit: p.Audit,
	})
	if errors.Is(err, chat.ErrBotStopped) {
		line, stopErr := b.Stopped(ctx, p.ConversationID, p.CreatorID)
		if stopErr != nil || line == "" {
			return "", time.Time{}, err
		}
		return "", time.Time{}, fmt.Errorf("%w: %s", apperrs.ErrForbidden, line)
	}
	if err != nil {
		return "", time.Time{}, err
	}
	return m.ID, m.CreatedAt, nil
}

func (b botwebhookPoster) Stopped(ctx context.Context, conversationID, creatorID string) (string, error) {
	stopped, name, err := b.svc.BotStopped(ctx, conversationID, creatorID)
	if err != nil {
		return "", err
	}
	creator := "its creator"
	u, err := b.users.GetUserByID(ctx, creatorID)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return "", err
	}
	if u != nil {
		creator = u.Login
		stopped = stopped || (u.AccountStatus != "" && u.AccountStatus != auth.AccountActive)
	}
	if u == nil {
		stopped = true
	}
	if !stopped {
		return "", nil
	}
	return fmt.Sprintf("Stopped: %s can't post in #%s", creator, name), nil
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
