package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// ListMessages returns a conversation's messages oldest-first; callerID gates a doc thread's messages.
func (s *Service) ListMessages(ctx context.Context, conversationID, callerID string, limit int) ([]*Message, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	if err := s.requireConversation(ctx, conversationID, callerID); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 50
	}
	ms, err := s.repo.ListMessages(ctx, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages for conversation %s: %w", conversationID, err)
	}
	return ms, nil
}

// ListMessagesSince returns non-deleted messages created strictly after since, oldest-first.
func (s *Service) ListMessagesSince(ctx context.Context, conversationID string, since time.Time) ([]*Message, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	ms, err := s.repo.ListMessagesSince(ctx, conversationID, since)
	if err != nil {
		return nil, fmt.Errorf("list messages since %s for conversation %s: %w", since, conversationID, err)
	}
	return ms, nil
}

// PostMessage posts a markdown message to a conversation, parsing @user/@Agent mentions; a doc thread
// gates the post behind docs:thread so the gateway and MCP tools agree (ADR 0057).
func (s *Service) PostMessage(ctx context.Context, conversationID, authorID, body string) (*Message, error) {
	if err := s.requireConversation(ctx, conversationID, authorID); err != nil {
		return nil, err
	}
	return s.postMessage(ctx, conversationID, authorID, body, AuthorUser)
}

// PostAgentMessage persists the Agent's final turn reply with the work it handed off; in-progress text only streams live.
func (s *Service) PostAgentMessage(ctx context.Context, conversationID, viaUserID, body string, handoffs []harness.Handoff) (*Message, error) {
	m, err := s.newMessage(conversationID, viaUserID, body, AuthorAgent)
	if err != nil {
		return nil, err
	}
	if m.Handoffs, err = storedHandoffs(handoffs); err != nil {
		return nil, err
	}
	return s.create(ctx, m)
}

// PostHarnessMessage relays a message userID wrote in their harness itself into the conversation, at the time they wrote
// it; key names it in that harness, so relaying it again changes nothing. Its mentions are not parsed: it already went
// to the harness, and an @Agent in it must not start a second turn.
func (s *Service) PostHarnessMessage(ctx context.Context, conversationID, userID, body, via, key string, at time.Time) error {
	m, err := s.newMessage(conversationID, userID, body, AuthorUser)
	if err != nil {
		return err
	}
	m.ID = ids.From("relayed-message\x00" + m.ConversationID + "\x00" + key)
	m.Mentions, m.Via, m.CreatedAt, m.UpdatedAt = []Mention{}, via, at.UTC(), at.UTC()
	_, err = s.create(ctx, m)
	if errors.Is(err, apperrs.ErrConflict) {
		return nil
	}
	return err
}

// BotPost is one post through a bot's URL as chat stores it; the URL that reached the bot is the credential.
type BotPost struct {
	ConversationID, BotID string
	// Name and AvatarURL are what the message shows, kept through the bot's rename or delete.
	Name, AvatarURL, Body string
	Embeds                json.RawMessage
	// Mentions are the @handles in Body the sender allowed; any other handle stays plain text and @Agent never fires.
	Mentions []string
	// Via is the bot's own name when Name overrides it, so the message says which bot sent it; empty otherwise.
	Via string
	// Audit is the action the post's audit row records under the bot.
	Audit string
}

// PostBotMessage posts a bot's message, counts it on its bot, and audits it, in one transaction; it checks no
// permission, and a bot gone meanwhile is ErrNotFound.
func (s *Service) PostBotMessage(ctx context.Context, p BotPost) (*Message, error) {
	conversationID, botID, name := strings.TrimSpace(p.ConversationID), strings.TrimSpace(p.BotID), strings.TrimSpace(p.Name)
	if conversationID == "" || botID == "" {
		return nil, fmt.Errorf("%w: a bot message needs its conversation and bot ids", apperrs.ErrInvalid)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: a bot message needs the name it shows", apperrs.ErrInvalid)
	}
	body := strings.TrimSpace(p.Body)
	if body == "" && len(p.Embeds) == 0 {
		return nil, fmt.Errorf("%w: a bot message needs content or an embed", apperrs.ErrInvalid)
	}
	now := s.now().UTC()
	m := &Message{
		ID: ids.New(), ConversationID: conversationID, AuthorID: botID, AuthorKind: AuthorBot,
		AuthorName: name, AuthorAvatarURL: strings.TrimSpace(p.AvatarURL), Body: body, Mentions: allowedMentions(body, p.Mentions),
		Embeds: p.Embeds, Via: strings.TrimSpace(p.Via), CreatedAt: now, UpdatedAt: now,
	}
	membersOnly, err := s.MembersOnly(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("post bot message to conversation %s: %w", conversationID, err)
	}
	if err := s.repo.CreateBotMessage(ctx, m, p.Audit, messageCreated(m, membersOnly)); err != nil {
		return nil, fmt.Errorf("post bot message to conversation %s: %w", conversationID, err)
	}
	return m, nil
}

