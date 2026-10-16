package logging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capture is a slog handler that keeps what it was handed, standing in for the OTLP handler.
type capture struct {
	mu   sync.Mutex
	recs []slog.Record
}

func (c *capture) Enabled(context.Context, slog.Level) bool { return true }
func (c *capture) WithAttrs([]slog.Attr) slog.Handler       { return c }
func (c *capture) WithGroup(string) slog.Handler            { return c }
func (c *capture) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recs = append(c.recs, r)
	return nil
}

func attrsOf(r slog.Record) map[string]any {
	out := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.Any()
		return true
	})
	return out
}

func TestRelay_Emit_StampsFromTheStampAndIgnoresWhatThePayloadClaims(t *testing.T) {
	c := &capture{}
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	NewRelay(c).Emit(t.Context(), Stamp{Source: "runner", Name: "edge", ID: "r-1", Machine: "box", Version: "v1"}, []Record{{
		Time: at, Level: "warn", Message: "disk almost full",
		Attrs: map[string]any{"source": "server", "name": "other", "source_id": "x", "machine": "elsewhere", "version": "v9", "path": "/data", "count": float64(3)},
	}})

	require.Len(t, c.recs, 1)
	r := c.recs[0]
	assert.Equal(t, at, r.Time)
	assert.Equal(t, slog.LevelWarn, r.Level)
	assert.Equal(t, "disk almost full", r.Message)
	assert.Equal(t, map[string]any{"source": "runner", "name": "edge", "source_id": "r-1", "machine": "box", "version": "v1", "path": "/data", "count": float64(3)}, attrsOf(r))
}

func TestRelay_Emit_RedactsCutsAndFlattensWhatItWrites(t *testing.T) {
	c := &capture{}
	token := "ses_" + strings.Repeat("a", 50)
	NewRelay(c).Emit(t.Context(), Stamp{Source: "automations"}, []Record{{
		Level: "error", Message: "failed with " + token,
		Attrs: map[string]any{"url": "wss://example.com/x?token=abc", "long": strings.Repeat("x", 3000), "ctx": map[string]any{"user": "alice", "source": "kept inside a group"}, "list": []any{"a", token}},
	}})

	r := c.recs[0]
	assert.Equal(t, "failed with [redacted token]", r.Message)
	got := attrsOf(r)
	assert.Equal(t, "wss://example.com/x?token=[redacted token]", got["url"])
	assert.Len(t, []rune(got["long"].(string)), maxChars+1)
	assert.Equal(t, "alice", got["ctx.user"])
	assert.Equal(t, "kept inside a group", got["ctx.source"], "only a top-level name is the server's")
	assert.Equal(t, `["a","[redacted token]"]`, got["list"])
}

func TestRelay_Emit_ClampsTheClockAndMapsLevels(t *testing.T) {
	c := &capture{}
	NewRelay(c).Emit(t.Context(), Stamp{}, []Record{
		{Time: time.Now().Add(24 * time.Hour), Level: "DEBUG", Message: "future"},
		{Level: "nonsense", Message: "no time"},
	})

	require.Len(t, c.recs, 2)
	assert.WithinDuration(t, time.Now(), c.recs[0].Time, time.Minute, "a record from the future is stamped now")
	assert.Equal(t, slog.LevelDebug, c.recs[0].Level)
	assert.WithinDuration(t, time.Now(), c.recs[1].Time, time.Minute)
	assert.Equal(t, slog.LevelInfo, c.recs[1].Level)
}

func TestRelay_NilRelayDropsRecords(t *testing.T) {
	var r *Relay
	r.Emit(t.Context(), Stamp{}, []Record{{Message: "x"}})
}

func queueLogger(level string) (*slog.Logger, *Queue) {
	q := NewQueue()
	return slog.New(q.Handler(level)), q
}

func TestQueue_Handler_KeepsLevelMessageAndAttrsAndGatesOnLevel(t *testing.T) {
	l, q := queueLogger("info")
	l.Debug("below the level")
	l.With("runner", "edge").WithGroup("job").Warn("rolled back", "id", "d1", "attempt", 2)

	recs, _, _ := q.Peek(MaxBatch, MaxBatchBytes)
	require.Len(t, recs, 1)
	assert.Equal(t, "warn", recs[0].Level)
	assert.Equal(t, "rolled back", recs[0].Message)
	assert.WithinDuration(t, time.Now(), recs[0].Time, time.Minute)
	assert.Equal(t, map[string]any{"runner": "edge", "job": map[string]any{"id": "d1", "attempt": float64(2)}}, recs[0].Attrs)
}

