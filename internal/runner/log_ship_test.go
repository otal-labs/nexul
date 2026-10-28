package runner

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// relayed keeps what the relay was handed, standing in for the log store.
type relayed struct {
	mu   sync.Mutex
	recs []slog.Record
}

func (r *relayed) Enabled(context.Context, slog.Level) bool { return true }
func (r *relayed) WithAttrs([]slog.Attr) slog.Handler       { return r }
func (r *relayed) WithGroup(string) slog.Handler            { return r }
func (r *relayed) Handle(_ context.Context, rec slog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recs = append(r.recs, rec)
	return nil
}

func (r *relayed) attrs(i int) map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]any{}
	r.recs[i].Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.Any()
		return true
	})
	return out
}

func (r *relayed) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.recs)
}

func TestFrame_Log_WireShapeAndLimits(t *testing.T) {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	f := Frame{Type: FrameLog, Records: []logging.Record{{Time: at, Level: "warn", Message: "slow", Attrs: map[string]any{"ms": 7}}}}
	data, err := f.Encode()
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"log","records":[{"time":"2026-10-01T12:00:00Z","level":"warn","message":"slow","attrs":{"ms":7}}]}`, string(data))

	for name, records := range map[string][]logging.Record{
		"no records": nil,
		"too many":   make([]logging.Record, logging.MaxBatch+1),
	} {
		_, err := (&Frame{Type: FrameLog, Records: records}).Encode()
		require.Error(t, err, name)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid), name)
	}
	_, err = (&Frame{Type: FrameLog, Records: make([]logging.Record, logging.MaxBatch)}).Encode()
	assert.NoError(t, err, "a full batch is allowed")
}

func logFrame(rec logging.Record) Frame {
	return Frame{Type: FrameLog, Records: []logging.Record{rec}}
}

func TestHandler_Log_AMachineRunnersRecordsAreStampedFromItsConnection(t *testing.T) {
	store := &relayed{}
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.Logs = logging.NewRelay(store) })
	c := &runnerConn{id: "r-1", name: "edge", machine: "box", machineID: "m-1", version: "v1.2.3"}

	spoof := logging.Record{Level: "error", Message: "pull failed", Attrs: map[string]any{"source": "server", "name": "other", "machine": "elsewhere", "version": "v9", "image": "acme/api"}}
	require.NoError(t, h.dispatchFrame(t.Context(), c, logFrame(spoof)))

	require.Equal(t, 1, store.count())
	assert.Equal(t, map[string]any{"source": "runner", "name": "edge", "source_id": "r-1", "machine": "box", "version": "v1.2.3", "image": "acme/api"}, store.attrs(0))
	assert.Equal(t, slog.LevelError, store.recs[0].Level)
}

func TestHandler_Log_AnUnresolvedMachineFallsBackToItsID(t *testing.T) {
	store := &relayed{}
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.Logs = logging.NewRelay(store) })

	require.NoError(t, h.dispatchFrame(t.Context(), &runnerConn{id: "r-1", name: "edge", machineID: "m-1"}, logFrame(logging.Record{Message: "up"})))

	assert.Equal(t, "m-1", store.attrs(0)["machine"])
}

func TestHandler_Log_APersonalRunnersRecordsAreDropped(t *testing.T) {
	store := &relayed{}
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo(), func(c *HandlerConfig) { c.Logs = logging.NewRelay(store) })

	require.NoError(t, h.dispatchFrame(t.Context(), &runnerConn{id: "r-1", name: "alice-laptop", personal: true}, logFrame(logging.Record{Message: "secret path /home/alice"})))

	assert.Zero(t, store.count())
}

func TestHandler_Log_WithNoLogStoreTheFrameIsDroppedQuietly(t *testing.T) {
	h := newTestHandler(newFakeBus(), newFakeRunnerRepo())

	assert.NoError(t, h.dispatchFrame(t.Context(), newTestConn(), logFrame(logging.Record{Message: "up"})))
}

// readFrames hands every frame the runner sends to frames.
func readFrames(frames chan<- Frame) func(context.Context, *websocket.Conn) {
	return func(ctx context.Context, conn *websocket.Conn) {
		for {
			var raw json.RawMessage
			if err := wsjson.Read(ctx, conn, &raw); err != nil {
				return
			}
			f, err := ParseFrame(raw)
			if err != nil {
				return
			}
			frames <- *f
		}
	}
}

// logMessages collects the messages of the log frames among frames until done reports it has what it wants.
func logMessages(t *testing.T, frames <-chan Frame, done func(seen []string) bool) []string {
	t.Helper()
	var seen []string
	timeout := time.After(5 * time.Second)
	for !done(seen) {
		select {
		case f := <-frames:
			if f.Type != FrameLog {
				continue
			}
			for _, r := range f.Records {
				seen = append(seen, r.Message)
			}
		case <-timeout:
			t.Fatalf("log records never arrived; saw %v", seen)
		}
	}
	return seen
}

func TestClient_ShipsQueuedLogRecordsAsLogFrames(t *testing.T) {
	frames := make(chan Frame, 64)
	srv := wsTestServer(t, readFrames(frames))
	q := logging.NewQueue()
	l := slog.New(q.Handler("info"))
	l.Info("before the connection", "runner", "alpha")

	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Logs, c.logEvery = q, 10*time.Millisecond
	cancel, done := runClient(t, c)

	logMessages(t, frames, func(seen []string) bool { return slices.Contains(seen, "before the connection") })
	l.Warn("while connected")
	logMessages(t, frames, func(seen []string) bool { return slices.Contains(seen, "while connected") })
	cancel()
	assert.NoError(t, <-done)
	require.Eventually(t, func() bool { return q.Len() == 0 }, time.Second, 5*time.Millisecond, "an acknowledged record leaves the queue")
}

func TestClient_KeepsRecordsAcrossAnOutageAndSendsThemOnReconnect(t *testing.T) {
	frames := make(chan Frame, 64)
	var tries atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tries.Add(1) == 1 {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		go func() {
			defer func() { _ = conn.CloseNow() }() // the client usually closed it already
			readFrames(frames)(context.Background(), conn)
		}()
	}))
	t.Cleanup(srv.Close)
	q := logging.NewQueue()
	slog.New(q.Handler("info")).Error("queued while down")

	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Logs, c.log = q, slog.New(q.Handler("info"))
	cancel, done := runClient(t, c)

	seen := logMessages(t, frames, func(seen []string) bool { return slices.Contains(seen, "runner connected") })
	assert.Contains(t, seen, "queued while down")
	assert.Contains(t, seen, "runner connection lost; reconnecting", "why the connection dropped is the line worth reading")
	cancel()
	assert.NoError(t, <-done)
}

func TestClient_APersonalRunnerNeverShipsEvenWithAQueue(t *testing.T) {
	frames := make(chan Frame, 64)
	srv := wsTestServer(t, readFrames(frames))
	q := logging.NewQueue()
	slog.New(q.Handler("info")).Info("stays on the computer")

	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Personal, c.cfg.Logs = true, q
	cancel, done := runClient(t, c)

	beats := 0
	for beats < 3 {
		f := <-frames
		require.NotEqual(t, FrameLog, f.Type, "no log frame leaves a personal runner")
		if f.Type == FrameHeartbeat {
			beats++
		}
	}
	cancel()
	assert.NoError(t, <-done)
	assert.Equal(t, 1, q.Len())
}
