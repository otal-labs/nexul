// Package deadletter persists events that exhausted retries or failed
// permanently, and replays them back onto the bus.
package deadletter

import (
	"context"
	"time"
)

// DeadLetter is a persisted failed event, queryable via the future MCP tools
// list_dead_letters / replay_dead_letter.
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
