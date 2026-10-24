package t3clientv2

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const runRunning = "running"

var (
	errNotQueued      = fmt.Errorf("%w: the message is no longer waiting in T3 Code's queue", apperrs.ErrConflict)
	errNothingToSteer = fmt.Errorf("%w: nothing is running on this T3 thread to take the message", apperrs.ErrConflict)
)

// promoteCommand is queued-message.promote-to-steer: T3 cancels the queued run and hands its message to the target run.
type promoteCommand struct {
	Type        string `json:"type"`
	CommandID   string `json:"commandId"`
	ThreadID    string `json:"threadId"`
	QueuedRunID string `json:"queuedRunId"`
	TargetRunID string `json:"targetRunId"`
}

// Steer implements harness.Client: the named turn's queued run joins the thread's running run, and the turn ends done.
func (h *Harness) Steer(ctx context.Context, target harness.Target) (err error) {
	if target.SessionID == "" || target.TurnID == "" {
		return fmt.Errorf("%w: no queued turn to steer", apperrs.ErrInvalid)
	}
	found, _ := h.turns.Load(turnKey{target.SessionID, target.TurnID})
	l, ok := found.(*runningTurn)
	if !ok {
		return errNotQueued
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
	queued := slices.IndexFunc(p.Runs, func(r run) bool { return r.UserMessageID == target.TurnID && r.Status == runQueued })
	if queued < 0 {
		return errNotQueued
	}
	running := slices.IndexFunc(p.Runs, func(r run) bool { return r.Status == runRunning })
	if running < 0 {
		return errNothingToSteer
	}
	// Marked first: T3 cancels the queued run within the call, and the turn's stream may see that before it returns.
	l.steered.Store(true)
	_, err = c.Call(ctx, dispatchCommand, promoteCommand{Type: "queued-message.promote-to-steer", CommandID: ids.New(),
		ThreadID: target.SessionID, QueuedRunID: p.Runs[queued].ID, TargetRunID: p.Runs[running].ID})
	if err != nil {
		l.steered.Store(false)
		return refused("steer the queued message into the running T3 run", err)
	}
	return nil
}
