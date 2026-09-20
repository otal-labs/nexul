package inprocess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/eventbus/processed"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// defaultChannelBuffer is the per-subscription delivery buffer size. Nothing
// in this codebase needs a different buffer depth, so it isn't an Option.
const defaultChannelBuffer = 256

// Options configures the in-process bus. Zero values get documented defaults.
type Options struct {
	// Logger for lifecycle and failure logs. Defaults to slog.Default().
	Logger *slog.Logger
	// DrainTimeout bounds how long Close waits for in-flight handlers.
	// Defaults to 30s.
	DrainTimeout time.Duration
	// RetryMaxAttempts is the total handler runs, first attempt included.
	// Defaults to 3.
	RetryMaxAttempts int
	// RetryBackoff supplies retry delays. Defaults to 100ms..30s exp + jitter.
	RetryBackoff *Backoff
	// DedupeStore enables the dedupe middleware. Nil disables it.
	DedupeStore processed.Storer
	// DeadLetterStore enables the dead-letter middleware. Nil disables it.
	DeadLetterStore deadletter.Storer
}

// Bus is the in-process event bus; each subscription runs its own middleware chain with its own dedupe scope.
type Bus struct {
	log         *slog.Logger
	drain       time.Duration
	retryMax    int
	backoff     *Backoff
	dedupeStore processed.Storer
	dlStore     deadletter.Storer
	ctx         context.Context
	cancel      context.CancelFunc

	mu     sync.Mutex
	topics map[string][]*subscriber
	closed bool
	wg     sync.WaitGroup
}

// New builds a Bus; each subscriber gets its own chain from addSubscriber, scoping dedupe per subscription.
func New(opts Options) *Bus {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.DrainTimeout <= 0 {
		opts.DrainTimeout = 30 * time.Second
	}
	if opts.RetryMaxAttempts <= 0 {
		opts.RetryMaxAttempts = 3
	}
	if opts.RetryBackoff == nil {
		opts.RetryBackoff = NewBackoff(100*time.Millisecond, 30*time.Second,
			rand.New(rand.NewSource(time.Now().UnixNano())))
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Bus{
		log:         opts.Logger,
		drain:       opts.DrainTimeout,
		retryMax:    opts.RetryMaxAttempts,
		backoff:     opts.RetryBackoff,
		dedupeStore: opts.DedupeStore,
		dlStore:     opts.DeadLetterStore,
		ctx:         ctx,
		cancel:      cancel,
		topics:      make(map[string][]*subscriber),
	}
}

// Publish marshals payload, builds an envelope (UUIDv7 ID, propagated
// trace_id) and fans it out to every subscriber of topic.
func (b *Bus) Publish(ctx context.Context, topic string, payload any) error {
	ev, err := b.buildEvent(ctx, topic, payload)
	if err != nil {
		return err
	}
	return b.publish(ctx, ev)
}

// PublishWithID publishes under a caller-supplied ID so a crash-redelivered outbox/dead-letter row still dedupes.
func (b *Bus) PublishWithID(ctx context.Context, id, topic string, payload any) error {
	if id == "" {
		return errors.New("eventbus: publish with empty id")
	}
	ev, err := b.buildEvent(ctx, topic, payload)
	if err != nil {
		return err
	}
	ev.ID = id
	return b.publish(ctx, ev)
}

// publish delivers a pre-built envelope to every subscriber of ev.Topic. It is
// the single fan-out path used by Publish and by replay/relay-style producers.
func (b *Bus) publish(ctx context.Context, ev eventbus.Event) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return eventbus.ErrClosed
	}
	subs := make([]*subscriber, 0, len(b.topics[ev.Topic]))
	subs = append(subs, b.topics[ev.Topic]...)
	b.mu.Unlock()

	for _, s := range subs {
		if err := b.deliver(ctx, s, ev); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a fan-out handler for topic; use SubscribeWithConsumer for an explicit dedupe identity.
func (b *Bus) Subscribe(ctx context.Context, topic string, h eventbus.Handler) error {
	return b.subscribe(ctx, topic, "", h)
}

// SubscribeWithConsumer scopes the dedupe middleware to consumer; the label must be stable across restarts.
func (b *Bus) SubscribeWithConsumer(ctx context.Context, consumer, topic string, h eventbus.Handler) error {
	return b.subscribe(ctx, topic, consumer, h)
}

func (b *Bus) subscribe(ctx context.Context, topic, consumer string, h eventbus.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return eventbus.ErrClosed
	}
	if consumer == "" {
		consumer = topic + "#" + strconv.Itoa(len(b.topics[topic]))
	}
	s := b.addSubscriber(topic, consumer, h)
	b.topics[topic] = append(b.topics[topic], s)
	go s.run()
	return nil
}

// Close stops accepting work, drains in-flight handlers under DrainTimeout,
// then cancels anything still unfinished (nacked for next boot). Idempotent.
func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	for _, s := range b.allSubscribers() {
		close(s.stop)
	}
	b.mu.Unlock()

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(b.drain):
		b.cancel()
		select {
		case <-done:
			return eventbus.ErrDrainTimeout
		case <-time.After(b.drain):
			return eventbus.ErrDrainTimeout
		}
	}
}

// addSubscriber builds the subscriber's own chain; a shared dedupe scope would make the first to run skip the rest.
func (b *Bus) addSubscriber(topic, consumer string, h eventbus.Handler) *subscriber {
	b.wg.Add(1)
	return &subscriber{
		topic: topic,
		ch:    make(chan eventbus.Event, defaultChannelBuffer),
		stop:  make(chan struct{}),
		chain: b.buildChain(h, consumer),
		bus:   b,
	}
}

func (b *Bus) buildChain(h eventbus.Handler, consumer string) eventbus.Handler {
	mw := make([]eventbus.Middleware, 0, 4)
	if b.dlStore != nil {
		mw = append(mw, NewDeadLetter(b.dlStore, b.log))
	}
	mw = append(mw, NewRetry(RetryConfig{MaxAttempts: b.retryMax, Backoff: b.backoff}))
	if b.dedupeStore != nil {
		mw = append(mw, NewDedupe(processed.Scope(b.dedupeStore, consumer)))
	}
	mw = append(mw, Recover())
	return eventbus.Chain(h, mw...)
}

func (b *Bus) allSubscribers() []*subscriber {
	out := make([]*subscriber, 0, len(b.topics))
	for _, subs := range b.topics {
		out = append(out, subs...)
	}
	return out
}

func (b *Bus) buildEvent(ctx context.Context, topic string, payload any) (eventbus.Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return eventbus.Event{}, fmt.Errorf("marshal payload: %w", err)
	}
	traceID := logging.TraceIDFromCtx(ctx)
	if traceID == "" {
		traceID = logging.NewTraceID()
	}
	return eventbus.Event{
		ID:        logging.NewTraceID(),
		TraceID:   traceID,
		Topic:     topic,
		Timestamp: time.Now().UTC(),
		Payload:   data,
	}, nil
}

// deliver sends ev to one subscriber, honoring ctx cancellation and the
// subscriber's stop signal so a concurrent Close never deadlocks a publish.
func (b *Bus) deliver(ctx context.Context, s *subscriber, ev eventbus.Event) error {
	select {
	case s.ch <- ev:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.stop:
		return nil
	}
}
