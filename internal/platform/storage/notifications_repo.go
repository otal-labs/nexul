package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
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

// CreateMany enqueues the outbox events only if a row was newly inserted or an unread one lifted by a repeat.
func (r *NotificationsRepo) CreateMany(ctx context.Context, ns []*workspace.Notification, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		changed := false
		for _, n := range ns {
			rows, err := q.CreateNotificationIfAbsent(ctx, sqlcgen.CreateNotificationIfAbsentParams{
				ID: n.ID, UserID: n.UserID, WorkspaceID: n.WorkspaceID, Kind: string(n.Kind), SubjectType: string(n.SubjectType),
				SubjectID: n.SubjectID, SubjectTitle: n.SubjectTitle, Read: int64(boolInt(n.Read)), ReadAt: nullUnixPtr(n.ReadAt), CreatedAt: n.CreatedAt.Unix(),
			})
			if err != nil {
				if errors.Is(classifyWriteErr(err), apperrs.ErrConflict) {
					continue // idempotent fan-out: same source event already created it
				}
				return fmt.Errorf("insert notification %s: %w", n.ID, err)
			}
			if rows == 0 && n.Kind.LiftsOnRepeat() {
				rows, err = q.BumpUnreadNotification(ctx, sqlcgen.BumpUnreadNotificationParams{
					ID: n.ID, SubjectTitle: n.SubjectTitle, CreatedAt: n.CreatedAt.Unix(), UserID: n.UserID, WorkspaceID: n.WorkspaceID,
					Kind: string(n.Kind), SubjectType: string(n.SubjectType), SubjectID: n.SubjectID,
				})
				if err != nil {
					return fmt.Errorf("lift notification %s: %w", n.ID, err)
				}
			}
			changed = changed || rows > 0
		}
		if !changed {
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

// Page reads one window of userID's inbox, newest first, and how many notifications it holds under the same filters.
func (r *NotificationsRepo) Page(ctx context.Context, userID string, f workspace.InboxFilter, scope *workspace.InboxScope, w paging.Window) ([]*workspace.Notification, int, error) {
	workspaceIDs, projectIDs := "[]", "[]"
	if scope != nil {
		workspaceIDs, projectIDs = idsJSON(scope.WorkspaceIDs), idsJSON(scope.ProjectIDs)
	}
	rows, err := r.q.ListNotificationsPage(ctx, sqlcgen.ListNotificationsPageParams{
		UserID: userID, WorkspaceID: f.WorkspaceID, UnreadOnly: f.UnreadOnly,
		Scoped: scope != nil, WorkspaceIds: workspaceIDs, ProjectIds: projectIDs,
		PageLimit: int64(w.Limit), PageOffset: int64(w.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("page notifications for %s: %w", userID, err)
	}
	total, err := r.q.CountNotificationsPage(ctx, sqlcgen.CountNotificationsPageParams{
		UserID: userID, WorkspaceID: f.WorkspaceID, UnreadOnly: f.UnreadOnly,
		Scoped: scope != nil, WorkspaceIds: workspaceIDs, ProjectIds: projectIDs,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications for %s: %w", userID, err)
	}
	return toNotifications(rows), int(total), nil
}

func (r *NotificationsRepo) UnreadByProject(ctx context.Context, userID, workspaceID string) ([]workspace.UnreadGroup, error) {
	rows, err := r.q.CountUnreadNotificationsByProject(ctx, sqlcgen.CountUnreadNotificationsByProjectParams{UserID: userID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, fmt.Errorf("unread count for %s: %w", userID, err)
	}
	out := make([]workspace.UnreadGroup, len(rows))
	for i, row := range rows {
		out[i] = workspace.UnreadGroup{WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, Unread: int(row.Unread)}
	}
	return out, nil
}

func (r *NotificationsRepo) MarkRead(ctx context.Context, userID, id string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MarkNotificationRead(ctx, sqlcgen.MarkNotificationReadParams{ID: id, UserID: userID, ReadAt: nullUnixPtr(&at)})
		if err != nil {
			return fmt.Errorf("mark notification %s read: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mark notification %s read: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *NotificationsRepo) MarkAllRead(ctx context.Context, userID, workspaceID string, at time.Time) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).MarkAllNotificationsRead(ctx, sqlcgen.MarkAllNotificationsReadParams{UserID: userID, WorkspaceID: workspaceID, ReadAt: nullUnixPtr(&at)}); err != nil {
			return fmt.Errorf("mark all notifications read for %s: %w", userID, err)
		}
		return nil
	})
}

// DeleteExpired deletes every notification created before createdBefore, then every one read before readBefore.
func (r *NotificationsRepo) DeleteExpired(ctx context.Context, readBefore, createdBefore time.Time) (read, old int64, err error) {
	err = r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if old, err = q.DeleteNotificationsCreatedBefore(ctx, createdBefore.Unix()); err != nil {
			return fmt.Errorf("delete notifications created before %s: %w", createdBefore, err)
		}
		if read, err = q.DeleteNotificationsReadBefore(ctx, nullUnixPtr(&readBefore)); err != nil {
			return fmt.Errorf("delete notifications read before %s: %w", readBefore, err)
		}
		return nil
	})
	return read, old, err
}

func toNotification(row sqlcgen.Notification) *workspace.Notification {
	return &workspace.Notification{
		ID:           row.ID,
		UserID:       row.UserID,
		WorkspaceID:  row.WorkspaceID,
		Kind:         workspace.Kind(row.Kind),
		SubjectType:  workspace.SubjectType(row.SubjectType),
		SubjectID:    row.SubjectID,
		SubjectTitle: row.SubjectTitle,
		Read:         row.Read != 0,
		ReadAt:       unixPtrFromNull(row.ReadAt),
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
	}
}

func toNotifications(rows []sqlcgen.ListNotificationsPageRow) []*workspace.Notification {
	var out []*workspace.Notification
	for _, row := range rows {
		n := toNotification(row.Notification)
		n.FolderID, n.FolderName, n.FolderIsDefault = row.FolderID, row.FolderName, row.FolderIsDefault != 0
		n.ProjectID = row.ProjectID
		out = append(out, n)
	}
	return out
}
