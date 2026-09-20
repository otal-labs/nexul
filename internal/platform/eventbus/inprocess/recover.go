// Package inprocess implements the Day-1 EventBus over in-process channels.
package inprocess

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Recover converts panics in the inner handler into retryable errors so the
// retry and dead-letter layers can route them instead of crashing the bus.
func Recover() eventbus.Middleware {
	return func(h eventbus.Handler) eventbus.Handler {
		return func(ctx context.Context, ev eventbus.Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = apperrs.Retryable(fmt.Errorf("panic: %v", r))
				}
			}()
			return h(ctx, ev)
		}
	}
}
