package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

const (
	// scheduleRetry is how soon a pass that failed runs again.
	scheduleRetry = 5 * time.Second
	// scheduleIdle is the longest the send loop sleeps; a save wakes it sooner.
	scheduleIdle = time.Hour
	// scheduleBatch is how many due messages one pass sends before it reads again.
	scheduleBatch = 50
)

// ScheduleInput is a message to post later: once at SendAt, or from SendAt on a repeat kept in Timezone.
type ScheduleInput struct {
	Body     string
	SendAt   time.Time
	Repeats  Repeat
	Timezone string
}

// ScheduleChange is a patch to a scheduled message; a nil field keeps its value.
type ScheduleChange struct {
	Body     *string
	SendAt   *time.Time
	Repeats  *Repeat
	Timezone *string
}

// ScheduleMessage saves a message to post in a conversation later as authorID, who must be able to post there now.
func (s *Service) ScheduleMessage(ctx context.Context, conversationID, authorID string, in ScheduleInput) (*ScheduledMessage, error) {
	c, err := s.repo.GetConversation(ctx, strings.TrimSpace(conversationID))
	if err != nil {
		return nil, fmt.Errorf("schedule message in conversation %s: %w", conversationID, err)
	}
	if err := s.requirePost(ctx, c, authorID); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	m := &ScheduledMessage{
		ID: ids.New(), ConversationID: c.ID, AuthorID: authorID, Body: in.Body, StartsAt: in.SendAt, SendAt: in.SendAt,
		Repeats: in.Repeats, Timezone: in.Timezone, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.validSchedule(m, true); err != nil {
		return nil, err
	}
	if err := s.repo.CreateScheduledMessage(ctx, m, scheduledChanged(m, c, false)); err != nil {
		return nil, fmt.Errorf("schedule message in conversation %s: %w", c.ID, err)
	}
	return m, nil
}

// PageScheduledMessages lists callerID's own scheduled messages in a conversation they read, soonest first.
func (s *Service) PageScheduledMessages(ctx context.Context, conversationID, callerID string, w paging.Window) ([]*ScheduledMessage, int, error) {
	c, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, 0, fmt.Errorf("list scheduled messages in conversation %s: %w", conversationID, err)
	}
	if err := s.requireRead(ctx, c, callerID); err != nil {
		return nil, 0, err
	}
	return s.repo.PageScheduledMessages(ctx, c.ID, callerID, w)
}

// UpdateScheduledMessage changes callerID's scheduled message; a new send time also becomes the start its repeat steps from.
func (s *Service) UpdateScheduledMessage(ctx context.Context, id, callerID string, ch ScheduleChange) (*ScheduledMessage, error) {
	m, c, err := s.ownScheduled(ctx, id, callerID)
	if err != nil {
		return nil, err
	}
	if ch.Body != nil {
		m.Body = *ch.Body
	}
	if ch.SendAt != nil {
		m.StartsAt, m.SendAt = *ch.SendAt, *ch.SendAt
	}
	if ch.Repeats != nil {
		m.Repeats = *ch.Repeats
	}
	if ch.Timezone != nil {
		m.Timezone = *ch.Timezone
	}
	m.UpdatedAt = s.now().UTC()
	if err := s.validSchedule(m, ch.SendAt != nil); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateScheduledMessage(ctx, m, scheduledChanged(m, c, false)); err != nil {
		return nil, fmt.Errorf("update scheduled message %s: %w", id, err)
	}
	return m, nil
}

