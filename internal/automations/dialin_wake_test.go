package automations

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/otal-labs/nexul/internal/platform/wake"
)

const deliverFallback = 5 * time.Second

// countingLog is an event log with nothing in it that counts every read.
type countingLog struct {
	mu    sync.Mutex
	reads int
}

func (c *countingLog) After(context.Context, []string, Cursor, int) ([]LogEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	return nil, nil
}

func (c *countingLog) Latest(context.Context) (Cursor, bool, error) { return Cursor{}, false, nil }

func (c *countingLog) Settled(context.Context) (Cursor, bool, error) { return Cursor{}, false, nil }

func (c *countingLog) readCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reads
}

type zeroCursors struct{}

func (zeroCursors) Get(context.Context, string) (Cursor, bool, error) { return Cursor{}, true, nil }
func (zeroCursors) Set(context.Context, string, Cursor) error         { return nil }

// startDeliverLoops runs one delivery loop per automation id over a shared log and commit broadcast.
func startDeliverLoops(t *testing.T, log *countingLog, commits *wake.Broadcast, automationIDs ...string) {
	t.Helper()
	repo := newFakeRepo()
	for _, id := range automationIDs {
		repo.rows[id] = &Automation{ID: id, Enabled: true, Subscriptions: []string{"ticket.created"}}
	}
	h := NewDialinHandler(nil, DialinConfig{Repo: repo, Cursors: zeroCursors{}, EventLog: log, Wake: commits.Next, Logger: testLogger()})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	for _, id := range automationIDs {
		go h.deliverLoop(ctx, &automationConn{id: id})
	}
	synctest.Wait()
}

func TestDeliverLoop_ACommitWakesEveryConnectionBeforeTheNextPoll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := &countingLog{}
		var commits wake.Broadcast
		startDeliverLoops(t, log, &commits, "a1", "a2", "a3")
		atStart := log.readCount()

		commits.Notify()
		synctest.Wait()

		assert.Equal(t, atStart+3, log.readCount(), "each of the three connections reads once, with no time passing")
	})
}

func TestDeliverLoop_AnIdleConnectionReadsOnlyAtTheFallbackInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := &countingLog{}
		var commits wake.Broadcast
		startDeliverLoops(t, log, &commits, "a1")
		atStart := log.readCount()

		time.Sleep(deliverFallback - time.Nanosecond)
		assert.Equal(t, atStart, log.readCount(), "no read before the fallback interval")

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, atStart+1, log.readCount(), "one read when the fallback interval passes")
	})
}

// pagedLog answers After from a fixed log by cursor and limit, as the outbox query does.
type pagedLog struct{ events []LogEvent }

func (l pagedLog) After(_ context.Context, _ []string, c Cursor, limit int) ([]LogEvent, error) {
	var out []LogEvent
	for _, ev := range l.events {
		if ev.ID > c.ID && len(out) < limit {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (pagedLog) Latest(context.Context) (Cursor, bool, error)  { return Cursor{}, false, nil }
func (pagedLog) Settled(context.Context) (Cursor, bool, error) { return Cursor{}, false, nil }

type memCursors struct {
	mu sync.Mutex
	m  map[string]Cursor
}

func (c *memCursors) Get(_ context.Context, id string) (Cursor, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur, ok := c.m[id]
	return cur, ok, nil
}

func (c *memCursors) Set(_ context.Context, id string, cur Cursor) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[id] = cur
	return nil
}

func TestDeliverLoop_ABacklogPastOneBatchDrainsWithoutACommitOrTheFallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		at := time.Now()
		var log pagedLog
		for i := range 125 {
			log.events = append(log.events, LogEvent{ID: fmt.Sprintf("ev-%03d", i+1), Topic: "ticket.created", Payload: []byte(`{"members_only":true}`), CreatedAt: at})
		}
		cursors := &memCursors{m: map[string]Cursor{}}
		repo := newFakeRepo()
		repo.rows["a1"] = &Automation{ID: "a1", Enabled: true, Subscriptions: []string{"ticket.created"}}
		var commits wake.Broadcast
		h := NewDialinHandler(nil, DialinConfig{Repo: repo, Cursors: cursors, EventLog: log, BatchSize: 50, Wake: commits.Next, Logger: testLogger()})
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)

		go h.deliverLoop(ctx, &automationConn{id: "a1"})
		synctest.Wait()

		cur, _, _ := cursors.Get(ctx, "a1")
		assert.Equal(t, "ev-125", cur.ID, "three batches go through at once, with no commit and no time passing")
	})
}
