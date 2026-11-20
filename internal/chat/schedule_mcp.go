package chat

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type scheduleIn struct {
	ID       string  `json:"id,omitempty" jsonschema:"Your scheduled message, from message_list with scheduled: changes it by the fields given instead."`
	SendAt   *string `json:"send_at,omitempty" jsonschema:"When it posts, an RFC 3339 time such as 2026-11-03T09:00:00Z."`
	Repeats  *string `json:"repeats,omitempty" jsonschema:"none, daily, weekly or monthly; monthly skips a month without its day."`
	Timezone *string `json:"timezone,omitempty" jsonschema:"The IANA timezone a repeat keeps its clock time in, such as Europe/London; UTC by default."`
	Cancel   bool    `json:"cancel,omitempty" jsonschema:"true cancels it."`
	SendNow  bool    `json:"send_now,omitempty" jsonschema:"true posts it now; a repeat keeps its next send."`
}

type scheduledResult struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Body           string    `json:"body"`
	SendAt         time.Time `json:"send_at"`
	Repeats        Repeat    `json:"repeats"`
	Timezone       string    `json:"timezone"`
}

func toScheduledResult(m *ScheduledMessage) scheduledResult {
	return scheduledResult{ID: m.ID, ConversationID: m.ConversationID, Body: m.Body, SendAt: m.SendAt, Repeats: m.Repeats, Timezone: m.Timezone}
}

// listScheduled is message_list with scheduled: the caller's own messages waiting to post in the conversation.
func listScheduled(ctx context.Context, s *Service, in messageListIn, id, caller string) (any, error) {
	ms, total, err := s.PageScheduledMessages(ctx, id, caller, in.Window())
	if err != nil {
		return nil, err
	}
	out := make([]scheduledResult, 0, len(ms))
	for _, m := range ms {
		out = append(out, toScheduledResult(m))
	}
	return mcptool.PageOf(out, total, in.Window()), nil
}

// schedule is message_post with schedule: a new message to post later, or a change to one of the caller's.
func schedule(ctx context.Context, s *Service, in messagePostIn, caller string) (any, error) {
	sc := in.Schedule
	if in.File != nil || in.Reaction != nil || in.Queue {
		return nil, fmt.Errorf("%w: a scheduled message takes only a body, no file, reaction or queue", apperrs.ErrInvalid)
	}
	if sc.ID == "" && (sc.Cancel || sc.SendNow) {
		return nil, fmt.Errorf("%w: cancel and send_now need schedule.id, from message_list with scheduled", apperrs.ErrInvalid)
	}
	ch, err := sc.change(in.Body)
	if err != nil {
		return nil, err
	}
	if sc.ID != "" {
		return changeScheduled(ctx, s, in, ch, caller)
	}
	id, err := threadFor(ctx, s, in.messageTarget, in.WorkspaceID, caller)
	if err != nil {
		return nil, err
	}
	m, err := s.ScheduleMessage(ctx, id, caller, ScheduleInput{
		Body: valueOf(ch.Body), SendAt: valueOf(ch.SendAt), Repeats: valueOf(ch.Repeats), Timezone: valueOf(ch.Timezone),
	})
	if err != nil {
		return nil, err
	}
	return toScheduledResult(m), nil
}

// changeScheduled cancels, sends now, or patches the caller's scheduled message sc.ID.
func changeScheduled(ctx context.Context, s *Service, in messagePostIn, ch ScheduleChange, caller string) (any, error) {
	sc := in.Schedule
	if in.messageTarget != (messageTarget{}) {
		return nil, fmt.Errorf("%w: with schedule.id, leave out the target; the scheduled message already has its conversation", apperrs.ErrInvalid)
	}
	if sc.Cancel {
		if err := s.CancelScheduledMessage(ctx, sc.ID, caller); err != nil {
			return nil, err
		}
		return map[string]any{"id": sc.ID, "cancelled": true}, nil
	}
	if sc.SendNow {
		m, err := s.SendScheduledMessageNow(ctx, sc.ID, caller)
		if err != nil {
			return nil, err
		}
		return toMessageResult(m, nil), nil
	}
	m, err := s.UpdateScheduledMessage(ctx, sc.ID, caller, ch)
	if err != nil {
		return nil, err
	}
	return toScheduledResult(m), nil
}

// change is the schedule's fields as a patch, body included when one was given.
func (sc *scheduleIn) change(body string) (ScheduleChange, error) {
	ch := ScheduleChange{Timezone: sc.Timezone}
	if body != "" {
		ch.Body = &body
	}
	if sc.Repeats != nil {
		r := Repeat(*sc.Repeats)
		ch.Repeats = &r
	}
	if sc.SendAt == nil {
		return ch, nil
	}
	at, err := time.Parse(time.RFC3339, *sc.SendAt)
	if err != nil {
		return ScheduleChange{}, fmt.Errorf("%w: send_at is an RFC 3339 time such as 2026-11-03T09:00:00Z", apperrs.ErrInvalid)
	}
	ch.SendAt = &at
	return ch, nil
}
