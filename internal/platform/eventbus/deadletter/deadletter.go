// Package deadletter persists events that exhausted retries or failed
// permanently, and replays them back onto the bus.
package deadletter

import (
	"context"
	"time"
)

// DeadLetter is a persisted failed event, listed and replayed by the dead_letter_list and dead_letter_replay MCP tools.
type DeadLetter struct {
	ID        string
	Topic     string
	Payload   []byte
	Error     string
	Attempts  int
	CreatedAt time.Time
}

// Storer persists and queries dead letters.
type Storer interface {
	Put(ctx context.Context, dl DeadLetter) error
	List(ctx context.Context, limit, offset int) ([]DeadLetter, error)
	Get(ctx context.Context, id string) (*DeadLetter, error)
	Delete(ctx context.Context, id string) error
}
