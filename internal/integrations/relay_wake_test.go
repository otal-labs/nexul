package integrations

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/otal-labs/nexul/internal/platform/wake"
)

const relayFallback = 5 * time.Second

// queueStore is a DeliveryStore whose due queue a test fills; it counts every read of the queue.
type queueStore struct {
	DeliveryStore
	mu        sync.Mutex
	due       []*Delivery
	reads     int
	delivered []string
}

func (q *queueStore) Due(context.Context, int, int64) ([]*Delivery, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.reads++
	return append([]*Delivery(nil), q.due...), nil
}

func (q *queueStore) MarkDelivered(_ context.Context, id string, _ int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.delivered = append(q.delivered, id)
	q.due = nil
	return nil
}

func (q *queueStore) enqueue(d *Delivery) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.due = append(q.due, d)
}

func (q *queueStore) readCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.reads
}

func (q *queueStore) deliveredIDs() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]string(nil), q.delivered...)
}

type okTransport struct{}

func (okTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func startRelay(t *testing.T, store *queueStore, commits *wake.Broadcast) {
	t.Helper()
	relay := NewRelay(store, RelayConfig{Wake: commits.Next, Logger: testDiscardLogger()})
	relay.client = &http.Client{Transport: okTransport{}}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	go func() { _ = relay.Run(ctx) }()
	synctest.Wait()
}

func TestRelay_Run_ACommitWakesItBeforeTheNextPoll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := &queueStore{}
		var commits wake.Broadcast
		startRelay(t, store, &commits)

		store.enqueue(&Delivery{ID: "d1", URL: "https://example.com/hook"})
		commits.Notify()
		synctest.Wait()

		assert.Equal(t, []string{"d1"}, store.deliveredIDs(), "delivered with no time passing")
	})
}

func TestRelay_Run_AnIdleRelayReadsOnlyAtTheFallbackInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := &queueStore{}
		var commits wake.Broadcast
		startRelay(t, store, &commits)
		atStart := store.readCount()

		time.Sleep(relayFallback - time.Nanosecond)
		assert.Equal(t, atStart, store.readCount(), "no read before the fallback interval")

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, atStart+1, store.readCount(), "one read when the fallback interval passes")
	})
}
