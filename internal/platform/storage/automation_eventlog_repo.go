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

var _ automations.EventLogReader = (*AutomationEventLogRepo)(nil)

// AutomationEventLogRepo lets automations replay the outbox, filtered by topic and ordered past a durable cursor.
type AutomationEventLogRepo struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

// After returns up to limit rows on any topic, strictly after cursor, ignoring the relay's own published flag.
func (r *AutomationEventLogRepo) After(ctx context.Context, topics []string, cursor automations.Cursor, limit int) ([]automations.LogEvent, error) {
	if len(topics) == 0 {
		return nil, nil
	}
	rows, err := r.q.ListOutboxAfterCursor(ctx, sqlcgen.ListOutboxAfterCursorParams{
		Topics:          topics,
		CursorCreatedAt: cursor.CreatedAt.Unix(),
		CursorID:        cursor.ID,
		Limit:           int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("read automation event log: %w", err)
	}
	var out []automations.LogEvent
	for _, row := range rows {
		out = append(out, automations.LogEvent{
			ID: row.ID, Topic: row.Topic, Payload: row.Payload,
			CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out, nil
}

// Latest returns the newest event's position, or ok=false when the log is
// empty.
func (r *AutomationEventLogRepo) Latest(ctx context.Context) (automations.Cursor, bool, error) {
	row, err := r.q.LatestOutboxPosition(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return automations.Cursor{}, false, nil
		}
		return automations.Cursor{}, false, fmt.Errorf("latest outbox position: %w", err)
	}
	return automations.Cursor{CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), ID: row.ID}, true, nil
}
