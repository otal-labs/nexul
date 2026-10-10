package plays

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// RefusalOffline is the harness refusal for a computer that did not answer; a queued run waits on it rather than failing.
const RefusalOffline = "offline"

const (
	// autoRunsPerPerson is how many runs a person's queue lets go at once; every active trail of theirs takes a slot.
	autoRunsPerPerson = 1
	// offlineRetry is when an item whose computer did not answer is tried again; nothing announces a computer coming back.
	offlineRetry = time.Minute
	// dispatchTimeout bounds one start, so a hung tunnel cannot stall everyone's queue.
	dispatchTimeout = 30 * time.Second
	// queueRetry is how soon a failed pass is tried again.
	queueRetry = 5 * time.Second
	// autoRunWindow is the rolling day the daily cap counts automatic runs over.
	autoRunWindow = 24 * time.Hour
)

// Kick wakes the dispatcher for another pass; it never blocks, and kicks during a pass fold into one more pass.
func (r *Runner) Kick() {
	select {
	case r.kick <- struct{}{}:
	default:
	}
}

// HandleRunFinished is registered on play.run_finished: a run ending frees its person's slot and its target.
func (r *Runner) HandleRunFinished(context.Context, eventbus.Event) error {
	r.Kick()
	return nil
}

// RunQueue starts queued auto runs until ctx ends. It passes once at boot, then on every kick, and on a timer to the
// next held item (an offline computer's retry, a paused target's oldest run leaving the day); it never polls.
func (r *Runner) RunQueue(ctx context.Context) {
	if r.queue == nil || r.facts == nil {
		return
	}
	if err := r.queue.RecoverDispatching(ctx); err != nil {
		r.log.Error("plays: recover the run queue", "error", err)
	}
	for {
		wait, err := r.dispatch(ctx)
		if err != nil {
			r.log.Error("plays: dispatch the run queue", "error", err)
			wait = queueRetry
		}
		if !r.waitForKick(ctx, wait) {
			return
		}
	}
}

// waitForKick waits out wait (forever when negative) or a kick, and reports false once ctx ends.
func (r *Runner) waitForKick(ctx context.Context, wait time.Duration) bool {
	var timeout <-chan time.Time
	if wait >= 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		timeout = timer.C
	}
	select {
	case <-ctx.Done():
		return false
	case <-timeout:
	case <-r.kick:
	}
	return true
}

// queuePass is one pass's clock and the earliest time an item it held back may go.
type queuePass struct {
	now  time.Time
	next time.Time
}

func (p *queuePass) wakeAt(t time.Time) {
	if t.After(p.now) && (p.next.IsZero() || t.Before(p.next)) {
		p.next = t
	}
}

// dispatch makes one pass over every person with a due item and returns how long to sleep, -1 for until kicked.
func (r *Runner) dispatch(ctx context.Context) (time.Duration, error) {
	pass := &queuePass{now: r.now().UTC()}
	people, err := r.queue.QueuedPeople(ctx, pass.now)
	if err != nil {
		return 0, fmt.Errorf("list people with queued runs: %w", err)
	}
	for _, person := range people {
		if err := r.dispatchPerson(ctx, pass, person); err != nil {
			return 0, err
		}
	}
	next, ok, err := r.queue.NextNotBefore(ctx, pass.now)
	if err != nil {
		return 0, fmt.Errorf("read the next held run: %w", err)
	}
	if ok {
		pass.wakeAt(next)
	}
	if pass.next.IsZero() {
		return -1, nil
	}
	return max(pass.next.Sub(r.now().UTC()), 0), nil
}

// dispatchPerson starts a person's due items, highest priority then oldest first, while they have a free slot.
func (r *Runner) dispatchPerson(ctx context.Context, pass *queuePass, person string) error {
	active, err := r.queue.CountActiveRuns(ctx, person)
	if err != nil {
		return fmt.Errorf("count the runs of %s: %w", person, err)
	}
	free := autoRunsPerPerson - active
	if free <= 0 {
		return nil
	}
	items, err := r.queue.ListDue(ctx, person, pass.now)
	if err != nil {
		return fmt.Errorf("list the queue of %s: %w", person, err)
	}
	for _, it := range items {
		if free == 0 {
			return nil
		}
		started, offline := r.dispatchItem(ctx, pass, it)
		if offline {
			return nil
		}
		if started {
			free--
		}
	}
	return nil
}

