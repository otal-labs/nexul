package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func testDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestFanoutHandler(t *testing.T) {
	setup := func(t *testing.T, scopes []Scope) (*Service, *fakeData, *Install) {
		t.Helper()
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		_, secret, install, err := svc.Install(context.Background(), "user-1", "discord", TrustCommunity, "https://example.com/hook", scopes)
		require.NoError(t, err)
		require.NotEmpty(t, secret)
		return svc, f, install
	}

	t.Run("enqueues signed delivery for subscribed topic", func(t *testing.T) {
		svc, _, install := setup(t, []Scope{ScopeEventsRead})
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))

		fh := NewFanoutHandler(svc)
		err := fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-1", Topic: "ticket.created", Timestamp: time.Now().UTC(),
			Payload: json.RawMessage(`{"ticket":{"id":"t1","title":"hi"}}`),
		})
		require.NoError(t, err)

		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
		d := deliveries[0]
		assert.Equal(t, DeliveryPending, d.Status)
		assert.Equal(t, "https://example.com/hook", d.URL)
		assert.Contains(t, d.Signature, "sha256=")

		var env deliveryEnvelope
		require.NoError(t, json.Unmarshal(d.Payload, &env))
		assert.Equal(t, "evt-1", env.EventID)
		assert.Equal(t, "ticket.created", env.Topic)
		assert.NotEmpty(t, env.DeliveryID)
		assert.False(t, env.Timestamp.IsZero())

		assert.True(t, VerifyWebhookSignature(install.WebhookSecret, d.Payload, d.Signature))
		assert.False(t, VerifyWebhookSignature("wrong-secret", d.Payload, d.Signature))
	})

	t.Run("deploy.requested env values are redacted to keys-only", func(t *testing.T) {
		svc, _, install := setup(t, []Scope{ScopeEventsRead})
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "deploy.requested"))

		fh := NewFanoutHandler(svc)
		raw := `{"id":"d1","kind":"deploy","service":"api","env":{"API_KEY":"supersecret","URL":"http://x"}}`
		err := fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-2", Topic: "deploy.requested", Timestamp: time.Now().UTC(), Payload: json.RawMessage(raw),
		})
		require.NoError(t, err)

		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
		var env deliveryEnvelope
		require.NoError(t, json.Unmarshal(deliveries[0].Payload, &env))
		var data map[string]any
		require.NoError(t, json.Unmarshal(env.Data, &data))
		envMap, ok := data["env"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "", envMap["API_KEY"])
		assert.Equal(t, "", envMap["URL"])
		assert.NotContains(t, string(env.Data), "supersecret")
	})

	t.Run("unsubscribed topic produces no delivery", func(t *testing.T) {
		svc, _, install := setup(t, []Scope{ScopeEventsRead})
		fh := NewFanoutHandler(svc)
		err := fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-3", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		})
		require.NoError(t, err)
		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		assert.Empty(t, deliveries)
	})

	t.Run("revoked install is skipped", func(t *testing.T) {
		svc, _, install := setup(t, []Scope{ScopeEventsRead})
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))
		require.NoError(t, svc.RevokeInstall(context.Background(), "user-1", install.ID))

		fh := NewFanoutHandler(svc)
		err := fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-4", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		})
		require.NoError(t, err)
		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		assert.Empty(t, deliveries)
	})

	t.Run("install without webhook url is skipped", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		_, _, install, err := svc.Install(context.Background(), "user-1", "callback-only", TrustCommunity, "", []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))

		fh := NewFanoutHandler(svc)
		err = fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-5", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		})
		require.NoError(t, err)
		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		assert.Empty(t, deliveries)
	})

	t.Run("duplicate event is idempotent", func(t *testing.T) {
		svc, _, install := setup(t, []Scope{ScopeEventsRead})
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))

		fh := NewFanoutHandler(svc)
		ev := eventbus.Event{ID: "evt-6", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{"ticket":{"id":"t1"}}`)}
		require.NoError(t, fh.HandleEvent(context.Background(), ev))
		require.NoError(t, fh.HandleEvent(context.Background(), ev))

		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		assert.Len(t, deliveries, 1)
	})
}

func TestRelay(t *testing.T) {
	t.Run("2xx marks delivered", func(t *testing.T) {
		var gotPayload, gotSig string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotSig = r.Header.Get("X-Nexul-Signature")
			buf := make([]byte, 1024)
			n, _ := r.Body.Read(buf)
			gotPayload = string(buf[:n])
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		_, secret, install, err := svc.Install(context.Background(), "user-1", "x", TrustCommunity, srv.URL, []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))

		fh := NewFanoutHandler(svc)
		require.NoError(t, fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-1", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{"ticket":{"id":"t1"}}`),
		}))

		relay := NewRelay(&fakeDeliveryStore{d: f}, RelayConfig{BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond})
		relay.client = srv.Client()
		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
		relay.deliver(context.Background(), deliveries[0])

		updated, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		assert.Equal(t, DeliveryDelivered, updated[0].Status)
		assert.NotNil(t, updated[0].DeliveredAt)
		assert.Equal(t, deliveries[0].Signature, gotSig)
		assert.Equal(t, string(deliveries[0].Payload), gotPayload)
		assert.True(t, VerifyWebhookSignature(secret, []byte(gotPayload), gotSig))
	})

	t.Run("5xx retries then dead-letters", func(t *testing.T) {
		var hits int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		_, _, install, err := svc.Install(context.Background(), "user-1", "x", TrustCommunity, srv.URL, []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))

		fh := NewFanoutHandler(svc)
		require.NoError(t, fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-2", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		}))

		relay := NewRelay(&fakeDeliveryStore{d: f}, RelayConfig{
			MaxAttempts: 3, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond,
		})
		relay.client = srv.Client()
		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)

		relay.deliver(context.Background(), deliveries[0])
		deliveries, _ = svc.ListDeliveries(context.Background(), "user-1", install.ID)
		assert.Equal(t, DeliveryFailed, deliveries[0].Status)
		assert.Equal(t, 1, deliveries[0].Attempts)

		deliveries[0].NextAttemptAt = time.Unix(0, 0)
		relay.deliver(context.Background(), deliveries[0])
		deliveries, _ = svc.ListDeliveries(context.Background(), "user-1", install.ID)
		assert.Equal(t, DeliveryFailed, deliveries[0].Status)
		assert.Equal(t, 2, deliveries[0].Attempts)

		deliveries[0].NextAttemptAt = time.Unix(0, 0)
		relay.deliver(context.Background(), deliveries[0])
		deliveries, _ = svc.ListDeliveries(context.Background(), "user-1", install.ID)
		assert.Equal(t, DeliveryDead, deliveries[0].Status)
		assert.Equal(t, 3, deliveries[0].Attempts)
		assert.Equal(t, 3, hits)
	})

	t.Run("connection error retries", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		_, _, install, err := svc.Install(context.Background(), "user-1", "x", TrustCommunity, "http://127.0.0.1:1", []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.Subscribe(context.Background(), "user-1", install.ID, "ticket.created"))

		fh := NewFanoutHandler(svc)
		require.NoError(t, fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-3", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		}))

		relay := NewRelay(&fakeDeliveryStore{d: f}, RelayConfig{MaxAttempts: 8, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond})
		relay.client = &http.Client{Timeout: 50 * time.Millisecond}
		deliveries, err := svc.ListDeliveries(context.Background(), "user-1", install.ID)
		require.NoError(t, err)
		relay.deliver(context.Background(), deliveries[0])
		deliveries, _ = svc.ListDeliveries(context.Background(), "user-1", install.ID)
		assert.Equal(t, DeliveryFailed, deliveries[0].Status)
		assert.GreaterOrEqual(t, deliveries[0].Attempts, 1)
	})
}

