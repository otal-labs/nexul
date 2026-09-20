package inprocess

import (
	"math/rand"
	"time"
)

// Backoff computes retry delay n as min(max, initial*2^n) plus jitter; rng is injectable for deterministic tests.
type Backoff struct {
	initial time.Duration
	max     time.Duration
	rng     *rand.Rand
}

func NewBackoff(initial, max time.Duration, rng *rand.Rand) *Backoff {
	return &Backoff{initial: initial, max: max, rng: rng}
}

// Delay returns the wait time for a failed attempt. attempt is zero-based.
func (b *Backoff) Delay(attempt int) time.Duration {
	d := b.initial
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
