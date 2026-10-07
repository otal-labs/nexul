package botwebhook

import (
	"cmp"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// ErrUnknownWebhook answers every refused URL alike: a wrong id, a wrong or old token, a deleted bot.
var ErrUnknownWebhook = fmt.Errorf("%w: unknown webhook", apperrs.ErrNotFound)

// Authenticate returns the live bot a URL's id and token name, comparing the token in constant time; anything else is
// ErrUnknownWebhook, so no answer tells a wrong token from a missing bot.
func (s *Service) Authenticate(ctx context.Context, id, token string) (*Bot, error) {
	b, err := s.repo.Get(ctx, id)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, ErrUnknownWebhook
	}
	if err != nil {
		return nil, fmt.Errorf("get bot %s: %w", id, err)
	}
	if b.DeletedAt != nil || subtle.ConstantTimeCompare([]byte(b.Token), []byte(token)) != 1 {
		return nil, ErrUnknownWebhook
	}
	return b, nil
}

// Execute posts p into b's conversation as b, the post's username and avatar_url overriding the bot's own; a bot
// deleted since Authenticate is ErrUnknownWebhook.
func (s *Service) Execute(ctx context.Context, b *Bot, p Payload) (*Message, error) {
	c, err := p.check()
	if err != nil {
		return nil, err
	}
	embeds, err := storedEmbeds(c.embeds)
	if err != nil {
		return nil, err
	}
	mentionable, err := s.mentionable(ctx, b.ConversationID, c.content, p.AllowedMentions.policy())
	if err != nil {
		return nil, err
	}
	name, avatar, via := cmp.Or(c.username, b.Name), cmp.Or(c.avatarURL, avatarPath(b)), ""
	if name != b.Name {
		via = b.Name
	}
	id, at, err := s.poster.PostBotMessage(ctx, Post{
		ConversationID: b.ConversationID, BotID: b.ID, Name: name, AvatarURL: avatar, Body: c.content, Embeds: embeds,
		Mentionable: mentionable, Via: via, Audit: "POST /api/botwebhooks/" + b.ID,
	})
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, ErrUnknownWebhook
	}
	if err != nil {
		return nil, fmt.Errorf("post as bot %s: %w", b.ID, err)
	}
	return &Message{
		ID: id, ChannelID: b.ConversationID, Content: c.content, Embeds: c.embeds, Timestamp: at.UTC().Format(time.RFC3339),
		WebhookID: b.ID, Author: MessageAuthor{ID: b.ID, Username: name, Avatar: avatar, Bot: true},
	}, nil
}

// mentionable is the logins of the workspace members the policy lets content mention; only an @ asks who they are.
func (s *Service) mentionable(ctx context.Context, conversationID, content string, policy mentionPolicy) ([]string, error) {
	if !strings.Contains(content, "@") || (!policy.all && len(policy.users) == 0) {
		return nil, nil
	}
	people, err := s.people.Members(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("list who conversation %s may mention: %w", conversationID, err)
	}
	var out []string
	for _, p := range people {
		if policy.allows(p.ID, p.Login) {
			out = append(out, p.Login)
		}
	}
	return out, nil
}

// Avatar returns a bot's own avatar to anyone who reads its conversation: every reader sees its messages, so no
// botwebhook permission is asked.
func (s *Service) Avatar(ctx context.Context, id string) (contentType string, data []byte, err error) {
	b, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return "", nil, fmt.Errorf("get bot %s: %w", id, err)
	}
	if _, err := s.conversations.Conversation(ctx, b.ConversationID); err != nil {
		return "", nil, fmt.Errorf("get conversation %s: %w", b.ConversationID, err)
	}
	if b.Avatar == "" {
		return "", nil, fmt.Errorf("%w: bot %s has no avatar", apperrs.ErrNotFound, b.ID)
	}
	return decodeAvatar(b.Avatar)
}

// decodeAvatar serves only raster images as themselves; anything else goes out as opaque bytes.
func decodeAvatar(dataURI string) (string, []byte, error) {
	meta, payload, _ := strings.Cut(strings.TrimPrefix(dataURI, "data:"), ",")
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, fmt.Errorf("decode stored avatar: %w", err)
	}
	switch contentType := strings.TrimSuffix(meta, ";base64"); contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif":
		return contentType, data, nil
	}
	return "application/octet-stream", data, nil
}