// dispatchItem starts it, holds it, or decides it; offline says the person's computer did not answer, so their pass stops.
func (r *Runner) dispatchItem(ctx context.Context, pass *queuePass, it *QueueItem) (started, offline bool) {
	trails, err := r.trails.ListTrailsByTarget(ctx, it.TargetType, it.TargetID)
	if err != nil {
		r.log.Error("plays: read the trails of a queued run's target", "item", it.ID, "error", err)
		return false, false
	}
	if slices.ContainsFunc(trails, func(t *Trail) bool { return t.State.Active() }) {
		r.hold(ctx, it, ReasonTicketBusy)
		return false, false
	}
	count, limit, frees, err := r.autoRunsToday(ctx, it.WorkspaceID, it.TargetType, it.TargetID)
	if err != nil {
		r.log.Error("plays: count a queued run's target's automatic runs", "item", it.ID, "error", err)
		return false, false
	}
	if count >= limit {
		pass.wakeAt(frees)
		r.hold(ctx, it, ReasonPaused)
		return false, false
	}
	play, tgt, v, err := r.recheck(ctx, it, trails)
	if err != nil {
		r.log.Error("plays: check a queued run again", "item", it.ID, "error", err)
		return false, false
	}
	if v.status == QueueDidntRun {
		r.failedTrail(ctx, it, v.reason)
	}
	if v.status != QueueQueued {
		r.logDecide(ctx, it, QueueQueued, v.status, v.reason)
		return false, false
	}
	return r.startQueued(ctx, it, play, tgt, pass.now)
}

// hold keeps it queued with reason, writing only when the reason changed so a pass over a busy ticket costs no write.
func (r *Runner) hold(ctx context.Context, it *QueueItem, reason string) {
	if it.Reason == reason {
		return
	}
	r.logDecide(ctx, it, QueueQueued, QueueQueued, reason)
}

func (r *Runner) logDecide(ctx context.Context, it *QueueItem, from, to QueueStatus, reason string) {
	if err := r.decide(ctx, it, from, to, reason); err != nil && !errors.Is(err, apperrs.ErrConflict) {
		r.log.Error("plays: move a queued run", "item", it.ID, "to", to, "error", err)
	}
}

// startQueued launches it on its person's computer under a trail id written onto the row first, so a crash mid-start
// leaves no second run behind.
func (r *Runner) startQueued(ctx context.Context, it *QueueItem, play *Play, tgt target, now time.Time) (started, offline bool) {
	held := it.Reason
	it.TrailID = ids.New()
	if err := r.move(ctx, it, QueueQueued, QueueDispatching, "", false); err != nil {
		it.TrailID = ""
		if !errors.Is(err, apperrs.ErrConflict) {
			r.log.Error("plays: mark a queued run dispatching", "item", it.ID, "error", err)
		}
		return false, false
	}
	runCtx, cancel := context.WithTimeout(identity.WithActor(ctx, identity.Actor{ID: it.PersonID}), dispatchTimeout)
	defer cancel()
	trail := &Trail{
		ID: it.TrailID, WorkspaceID: it.WorkspaceID, PlayID: play.ID, PlayLabel: play.Label, TargetType: it.TargetType,
		TargetID: it.TargetID, ProjectID: it.ProjectID, StarterID: it.PersonID, Via: it.Via, SelectedMemoryIDs: []string{},
		State: TrailStarting, StartedAt: now,
	}
	_, err := r.launch(runCtx, play, trail, tgt, HarnessChoice{}, launchQueued)
	if err == nil {
		r.logDecide(ctx, it, QueueDispatching, QueueStarted, "")
		return true, false
	}
	var refusal *HarnessRefusal
	if errors.As(err, &refusal) && refusal.Reason == RefusalOffline {
		r.requeue(ctx, it, ReasonOffline, held, now.Add(offlineRetry))
		return false, true
	}
	if errors.Is(err, apperrs.ErrConflict) {
		r.requeue(ctx, it, ReasonTicketBusy, held, now)
		return false, false
	}
	if _, gerr := r.trails.GetTrail(ctx, it.TrailID); gerr != nil {
		it.TrailID = ""
	}
	r.logDecide(ctx, it, QueueDispatching, QueueDidntRun, err.Error())
	return false, false
}

// requeue puts a dispatching item back in its queue without the trail it never got, announcing it only when it waits
// for a new reason, so an offline computer's retry every minute stays quiet.
func (r *Runner) requeue(ctx context.Context, it *QueueItem, reason, held string, notBefore time.Time) {
	it.TrailID, it.NotBefore = "", notBefore
	if err := r.move(ctx, it, QueueDispatching, QueueQueued, reason, reason != held); err != nil {
		r.log.Error("plays: put a queued run back", "item", it.ID, "error", err)
	}
}

