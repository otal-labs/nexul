package plays

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type trailObserver struct {
	r           *Runner
	trail       *Trail
	targetTitle string
	resumed     bool
	ctx         context.Context // detached, so the terminal state is saved even after the turn is cancelled
	cancel      context.CancelCauseFunc

	mu         sync.Mutex
	timer      *time.Timer
	done       bool
	stopReason string
	// opened hears, once, whether a continued turn found its harness thread gone (true) or got going (false); nil otherwise.
	opened chan bool
	// before is a continued trail as it had ended, which a gone thread puts back.
	before *Trail
}

// signalOpen tells a waiting Continue how the turn opened; only the first word counts.
func (o *trailObserver) signalOpen(gone bool) {
	if o.opened == nil {
		return
	}
	select {
	case o.opened <- gone:
	default:
	}
}

func (o *trailObserver) OnStarted(sessionID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.timer.Reset(o.r.silence)
	o.trail.State = TrailRunning
	o.trail.HarnessSessionID = sessionID
	defer o.signalOpen(false)
	if o.resumed {
		o.r.save(o.ctx, o.trail)
		return
	}
	o.r.save(o.ctx, o.trail, o.r.startedEvent(o.trail, o.targetTitle))
}

// ponytail: one trail write per activity step; batch them if a busy turn ever makes the writer visible.
func (o *trailObserver) OnActivity(a harness.Activity) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.resetSilence()
	o.trail.AppendActivity(ActivityEntry(a))
	o.r.save(o.ctx, o.trail)
}

func (o *trailObserver) OnSnapshot() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.resetSilence()
}

// resetSilence restarts the silence window unless the run is waiting on the user, whose silence is not the harness's.
func (o *trailObserver) resetSilence() {
	if o.trail.State == TrailWaiting {
		return
	}
	o.timer.Reset(o.r.silence)
}

// OnQuestion parks the run: the trail waits, the silence timer stops, and the starter is told the run needs them.
func (o *trailObserver) OnQuestion(q harness.Question) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.timer.Stop()
	o.trail.State = TrailWaiting
	o.trail.Question = &TrailQuestion{Question: q, AskedAt: o.r.now().UTC()}
	o.r.save(o.ctx, o.trail, o.r.waitingEvent(o.trail, o.targetTitle))
}

// OnAnswered takes the answer given in the harness's own app to the question the trail waits on, and sets the run going again.
func (o *trailObserver) OnAnswered(a harness.AnsweredQuestion) {
	o.mu.Lock()
	defer o.mu.Unlock()
	q := o.trail.Question
	if o.done || q == nil || q.RequestID != a.RequestID || q.Answer != nil {
		return
	}
	q.Answer = &a.Answer
	o.trail.State = TrailRunning
	o.timer.Reset(o.r.silence)
	o.r.save(o.ctx, o.trail)
	o.r.recordFollowUps(o.ctx, o.trail, o.trail.StarterID, a.Answer)
}

// answer hands the answer to the live turn and sets the run going again; errTurnGone when the turn is no longer here.
func (o *trailObserver) answer(ctx context.Context, answer harness.QuestionAnswer) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done || o.trail.Question == nil {
		return errTurnGone
	}
	err := o.r.turns.Answer(ctx, o.trail.ConversationID, o.trail.Question.RequestID, answer)
	if errors.Is(err, apperrs.ErrNotFound) {
		return errTurnGone
	}
	// Answered already, in the harness or by a racing submit: the turn has its answer, so the run carries on.
	if err != nil && !errors.Is(err, apperrs.ErrConflict) {
		return fmt.Errorf("answer harness question: %w", err)
	}
	o.trail.Question.Answer = &answer
	o.trail.State = TrailRunning
	o.timer.Reset(o.r.silence)
	o.r.save(o.ctx, o.trail)
	return nil
}

func (o *trailObserver) OnFinished(result harness.TurnResult, replyMessageID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.done = true
	o.timer.Stop()
	if result.SessionGone && o.before != nil {
		o.putBack()
		return
	}
	defer o.signalOpen(false)
	if o.trail.State == TrailWaiting && o.stopReason == "" && result.State != harness.TurnInterrupted {
		// The harness closed the turn under an unanswered question; the trail keeps waiting and the answer resumes it.
		return
	}
	if o.stopReason != "" {
		// Stop was requested on this trail; a race with the harness's own terminal state never reads as done.
		result.State = harness.TurnInterrupted
		if result.LastError == "" {
			result.LastError = o.stopReason
		}
	}
	o.r.finish(o.ctx, o.trail, o.targetTitle, result, replyMessageID, "")
}

// The lines a continued trail ends on when its harness thread was deleted, once the play did or didn't start again (ADR 0128).
const (
	threadGoneNote             = "This run's thread is gone from T3 Code; the play started again in a new run."
	threadGoneNotStartedNote   = "This run's thread is gone from T3 Code, and the play didn't start again: pick where it runs."
	threadGoneNotStartedPrefix = "This run's thread is gone from T3 Code, and the play didn't start again: "
)

