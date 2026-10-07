package botwebhook

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Gate is the workspace permission check every use-case passes for its own botwebhook action.
type Gate interface {
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
}

// Conversations reads a conversation as the actor on ctx through chat's read rule: a DM's participants, a private
// channel's members, a ticket or doc thread's own gate.
type Conversations interface {
	Conversation(ctx context.Context, id string) (*Conversation, error)
}

// InstanceURL is the address senders reach this instance at, the base of every bot's URL.
type InstanceURL interface {
	GetInstanceURL(ctx context.Context) (string, error)
}

// Poster posts a bot's message into its conversation; chat satisfies it, so chat never learns what a webhook is.
type Poster interface {
	PostBotMessage(ctx context.Context, conversationID, botID, name, avatarURL, body string, embeds json.RawMessage) (string, time.Time, error)
}

// Config wires Service's seams; the adapters live in the composition root.
type Config struct {
	Repo          Repo
	Gate          Gate
	Conversations Conversations
	Instance      InstanceURL
}

// Service is the botwebhook use-case layer; every mutation enqueues its event through the transactional outbox.
type Service struct {
	repo          Repo
	gate          Gate
	conversations Conversations
	instance      InstanceURL
	now           func() time.Time
}

// NewService wires the bot use-cases over the given seams.
func NewService(cfg Config) *Service {
	return &Service{repo: cfg.Repo, gate: cfg.Gate, conversations: cfg.Conversations, instance: cfg.Instance, now: time.Now}
}

// List returns a conversation's live bots, URLs only for botwebhook:write; the deleted list is for editors alone.
func (s *Service) List(ctx context.Context, conversationID string, deleted bool) ([]*Bot, error) {
	c, err := s.conversation(ctx, conversationID, permissions.BotwebhookRead)
	if err != nil {
		return nil, err
	}
	if deleted {
		if err := s.gate.Require(ctx, c.WorkspaceID, permissions.BotwebhookWrite); err != nil {
			return nil, err
		}
	}
	bots, err := s.repo.List(ctx, c.ID, deleted)
	if err != nil {
		return nil, fmt.Errorf("list bots of conversation %s: %w", c.ID, err)
	}
	if deleted {
		return bots, nil
	}
	err = s.gate.Require(ctx, c.WorkspaceID, permissions.BotwebhookWrite)
	if permissions.Refused(err) {
		return bots, nil
	}
	if err != nil {
		return nil, err
	}
	for _, b := range bots {
		if err := s.withURL(ctx, b); err != nil {
			return nil, err
		}
	}
	return bots, nil
}

// Create adds a bot to a conversation with a fresh token, under botwebhook:write and the conversation's own gate.
func (s *Service) Create(ctx context.Context, conversationID, name, avatar string) (*Bot, error) {
	name, err := botName(name)
	if err != nil {
		return nil, err
	}
	if err := checkAvatar(avatar); err != nil {
		return nil, err
	}
	c, err := s.conversation(ctx, conversationID, permissions.BotwebhookWrite)
	if err != nil {
		return nil, err
	}
	if err := s.roomFor(ctx, c.ID, "", name, true); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	b := &Bot{ID: ids.New(), ConversationID: c.ID, Name: name, Avatar: avatar, CreatedBy: actorID(ctx), CreatedAt: now, UpdatedAt: now, Token: newToken()}
	if err := s.repo.Create(ctx, b, event(ctx, TopicCreated, b, c, nil)); err != nil {
		return nil, fmt.Errorf("create bot in conversation %s: %w", c.ID, err)
	}
	return b, s.withURL(ctx, b)
}

// Update renames, re-avatars, regenerates, restores, or deletes a bot; a restore issues a fresh token like a create.
func (s *Service) Update(ctx context.Context, id string, ch Changes) (*Bot, error) {
	if ch.Deleted != nil && *ch.Deleted {
		return s.deleteOnly(ctx, id, ch)
	}
	b, c, err := s.bot(ctx, id, permissions.BotwebhookWrite)
	if err != nil {
		return nil, err
	}
	restoring := ch.Deleted != nil && b.DeletedAt != nil
	if b.DeletedAt != nil && !restoring {
		return nil, fmt.Errorf("%w: bot %s is deleted; restore it first", apperrs.ErrInvalid, b.ID)
	}
	next, changes, err := changed(b, ch)
	if err != nil {
		return nil, err
	}
	if restoring || slices.Contains(changes, ChangeRenamed) {
		if err := s.roomFor(ctx, c.ID, b.ID, next.Name, restoring); err != nil {
			return nil, err
		}
	}
	var evts []eventbus.OutboxEvent
	if restoring {
		next.DeletedAt, next.DeletedBy, next.Token = nil, "", newToken()
		evts = append(evts, event(ctx, TopicRestored, next, c, nil))
	}
	if len(changes) > 0 {
		evts = append(evts, event(ctx, TopicUpdated, next, c, changes))
	}
	if len(evts) == 0 {
		return b, s.withURL(ctx, b)
	}
	next.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, next, evts...); err != nil {
		return nil, fmt.Errorf("update bot %s: %w", b.ID, err)
	}
	return next, s.withURL(ctx, next)
}

