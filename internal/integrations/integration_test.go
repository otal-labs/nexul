//go:build integration

package integrations_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

type ownerGate struct{}

func (ownerGate) CanCreateWorkspace(ctx context.Context, userID string) (bool, error) {
	return true, nil
}

func newSvc(store *storage.Store) *integrations.Service {
	return integrations.NewService(integrations.Config{
		Installs:   store.IntegrationInstalls,
		Tokens:     store.IntegrationTokens,
		Subs:       store.IntegrationSubs,
		Deliveries: store.IntegrationDeliveries,
		Schemas:    store.EventSchemas,
		Audit:      store.Audit,
		Owner:      ownerGate{},
		Now:        func() time.Time { return time.Now().UTC() },
	})
}

// TestEndToEndWebhookDelivery is the ws-28 acceptance scenario: install ->
// subscribe -> event published on the real bus -> fan-out enqueues a signed
// delivery -> the relay POSTs it -> the integration verifies the HMAC.
func TestEndToEndWebhookDelivery(t *testing.T) {
	ctx := context.Background()

	store := testutil.NewStore(t)

	bus := inprocess.New(inprocess.Options{
		Logger:           testutil.DiscardLogger(),
		RetryMaxAttempts: 2,
		RetryBackoff:     inprocess.NewBackoff(time.Millisecond, time.Millisecond, nil),
		DedupeStore:      store.ProcessedEvents,
	})
	t.Cleanup(func() { require.NoError(t, bus.Close()) })

	svc := newSvc(store)

	received := make(chan string, 1)
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("X-Nexul-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	raw, secret, install, err := svc.Install(ctx, "owner-1", "discord", integrations.TrustCommunity, hook.URL, []integrations.Scope{integrations.ScopeEventsRead})
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.NotEmpty(t, secret)
	require.NoError(t, svc.Subscribe(ctx, "owner-1", install.ID, "ticket.created"))

	fanout := integrations.NewFanoutHandler(svc)
	require.NoError(t, bus.Subscribe(ctx, "ticket.created", fanout.HandleEvent))

	relay := integrations.NewRelay(store.IntegrationDeliveries, integrations.RelayConfig{
		Interval: 10 * time.Millisecond, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond,
		MaxAttempts: 3, Logger: testutil.DiscardLogger(),
	})
	relayCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go relay.Run(relayCtx)

	require.NoError(t, bus.Publish(ctx, "ticket.created", map[string]any{"ticket": map[string]any{"id": "t1", "title": "hi"}}))

	select {
	case sig := <-received:
		require.NotEmpty(t, sig)
		deliveries, err := svc.ListDeliveries(ctx, "owner-1", install.ID)
		require.NoError(t, err)
		require.Len(t, deliveries, 1)
		assert.True(t, integrations.VerifyWebhookSignature(secret, deliveries[0].Payload, sig))
	case <-time.After(3 * time.Second):
		t.Fatal("webhook was never delivered")
	}

	require.Eventually(t, func() bool {
		deliveries, _ := svc.ListDeliveries(ctx, "owner-1", install.ID)
		return len(deliveries) == 1 && deliveries[0].Status == integrations.DeliveryDelivered
	}, 3*time.Second, 20*time.Millisecond)
}

// TestEndToEndDeployEnvRedaction verifies that a deploy.requested event
// delivered to an integration carries keys-only env values (R5 enforced at the
// webhook boundary).
func TestEndToEndDeployEnvRedaction(t *testing.T) {
	ctx := context.Background()

	store := testutil.NewStore(t)

	var body []byte
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body = buf[:n]
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	svc := newSvc(store)
	_, _, install, err := svc.Install(ctx, "owner-1", "deployer", integrations.TrustCommunity, hook.URL, []integrations.Scope{integrations.ScopeEventsRead})
	require.NoError(t, err)
	require.NoError(t, svc.Subscribe(ctx, "owner-1", install.ID, "deploy.requested"))

	fanout := integrations.NewFanoutHandler(svc)
	require.NoError(t, fanout.HandleEvent(ctx, eventbus.Event{
		ID: "evt-1", Topic: "deploy.requested", Timestamp: time.Now().UTC(),
		Payload: json.RawMessage(`{"id":"d1","kind":"deploy","env":{"API_KEY":"supersecret"}}`),
	}))

	relay := integrations.NewRelay(store.IntegrationDeliveries, integrations.RelayConfig{
		Interval: 10 * time.Millisecond, BackoffInitial: time.Millisecond, BackoffMax: time.Millisecond,
		Logger: testutil.DiscardLogger(),
	})
	require.NoError(t, relay.FlushOnce(ctx))

	require.NotContains(t, string(body), "supersecret")
	require.Contains(t, string(body), "API_KEY")
}
