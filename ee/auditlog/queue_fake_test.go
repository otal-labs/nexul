package auditlog

import (
	"context"
	"slices"
	"sync"
	"time"
)

type queued struct {
	Pending
	status    string
	next      time.Time
	delivered time.Time
	reason    string
}

// fakeQueue is the stream's queue in memory, with the storage queue's rules: pending rows due by now, oldest first.
type fakeQueue struct {
	mu   sync.Mutex
	rows map[string]*queued
	// failDelivered makes the next MarkDelivered fail, as a crash after the receiver's 2xx would.
	failDelivered error
}

func newFakeQueue() *fakeQueue { return &fakeQueue{rows: map[string]*queued{}} }

func (q *fakeQueue) add(p Pending) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.rows[p.ID] = &queued{Pending: p, status: "pending", next: p.CreatedAt}
}

func (q *fakeQueue) row(id string) queued {
	q.mu.Lock()
	defer q.mu.Unlock()
	return *q.rows[id]
}

func (q *fakeQueue) Due(_ context.Context, limit int, now time.Time) ([]Pending, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []Pending
	for _, r := range q.rows {
		if r.status == "pending" && !r.next.After(now) {
			out = append(out, r.Pending)
		}
	}
	slices.SortFunc(out, func(a, b Pending) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out[:min(limit, len(out))], nil
}

func (q *fakeQueue) MarkDelivered(_ context.Context, id string, at time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.failDelivered; err != nil {
		q.failDelivered = nil
		return err
	}
	r := q.rows[id]
	r.status, r.delivered, r.reason = "delivered", at, ""
	return nil
}

func (q *fakeQueue) MarkFailed(_ context.Context, id string, attempts int, next time.Time, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	r := q.rows[id]
	r.Attempts, r.next, r.reason = attempts, next, reason
	return nil
}

func (q *fakeQueue) MarkDead(_ context.Context, id string, attempts int, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	r := q.rows[id]
	r.status, r.Attempts, r.reason = "dead", attempts, reason
	return nil
}

func (q *fakeQueue) RetryDead(_ context.Context, now time.Time) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for _, r := range q.rows {
		if r.status == "dead" {
			r.status, r.Attempts, r.next = "pending", 0, now
			n++
		}
	}
	return n, nil
}

func (q *fakeQueue) Stats(context.Context) (Stats, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var st Stats
	for _, r := range q.rows {
		if r.status == "pending" {
			st.Pending++
		}
		if r.status == "dead" {
			st.Dead++
		}
		if r.reason != "" {
			st.LastError = r.reason
		}
	}
	return st, nil
}
