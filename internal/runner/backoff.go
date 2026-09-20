package runner

import (
	"context"
	"math/rand"
	"time"
)

// Backoff computes exponential delay with jitter for reconnect attempts.
type Backoff struct {
	base time.Duration
	max  time.Duration
	rng  *rand.Rand
}

func NewBackoff(base, max time.Duration, rng *rand.Rand) *Backoff {
	return &Backoff{base: base, max: max, rng: rng}
}

// Delay returns the wait before attempt n (zero-based).
func (b *Backoff) Delay(attempt int) time.Duration {
	d := b.base
	for i := 0; i < attempt && d < b.max; i++ {
		d *= 2
	}
	if d > b.max {
		d = b.max
	}
	if d <= 0 || b.rng == nil {
		return d
	}
	return d + time.Duration(b.rng.Int63n(int64(d/2)))
}

// Wait blocks for the attempt delay or until ctx is cancelled, returning false when ctx fired first.
func (b *Backoff) Wait(ctx context.Context, attempt int) bool {
	t := time.NewTimer(b.Delay(attempt))
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
