package t3clientv2

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

const subscribeShell = "orchestration.subscribeShell"

// workingActivity are the shell's activityRunStatus values of a run still doing work; waiting is a reply T3 has.
var workingActivity = []string{"preparing", "starting", "running"}

// shellItem is one orchestration.subscribeShell item, read only for its threads.
type shellItem struct {
	Kind     string `json:"kind"`
	Snapshot struct {
		Threads []shellThread `json:"threads"`
	} `json:"snapshot"`
	Thread   shellThread `json:"thread"`
	ThreadID string      `json:"threadId"`
}

// shellThread is the slice of T3's sidebar row for a thread that says whether it has news.
type shellThread struct {
	ID                string  `json:"id"`
	LatestRunID       *string `json:"latestRunId"`
	ActivityRunStatus *string `json:"activityRunStatus"`
	DeletedAt         *string `json:"deletedAt"`
	// Status is the latest run's status, "idle" without one.
	Status             string  `json:"status"`
	LatestRunStartedAt *string `json:"latestRunStartedAt"`
}

func (t shellThread) update() harness.SessionUpdate {
	u := harness.SessionUpdate{SessionID: t.ID, Gone: t.DeletedAt != nil}
	if t.LatestRunID != nil && t.ran() {
		u.Latest = *t.LatestRunID
	}
	if t.ActivityRunStatus != nil {
		u.Working = slices.Contains(workingActivity, *t.ActivityRunStatus)
	}
	return u
}

// ran is T3's latestExecutedRun rule for the latest run: a queued one, or one cancelled before it started, never ran.
func (t shellThread) ran() bool {
	return t.Status != runQueued && (t.Status != "cancelled" || t.LatestRunStartedAt != nil)
}

// WatchSessions implements harness.SessionWatcher on the shell stream T3's own apps keep their sidebar on.
func (h *Harness) WatchSessions(ctx context.Context, s harness.Session, onUpdate func(harness.SessionUpdate)) (harness.Conn, error) {
	c, err := h.connect(ctx, s)
	if err != nil {
		return nil, err
	}
	stream, err := c.Stream(ctx, subscribeShell, nil)
	if err != nil {
		_ = c.Close() // the subscribe error is the one worth reporting
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	w := &sessionsConn{conn: c, cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(w.done)
		defer stream.Close()
		readShell(ctx, stream, h.log(), onUpdate)
	}()
	return w, nil
}

// readShell reports each thread's change in what Nexul follows it by, until the stream or ctx ends.
func readShell(ctx context.Context, stream *t3rpc.Stream, log *slog.Logger, onUpdate func(harness.SessionUpdate)) {
	last := map[string]harness.SessionUpdate{}
	report := func(u harness.SessionUpdate) {
		if u.SessionID == "" || last[u.SessionID] == u {
			return
		}
		last[u.SessionID] = u
		onUpdate(u)
	}
	for {
		values, err := stream.Next(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Warn("t3clientv2: shell stream ended", "error", err)
			}
			return
		}
		for _, raw := range values {
			var item shellItem
			if json.Unmarshal(raw, &item) != nil {
				continue
			}
			for _, u := range shellUpdates(item) {
				report(u)
			}
		}
	}
}

// shellUpdates are the thread changes one shell item carries.
func shellUpdates(item shellItem) []harness.SessionUpdate {
	switch item.Kind {
	case "snapshot":
		out := make([]harness.SessionUpdate, 0, len(item.Snapshot.Threads))
		for _, t := range item.Snapshot.Threads {
			out = append(out, t.update())
		}
		return out
	case "thread.updated":
		return []harness.SessionUpdate{item.Thread.update()}
	case "thread.removed":
		return []harness.SessionUpdate{{SessionID: item.ThreadID, Gone: true}}
	}
	return nil
}

// sessionsConn is the held connection and its shell reader; Done fires once either ends.
type sessionsConn struct {
	conn   *t3rpc.Conn
	cancel context.CancelFunc
	done   chan struct{}
}

func (w *sessionsConn) Done() <-chan struct{} { return w.done }

func (w *sessionsConn) Close() error {
	w.cancel()
	return w.conn.Close()
}

var _ harness.SessionWatcher = (*Harness)(nil)