func TestRelayRun(t *testing.T) {
	t.Run("run loops until context cancelled", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		_, _, install, err := svc.Install(context.Background(), "owner-1", "x", TrustCommunity, "http://127.0.0.1:1", []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.Subscribe(context.Background(), "owner-1", install.ID, "ticket.created"))
		fh := NewFanoutHandler(svc)
		require.NoError(t, fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-run", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		}))

		relay := NewRelay(&fakeDeliveryStore{d: f}, RelayConfig{
			Interval: 5 * time.Millisecond, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond,
			MaxAttempts: 2, Logger: testDiscardLogger(),
		})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- relay.Run(ctx) }()
		time.Sleep(30 * time.Millisecond)
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(time.Second):
			t.Fatal("Run did not return after cancel")
		}
	})

	t.Run("flush delivers and returns nil", func(t *testing.T) {
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		received := make(chan struct{}, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received <- struct{}{}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		_, _, install, err := svc.Install(context.Background(), "owner-1", "x", TrustCommunity, srv.URL, []Scope{ScopeEventsRead})
		require.NoError(t, err)
		require.NoError(t, svc.Subscribe(context.Background(), "owner-1", install.ID, "ticket.created"))
		fh := NewFanoutHandler(svc)
		require.NoError(t, fh.HandleEvent(context.Background(), eventbus.Event{
			ID: "evt-flush", Topic: "ticket.created", Timestamp: time.Now().UTC(), Payload: json.RawMessage(`{}`),
		}))

		relay := NewRelay(&fakeDeliveryStore{d: f}, RelayConfig{Logger: testDiscardLogger()})
		relay.client = srv.Client()
		require.NoError(t, relay.FlushOnce(context.Background()))
		select {
		case <-received:
		case <-time.After(time.Second):
			t.Fatal("flush did not POST")
		}
	})

	t.Run("flush error path surfaces store failure", func(t *testing.T) {
		relay := NewRelay(&errorDeliveryStore{}, RelayConfig{Logger: testDiscardLogger()})
		require.Error(t, relay.FlushOnce(context.Background()))
	})
}