func TestQueue_Handler_RedactsBeforeAnythingLeaves(t *testing.T) {
	l, q := queueLogger("info")
	token := "dep_" + strings.Repeat("b", 50)
	l.Info("dialing with "+token, "url", "wss://example.com/live?token=abc", "nested", map[string]any{"auth": token})

	recs, _, _ := q.Peek(MaxBatch, MaxBatchBytes)
	require.Len(t, recs, 1)
	assert.Equal(t, "dialing with [redacted token]", recs[0].Message)
	assert.Equal(t, "wss://example.com/live?token=[redacted token]", recs[0].Attrs["url"])
	assert.Equal(t, map[string]any{"auth": "[redacted token]"}, recs[0].Attrs["nested"])
}

func TestQueue_Overflow_KeepsTheNewestAndCountsWhatItDropped(t *testing.T) {
	l, q := queueLogger("info")
	for i := range QueueLimit + 5 {
		l.Info(fmt.Sprint("line ", i))
	}

	require.Equal(t, QueueLimit, q.Len())
	recs, next, dropped := q.Peek(3, MaxBatchBytes)
	require.Len(t, recs, 3)
	assert.Equal(t, 5, dropped)
	assert.Equal(t, "dropped 5 log lines", recs[0].Message)
	assert.Equal(t, "warn", recs[0].Level)
	assert.Equal(t, "line 5", recs[1].Message, "the oldest five went")
	assert.Equal(t, "line 6", recs[2].Message)

	q.Ack(next, dropped)
	recs, _, dropped = q.Peek(1, MaxBatchBytes)
	assert.Zero(t, dropped, "the warning is sent once")
	assert.Equal(t, "line 7", recs[0].Message)
	assert.Equal(t, QueueLimit-2, q.Len())
}

func TestQueue_AFailedBatchStaysQueuedAndTheDropCountSurvivesIt(t *testing.T) {
	l, q := queueLogger("info")
	for i := range QueueLimit + 2 {
		l.Info(fmt.Sprint("line ", i))
	}
	first, _, dropped := q.Peek(MaxBatch, MaxBatchBytes)
	require.Equal(t, 2, dropped)

	again, _, droppedAgain := q.Peek(MaxBatch, MaxBatchBytes)
	assert.Equal(t, messages(first), messages(again), "nothing left the queue without an Ack")
	assert.Equal(t, 2, droppedAgain)
	assert.Equal(t, QueueLimit, q.Len())
}

func messages(recs []Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.Message
	}
	return out
}

func TestQueue_AckAfterMoreOverflowOnlyRemovesWhatWasSent(t *testing.T) {
	l, q := queueLogger("info")
	for i := range QueueLimit {
		l.Info(fmt.Sprint("line ", i))
	}
	recs, next, dropped := q.Peek(10, MaxBatchBytes)
	require.Len(t, recs, 10)

	for i := range 4 {
		l.Info(fmt.Sprint("late ", i))
	}
	q.Ack(next, dropped)

	rest, _, droppedNow := q.Peek(1, MaxBatchBytes)
	assert.Equal(t, 4, droppedNow, "the four that overflowed during the send are still owed a warning")
	assert.Equal(t, "dropped 4 log lines", rest[0].Message)
	assert.Equal(t, QueueLimit-6, q.Len(), "four of the ten sent were already gone, so only six left the queue")
}

func TestQueue_PeekBoundsABatchByCountAndSize(t *testing.T) {
	l, q := queueLogger("info")
	for range 5 {
		l.Info(strings.Repeat("x", 1500))
	}

	recs, _, _ := q.Peek(MaxBatch, 4000)
	assert.Len(t, recs, 2, "two 1.5 KB records fit 4000 bytes")
	recs, _, _ = q.Peek(MaxBatch, 10)
	assert.Len(t, recs, 1, "a batch always carries at least one record")
	recs, _, _ = q.Peek(4, MaxBatchBytes)
	assert.Len(t, recs, 4)
}

func TestQueue_ARecordTooBigToSendKeepsItsMessageAndLosesItsAttrs(t *testing.T) {
	l, q := queueLogger("info")
	big := map[string]any{}
	for i := range 40 {
		big[fmt.Sprint("k", i)] = strings.Repeat("v", 1900)
	}
	l.Info("huge", "payload", big)

	recs, _, _ := q.Peek(MaxBatch, MaxBatchBytes)
	require.Len(t, recs, 1)
	assert.Equal(t, "huge", recs[0].Message)
	assert.Nil(t, recs[0].Attrs)
}

func TestQueue_AcceptsRecordsFromManyGoroutines(t *testing.T) {
	l, q := queueLogger("info")
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 200 {
				l.Info("x")
			}
		})
	}
	wg.Wait()
	assert.Equal(t, QueueLimit, q.Len())
}
