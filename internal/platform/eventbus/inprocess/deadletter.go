package inprocess

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
)

// NewDeadLetter persists an escaping error and acks the event; cancellation nacks for redelivery instead.
func NewDeadLetter(store deadletter.Storer, log *slog.Logger) eventbus.Middleware {
	return func(h eventbus.Handler) eventbus.Handler {
		return func(ctx context.Context, ev eventbus.Event) error {
			err := h(ctx, ev)
			if err == nil {
				return nil
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			attempts := 1
			var exhausted *RetryExhausted
			if errors.As(err, &exhausted) {
				attempts = exhausted.Attempts
			}
			dl := deadletter.DeadLetter{
				ID:        ev.ID,
				Topic:     ev.Topic,
				Payload:   ev.Payload,
				Error:     err.Error(),
				Attempts:  attempts,
				CreatedAt: time.Now().UTC(),
			}
			if perr := store.Put(ctx, dl); perr != nil {
				log.Error("dead letter persist", "id", ev.ID, "topic", ev.Topic, "err", perr)
				return fmt.Errorf("dead letter %s: %w", ev.ID, perr)
			}
			log.Warn("event dead-lettered", "id", ev.ID, "topic", ev.Topic, "attempts", attempts, "err", err)
			return nil
		}
	}
}
