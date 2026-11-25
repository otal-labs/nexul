package auditlog

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	relayBatch       = 50
	relayInterval    = 5 * time.Second
	relayMaxAttempts = 8
	backoffFirst     = time.Second
	backoffMax       = 30 * time.Second
)

// Relay sends the stream's queue to its destination, shaped like the webhook relay (ADR 0190): a row is marked
// delivered only after a 2xx, so a crash sends it again and a receiver dedupes on its id.
type Relay struct {
	svc  *Service
	wake func() <-chan struct{}
	log  *slog.Logger
}

// NewRelay wires the relay; wake returns a channel the next commit that queues or changes something closes.
func (s *Service) NewRelay(wake func() <-chan struct{}, log *slog.Logger) *Relay {
	if wake == nil {
		wake = func() <-chan struct{} { return nil }
	}
	return &Relay{svc: s, wake: wake, log: log}
}

// Run sends what is due at start, after every commit that wakes it, and every 5 seconds for the retries whose time
// has come, until ctx is cancelled; a failed pass is logged and the next one tries again.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(relayInterval)
	defer ticker.Stop()
	for {
		committed := r.wake()
		if err := r.Flush(ctx); err != nil {
			r.log.Warn("audit stream relay", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-committed:
		}
	}
}

// Flush sends one batch of due rows, oldest first, while the stream is on, has a destination and the key unlocks
// audit; otherwise the rows stay queued.
func (r *Relay) Flush(ctx context.Context) error {
	st, err := r.svc.cfg.Repo.Get(ctx)
	if err != nil {
		return fmt.Errorf("read audit stream settings: %w", err)
	}
	if !st.StreamEnabled || st.StreamURL == "" || !r.svc.unlocked(ctx) {
		return nil
	}
	due, err := r.svc.cfg.Queue.Due(ctx, relayBatch, r.svc.cfg.Now())
	if err != nil {
		return fmt.Errorf("read due audit deliveries: %w", err)
	}
	for _, row := range due {
		if ctx.Err() != nil {
			return nil
		}
		r.deliver(ctx, st, row)
	}
	return nil
}

func (r *Relay) deliver(ctx context.Context, st Settings, row Pending) {
	_, err := r.svc.send(ctx, st, row)
	if err == nil {
		if err := r.svc.cfg.Queue.MarkDelivered(ctx, row.ID, r.svc.cfg.Now()); err != nil {
			r.log.Error("mark audit delivery sent", "audit_id", row.ID, "error", err)
		}
		return
	}
	attempts := row.Attempts + 1
	if attempts >= relayMaxAttempts {
		r.log.Warn("audit delivery dead", "audit_id", row.ID, "attempts", attempts, "error", err)
		if err := r.svc.cfg.Queue.MarkDead(ctx, row.ID, attempts, err.Error()); err != nil {
			r.log.Error("mark audit delivery dead", "audit_id", row.ID, "error", err)
		}
		return
	}
	next := r.svc.cfg.Now().Add(backoff(attempts))
	if err := r.svc.cfg.Queue.MarkFailed(ctx, row.ID, attempts, next, err.Error()); err != nil {
		r.log.Error("mark audit delivery failed", "audit_id", row.ID, "error", err)
	}
}

// backoff is the wait after the nth failed attempt: 1 second, doubling, at most 30.
func backoff(attempts int) time.Duration {
	d := backoffFirst
	for i := 1; i < attempts && d < backoffMax; i++ {
		d *= 2
	}
	return min(d, backoffMax)
}
