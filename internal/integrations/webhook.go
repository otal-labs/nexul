package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// FanoutHandler turns every published event into signed deliveries for every subscribed install (ADR 0043), idempotently.
type FanoutHandler struct {
	service *Service
	now     func() time.Time
}

// NewFanoutHandler wires the fan-out consumer over the service.
func NewFanoutHandler(s *Service) *FanoutHandler {
	return &FanoutHandler{service: s, now: time.Now}
}

// deliveryEnvelope's delivery_id/timestamp let retries dedupe (ADR 0043).
type deliveryEnvelope struct {
	DeliveryID string          `json:"delivery_id"`
	EventID    string          `json:"event_id"`
	Topic      string          `json:"topic"`
	Timestamp  time.Time       `json:"timestamp"`
	Data       json.RawMessage `json:"data"`
}

// HandleEvent enqueues one delivery per active subscription on the event's topic.
func (f *FanoutHandler) HandleEvent(ctx context.Context, ev eventbus.Event) error {
	subs, err := f.service.cfg.Subs.ListByTopic(ctx, ev.Topic)
	if err != nil {
		return fmt.Errorf("fanout: list subscriptions for %s: %w", ev.Topic, err)
	}
	for _, sub := range subs {
		if err := f.enqueue(ctx, sub.InstallID, ev); err != nil {
			return err
		}
	}
	return nil
}

// enqueue builds one signed delivery for an install and writes it idempotently.
func (f *FanoutHandler) enqueue(ctx context.Context, installID string, ev eventbus.Event) error {
	install, err := f.service.cfg.Installs.GetByID(ctx, installID)
	if err != nil {
		return fmt.Errorf("fanout: get install %s: %w", installID, err)
	}
	if install.RevokedAt != nil || install.WebhookURL == "" || install.WebhookSecret == "" {
		return nil
	}
	data, err := redactEventPayload(ev.Topic, ev.Payload)
	if err != nil {
		return fmt.Errorf("fanout: redact %s: %w", ev.Topic, err)
	}
	now := f.now().UTC()
	envelope := deliveryEnvelope{
		DeliveryID: ids.New(),
		EventID:    ev.ID,
		Topic:      ev.Topic,
		Timestamp:  now,
		Data:       data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("fanout: marshal envelope: %w", err)
	}
	delivery := &Delivery{
		ID:            ids.New(),
		InstallID:     install.ID,
		Topic:         ev.Topic,
		EventID:       ev.ID,
		Payload:       payload,
		Signature:     signWebhook(install.WebhookSecret, payload),
		URL:           install.WebhookURL,
		Status:        DeliveryPending,
		NextAttemptAt: now,
		CreatedAt:     now,
	}
	if err := f.service.cfg.Deliveries.Create(ctx, delivery); err != nil {
		return fmt.Errorf("fanout: enqueue delivery: %w", err)
	}
	return nil
}

// signWebhook computes the "sha256=<hex>" HMAC header value the integration verifies with the shared webhook secret.
func signWebhook(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature is the verification side of the outbound HMAC signing.
func VerifyWebhookSignature(secret string, payload []byte, header string) bool {
	want := signWebhook(secret, payload)
	return hmac.Equal([]byte(header), []byte(want))
}

// redactEventPayload blanks deploy.requested's Env values (secrets), keeping keys; other topics pass through (R5).
func redactEventPayload(topic string, payload json.RawMessage) (json.RawMessage, error) {
	if topic != "deploy.requested" {
		return payload, nil
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil, err
	}
	env, ok := m["env"].(map[string]any)
	if !ok {
		return payload, nil
	}
	redacted := make(map[string]any, len(env))
	for k := range env {
		redacted[k] = ""
	}
	m["env"] = redacted
	return json.Marshal(m)
}

// RelayConfig tunes the webhook delivery relay (ADR 0043); zero values get the documented defaults.
type RelayConfig struct {
	// Interval between polls of the due queue; default 1s.
	Interval time.Duration
	// BatchSize is the max deliveries fetched per poll; default 50.
	BatchSize int
	// MaxAttempts is the delivery attempt budget before a row is dead; default 8.
	MaxAttempts int
	// BackoffInitial / BackoffMax bound the exponential retry delay.
	BackoffInitial time.Duration
	BackoffMax     time.Duration
	// Logger for failed deliveries.
	Logger *slog.Logger
}

// Relay polls due deliveries, POSTing at-least-once: marked delivered only after a 2xx, so a crash re-delivers (ADR 0043).
type Relay struct {
	store    DeliveryStore
	client   *http.Client
	interval time.Duration
	batch    int
	max      int
	backoff  *RelayBackoff
	log      *slog.Logger
}

// NewRelay wires a delivery relay over the store and a real HTTP client.
func NewRelay(store DeliveryStore, cfg RelayConfig) *Relay {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 8
	}
	if cfg.BackoffInitial <= 0 {
		cfg.BackoffInitial = time.Second
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = 30 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Relay{
		store:    store,
		client:   &http.Client{Timeout: 15 * time.Second},
		interval: cfg.Interval,
		batch:    cfg.BatchSize,
		max:      cfg.MaxAttempts,
		backoff:  NewRelayBackoff(cfg.BackoffInitial, cfg.BackoffMax, rand.New(rand.NewSource(time.Now().UnixNano()))),
		log:      cfg.Logger,
	}
}

// Run polls the due queue until cancelled; a failed poll is logged and retried next tick, never dropping a row.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.flush(ctx); err != nil {
				r.log.Error("webhook relay flush", "err", err)
			}
		}
	}
}

