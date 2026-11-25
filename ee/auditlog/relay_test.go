package auditlog

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/webhooksig"
)

// receiver is a destination that records each request and answers with status.
type receiver struct {
	mu     sync.Mutex
	status int
	bodies [][]byte
	sigs   []string
}

func (rc *receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.bodies = append(rc.bodies, body)
	rc.sigs = append(rc.sigs, r.Header.Get(webhooksig.Header))
	if rc.status == http.StatusFound {
		w.Header().Set("Location", "https://elsewhere.example.com")
	}
	w.WriteHeader(rc.status)
}

func (rc *receiver) received() []delivery {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	out := make([]delivery, 0, len(rc.bodies))
	for _, b := range rc.bodies {
		var d delivery
		_ = json.Unmarshal(b, &d)
		out = append(out, d)
	}
	return out
}

func serveReceiver(t *testing.T, status int) (*receiver, string) {
	t.Helper()
	rc := &receiver{status: status}
	srv := httptest.NewServer(rc)
	t.Cleanup(srv.Close)
	return rc, srv.URL
}

// streamingTo is a harness whose stream is on, unlocked, and sends to url signed with s3cret.
func streamingTo(t *testing.T, url string) harness {
	t.Helper()
	h := newHarness(t, true)
	h.repo.st = Settings{StreamURL: url, StreamSecret: "s3cret", StreamEnabled: true}
	return h
}

func row(id string, at time.Time) Pending {
	return Pending{ID: id, ActorType: "user", ActorID: "u-alice", Action: "PATCH /api/tickets/t1", CreatedAt: at}
}

func quietRelay(h harness) *Relay { return h.svc.NewRelay(nil, slog.New(slog.DiscardHandler)) }

func TestRelay_ADestinationThatDoesNotTake_TheRow_BacksOffAndKeepsIt(t *testing.T) {
	t.Parallel()
	refused := httptest.NewServer(http.NotFoundHandler())
	refusedURL := refused.URL
	refused.Close()
	tests := []struct {
		name   string
		status int
		url    string
		want   string
	}{
		{"a 500", http.StatusInternalServerError, "", "answered 500"},
		{"a 4xx", http.StatusUnauthorized, "", "answered 401"},
		{"a redirect, never followed", http.StatusFound, "", "answered 302"},
		{"a refused connection", 0, refusedURL, "could not be reached"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			url := tt.url
			var rc *receiver
			if url == "" {
				rc, url = serveReceiver(t, tt.status)
			}
			h := streamingTo(t, url)
			h.queue.add(row("a-1", now))

			require.NoError(t, quietRelay(h).Flush(t.Context()))

			got := h.queue.row("a-1")
			assert.Equal(t, "pending", got.status)
			assert.Equal(t, 1, got.Attempts)
			assert.Equal(t, now.Add(time.Second), got.next, "the first retry waits a second")
			assert.Contains(t, got.reason, tt.want)
			assert.NotContains(t, got.reason, url, "the reason never repeats the destination")
			if rc != nil {
				assert.Len(t, rc.received(), 1, "a redirect is not followed to a second request")
			}

			require.NoError(t, quietRelay(h).Flush(t.Context()))
			assert.Equal(t, 1, h.queue.row("a-1").Attempts, "a row is not sent again before its retry is due")
		})
	}
}

// hang is a transport that never answers, until the request gives up.
type hang struct{}

func (hang) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()
	return nil, r.Context().Err()
}

func TestRelay_ADestinationThatNeverAnswers_TimesOutAfter15Seconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := streamingTo(t, "https://logs.example.com/nexul")
		h.svc.cfg.Client = &http.Client{Timeout: deliveryTimeout, Transport: hang{}}
		h.queue.add(row("a-1", now))
		start := time.Now()

		require.NoError(t, quietRelay(h).Flush(t.Context()))

		assert.Equal(t, deliveryTimeout, time.Since(start))
		assert.Contains(t, h.queue.row("a-1").reason, "did not answer within 15s")
	})
}

func TestRelay_SendsSignedRowsOldestFirst_AndARestartedRelaySendsWhatWasQueued(t *testing.T) {
	t.Parallel()
	rc, url := serveReceiver(t, http.StatusNoContent)
	h := streamingTo(t, url)
	h.queue.add(row("a-2", now.Add(-time.Minute)))
	h.queue.add(row("a-1", now.Add(-2*time.Minute)))

	require.NoError(t, quietRelay(h).Flush(t.Context()), "a relay that never saw the rows written sends them")

	got := rc.received()
	require.Len(t, got, 2)
	assert.Equal(t, []string{"a-1", "a-2"}, []string{got[0].ID, got[1].ID}, "a batch goes out in created_at order")
	assert.Equal(t, delivery{
		DeliveryID: got[0].DeliveryID, ID: "a-1", CreatedAt: now.Add(-2 * time.Minute), ActorType: "user", ActorID: "u-alice",
		Action: "PATCH /api/tickets/t1", InstanceURL: "https://nexul.example.com",
	}, got[0])
	for i, body := range rc.bodies {
		assert.True(t, webhooksig.Verify("s3cret", body, rc.sigs[i]), "the signature verifies against the body")
	}
	assert.Equal(t, "delivered", h.queue.row("a-1").status)
	assert.Equal(t, now, h.queue.row("a-1").delivered)
}

