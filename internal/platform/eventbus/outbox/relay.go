package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Publisher delivers an entry onto the bus; the relay uses PublishWithID so a redelivered row keeps its ID.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
	PublishWithID(ctx context.Context, id, topic string, payload any) error
}

// Relay polls the outbox store and publishes unpublished entries. It is a
// goroutine started by the composition root.
type Relay struct {
	store     Storer
	publisher Publisher
	interval  time.Duration
	batchSize int
	log       *slog.Logger
}

// RelayConfig tunes the outbox relay. Zero values get the documented
// defaults.
type RelayConfig struct {
	// Interval between polls of the outbox. Default 100ms.
	Interval time.Duration
	// BatchSize is the max entries relayed per poll. Default 100.
	BatchSize int
	// Logger for failed flushes. Default slog.Default().
	Logger *slog.Logger
}

// NewRelay builds a Relay. Defaults: 100ms poll, batch of 100, slog.Default.
func NewRelay(store Storer, p Publisher, cfg RelayConfig) *Relay {
	if cfg.Interval <= 0 {
		cfg.Interval = 100 * time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Relay{
		store:     store,
		publisher: p,
		interval:  cfg.Interval,
		batchSize: cfg.BatchSize,
		log:       cfg.Logger,
	}
}

// Run polls until ctx is cancelled. A failed poll is logged and retried on the
// next tick so a transient publish error never loses a row.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.flush(ctx); err != nil {
				r.log.Error("outbox relay flush", "err", err)
			}
		}
	}
}

// flush marks entries published only after the publish succeeds, preserving at-least-once delivery.
func (r *Relay) flush(ctx context.Context) error {
	batch, err := r.store.Unpublished(ctx, r.batchSize)
	if err != nil {
		return fmt.Errorf("outbox next batch: %w", err)
	}
	for _, e := range batch {
		if err := r.publisher.PublishWithID(ctx, e.ID, e.Topic, json.RawMessage(e.Payload)); err != nil {
			return fmt.Errorf("outbox publish %s: %w", e.ID, err)
		}
		if err := r.store.MarkPublished(ctx, e.ID); err != nil {
			return fmt.Errorf("outbox mark %s: %w", e.ID, err)
		}
	}
	return nil
}