// allowedMentions is body's person mentions whose handle is in allowed, ignoring case; never the Agent.
func allowedMentions(body string, allowed []string) []Mention {
	out := []Mention{}
	for _, m := range ParseMentions(body) {
		if m.Kind == MentionUser && slices.ContainsFunc(allowed, func(h string) bool { return strings.EqualFold(h, m.Handle) }) {
			out = append(out, m)
		}
	}
	return out
}

// maxHandoffBytes caps the hand-offs one reply stores, as JSON.
const maxHandoffBytes = 256 << 10

// storedHandoffs is hs as a reply stores them, nil for none, within maxHandoffBytes: the oldest steps go first, then the longest replies.
func storedHandoffs(hs []harness.Handoff) ([]Handoff, error) {
	if len(hs) == 0 {
		return nil, nil
	}
	out := make([]Handoff, 0, len(hs))
	for _, h := range hs {
		out = append(out, NewHandoff(h))
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("measure hand-offs: %w", err)
	}
	type stepRef struct {
		handoff, step int
		at            time.Time
		size          int
	}
	var refs []stepRef
	for i, h := range out {
		for j, s := range h.Steps {
			step, err := json.Marshal(s)
			if err != nil {
				return nil, fmt.Errorf("measure a hand-off step: %w", err)
			}
			refs = append(refs, stepRef{i, j, s.At, len(step)})
		}
	}
	slices.SortStableFunc(refs, func(a, b stepRef) int { return a.at.Compare(b.at) })
	size, drop := len(raw), map[[2]int]bool{}
	for _, r := range refs {
		if size <= maxHandoffBytes {
			break
		}
		size -= r.size
		drop[[2]int{r.handoff, r.step}] = true
	}
	for i := range out {
		kept := out[i].Steps[:0]
		for j, s := range out[i].Steps {
			if !drop[[2]int{i, j}] {
				kept = append(kept, s)
			}
		}
		out[i].Steps = kept
	}
	cutReplies(out, size)
	return out, nil
}

// cutReplies cuts out's longest replies to even shares of what the rest leaves under maxHandoffBytes, size being out's JSON length.
func cutReplies(out []Handoff, size int) {
	if size <= maxHandoffBytes {
		return
	}
	lens, order := make([]int, len(out)), make([]int, len(out))
	budget := maxHandoffBytes - size
	for i, h := range out {
		lens[i], order[i] = jsonLen(h.Reply), i
		budget += lens[i]
	}
	slices.SortFunc(order, func(a, b int) int { return lens[a] - lens[b] })
	for k, i := range order {
		share := budget / (len(order) - k)
		if lens[i] > share {
			// An escape never shrinks a byte, so cutting the excess as raw bytes cuts at least as much JSON.
			out[i].Reply = harness.CapBytes(out[i].Reply, max(len(out[i].Reply)-(lens[i]-share), 0))
			lens[i] = share
		}
		budget -= lens[i]
	}
}

func jsonLen(s string) int {
	raw, _ := json.Marshal(s) // marshalling a string cannot fail
	return len(raw)
}

// PostSystemMessage posts an inline system message attributed to the user it's for.
func (s *Service) PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) (*Message, error) {
	return s.postMessage(ctx, conversationID, viaUserID, body, AuthorSystem)
}

func (s *Service) postMessage(ctx context.Context, conversationID, authorID, body string, kind AuthorKind) (*Message, error) {
	m, err := s.newMessage(conversationID, authorID, body, kind)
	if err != nil {
		return nil, err
	}
	return s.create(ctx, m)
}

func (s *Service) create(ctx context.Context, m *Message) (*Message, error) {
	membersOnly, err := s.MembersOnly(ctx, m.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("post message to conversation %s: %w", m.ConversationID, err)
	}
	if err := s.repo.CreateMessage(ctx, m, messageCreated(m, membersOnly)); err != nil {
		return nil, fmt.Errorf("post message to conversation %s: %w", m.ConversationID, err)
	}
	return m, nil
}

func (s *Service) newMessage(conversationID, authorID, body string, kind AuthorKind) (*Message, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return nil, fmt.Errorf("%w: author id is required", apperrs.ErrInvalid)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: message body is required", apperrs.ErrInvalid)
	}
	now := s.now().UTC()
	return &Message{
		ID:             ids.New(),
		ConversationID: conversationID,
		AuthorID:       authorID,
		AuthorKind:     kind,
		Body:           body,
		Mentions:       ParseMentions(body),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func messageCreated(m *Message, membersOnly bool) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageCreated, Payload: MessageCreatedEvent{Message: *m, MembersOnly: membersOnly}}
}

// MembersOnly reports whether a conversation is read by its members alone; the server's own seam, so unchecked.
func (s *Service) MembersOnly(ctx context.Context, conversationID string) (bool, error) {
	c, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return false, err
	}
	return c.MembersOnly(), nil
}

