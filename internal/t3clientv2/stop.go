package t3clientv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

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

// disposeCommand drops a delegated task's undelivered result, so the task's end cannot wake its parent thread.
type disposeCommand struct {
	Type           string `json:"type"`
	CommandID      string `json:"commandId"`
	ParentThreadID string `json:"parentThreadId"`
	TaskID         string `json:"taskId"`
}

// settledDeliveries are the states of a delegated task's result that can no longer wake its parent.
var settledDeliveries = []string{"acknowledged", "delivered", "disposed"}

// live is a turn as Stop sees it from its own connection, while the turn's pump owns the watch.
type live struct {
	messageID string
	// halted ends the pump's wait on T3 once Stop has stopped the turn.
	halted context.Context
	halt   context.CancelFunc
	mu     sync.Mutex
	work   handoffs
	// notes are what Stop could not stop, for the turn to show before it ends.
	notes []string
}

// handoffs is what a turn follows besides its own run: the runs its hand-offs woke, and every followed run's subagents.
type handoffs struct {
	runs      []string
	subagents []subagent
}

func newLive(ctx context.Context, messageID string) *live {
	halted, halt := context.WithCancel(ctx)
	return &live{messageID: messageID, halted: halted, halt: halt}
}

func (l *live) publish(work handoffs) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.work = work
}

func (l *live) handoffs() handoffs {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.work
}

func (l *live) note(summary string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.notes = append(l.notes, summary)
}

func (l *live) takeNotes() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	notes := l.notes
	l.notes = nil
	return notes
}

// Interrupt implements harness.Client: it stops the watching turn's handed-off work, then its run, and ends it interrupted.
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
	found, _ := h.turns.Load(target.SessionID)
	l, ok := found.(*live)
	if !ok {
		return stopRun(ctx, c, target.SessionID, p.Runs, "")
	}
	handedOff := l.stopHandoffs(ctx, c, target.SessionID, p.Runs, h.log())
	err = stopRun(ctx, c, target.SessionID, p.Runs, l.messageID)
	if errors.Is(err, errNothingToStop) && handedOff {
		err = nil
	}
	if err != nil {
		return err
	}
	l.halt()
	return nil
}

// stopHandoffs drops wakeable results, then stops children's runs, then wake runs (ADR 0116), noting failures; true if any stopped.
func (l *live) stopHandoffs(ctx context.Context, c *t3rpc.Conn, threadID string, runs []run, log *slog.Logger) bool {
	work := l.handoffs()
	stopped := false
	tally := func(err error) {
		stopped = stopped || err == nil
		if err == nil || errors.Is(err, errNothingToStop) {
			return
		}
		log.Warn("t3clientv2: could not stop handed-off work", "thread", threadID, "error", err)
		l.note(fmt.Sprintf("Could not stop handed-off work in T3 Code: %v", err))
	}
	for _, s := range work.subagents {
		if s.Origin == "app_owned" && (s.CompletionDelivery == nil || !slices.Contains(settledDeliveries, s.CompletionDelivery.State)) {
			tally(dispose(ctx, c, threadID, s.ID))
		}
	}
	for _, s := range work.subagents {
		if slices.Contains(openItems, s.Status) && s.ChildThreadID != "" {
			tally(interruptChild(ctx, c, s.ChildThreadID))
		}
	}
	for _, id := range work.runs {
		if i := slices.IndexFunc(runs, func(r run) bool { return r.ID == id }); i >= 0 {
			tally(halt(ctx, c, threadID, runs[i], i == len(runs)-1))
		}
	}
	return stopped
}

func dispose(ctx context.Context, c *t3rpc.Conn, threadID, taskID string) error {
	_, err := c.Call(ctx, dispatchCommand, disposeCommand{Type: "delegated_task.completion-delivery.dispose", CommandID: ids.New(),
		ParentThreadID: threadID, TaskID: taskID})
	if err != nil {
		return refused("drop a handed-off result", err)
	}
	return nil
}

// interruptChild interrupts the newest started run of a handed-off agent's own thread.
func interruptChild(ctx context.Context, c *t3rpc.Conn, childThreadID string) error {
	p, err := readProjection(ctx, c, childThreadID)
	if err != nil {
		return err
	}
	for i := len(p.Runs) - 1; i >= 0; i-- {
		if p.Runs[i].Status != runQueued && slices.Contains(workingRuns, p.Runs[i].Status) {
			return halt(ctx, c, childThreadID, p.Runs[i], i == len(p.Runs)-1)
		}
	}
	return errNothingToStop
}

// stopRun stops the run Stop is for.
func stopRun(ctx context.Context, c *t3rpc.Conn, threadID string, runs []run, messageID string) error {
	r, ok := toStop(runs, messageID)
	if !ok {
		return errNothingToStop
	}
	return halt(ctx, c, threadID, r, r.ID == runs[len(runs)-1].ID)
}

// halt cancels a queued run or interrupts a live one; a waiting run already replied, so only the latest has background work to stop.
func halt(ctx context.Context, c *t3rpc.Conn, threadID string, r run, latest bool) error {
	if !slices.Contains(liveRuns, r.Status) || (r.Status == runWaiting && !latest) {
		return errNothingToStop
	}
	if r.Status == runQueued {
		return cancelRun(ctx, c, threadID, r.ID)
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

// toStop is the run messageID started, else the thread's newest unfinished run.
func toStop(runs []run, messageID string) (run, bool) {
	for i := len(runs) - 1; i >= 0 && messageID != ""; i-- {
		if runs[i].UserMessageID == messageID {
			return runs[i], true
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
