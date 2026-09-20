package inprocess

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

var errBoom = errors.New("boom")

func newTestRetry(max int) eventbus.Middleware {
	return NewRetry(RetryConfig{
		MaxAttempts: max,
		Backoff:     NewBackoff(time.Nanosecond, time.Nanosecond, nil),
	})
}

func TestRetry_RoutesErrors(t *testing.T) {
	tests := []struct {
		name        string
		failures    int // failures before success; >=max means never succeeds
		fatal       bool
		maxAttempts int
		wantErr     bool
		wantFatal   bool
		wantExhaust bool
		wantCalls   int
	}{
		{"succeeds first attempt", 0, false, 3, false, false, false, 1},
		{"recovers after retryable failure", 1, false, 3, false, false, false, 2},
		{"exhausts retries", 99, false, 3, true, false, true, 3},
		{"single attempt exhaustion", 99, false, 1, true, false, true, 1},
		{"fatal skips retries", 99, true, 3, true, true, false, 1},
		{"plain error is retried", 1, false, 3, false, false, false, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := newTestRetry(tt.maxAttempts)
			calls := 0
			h := mw(func(ctx context.Context, ev eventbus.Event) error {
				calls++
				if tt.fatal {
					return apperrs.Fatal(errBoom)
				}
				if calls <= tt.failures {
					return apperrs.Retryable(errBoom)
				}
				return nil
			})
			err := h(context.Background(), eventbus.Event{ID: "e1"})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantFatal {
				assert.ErrorIs(t, err, apperrs.ErrFatal)
			}
			if tt.wantExhaust {
				var exhausted *RetryExhausted
				require.ErrorAs(t, err, &exhausted)
				assert.Equal(t, tt.maxAttempts, exhausted.Attempts)
				assert.ErrorIs(t, exhausted, errBoom)
			}
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestRetry_PlainErrorIsTreatedAsRetryable(t *testing.T) {
	mw := newTestRetry(2)
	calls := 0
	h := mw(func(ctx context.Context, ev eventbus.Event) error {
		calls++
		if calls == 1 {
			return errors.New("transient hiccup")
		}
		return nil
	})
	require.NoError(t, h(context.Background(), eventbus.Event{ID: "e1"}))
	assert.Equal(t, 2, calls)
}

func TestRetry_CancelledContext_AbortsBackoff(t *testing.T) {
	mw := NewRetry(RetryConfig{MaxAttempts: 3, Backoff: NewBackoff(time.Hour, time.Hour, nil)})
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	h := mw(func(ctx context.Context, ev eventbus.Event) error {
		calls++
		cancel()
		return apperrs.Retryable(errBoom)
	})
	err := h(ctx, eventbus.Event{ID: "e1"})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}