// EditMessage edits a message's body in place; only its author may edit it, and never a deleted one.
func (s *Service) EditMessage(ctx context.Context, messageID, authorID, body string) (*Message, error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil, fmt.Errorf("%w: message id is required", apperrs.ErrInvalid)
	}
	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return nil, fmt.Errorf("%w: author id is required", apperrs.ErrInvalid)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: message body is required", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("edit message %s: %w", messageID, err)
	}
	if current.DeletedAt != nil {
		return nil, fmt.Errorf("%w: cannot edit a deleted message", apperrs.ErrInvalid)
	}
	if current.AuthorID != authorID {
		return nil, fmt.Errorf("%w: only the author may edit this message", apperrs.ErrForbidden)
	}
	membersOnly, err := s.MembersOnly(ctx, current.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("edit message %s: %w", messageID, err)
	}
	now := s.now().UTC()
	updated := *current
	updated.Body = body
	updated.Mentions = ParseMentions(body)
	updated.EditedAt = &now
	updated.UpdatedAt = now
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageUpdated, Payload: MessageUpdatedEvent{Message: updated, MembersOnly: membersOnly}}
	if err := s.repo.UpdateMessage(ctx, messageID, updated.Body, updated.Mentions, now, evt); err != nil {
		return nil, fmt.Errorf("edit message %s: %w", messageID, err)
	}
	return &updated, nil
}

// maxEmojiBytes fits the longest emoji sequences, such as a family joined with skin tones.
const maxEmojiBytes = 64

// React adds or removes callerID's emoji on a message; whoever reads its conversation may react, never on a deleted one.
func (s *Service) React(ctx context.Context, messageID, callerID, emoji string, reacted bool) (*Message, error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil, fmt.Errorf("%w: message id is required", apperrs.ErrInvalid)
	}
	callerID = strings.TrimSpace(callerID)
	if callerID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	emoji = strings.TrimSpace(emoji)
	if emoji == "" || len(emoji) > maxEmojiBytes || strings.ContainsFunc(emoji, unicode.IsSpace) {
		return nil, fmt.Errorf("%w: emoji must be a single emoji such as 👍", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	if current.DeletedAt != nil {
		return nil, fmt.Errorf("%w: cannot react to a deleted message", apperrs.ErrInvalid)
	}
	if err := s.requireConversation(ctx, current.ConversationID, callerID); err != nil {
		return nil, err
	}
	membersOnly, err := s.MembersOnly(ctx, current.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageReactionsChanged, Payload: MessageReactionsChangedEvent{
		ConversationID: current.ConversationID, MessageID: messageID, UserID: callerID, Emoji: emoji, Reacted: reacted, MembersOnly: membersOnly,
	}}
	if err := s.repo.SetReaction(ctx, messageID, callerID, emoji, reacted, s.now().UTC(), evt); err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	return m, nil
}

// DeleteMessage soft-deletes a message (author-only, a note under tickets:write); deleting an already-deleted one is a no-op.
func (s *Service) DeleteMessage(ctx context.Context, messageID, authorID string) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return fmt.Errorf("%w: message id is required", apperrs.ErrInvalid)
	}
	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return fmt.Errorf("%w: author id is required", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	if current.DeletedAt != nil {
		return nil
	}
	if current.AttachmentID != "" {
		return s.deleteNote(ctx, current, authorID)
	}
	if current.AuthorID != authorID {
		return fmt.Errorf("%w: only the author may delete this message", apperrs.ErrForbidden)
	}
	membersOnly, err := s.MembersOnly(ctx, current.ConversationID)
	if err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	now := s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageDeleted, Payload: MessageDeletedEvent{ConversationID: current.ConversationID, MessageID: messageID, DeletedAt: now, MembersOnly: membersOnly}}
	if err := s.repo.DeleteMessage(ctx, messageID, now, evt); err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	return nil
}

// MarkRead advances userID's read cursor on a conversation to now.
func (s *Service) MarkRead(ctx context.Context, conversationID, userID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if err := s.requireConversation(ctx, conversationID, userID); err != nil {
		return err
	}
	if err := s.repo.MarkRead(ctx, conversationID, userID, s.now().UTC()); err != nil {
		return fmt.Errorf("mark conversation %s read for user %s: %w", conversationID, userID, err)
	}
	return nil
}

// UnreadCounts returns userID's unread message count per conversation their list shows, the nav badge's data source.
func (s *Service) UnreadCounts(ctx context.Context, workspaceID, userID string) (map[string]int, error) {
	listed, err := s.ListConversations(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.UnreadCounts(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("unread counts for user %s: %w", userID, err)
	}
	shown := make(map[string]bool, len(listed))
	for _, c := range listed {
		shown[c.ID] = true
	}
	counts := make(map[string]int, len(rows))
	for _, r := range rows {
		if shown[r.ConversationID] {
			counts[r.ConversationID] = r.Count
		}
	}
	return counts, nil
}