// errorDeliveryStore fails Due to exercise the flush error path.
type errorDeliveryStore struct{}

func (e *errorDeliveryStore) Create(ctx context.Context, d *Delivery) error { return nil }
func (e *errorDeliveryStore) Due(ctx context.Context, limit int, now int64) ([]*Delivery, error) {
	return nil, fmt.Errorf("boom")
}
func (e *errorDeliveryStore) MarkDelivered(ctx context.Context, id string, at int64) error {
	return nil
}
func (e *errorDeliveryStore) MarkFailed(ctx context.Context, id string, attempts int, nextAttemptAt int64) error {
	return nil
}
func (e *errorDeliveryStore) MarkDead(ctx context.Context, id string) error { return nil }
func (e *errorDeliveryStore) ListByInstall(ctx context.Context, installID string) ([]*Delivery, error) {
	return nil, nil
}

func TestRelayBackoff(t *testing.T) {
	b := NewRelayBackoff(time.Second, 30*time.Second, nil)
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{10, 30 * time.Second},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, b.Delay(tt.attempt))
	}
}

func TestRedactEventPayload(t *testing.T) {
	t.Run("deploy.requested blanks env values", func(t *testing.T) {
		got, err := redactEventPayload("deploy.requested", json.RawMessage(`{"env":{"A":"secret"}}`))
		require.NoError(t, err)
		assert.JSONEq(t, `{"env":{"A":""}}`, string(got))
	})

	t.Run("deploy.requested without env passes through", func(t *testing.T) {
		got, err := redactEventPayload("deploy.requested", json.RawMessage(`{"id":"x"}`))
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":"x"}`, string(got))
	})

	t.Run("other topics untouched", func(t *testing.T) {
		raw := json.RawMessage(`{"env":{"A":"keepme"}}`)
		got, err := redactEventPayload("ticket.created", raw)
		require.NoError(t, err)
		assert.Equal(t, raw, got)
	})

	t.Run("invalid payload errors", func(t *testing.T) {
		_, err := redactEventPayload("deploy.requested", json.RawMessage(`{not-json`))
		require.Error(t, err)
	})
}

func TestSignWebhook(t *testing.T) {
	sig := signWebhook("secret", []byte("payload"))
	require.NotEmpty(t, sig)
	assert.Contains(t, sig, "sha256=")
	assert.True(t, VerifyWebhookSignature("secret", []byte("payload"), sig))
	assert.False(t, VerifyWebhookSignature("other", []byte("payload"), sig))
	assert.False(t, VerifyWebhookSignature("secret", []byte("payload2"), sig))
	assert.False(t, VerifyWebhookSignature("secret", []byte("payload"), "sha256=deadbeef"))
}