// verdict is what the check at the front of the queue decided: queued to go ahead, else skipped or didn't run, and why.
type verdict struct {
	status QueueStatus
	reason string
}

var goAhead = verdict{status: QueueQueued}

func skip(reason string) verdict { return verdict{status: QueueSkipped, reason: reason} }

// recheck checks a run again at the front of its queue against current data. An auto run skips the play's show-when
// stage, which only places its button; a stage limit is a "Stage is" condition.
func (r *Runner) recheck(ctx context.Context, it *QueueItem, trails []*Trail) (*Play, target, verdict, error) {
	a, play, v, err := r.recheckRecords(ctx, it)
	if err != nil || v != goAhead {
		return nil, target{}, v, err
	}
	f, err := r.facts.Facts(ctx, it.TargetType, it.TargetID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, target{}, skip(fmt.Sprintf("the %s was deleted", it.TargetType)), nil
	}
	if err != nil {
		return nil, target{}, goAhead, fmt.Errorf("read %s %s: %w", it.TargetType, it.TargetID, err)
	}
	if err := r.checkPlay(ctx, it.PersonID, play, f.ProjectID); err != nil {
		if errors.Is(err, apperrs.ErrInvalid) {
			return nil, target{}, skip(err.Error()), nil
		}
		return nil, target{}, verdict{status: QueueDidntRun, reason: err.Error()}, nil
	}
	if v := stillMatches(a, it, f); v != goAhead {
		return nil, target{}, v, nil
	}
	if slices.ContainsFunc(trails, func(t *Trail) bool { return t.PlayID == it.PlayID && !t.StartedAt.Before(it.QueuedAt) }) {
		return nil, target{}, skip("already ran"), nil
	}
	tgt, err := r.readTarget(identity.WithActor(ctx, identity.Actor{ID: it.PersonID}), it.TargetType, it.TargetID)
	if err != nil {
		return nil, target{}, verdict{status: QueueDidntRun, reason: err.Error()}, nil
	}
	return play, tgt, goAhead, nil
}

// recheckRecords reads the item's auto play and play again: removed, switched off, or waiting for another moment skips it.
func (r *Runner) recheckRecords(ctx context.Context, it *QueueItem) (*AutoPlay, *Play, verdict, error) {
	a, err := r.plays.GetAutoPlay(ctx, it.AutoPlayID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil, skip("its auto play was removed"), nil
	}
	if err != nil {
		return nil, nil, goAhead, fmt.Errorf("get auto play %s: %w", it.AutoPlayID, err)
	}
	if !a.Enabled {
		return nil, nil, skip("its auto play was switched off"), nil
	}
	if a.Moment != it.Moment {
		return nil, nil, skip("its auto play now waits for another moment"), nil
	}
	play, err := r.plays.Get(ctx, it.PlayID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil, skip("the play was removed"), nil
	}
	if err != nil {
		return nil, nil, goAhead, fmt.Errorf("get play %s: %w", it.PlayID, err)
	}
	return a, play, goAhead, nil
}

// stillMatches checks the moment, the conditions, and the person against the target as it stands now.
func stillMatches(a *AutoPlay, it *QueueItem, f Facts) verdict {
	if f.Archived {
		return skip("the doc was archived")
	}
	if reason := momentGone(a, f); reason != "" {
		return skip(reason)
	}
	if !a.Conditions.holds(f) {
		return skip("its conditions no longer hold")
	}
	if a.RunOn == RunOnDeveloper && f.Developer != it.PersonID {
		return skip("the developer changed")
	}
	if a.RunOn == RunOnTester && f.Tester != it.PersonID {
		return skip("the tester changed")
	}
	return goAhead
}

// momentGone says why a's moment no longer holds on the target, "" while it does.
func momentGone(a *AutoPlay, f Facts) string {
	switch a.Moment {
	case MomentTicketUnblocked:
		if f.Blocked {
			return "no longer unblocked"
		}
	case MomentTicketEnteredStage:
		if a.MomentStage != nil && f.Stage != *a.MomentStage {
			return "no longer in " + string(*a.MomentStage)
		}
	case MomentTicketDeveloperSet:
		if f.Developer == "" {
			return "no longer has a developer"
		}
	case MomentTicketTesterSet:
		if f.Tester == "" {
			return "no longer has a tester"
		}
	}
	return ""
}
