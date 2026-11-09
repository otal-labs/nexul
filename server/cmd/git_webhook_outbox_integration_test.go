package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

type keyResolver map[string]string

func (k keyResolver) TicketIDs(_ context.Context, _, _ string, keys []string) ([]string, error) {
	var ids []string
	for _, key := range keys {
		if id, ok := k[key]; ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func signedPROpened(t *testing.T, secret, delivery string) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"action": "opened",
		"pull_request": map[string]any{
			"number": 1, "title": "ACME-7 Fix login", "state": "open",
			"head": map[string]any{"ref": "ACME-7-fix-login", "sha": "abc123"},
			"base": map[string]any{"ref": "main"},
			"user": map[string]any{"login": "alice"},
		},
		"repository": map[string]any{"name": "app", "owner": map[string]any{"login": "acme"}},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/hooks/github", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-GitHub-Delivery", delivery)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	return req
}

// TestGitWebhook_ReachesTheAutomationLogOncePerDelivery: an opened PR lands in the outbox automations read, with its tickets
// resolved; GitHub redelivering it writes nothing new, and the in-process subscriber still runs exactly once.
func TestGitWebhook_ReachesTheAutomationLogOncePerDelivery(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	store := storage.New(migrated.DB(t), []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	var delivered atomic.Int32
	require.NoError(t, bus.SubscribeWithConsumer(ctx, "test.pr_opened", gitprovider.TopicPROpened, func(context.Context, eventbus.Event) error {
		delivered.Add(1)
		return nil
	}))
	runCtx, cancel := context.WithCancel(ctx)
	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Wake: store.Commits().Next, Logger: logger})
	var running sync.WaitGroup
	running.Go(func() { _ = relay.Run(runCtx) })
	t.Cleanup(func() { cancel(); running.Wait() })

	h := gitprovider.NewWebhookHandler("shhh", store.Outbox).WithTicketResolver(keyResolver{"ACME-7": "t-7"})
	for range 2 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, signedPROpened(t, "shhh", "delivery-1"))
		require.Equal(t, http.StatusOK, rec.Code)
	}

	events, err := store.AutomationEventLog.After(ctx, []string{gitprovider.TopicPROpened, gitprovider.TopicProviderEvent}, automations.Cursor{}, 10)
	require.NoError(t, err)
	require.Len(t, events, 2, "one envelope and one typed event, not one per delivery attempt")
	var opened gitprovider.PREvent
	for _, ev := range events {
		if ev.Topic == gitprovider.TopicPROpened {
			require.NoError(t, json.Unmarshal(ev.Payload, &opened))
		}
	}
	assert.Equal(t, []string{"t-7"}, opened.PR.LinkedTicketIDs)

	require.Eventually(t, func() bool {
		pending, err := store.Outbox.Unpublished(ctx, 10)
		return err == nil && len(pending) == 0 && delivered.Load() == 1
	}, 10*time.Second, 20*time.Millisecond)
	assert.Equal(t, int32(1), delivered.Load())
}
