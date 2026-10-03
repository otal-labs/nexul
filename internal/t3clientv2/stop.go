package t3clientv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/t3rpc"
)

const (
	getThreadProjection = "orchestration.getThreadProjection"
	stopReason          = "Stopped from Nexul"
)

var errNothingToStop = fmt.Errorf("%w: nothing is running on this T3 thread", apperrs.ErrConflict)

// runCommand is run.interrupt or queued-run.cancel; holdQueue is never sent, since a held queue waits for queue.resume.
type runCommand struct {
	Type      string `json:"type"`
	CommandID string `json:"commandId"`
	ThreadID  string `json:"threadId"`
	RunID     string `json:"runId"`
	Reason    string `json:"reason,omitempty"`
}

// Interrupt implements harness.Client: it stops the run of the turn watching the thread, else its newest unfinished run.
func (h *Harness) Interrupt(ctx context.Context, target harness.Target) (err error) {
	if target.SessionID == "" {
		return fmt.Errorf("%w: no active session to interrupt", apperrs.ErrInvalid)
	}
	c, err := h.connect(ctx, target.Session)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	p, err := readProjection(ctx, c, target.SessionID)
	if err != nil {
		return err
	}
	messageID, _ := h.messages.Load(target.SessionID)
	id, _ := messageID.(string)
	return stopRun(ctx, c, target.SessionID, p.Runs, id)
}

// stopRun stops the run Stop is for; a waiting run already replied, so only the latest one has background work to stop.
func stopRun(ctx context.Context, c *t3rpc.Conn, threadID string, runs []run, messageID string) error {
	r, ok := toStop(runs, messageID)
	if !ok {
		return errNothingToStop
	}
	if r.Status == runQueued {
		return cancelRun(ctx, c, threadID, r.ID)
	}
	if r.Status == runWaiting && r.ID != runs[len(runs)-1].ID {
		return errNothingToStop
	}
	_, err := c.Call(ctx, dispatchCommand, runCommand{Type: "run.interrupt", CommandID: ids.New(), ThreadID: threadID, RunID: r.ID, Reason: stopReason})
	if r.Status == runWaiting && strings.HasSuffix(t3Message(err), " is not interruptible.") {
		return nil
	}
	if err != nil {
		return refused("stop the T3 run", err)
	}
	return nil
}

// toStop is the run messageID started, else the thread's newest unfinished run; ok is false when that run is over.
func toStop(runs []run, messageID string) (run, bool) {
	for i := len(runs) - 1; i >= 0 && messageID != ""; i-- {
		if runs[i].UserMessageID == messageID {
			return runs[i], slices.Contains(liveRuns, runs[i].Status)
		}
	}
	for i := len(runs) - 1; i >= 0; i-- {
		if slices.Contains(liveRuns, runs[i].Status) {
			return runs[i], true
		}
	}
	return run{}, false
}

func cancelRun(ctx context.Context, c *t3rpc.Conn, threadID, runID string) error {
	_, err := c.Call(ctx, dispatchCommand, runCommand{Type: "queued-run.cancel", CommandID: ids.New(), ThreadID: threadID, RunID: runID})
	if err != nil {
		return refused("cancel the queued T3 run", err)
	}
	return nil
}

// readProjection reads the thread's runs and open requests; every queued or live run is in it, however long the thread.
func readProjection(ctx context.Context, c *t3rpc.Conn, threadID string) (projection, error) {
	raw, err := c.Call(ctx, getThreadProjection, map[string]string{"threadId": threadID})
	if err != nil {
		return projection{}, refused("read the T3 thread", err)
	}
	var p projection
	if err := json.Unmarshal(raw, &p); err != nil {
		return projection{}, fmt.Errorf("decode the T3 thread %s: %w", threadID, err)
	}
	return p, nil
}
