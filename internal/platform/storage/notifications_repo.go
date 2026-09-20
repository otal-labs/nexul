package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/workspace"
)

var _ workspace.NotificationRepo = (*NotificationsRepo)(nil)

// NotificationsRepo persists notifications; inserts are idempotent since fan-out derives IDs from the source event.
type NotificationsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// CreateMany enqueues the outbox event only if a row was newly inserted.
func (r *NotificationsRepo) CreateMany(ctx context.Context, ns []*workspace.Notification, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		inserted := false
		for _, n := range ns {
			rows, err := q.CreateNotificationIfAbsent(ctx, sqlcgen.CreateNotificationIfAbsentParams{
				ID: n.ID, UserID: n.UserID, Kind: string(n.Kind), SubjectType: string(n.SubjectType),
				SubjectID: n.SubjectID, SubjectTitle: n.SubjectTitle, Read: int64(boolInt(n.Read)), CreatedAt: n.CreatedAt.Unix(),
			})
			if err != nil {
				if errors.Is(classifyWriteErr(err), apperrs.ErrConflict) {
					continue // idempotent fan-out: same source event already created it
				}
				return fmt.Errorf("insert notification %s: %w", n.ID, err)
			}
			if rows > 0 {
				inserted = true
			}
		}
		if !inserted {
			return nil
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *NotificationsRepo) List(ctx context.Context, userID string, limit int) ([]*workspace.Notification, error) {
	rows, err := r.q.ListNotifications(ctx, sqlcgen.ListNotificationsParams{UserID: userID, Limit: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list notifications for %s: %w", userID, err)
	}
	return toNotifications(rows), nil
}

func (r *NotificationsRepo) UnreadCount(ctx context.Context, userID string) (int, error) {
	n, err := r.q.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("unread count for %s: %w", userID, err)
	}
	return int(n), nil
}

func (r *NotificationsRepo) MarkRead(ctx context.Context, userID, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MarkNotificationRead(ctx, sqlcgen.MarkNotificationReadParams{ID: id, UserID: userID})
		if err != nil {
			return fmt.Errorf("mark notification %s read: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mark notification %s read: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *NotificationsRepo) MarkAllRead(ctx context.Context, userID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).MarkAllNotificationsRead(ctx, userID); err != nil {
			return fmt.Errorf("mark all notifications read for %s: %w", userID, err)
		}
		return nil
	})
}

func toNotification(row sqlcgen.Notification) *workspace.Notification {
	return &workspace.Notification{
		ID:           row.ID,
		UserID:       row.UserID,
		Kind:         workspace.Kind(row.Kind),
		SubjectType:  workspace.SubjectType(row.SubjectType),
		SubjectID:    row.SubjectID,
		SubjectTitle: row.SubjectTitle,
		Read:         row.Read != 0,
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
	}
}

func toNotifications(rows []sqlcgen.Notification) []*workspace.Notification {
	var out []*workspace.Notification
	for _, row := range rows {
		out = append(out, toNotification(row))
	}
	return out
}