// deleteOnly is an Update that deletes, which takes no other change in the same call.
func (s *Service) deleteOnly(ctx context.Context, id string, ch Changes) (*Bot, error) {
	if ch.Name != nil || ch.Avatar != nil || ch.Regenerate {
		return nil, fmt.Errorf("%w: deleting a bot takes no other change", apperrs.ErrInvalid)
	}
	return s.Delete(ctx, id)
}

// changed applies ch to a copy of b and names what it changed; a regenerate always changes the token.
func changed(b *Bot, ch Changes) (*Bot, []Change, error) {
	next := *b
	var changes []Change
	if ch.Name != nil {
		name, err := botName(*ch.Name)
		if err != nil {
			return nil, nil, err
		}
		if name != b.Name {
			next.Name, changes = name, append(changes, ChangeRenamed)
		}
	}
	if ch.Avatar != nil {
		if err := checkAvatar(*ch.Avatar); err != nil {
			return nil, nil, err
		}
		if *ch.Avatar != b.Avatar {
			next.Avatar, changes = *ch.Avatar, append(changes, ChangeAvatar)
		}
	}
	if ch.Regenerate {
		next.Token, changes = newToken(), append(changes, ChangeRegenerated)
	}
	return &next, changes, nil
}

// Delete soft-deletes a bot under botwebhook:delete: its URL dies at once and its name is freed; deleting it again is a no-op.
func (s *Service) Delete(ctx context.Context, id string) (*Bot, error) {
	b, c, err := s.bot(ctx, id, permissions.BotwebhookDelete)
	if err != nil {
		return nil, err
	}
	if b.DeletedAt != nil {
		return b, nil
	}
	now := s.now().UTC()
	next := *b
	// The new token is never shown, so the old URL stops working even on a path that forgets to check deleted_at.
	next.DeletedAt, next.DeletedBy, next.UpdatedAt, next.Token = &now, actorID(ctx), now, newToken()
	if err := s.repo.Update(ctx, &next, event(ctx, TopicDeleted, &next, c, nil)); err != nil {
		return nil, fmt.Errorf("delete bot %s: %w", b.ID, err)
	}
	return &next, nil
}

// roomFor refuses a name a live bot of the conversation other than selfID already has and, for a bot about to become
// live, a conversation already holding MaxLive.
func (s *Service) roomFor(ctx context.Context, conversationID, selfID, name string, joining bool) error {
	// ponytail: checked before the write, so two racing creates can land an eleventh bot; the unique index still holds names.
	live, err := s.repo.List(ctx, conversationID, false)
	if err != nil {
		return fmt.Errorf("list bots of conversation %s: %w", conversationID, err)
	}
	if joining && len(live) >= MaxLive {
		return fmt.Errorf("%w: a conversation holds %d bots; delete one to add another", apperrs.ErrConflict, MaxLive)
	}
	if slices.ContainsFunc(live, func(o *Bot) bool { return o.ID != selfID && strings.EqualFold(o.Name, name) }) {
		return fmt.Errorf("%w: this conversation already has a bot named %s", apperrs.ErrConflict, name)
	}
	return nil
}

// bot loads a bot and checks, in order, that the caller reads its conversation and holds action.
func (s *Service) bot(ctx context.Context, id string, action permissions.Action) (*Bot, *Conversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil, fmt.Errorf("%w: bot id is required", apperrs.ErrInvalid)
	}
	b, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("get bot %s: %w", id, err)
	}
	c, err := s.conversation(ctx, b.ConversationID, action)
	if err != nil {
		return nil, nil, err
	}
	return b, c, nil
}

// conversation checks, in order, that the caller reads the conversation and holds action in its workspace.
func (s *Service) conversation(ctx context.Context, id string, action permissions.Action) (*Conversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	c, err := s.conversations.Conversation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", id, err)
	}
	if err := s.gate.Require(ctx, c.WorkspaceID, action); err != nil {
		return nil, err
	}
	return c, nil
}

// withURL fills b's URL, the instance address plus Discord's path shape; before an instance URL is set it is the path alone.
func (s *Service) withURL(ctx context.Context, b *Bot) error {
	base, err := s.instance.GetInstanceURL(ctx)
	if err != nil {
		return fmt.Errorf("read the instance URL: %w", err)
	}
	b.URL = strings.TrimRight(base, "/") + "/api/botwebhooks/" + b.ID + "/" + b.Token
	return nil
}

func event(ctx context.Context, topic string, b *Bot, c *Conversation, changes []Change) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: Event{
		BotwebhookID: b.ID, ConversationID: c.ID, WorkspaceID: c.WorkspaceID, Name: b.Name, ActorID: actorID(ctx),
		Changes: changes, MembersOnly: c.MembersOnly,
	}}
}

// newToken is the scoped-token shape: 32 random bytes as 43 URL-safe characters.
func newToken() string {
	b := make([]byte, 32)
	// crypto/rand.Read never returns an error; it crashes the program instead.
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
}
