package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ automations.CursorRepo = (*AutomationCursorsRepo)(nil)

// AutomationCursorsRepo persists each automation's last-acked log position, so a reconnect resumes without loss.
type AutomationCursorsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AutomationCursorsRepo) Get(ctx context.Context, automationID string) (automations.Cursor, bool, error) {
	row, err := r.q.GetAutomationCursor(ctx, automationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automations.Cursor{}, false, nil
		}
		return automations.Cursor{}, false, fmt.Errorf("get automation cursor %s: %w", automationID, err)
	}
	return automations.Cursor{CreatedAt: time.Unix(row.LastCreatedAt, 0).UTC(), ID: row.LastEventID}, true, nil
}

func (r *AutomationCursorsRepo) Set(ctx context.Context, automationID string, cursor automations.Cursor) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetAutomationCursor(ctx, sqlcgen.SetAutomationCursorParams{
			AutomationID:  automationID,
			LastCreatedAt: cursor.CreatedAt.Unix(),
			LastEventID:   cursor.ID,
			UpdatedAt:     time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("set automation cursor %s: %w", automationID, err)
		}
		return nil
	})
}