func TestRelay_ARowWhose2xxWasNotRecorded_IsSentAgainWithTheSameID(t *testing.T) {
	t.Parallel()
	rc, url := serveReceiver(t, http.StatusOK)
	h := streamingTo(t, url)
	h.queue.add(row("a-1", now))
	h.queue.failDelivered = assertErr

	require.NoError(t, quietRelay(h).Flush(t.Context()))
	require.Equal(t, "pending", h.queue.row("a-1").status)
	require.NoError(t, quietRelay(h).Flush(t.Context()))

	got := rc.received()
	require.Len(t, got, 2, "at least once")
	assert.Equal(t, got[0].ID, got[1].ID, "the receiver dedupes on id")
	assert.NotEqual(t, got[0].DeliveryID, got[1].DeliveryID)
	assert.Equal(t, "delivered", h.queue.row("a-1").status)
}

var assertErr = io.ErrUnexpectedEOF

func TestRelay_EightFailures_MakeARowDead_AndRetryFailedSendsItAgain(t *testing.T) {
	t.Parallel()
	rc, url := serveReceiver(t, http.StatusServiceUnavailable)
	h := streamingTo(t, url)
	h.queue.add(row("a-1", now))
	relay := quietRelay(h)

	waits := []time.Duration{}
	for range relayMaxAttempts {
		require.NoError(t, relay.Flush(t.Context()))
		got := h.queue.row("a-1")
		if got.status == "dead" {
			break
		}
		waits = append(waits, got.next.Sub(*h.clock))
		*h.clock = got.next
	}

	got := h.queue.row("a-1")
	assert.Equal(t, "dead", got.status)
	assert.Equal(t, relayMaxAttempts, got.Attempts)
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second}, waits)
	v, err := h.svc.View(asOwner(t))
	require.NoError(t, err)
	assert.Equal(t, 1, v.Dead)
	assert.Contains(t, v.LastError, "answered 503")
	require.NoError(t, relay.Flush(t.Context()))
	assert.Len(t, rc.received(), relayMaxAttempts, "a dead row is not sent again by itself")

	rc.mu.Lock()
	rc.status = http.StatusOK
	rc.mu.Unlock()
	v, err = h.svc.RetryFailed(asOwner(t))
	require.NoError(t, err)
	assert.Equal(t, 0, v.Dead)
	assert.Equal(t, 1, v.Pending)
	require.NoError(t, relay.Flush(t.Context()))
	assert.Equal(t, "delivered", h.queue.row("a-1").status)
}

func TestRelay_NothingIsSent_WhileTheStreamCannotRun(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(h harness)
	}{
		{"the destination was removed", func(h harness) { h.repo.st.StreamURL = "" }},
		{"the stream is switched off", func(h harness) { h.repo.st.StreamEnabled = false }},
		{"the key is locked", func(h harness) { h.licenses.on = false }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rc, url := serveReceiver(t, http.StatusOK)
			h := streamingTo(t, url)
			h.queue.add(row("a-1", now))
			tt.setup(h)

			require.NoError(t, quietRelay(h).Flush(t.Context()))

			assert.Empty(t, rc.received())
			assert.Equal(t, "pending", h.queue.row("a-1").status, "queued rows stay for when the stream runs again")
			assert.False(t, h.svc.Streaming(t.Context()), "and no new row is queued")
		})
	}
}

func TestRelay_SettingsThatCannotBeRead_AreReturned(t *testing.T) {
	t.Parallel()
	h := streamingTo(t, "https://logs.example.com")
	h.repo.err = assertErr

	require.ErrorIs(t, quietRelay(h).Flush(t.Context()), assertErr)
	assert.False(t, h.svc.Streaming(t.Context()), "a row written now is kept, unstreamed")
}

func TestStreaming_IsOnOnlyWithADestinationAndTheKey(t *testing.T) {
	t.Parallel()
	h := streamingTo(t, "https://logs.example.com")
	assert.True(t, h.svc.Streaming(t.Context()))
}

// roundTrip answers every request with 200 and records its body, inside a synctest bubble.
type roundTrip struct {
	mu  sync.Mutex
	ids []string
}

func (rt *roundTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	var d delivery
	_ = json.NewDecoder(r.Body).Decode(&d)
	rt.mu.Lock()
	rt.ids = append(rt.ids, d.ID)
	rt.mu.Unlock()
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
}

func (rt *roundTrip) sent() []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return append([]string{}, rt.ids...)
}

func TestRelayRun_SendsAtStart_OnACommit_AndEvery5Seconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := streamingTo(t, "https://logs.example.com")
		rt := &roundTrip{}
		h.svc.cfg.Client = &http.Client{Transport: rt}
		h.svc.cfg.Now = time.Now
		var mu sync.Mutex
		current := make(chan struct{})
		wake := func() <-chan struct{} {
			mu.Lock()
			defer mu.Unlock()
			return current
		}
		commit := func() {
			mu.Lock()
			close(current)
			current = make(chan struct{})
			mu.Unlock()
		}
		h.queue.add(row("a-1", time.Now()))
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			h.svc.NewRelay(wake, slog.New(slog.DiscardHandler)).Run(ctx)
			close(done)
		}()

		synctest.Wait()
		assert.Equal(t, []string{"a-1"}, rt.sent(), "what is queued goes out at start")

		h.queue.add(row("a-2", time.Now()))
		commit()
		synctest.Wait()
		assert.Equal(t, []string{"a-1", "a-2"}, rt.sent(), "a commit wakes the relay at once")

		h.queue.add(row("a-3", time.Now()))
		time.Sleep(relayInterval - time.Millisecond)
		synctest.Wait()
		assert.Len(t, rt.sent(), 2)
		time.Sleep(time.Millisecond)
		synctest.Wait()
		assert.Equal(t, []string{"a-1", "a-2", "a-3"}, rt.sent(), "the fallback poll finds what no commit announced")

		cancel()
		<-done
	})
}
