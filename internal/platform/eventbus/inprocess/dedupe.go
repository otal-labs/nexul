package inprocess

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/processed"
)

// NewDedupe short-circuits events already in the processed store; recorded only after the handler returns nil.
func NewDedupe(store processed.Storer) eventbus.Middleware {
	return func(h eventbus.Handler) eventbus.Handler {
		return func(ctx context.Context, ev eventbus.Event) error {
			seen, err := store.Seen(ctx, ev.ID)
			if err != nil {
				return fmt.Errorf("dedupe seen %s: %w", ev.ID, err)
			}
			if seen {
				return nil
			}
			if err := h(ctx, ev); err != nil {
				return err
			}
			if err := store.Record(ctx, ev.ID); err != nil {
				return fmt.Errorf("dedupe record %s: %w", ev.ID, err)
			}
			return nil
		}
	}
}