// CancelScheduledMessage removes callerID's scheduled message, so it never posts again.
func (s *Service) CancelScheduledMessage(ctx context.Context, id, callerID string) error {
	m, c, err := s.ownScheduled(ctx, id, callerID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteScheduledMessage(ctx, m.ID, scheduledChanged(m, c, true)); err != nil {
		return fmt.Errorf("cancel scheduled message %s: %w", id, err)
	}
	return nil
}

// SendScheduledMessageNow posts callerID's scheduled message at once; a repeating one keeps its next send.
func (s *Service) SendScheduledMessageNow(ctx context.Context, id, callerID string) (*Message, error) {
	m, c, err := s.ownScheduled(ctx, id, callerID)
	if err != nil {
		return nil, err
	}
	if err := s.requirePost(ctx, c, callerID); err != nil {
		return nil, err
	}
	var next time.Time
	if m.Repeats != RepeatNone {
		next = m.SendAt
	}
	return s.sendScheduled(ctx, m, c, next)
}

// RunScheduleLoop posts each scheduled message when it falls due, sleeping until the next one or a commit that may
// plan an earlier one, until ctx ends.
func (s *Service) RunScheduleLoop(ctx context.Context, wake func() <-chan struct{}, logger *slog.Logger) {
	for {
		committed := wake()
		wait, err := s.SendDueScheduledMessages(ctx, logger)
		if err != nil {
			logger.Error("send scheduled messages", "error", err)
			wait = scheduleRetry
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-committed:
		case <-timer.C:
		}
		timer.Stop()
	}
}

// SendDueScheduledMessages posts every scheduled message due by now and returns how long until the next one. One that
// went unsent while the server was down posts once late, and a repeat then skips to its first send still ahead.
func (s *Service) SendDueScheduledMessages(ctx context.Context, logger *slog.Logger) (time.Duration, error) {
	now := s.now().UTC()
	due, err := s.repo.ScheduledMessagesDue(ctx, now, scheduleBatch)
	if err != nil {
		return 0, err
	}
	var errs []error
	for _, m := range due {
		errs = append(errs, s.sendDue(ctx, m, now, logger))
	}
	if err := errors.Join(errs...); err != nil {
		return 0, err
	}
	if len(due) == scheduleBatch {
		return 0, nil
	}
	at, ok, err := s.repo.NextScheduledMessage(ctx)
	if err != nil || !ok {
		return scheduleIdle, err
	}
	return min(max(at.Sub(now), 0), scheduleIdle), nil
}

// sendDue posts m as its author; one its author can no longer post is dropped, since it would never go.
func (s *Service) sendDue(ctx context.Context, m *ScheduledMessage, now time.Time, logger *slog.Logger) error {
	c, err := s.repo.GetConversation(ctx, m.ConversationID)
	if err != nil {
		return fmt.Errorf("send scheduled message %s: %w", m.ID, err)
	}
	err = s.requirePost(identity.WithActor(ctx, identity.Actor{ID: m.AuthorID}), c, m.AuthorID)
	if errors.Is(err, apperrs.ErrForbidden) || errors.Is(err, apperrs.ErrNotFound) {
		logger.Warn("drop scheduled message its author can no longer post", "scheduled_message_id", m.ID, "conversation_id", c.ID, "error", err)
		return ignoreConflict(s.repo.DeleteScheduledMessage(ctx, m.ID, scheduledChanged(m, c, true)))
	}
	if err != nil {
		return fmt.Errorf("send scheduled message %s: %w", m.ID, err)
	}
	next, _ := m.NextAfter(now)
	_, err = s.sendScheduled(ctx, m, c, next)
	return ignoreConflict(err)
}

// sendScheduled posts m's text as its author and moves it to next, or removes it when next is zero.
func (s *Service) sendScheduled(ctx context.Context, m *ScheduledMessage, c *Conversation, next time.Time) (*Message, error) {
	msg, err := s.newMessage(c.ID, m.AuthorID, m.Body, AuthorUser)
	if err != nil {
		return nil, err
	}
	sent := *m
	sent.SendAt = next
	evts := []eventbus.OutboxEvent{messageCreated(msg, c, false), scheduledChanged(&sent, c, next.IsZero())}
	if err := s.repo.SendScheduledMessage(ctx, m, msg, next, evts...); err != nil {
		return nil, fmt.Errorf("send scheduled message %s: %w", m.ID, err)
	}
	return msg, nil
}

// ownScheduled is callerID's scheduled message and its conversation; anyone else's reads as missing.
func (s *Service) ownScheduled(ctx context.Context, id, callerID string) (*ScheduledMessage, *Conversation, error) {
	m, err := s.repo.GetScheduledMessage(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, nil, err
	}
	if m.AuthorID != callerID {
		return nil, nil, fmt.Errorf("%w: scheduled message %s", apperrs.ErrNotFound, id)
	}
	c, err := s.repo.GetConversation(ctx, m.ConversationID)
	if err != nil {
		return nil, nil, fmt.Errorf("scheduled message %s: %w", id, err)
	}
	return m, c, nil
}

// validSchedule trims m and checks it; a send time it was given must still be ahead.
func (s *Service) validSchedule(m *ScheduledMessage, newTime bool) error {
	m.Body = strings.TrimSpace(m.Body)
	if m.Body == "" {
		return fmt.Errorf("%w: message body is required", apperrs.ErrInvalid)
	}
	if m.Repeats == "" {
		m.Repeats = RepeatNone
	}
	if !m.Repeats.valid() {
		return fmt.Errorf("%w: repeats is none, daily, weekly or monthly, not %q", apperrs.ErrInvalid, m.Repeats)
	}
	if m.Timezone == "" {
		m.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(m.Timezone); err != nil {
		return fmt.Errorf("%w: unknown timezone %q, such as Europe/London", apperrs.ErrInvalid, m.Timezone)
	}
	m.StartsAt, m.SendAt = m.StartsAt.UTC().Truncate(time.Second), m.SendAt.UTC().Truncate(time.Second)
	if newTime && !m.SendAt.After(s.now()) {
		return fmt.Errorf("%w: the send time must be in the future", apperrs.ErrInvalid)
	}
	return nil
}

// ignoreConflict passes over a schedule changed while it was being sent; the next pass reads it again.
func ignoreConflict(err error) error {
	if errors.Is(err, apperrs.ErrConflict) || errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	return err
}

func scheduledChanged(m *ScheduledMessage, c *Conversation, removed bool) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicScheduledMessageChanged, Payload: ScheduledMessageChangedEvent{
		ID: m.ID, ConversationID: c.ID, WorkspaceID: c.WorkspaceID, UserID: m.AuthorID, Removed: removed, MembersOnly: true,
	}}
}
