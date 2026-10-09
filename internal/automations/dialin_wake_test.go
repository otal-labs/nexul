package automations

import (
	"context"
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
