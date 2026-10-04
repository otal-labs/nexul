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

// runningTurn is a turn as Stop sees it from its own connection, while the turn's pump owns the watch.
type runningTurn struct {
	messageID string
	// halted ends the pump's wait on T3 once Stop has stopped the turn.
	halted context.Context
	halt   context.CancelFunc
	mu     sync.Mutex
	work   handoffs
	// notes are what the latest Stop could not stop, for the turn to show before it ends.
	notes []string
}

// handoffs is what a turn follows besides its own run: the runs its hand-offs woke, and every followed run's subagents.
type handoffs struct {
	runs      []string
	subagents []subagent
}

func newRunningTurn(ctx context.Context, messageID string) *runningTurn {
	halted, halt := context.WithCancel(ctx)
	return &runningTurn{messageID: messageID, halted: halted, halt: halt}
}

func (l *runningTurn) publish(work handoffs) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.work = work
}

func (l *runningTurn) handoffs() handoffs {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.work
}

func (l *runningTurn) setNotes(notes []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.notes = notes
}

func (l *runningTurn) takeNotes() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	notes := l.notes
	l.notes = nil
	return notes
}

// Interrupt implements harness.Client: it stops the named turn's handed-off work, then its run, and ends it interrupted.
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
	found, _ := h.turns.Load(turnKey{target.SessionID, target.TurnID})
	l, ok := found.(*runningTurn)
	if !ok {
		return stopRun(ctx, c, target.SessionID, p.Runs, "")
	}
	runs, handedOff, err := l.stopHandoffs(ctx, c, target.SessionID, p.Runs, h.log())
	if err != nil {
		return err
	}
	err = stopRun(ctx, c, target.SessionID, runs, l.messageID)
	if errors.Is(err, errNothingToStop) {
		err = handedOff
	}
	if err != nil {
		return err
	}
	l.halt()
	return nil
}

// stopHandoffs stops handed-off work in ADR 0116's order; it returns the runs as they then stand and the steps' outcome.
func (l *runningTurn) stopHandoffs(ctx context.Context, c *t3rpc.Conn, threadID string, runs []run, log *slog.Logger) (_ []run, handedOff, err error) {
	work := l.handoffs()
	steps := &tally{log: log, threadID: threadID}
	defer func() { l.setNotes(steps.notes) }()
	for _, s := range work.subagents {
		if s.Origin == "app_owned" && (s.CompletionDelivery == nil || !slices.Contains(settledDeliveries, s.CompletionDelivery.State)) {
			steps.count(dispose(ctx, c, threadID, s.ID))
		}
	}
	for _, s := range work.subagents {
		if slices.Contains(openItems, s.Status) && s.ChildThreadID != "" {
			steps.count(interruptChild(ctx, c, s.ChildThreadID))
		}
	}
	if len(work.subagents) > 0 {
		// Dropping the last result of a queued wake cancels that wake in T3, which then refuses to cancel it again.
		p, err := readProjection(ctx, c, threadID)
		if err != nil {
			return nil, steps.outcome(), err
		}
		runs = p.Runs
	}
	for _, id := range work.runs {
		if i := slices.IndexFunc(runs, func(r run) bool { return r.ID == id }); i >= 0 {
			steps.count(halt(ctx, c, threadID, runs[i], i == len(runs)-1))
		}
	}
	return runs, steps.outcome(), nil
}

// tally is what Stop's hand-off steps came to: whether any stopped something, the first failure, a note per failure.
type tally struct {
	log      *slog.Logger
	threadID string
	stopped  bool
	first    error
	notes    []string
}

func (t *tally) count(err error) {
	t.stopped = t.stopped || err == nil
	if err == nil || errors.Is(err, errNothingToStop) {
		return
	}
	t.log.Warn("t3clientv2: could not stop handed-off work", "thread", t.threadID, "error", err)
	if t.first == nil {
		t.first = err
	}
	t.notes = append(t.notes, "Could not stop handed-off work in T3 Code: "+strings.TrimPrefix(err.Error(), apperrs.ErrInvalid.Error()+": "))
}

// outcome is nil once a step stopped something, else the first failure, else errNothingToStop.
func (t *tally) outcome() error {
	if t.stopped {
		return nil
	}
	if t.first != nil {
		return t.first
	}
	return errNothingToStop
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

// movedOn is T3 refusing to stop a run that started or replied since Stop read it.
type movedOn struct{ error }

func (m movedOn) Unwrap() error { return m.error }

// stopRun stops the run Stop is for, reading the thread once more when that run moved on before the command landed.
func stopRun(ctx context.Context, c *t3rpc.Conn, threadID string, runs []run, messageID string) error {
	err := stopRunIn(ctx, c, threadID, runs, messageID)
	if !errors.As(err, new(movedOn)) {
		return err
	}
	p, err := readProjection(ctx, c, threadID)
	if err != nil {
		return err
	}
	return stopRunIn(ctx, c, threadID, p.Runs, messageID)
}

func stopRunIn(ctx context.Context, c *t3rpc.Conn, threadID string, runs []run, messageID string) error {
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
	if strings.HasSuffix(t3Message(err), " is not interruptible.") {
		if r.Status == runWaiting {
			return nil
		}
		return movedOn{refused("stop the T3 run", err)}
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
	if strings.HasSuffix(t3Message(err), " is not queued.") {
		return movedOn{refused("cancel the queued T3 run", err)}
	}
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