// flush delivers every due row in one batch.
func (r *Relay) flush(ctx context.Context) error {
	due, err := r.store.Due(ctx, r.batch, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("webhook due batch: %w", err)
	}
	for _, d := range due {
		r.deliver(ctx, d)
	}
	return nil
}

// FlushOnce runs a single poll of the due queue for tests; production uses Run.
func (r *Relay) FlushOnce(ctx context.Context) error {
	return r.flush(ctx)
}

// deliver POSTs one signed delivery and advances its state.
func (r *Relay) deliver(ctx context.Context, d *Delivery) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.URL, bytes.NewReader(d.Payload))
	if err != nil {
		r.log.Error("webhook relay build request", "delivery", d.ID, "install", d.InstallID, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nexul-Signature", d.Signature)
	req.Header.Set("X-Nexul-Delivery-Id", d.ID)

	resp, err := r.client.Do(req)
	if err != nil {
		r.recordFailure(ctx, d, err)
		return
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			r.log.Debug("webhook relay close response body", "delivery", d.ID, "err", cerr)
		}
	}()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := r.store.MarkDelivered(ctx, d.ID, time.Now().Unix()); err != nil {
			r.log.Error("webhook relay mark delivered", "delivery", d.ID, "err", err)
		}
		return
	}
	r.recordFailure(ctx, d, fmt.Errorf("webhook returned %d", resp.StatusCode))
}

// recordFailure advances a delivery one retry step, or dead-letters it after
// the attempt budget is exhausted.
func (r *Relay) recordFailure(ctx context.Context, d *Delivery, cause error) {
	attempts := d.Attempts + 1
	if attempts >= r.max {
		if err := r.store.MarkDead(ctx, d.ID); err != nil {
			r.log.Error("webhook relay mark dead", "delivery", d.ID, "err", err)
		}
		r.log.Warn("webhook delivery dead", "delivery", d.ID, "install", d.InstallID, "topic", d.Topic, "attempts", attempts, "cause", cause)
		return
	}
	next := time.Now().Add(r.backoff.Delay(attempts)).Unix()
	if err := r.store.MarkFailed(ctx, d.ID, attempts, next); err != nil {
		r.log.Error("webhook relay mark failed", "delivery", d.ID, "err", err)
	}
	r.log.Warn("webhook delivery failed", "delivery", d.ID, "install", d.InstallID, "topic", d.Topic, "attempts", attempts, "next", next, "cause", cause)
}

// RelayBackoff computes exponential retry delays with jitter: min(max, initial*2^n) + jitter in [0, delay/2].
type RelayBackoff struct {
	initial time.Duration
	max     time.Duration
	rng     *rand.Rand
}

// NewRelayBackoff wires a backoff with an injectable rng for deterministic tests.
func NewRelayBackoff(initial, max time.Duration, rng *rand.Rand) *RelayBackoff {
	return &RelayBackoff{initial: initial, max: max, rng: rng}
}

// Delay returns the wait after attempt n failures (attempt is 1-based).
func (b *RelayBackoff) Delay(attempt int) time.Duration {
	d := b.initial
	for i := 1; i < attempt && d < b.max; i++ {
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
