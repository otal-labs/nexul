package automations

import (
	"context"
	"time"
)

// LogEvent is one durable event read from the outbox/event log for delivery
// to a dialed-in automation (ADR 0046).
type LogEvent struct {
	ID        string
	Topic     string
	Payload   []byte
	CreatedAt time.Time
}

// Cursor is a strictly-ordered log position: (created_at, id), id as tiebreaker; zero means "from the beginning".
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// EventLogReader is automations' view onto the durable outbox log, so a disconnect catches up on what it missed.
type EventLogReader interface {
	// After returns up to limit events on any of topics, strictly after cursor, oldest first.
	After(ctx context.Context, topics []string, cursor Cursor, limit int) ([]LogEvent, error)
	// Latest returns the newest event's position, or ok=false if empty; starts a new automation's cursor at "now" (ADR 0046).
	Latest(ctx context.Context) (cursor Cursor, ok bool, err error)
}

// CursorRepo persists the per-automation delivery cursor (ADR 0046): reconnect
// resumes from here so downtime loses nothing.
type CursorRepo interface {
	Get(ctx context.Context, automationID string) (cursor Cursor, ok bool, err error)
	Set(ctx context.Context, automationID string, cursor Cursor) error
}
