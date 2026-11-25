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

// passStatements is what one delivery pass over an empty log runs: the automation, its cursor, the settled position
// and the log.
const passStatements = 4

// statements counts every read the delivery pass makes, one statement each.
type statements struct {
	mu sync.Mutex
	n  int
}

func (s *statements) add() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
}

func (s *statements) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// countingLog is an event log with nothing in it.
type countingLog struct{ *statements }

func (c countingLog) After(context.Context, []string, Cursor, int) ([]LogEvent, error) {
	c.add()
	return nil, nil
}

func (c countingLog) Latest(context.Context) (Cursor, bool, error) {
	c.add()
	return Cursor{}, false, nil
}

func (c countingLog) Settled(context.Context) (Cursor, bool, error) {
	c.add()
	return Cursor{}, false, nil
}

type countingCursors struct{ *statements }

func (c countingCursors) Get(context.Context, string) (Cursor, bool, error) {
	c.add()
	return Cursor{}, true, nil
}

func (c countingCursors) Set(context.Context, string, Cursor) error {
	c.add()
	return nil
}

type countingRepo struct {
	Repo
	*statements
}

func (r countingRepo) Get(ctx context.Context, id string) (*Automation, error) {
	r.add()
	return r.Repo.Get(ctx, id)
}

// startDeliverLoops runs one delivery loop per automation id over a shared empty log and commit broadcast, and counts
// the statements they run.
func startDeliverLoops(t *testing.T, commits *wake.Broadcast, automationIDs ...string) *statements {
	t.Helper()
	repo := newFakeRepo()
	for _, id := range automationIDs {
		repo.rows[id] = &Automation{ID: id, Enabled: true, Subscriptions: []string{"ticket.created"}}
	}
	st := &statements{}
	h := NewDialinHandler(nil, DialinConfig{
		Repo: countingRepo{repo, st}, Cursors: countingCursors{st}, EventLog: countingLog{st}, Wake: commits.Next, Logger: testLogger(),
	})
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	for _, id := range automationIDs {
		go h.deliverLoop(ctx, &automationConn{id: id})
	}
	synctest.Wait()
	return st
}

func TestDeliverLoop_ACommitWakesEveryConnectionBeforeTheNextPoll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var commits wake.Broadcast
		st := startDeliverLoops(t, &commits, "a1", "a2", "a3")
		atStart := st.count()

		commits.Notify()
		synctest.Wait()

		assert.Equal(t, atStart+3*passStatements, st.count(), "each of the three connections runs one pass, with no time passing")
	})
}

func TestDeliverLoop_TenIdleMinutesRunAHandfulOfStatements(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var commits wake.Broadcast
		st := startDeliverLoops(t, &commits, "a1")

		time.Sleep(10 * time.Minute)
		synctest.Wait()

		assert.Greater(t, st.count(), passStatements, "the safety net still reads an idle connection")
		assert.LessOrEqual(t, st.count(), 3*passStatements, "an idle connection runs a pass at connect and every few minutes, not every few seconds")
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