// putBack returns a continued trail to how it had ended and tells Continue to run again, which notes how that went.
func (o *trailObserver) putBack() {
	o.trail.State, o.trail.EndedAt, o.trail.LastError = o.before.State, o.before.EndedAt, o.before.LastError
	o.r.save(o.ctx, o.trail)
	o.signalOpen(true)
}

// onSilence fails the run when the window elapses with nothing from the harness and stops its turn there; done makes the pipeline's later terminal a no-op.
func (o *trailObserver) onSilence() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return
	}
	o.done = true
	reason := "no harness update for " + formatDuration(o.r.silence)
	o.r.finish(o.ctx, o.trail, o.targetTitle, harness.TurnResult{State: harness.TurnError, LastError: reason}, "", "Run failed: "+reason)
	ctx, cancel := context.WithTimeout(o.ctx, silenceInterruptTimeout)
	defer cancel()
	if err := o.r.turns.Interrupt(ctx, o.trail.ConversationID); err != nil {
		o.r.log.Warn("plays: harness interrupt after silence failed", "trail", o.trail.ID, "error", err)
	}
	o.cancel(errors.New(reason))
}

// stop asks the harness to interrupt; when it cannot, the run is closed here and the turn's context cancelled.
func (o *trailObserver) stop(ctx context.Context, reason string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.done {
		return fmt.Errorf("%w: the run has already ended", apperrs.ErrConflict)
	}
	o.stopReason = reason
	o.r.note(ctx, o.trail, "Run "+reason+".")
	if err := o.r.turns.Interrupt(ctx, o.trail.ConversationID); err != nil {
		o.r.log.Warn("plays: harness interrupt failed, closing the trail", "trail", o.trail.ID, "error", err)
		o.done = true
		o.timer.Stop()
		o.r.finish(o.ctx, o.trail, o.targetTitle, harness.TurnResult{State: harness.TurnInterrupted, LastError: reason}, "", "")
		o.cancel(errors.New(reason))
	}
	return nil
}

// Stop interrupts an active run; the starter or a plays:write holder may, and the trail keeps its record.
func (r *Runner) Stop(ctx context.Context, trailID string) (*Trail, error) {
	trailID = strings.TrimSpace(trailID)
	if trailID == "" {
		return nil, fmt.Errorf("%w: trail id is required", apperrs.ErrInvalid)
	}
	trail, err := r.trails.GetTrail(ctx, trailID)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", trailID, err)
	}
	actor := actorID(ctx)
	if actor != trail.StarterID && !r.perm.HasPermission(ctx, actor, trail.WorkspaceID, permissions.PlaysWrite, "", "") {
		return nil, fmt.Errorf("%w: only the starter or a %s holder can stop a run", apperrs.ErrForbidden, permissions.PlaysWrite)
	}
	if !trail.State.Active() {
		return nil, fmt.Errorf("%w: the run has already ended", apperrs.ErrConflict)
	}
	reason := "stopped by " + r.login(ctx, actor)
	if o := r.run(trailID); o != nil {
		if err := o.stop(ctx, reason); err != nil {
			return nil, err
		}
		return r.trails.GetTrail(ctx, trailID)
	}
	// No live turn on this server (a restart mid-run): closing the record here frees the target again.
	tgt, err := r.readTarget(ctx, trail.TargetType, trail.TargetID)
	if err != nil {
		r.log.Warn("plays: stop could not read the target", "trail", trailID, "error", err)
	}
	r.finish(ctx, trail, tgt.title, harness.TurnResult{State: harness.TurnInterrupted, LastError: reason}, "", "Run "+reason+".")
	return trail, nil
}

// ResumeRunsAfterRestart follows again the runs whose turns only the previous process watched; call it at boot, before any run starts.
func (r *Runner) ResumeRunsAfterRestart(ctx context.Context) error {
	trails, err := r.trails.ListRunningTrails(ctx)
	if err != nil {
		return fmt.Errorf("list running trails: %w", err)
	}
	for _, trail := range trails {
		// Boot has no signed-in actor and the target reads check permissions, so read as the run's starter.
		tgt, err := r.readTarget(identity.WithActor(ctx, identity.Actor{ID: trail.StarterID}), trail.TargetType, trail.TargetID)
		if err != nil {
			r.log.Warn("plays: restart could not read the run's target", "trail", trail.ID, "error", err)
		}
		// A run the harness never accepted has no thread to follow.
		if trail.HarnessSessionID == "" || trail.ConversationID == "" {
			const reason = "Nexul restarted before the run started"
			r.finish(ctx, trail, tgt.title, harness.TurnResult{State: harness.TurnInterrupted, LastError: reason}, "", reason+".")
			continue
		}
		r.note(ctx, trail, "Nexul restarted; following the run in T3 Code again.")
		r.save(ctx, trail)
		r.startTurn(ctx, trail, tgt.title, agent.TurnRequest{
			ConversationID: trail.ConversationID, ViaUserID: trail.StarterID, Watch: true,
			Target: trailTarget(trail),
		}, true)
	}
	return nil
}

// finish is the run's terminal step; note is the runner's own reason for the thread, empty when the pipeline already posted one.
