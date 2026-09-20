package inprocess

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// RetryConfig tunes the retry middleware.
type RetryConfig struct {
	// MaxAttempts is the total number of times the handler runs, including
	// the first attempt.
	MaxAttempts int
	// Backoff supplies the delay between attempts.
	Backoff *Backoff
}

// RetryExhausted wraps the final error once MaxAttempts have been tried. The
// dead-letter middleware reads Attempts from it to persist an accurate count.
type RetryExhausted struct {
	Err      error
	Attempts int
}

func (e *RetryExhausted) Error() string {
	return fmt.Sprintf("retries exhausted (%d attempts): %v", e.Attempts, e.Err)
}

func (e *RetryExhausted) Unwrap() error { return e.Err }

// NewRetry reruns the handler with exponential backoff; ErrFatal skips retrying, cancelling ctx aborts the wait.
func NewRetry(cfg RetryConfig) eventbus.Middleware {
	return func(h eventbus.Handler) eventbus.Handler {
		return func(ctx context.Context, ev eventbus.Event) error {
			var err error
			for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
				err = h(ctx, ev)
				if err == nil {
					return nil
				}
				if errors.Is(err, apperrs.ErrFatal) {
					return err
				}
				if attempt == cfg.MaxAttempts-1 {
					break
				}
				if err := sleep(ctx, cfg.Backoff.Delay(attempt)); err != nil {
					return err
				}
			}
			return &RetryExhausted{Err: err, Attempts: cfg.MaxAttempts}
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
