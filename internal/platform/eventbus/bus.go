package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrClosed       = errors.New("eventbus: closed")
	ErrDrainTimeout = errors.New("eventbus: drain timeout")
)

// Event is the envelope every message carries. ID and TraceID
// are time-sortable UUIDv7s so consumers can dedupe and correlate.
type Event struct {
	ID        string          `json:"id"`
	TraceID   string          `json:"trace_id"`
	Topic     string          `json:"topic"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// Handler processes a single event; ErrRetryable/ErrFatal route it, anything else is treated as retryable.
type Handler func(ctx context.Context, ev Event) error

// OutboxEvent is what a domain mutation hands its repo to write to the outbox table in the same transaction.
type OutboxEvent struct {
	ID      string
	Topic   string
	Payload any
}

// Middleware wraps a Handler. The stack is composed with Chain; each
// middleware runs its inner handler and may transform errors or short-circuit.
type Middleware func(Handler) Handler

// Chain composes middleware outermost-first: ms[0] is the first to see the event.
func Chain(h Handler, ms ...Middleware) Handler {
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}
