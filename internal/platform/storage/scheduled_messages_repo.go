package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

func (r *ChatRepo) CreateScheduledMessage(ctx context.Context, m *chat.ScheduledMessage, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).ScheduledMessageInsert(ctx, sqlcgen.ScheduledMessageInsertParams{
			ID: m.ID, ConversationID: m.ConversationID, AuthorID: m.AuthorID, Body: m.Body, StartsAt: m.StartsAt.Unix(),
			SendAt: m.SendAt.Unix(), Repeats: string(m.Repeats), Timezone: m.Timezone, CreatedAt: m.CreatedAt.Unix(),
			UpdatedAt: m.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("create scheduled message %s: %w", m.ID, classifyWriteErr(err))
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) GetScheduledMessage(ctx context.Context, id string) (*chat.ScheduledMessage, error) {
	row, err := r.q.ScheduledMessageGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get scheduled message %s: %w", id, notFoundIfNoRows(err))
	}
	return toScheduledMessage(row), nil
}

func (r *ChatRepo) UpdateScheduledMessage(ctx context.Context, m *chat.ScheduledMessage, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).ScheduledMessageUpdate(ctx, sqlcgen.ScheduledMessageUpdateParams{
			Body: m.Body, StartsAt: m.StartsAt.Unix(), SendAt: m.SendAt.Unix(), Repeats: string(m.Repeats),
			Timezone: m.Timezone, UpdatedAt: m.UpdatedAt.Unix(), ID: m.ID,
		})
		if err != nil {
			return fmt.Errorf("update scheduled message %s: %w", m.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update scheduled message %s: %w", m.ID, apperrs.ErrNotFound)
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) DeleteScheduledMessage(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).ScheduledMessageDelete(ctx, id)
		if err != nil {
			return fmt.Errorf("delete scheduled message %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete scheduled message %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func (r *ChatRepo) PageScheduledMessages(ctx context.Context, conversationID, authorID string, w paging.Window) ([]*chat.ScheduledMessage, int, error) {
	w = w.Clamped()
	total, err := r.q.ScheduledMessagesCount(ctx, sqlcgen.ScheduledMessagesCountParams{ConversationID: conversationID, AuthorID: authorID})
	if err != nil {
		return nil, 0, fmt.Errorf("count scheduled messages in conversation %s: %w", conversationID, err)
	}
	rows, err := r.q.ScheduledMessagesPage(ctx, sqlcgen.ScheduledMessagesPageParams{
		ConversationID: conversationID, AuthorID: authorID, Limit: int64(w.Limit), Offset: int64(w.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list scheduled messages in conversation %s: %w", conversationID, err)
	}
	return toScheduledMessages(rows), int(total), nil
}

func (r *ChatRepo) ScheduledMessagesDue(ctx context.Context, now time.Time, limit int) ([]*chat.ScheduledMessage, error) {
	rows, err := r.q.ScheduledMessagesDue(ctx, sqlcgen.ScheduledMessagesDueParams{SendAt: now.Unix(), Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("scheduled messages due: %w", err)
	}
	return toScheduledMessages(rows), nil
}

func (r *ChatRepo) NextScheduledMessage(ctx context.Context) (time.Time, bool, error) {
	v, err := r.q.ScheduledMessageNext(ctx)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("next scheduled message: %w", err)
	}
	at, ok, err := optionalInt(v)
	if err != nil || !ok {
		return time.Time{}, false, err
	}
	return time.Unix(at, 0).UTC(), true, nil
}

// SendScheduledMessage moves or removes the schedule first, so one changed since it was read posts nothing.
func (r *ChatRepo) SendScheduledMessage(ctx context.Context, m *chat.ScheduledMessage, msg *chat.Message, next time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := moveScheduledMessage(ctx, q, m, next, msg.CreatedAt)
		if err != nil {
			return fmt.Errorf("send scheduled message %s: %w", m.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("send scheduled message %s: %w", m.ID, apperrs.ErrConflict)
		}
		if err := insertMessage(ctx, q, msg); err != nil {
			return err
		}
		return enqueueChatOutbox(ctx, tx, evts)
	})
}

func toScheduledMessage(row sqlcgen.ScheduledMessage) *chat.ScheduledMessage {
	return &chat.ScheduledMessage{
		ID: row.ID, ConversationID: row.ConversationID, AuthorID: row.AuthorID, Body: row.Body,
		StartsAt: time.Unix(row.StartsAt, 0).UTC(), SendAt: time.Unix(row.SendAt, 0).UTC(), Repeats: chat.Repeat(row.Repeats),
		Timezone: row.Timezone, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toScheduledMessages(rows []sqlcgen.ScheduledMessage) []*chat.ScheduledMessage {
	out := make([]*chat.ScheduledMessage, 0, len(rows))
	for _, row := range rows {
		out = append(out, toScheduledMessage(row))
	}
	return out
}

// moveScheduledMessage moves m to next, or removes it when next is zero, only while it is still as it was read.
func moveScheduledMessage(ctx context.Context, q *sqlcgen.Queries, m *chat.ScheduledMessage, next, at time.Time) (int64, error) {
	if next.IsZero() {
		return q.ScheduledMessageDeleteSent(ctx, sqlcgen.ScheduledMessageDeleteSentParams{
			ID: m.ID, SendAt: m.SendAt.Unix(), Body: m.Body, Repeats: string(m.Repeats),
		})
	}
	return q.ScheduledMessageAdvance(ctx, sqlcgen.ScheduledMessageAdvanceParams{
		NextSendAt: next.Unix(), UpdatedAt: at.Unix(), ID: m.ID, SendAt: m.SendAt.Unix(), Body: m.Body, Repeats: string(m.Repeats),
	})
}
