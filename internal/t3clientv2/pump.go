package t3clientv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
)

// source is one subscription's items; *t3rpc.Stream is the real one.
type source interface {
	Next(ctx context.Context) ([]json.RawMessage, error)
	Close()
}

const (
	// noteEvery repeats the standing note, which keeps the callers' silence windows open while the turn waits on T3.
	noteEvery = 5 * time.Minute
	// handoffCap is how long a turn whose run is done waits for the work it handed off.
	handoffCap = 60 * time.Minute
	// updatedNote ends a turn whose resubscribe finds T3 Code on another protocol.
	updatedNote = "T3 Code was updated during this turn; ask again"
)

// resubscribeBackoff is the wait before each resubscribe in a row; its length is how many are tried.
var resubscribeBackoff = []time.Duration{time.Second, 5 * time.Second, 25 * time.Second}

// pump feeds a turn's stream through its watch until the watch ends the turn, resubscribing when the stream ends.
type pump struct {
	w       *watch
	open    func(ctx context.Context, after int64) (source, error)
	decline func(ctx context.Context, requestID string) error
	log     *slog.Logger
	live    *runningTurn
	// failures counts resubscribes since the stream last delivered anything.
	failures int
	// note is the standing note last shown, due when it is shown again.
	note string
	due  time.Time
	// capAt is when the turn stops waiting for handed-off work, zero until its own run is done.
	capAt time.Time
}

func (p *pump) run(ctx context.Context, src source, out chan<- harness.Update) {
	defer close(out)
	for src != nil {
		done, err := p.drain(ctx, src, out)
		src.Close()
		if done {
			return
		}
		p.log.Warn("t3clientv2: thread stream ended mid-turn", "after_sequence", p.w.cursor, "error", err)
		src = p.resubscribe(ctx, err, out)
	}
}

// drain reads src until the turn is over (true) or the stream ends (its error).
func (p *pump) drain(ctx context.Context, src source, out chan<- harness.Update) (bool, error) {
	for {
		if p.capped(ctx, out) || !p.remind(ctx, out) {
			return true, nil
		}
		values, err := p.next(p.live.halted, src)
		if ctx.Err() != nil {
			return true, nil
		}
		if p.live.halted.Err() != nil {
			p.finish(ctx, out, p.w.stop())
			return true, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		if err != nil {
			return false, err
		}
		p.failures = 0
		if p.fold(ctx, values, out) {
			return true, nil
		}
	}
}

// next waits for the stream's next items, but only until the standing note is due again or the cap is reached.
func (p *pump) next(ctx context.Context, src source) ([]json.RawMessage, error) {
	deadline := p.due
	if !p.capAt.IsZero() && (deadline.IsZero() || p.capAt.Before(deadline)) {
		deadline = p.capAt
	}
	if deadline.IsZero() {
		return src.Next(ctx)
	}
	waitCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	return src.Next(waitCtx)
}

// fold applies items to the watch and forwards what it emits; true once the turn is over.
func (p *pump) fold(ctx context.Context, values []json.RawMessage, out chan<- harness.Update) bool {
	for _, raw := range values {
		updates, end := p.w.apply(decodeItem(p.log, raw))
		p.declineApprovals(ctx)
		for _, u := range updates {
			if !send(ctx, out, u) {
				return true
			}
		}
		if end != nil {
			p.finish(ctx, out, end)
			return true
		}
	}
	p.live.publish(p.w.handoffs())
	return false
}

// finish ends the turn with end, after the notes of what Stop could not stop.
func (p *pump) finish(ctx context.Context, out chan<- harness.Update, end *harness.TurnResult) {
	for _, summary := range p.live.takeNotes() {
		if !send(ctx, out, note(summary)) {
			return
		}
	}
	send(ctx, out, harness.Update{Terminal: end})
}

// declineApprovals refuses the approvals the watch just raised before they are forwarded, as protocol 1's client does.
func (p *pump) declineApprovals(ctx context.Context) {
	for _, id := range p.w.declines {
		if err := p.decline(ctx, id); err != nil {
			p.log.Warn("t3clientv2: auto-decline failed", "request", id, "error", err)
		}
	}
	p.w.declines = nil
}

// remind shows the standing note when it changes and again every noteEvery while it stands; false once ctx ends.
func (p *pump) remind(ctx context.Context, out chan<- harness.Update) bool {
	note := p.w.standingNote()
	if note == nil {
		p.note, p.due = "", time.Time{}
		return true
	}
	now := time.Now()
	if note.Summary == p.note && now.Before(p.due) {
		return true
	}
	p.note, p.due = note.Summary, now.Add(noteEvery)
	note.At = now.UTC()
	return send(ctx, out, harness.Update{Activity: note})
}

// capped ends the turn done handoffCap after its own run finished, while work it handed off still runs; true once it did.
func (p *pump) capped(ctx context.Context, out chan<- harness.Update) bool {
	if !p.w.waiting() {
		return false
	}
	if p.capAt.IsZero() {
		p.capAt = time.Now().Add(handoffCap)
	}
	if time.Now().Before(p.capAt) {
		return false
	}
	p.finish(ctx, out, p.w.leave())
	return true
}

// resubscribe opens the stream again after the cursor, ending the turn when it cannot.
func (p *pump) resubscribe(ctx context.Context, cause error, out chan<- harness.Update) source {
	for p.failures < len(resubscribeBackoff) {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(resubscribeBackoff[p.failures]):
		}
		p.failures++
		src, err := p.open(ctx, p.w.cursor)
		if errors.Is(err, harness.ErrProtocol) {
			send(ctx, out, harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: updatedNote}})
			return nil
		}
		if err == nil {
			return src
		}
		p.log.Warn("t3clientv2: resubscribe failed", "attempt", p.failures, "error", err)
		cause = err
	}
	send(ctx, out, harness.Update{Terminal: &harness.TurnResult{State: harness.TurnError, LastError: fmt.Sprintf(
		"Lost the connection to T3 Code and couldn't resume the turn after %d tries: %v", p.failures, cause)}})
	return nil
}

func send(ctx context.Context, out chan<- harness.Update, u harness.Update) bool {
	select {
	case out <- u:
		return true
	case <-ctx.Done():
		return false
	}
}
