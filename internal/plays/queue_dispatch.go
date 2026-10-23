package plays

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const (
	// heldRetry is when an item none of its person's computers could take is tried again.
	heldRetry = time.Minute
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

// HandleSlotsFreed is registered on play.run_finished, where a run ending frees its computer's room and its target, and on
// auto_play.limits_updated, where a workspace's daily cap may now let a paused target go.
func (r *Runner) HandleSlotsFreed(context.Context, eventbus.Event) error {
	r.Kick()
	return nil
}

// ComputerTopics are the pairing changes that may let one of a person's computers take a waiting run;
// HandleComputerChanged is subscribed to each.
var ComputerTopics = []string{"computer.paired", "computer.setup_confirmed", "computer.facts_changed", "computer.placement_changed"}

// HandleComputerChanged makes the person's queued runs due now and wakes the dispatcher, so a fixed gap need not wait
// out the minute.
func (r *Runner) HandleComputerChanged(ctx context.Context, ev eventbus.Event) error {
	if r.queue == nil {
		return nil
	}
	var p struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.UserID == "" {
		return nil
	}
	if err := r.queue.WakePerson(ctx, p.UserID, r.now().UTC()); err != nil {
		return apperrs.Retryable(fmt.Errorf("wake the queue of %s: %w", p.UserID, err))
	}
	r.Kick()
	return nil
}

// AutoPlayOffTopics are the changes that can end an auto play's say over its queued runs; HandleAutoPlayOff is subscribed to each.
var AutoPlayOffTopics = []string{TopicAutoPlayUpdated, TopicAutoPlayDeleted}

// HandleAutoPlayOff skips the queued runs of an auto play that was switched off or deleted, so a run held behind a
// busy ticket or an offline computer does not keep retrying for a rule that is gone. A run already dispatching goes on.
func (r *Runner) HandleAutoPlayOff(ctx context.Context, ev eventbus.Event) error {
	if r.queue == nil {
		return nil
	}
	var p struct {
		ID       string   `json:"id"`
		PlayID   string   `json:"play_id"`
		AutoPlay AutoPlay `json:"auto_play"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	id, playID, reason := p.ID, p.PlayID, "its auto play was removed"
	if ev.Topic == TopicAutoPlayUpdated {
		if p.AutoPlay.Enabled {
			return nil
		}
		id, playID, reason = p.AutoPlay.ID, p.AutoPlay.PlayID, "its auto play was switched off"
	}
	items, err := r.queue.ListQueuedByPlay(ctx, playID, ProjectScope{All: true})
	if err != nil {
		return apperrs.Retryable(fmt.Errorf("list the queued runs of play %s: %w", playID, err))
	}
	for _, it := range items {
		if it.AutoPlayID == id {
			r.logDecide(ctx, it, QueueQueued, QueueSkipped, reason)
		}
	}
	r.Kick()
	return nil
}

// RunQueue starts queued auto runs until ctx ends. It passes once at boot, then on every kick, and on a timer to the
// next held item (a held computer's retry, a paused target's oldest run leaving the day); it never polls.
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

// dispatchPerson starts a person's due items, highest priority then oldest first, each on the first of their computers
// that can take it (ADR 0165); a started run takes its computer's room, which the next item's count sees.
func (r *Runner) dispatchPerson(ctx context.Context, pass *queuePass, person string) error {
	items, err := r.queue.ListDue(ctx, person, pass.now)
	if err != nil {
		return fmt.Errorf("list the queue of %s: %w", person, err)
	}
	for _, it := range items {
		r.dispatchItem(ctx, pass, it)
	}
	return nil
}

// dispatchItem starts it, holds it, or decides it.
func (r *Runner) dispatchItem(ctx context.Context, pass *queuePass, it *QueueItem) {
	trails, err := r.trails.ListTrailsByTarget(ctx, it.TargetType, it.TargetID)
	if err != nil {
		r.log.Error("plays: read the trails of a queued run's target", "item", it.ID, "error", err)
		return
	}
	if slices.ContainsFunc(trails, func(t *Trail) bool { return t.State.Active() }) {
		r.hold(ctx, it, ReasonTicketBusy)
		return
	}
	count, limit, frees, err := r.autoRunsToday(ctx, it.WorkspaceID, it.TargetType, it.TargetID)
	if err != nil {
		r.log.Error("plays: count a queued run's target's automatic runs", "item", it.ID, "error", err)
		return
	}
	if count >= limit {
		pass.wakeAt(frees)
		r.hold(ctx, it, ReasonPaused)
		return
	}
	play, tgt, v, err := r.recheck(ctx, it, trails)
	if err != nil {
		r.log.Error("plays: check a queued run again", "item", it.ID, "error", err)
		return
	}
	if v.status == QueueDidntRun {
		r.failedTrail(ctx, it, v.reason)
	}
	if v.status != QueueQueued {
		r.logDecide(ctx, it, QueueQueued, v.status, v.reason)
		return
	}
	r.startQueued(ctx, it, play, tgt, pass.now)
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

// startQueued launches it on its person's first ready computer under a trail id written onto the row first, so a crash
// mid-start leaves no second run behind.
func (r *Runner) startQueued(ctx context.Context, it *QueueItem, play *Play, tgt target, now time.Time) {
	held := *it
	it.TrailID = ids.New()
	if err := r.move(ctx, it, QueueQueued, QueueDispatching, "", false); err != nil {
		it.TrailID = ""
		if !errors.Is(err, apperrs.ErrConflict) {
			r.log.Error("plays: mark a queued run dispatching", "item", it.ID, "error", err)
		}
		return
	}
	runCtx, cancel := context.WithTimeout(identity.WithActor(ctx, identity.Actor{ID: it.PersonID}), dispatchTimeout)
	defer cancel()
	trail := &Trail{
		ID: it.TrailID, WorkspaceID: it.WorkspaceID, PlayID: play.ID, PlayLabel: play.Label, TargetType: it.TargetType,
		TargetID: it.TargetID, ProjectID: it.ProjectID, StarterID: it.PersonID, Via: it.Via, SelectedMemoryIDs: normalizeIDs(tgt.memoryIDs),
		State: TrailStarting, StartedAt: now,
	}
	_, err := r.launch(runCtx, play, trail, tgt, HarnessChoice{ComputerID: it.ComputerID, ComputerIDs: it.ComputerIDs}, launchQueued)
	if err == nil {
		r.logDecide(ctx, it, QueueDispatching, QueueStarted, "")
		return
	}
	var refusal *HarnessRefusal
	if errors.As(err, &refusal) && onlyFull(refusal.PassedOver) {
		it.WaitingOn = refusal.PassedOver
		r.requeue(ctx, it, refusal.Reason, held, now)
		return
	}
	if refusal != nil {
		it.WaitingOn = refusal.PassedOver
		r.requeue(ctx, it, refusal.Reason, held, now.Add(heldRetry))
		return
	}
	if errors.Is(err, apperrs.ErrConflict) {
		r.requeue(ctx, it, ReasonTicketBusy, held, now)
		return
	}
	if _, gerr := r.trails.GetTrail(ctx, it.TrailID); gerr != nil {
		it.TrailID = ""
	}
	r.logDecide(ctx, it, QueueDispatching, QueueDidntRun, err.Error())
}

// onlyFull is whether every computer that could not take a run was only full: a run ending frees one and kicks the
// queue, so the run waits for that rather than a retry.
func onlyFull(passed []PassedOver) bool {
	return len(passed) > 0 && !slices.ContainsFunc(passed, func(p PassedOver) bool { return p.Why != "full" })
}

// requeue puts a dispatching item back in its queue without the trail it never got, announcing it only when it waits
// for a new reason or on different computers, so a held run's retry every minute stays quiet.
func (r *Runner) requeue(ctx context.Context, it *QueueItem, reason string, held QueueItem, notBefore time.Time) {
	it.TrailID, it.NotBefore = "", notBefore
	changed := reason != held.Reason || !slices.Equal(it.WaitingOn, held.WaitingOn)
	if err := r.move(ctx, it, QueueDispatching, QueueQueued, reason, changed); err != nil {
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
	f.Service = it.Service
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
	if reason := r.whyNoComputer(ctx, it); reason != "" {
		return nil, target{}, verdict{status: QueueDidntRun, reason: reason}, nil
	}
	tgt, err := r.readTarget(identity.WithActor(ctx, identity.Actor{ID: it.PersonID}), it.TargetType, it.TargetID)
	if err != nil {
		return nil, target{}, verdict{status: QueueDidntRun, reason: err.Error()}, nil
	}
	tgt.service, tgt.outageID = it.Service, it.OutageID
	if a != nil {
		tgt.memoryIDs = a.MemoryIDs
	}
	return play, tgt, goAhead, nil
}

// recheckRecords reads the item's auto play and play again: removed, switched off, or waiting for another moment skips
// it. A run an automation queued has no auto play, so only its play is read.
func (r *Runner) recheckRecords(ctx context.Context, it *QueueItem) (*AutoPlay, *Play, verdict, error) {
	if it.AutoPlayID == "" {
		play, v, err := r.recheckPlay(ctx, it)
		return nil, play, v, err
	}
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
	play, v, err := r.recheckPlay(ctx, it)
	return a, play, v, err
}

func (r *Runner) recheckPlay(ctx context.Context, it *QueueItem) (*Play, verdict, error) {
	play, err := r.plays.Get(ctx, it.PlayID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, skip("the play was removed"), nil
	}
	if err != nil {
		return nil, goAhead, fmt.Errorf("get play %s: %w", it.PlayID, err)
	}
	return play, goAhead, nil
}

// stillMatches checks the moment, the conditions, and the person against the target as it stands now; a run an
// automation queued has no moment or conditions, only its person.
func stillMatches(a *AutoPlay, it *QueueItem, f Facts) verdict {
	if a == nil {
		return samePerson(it.RunOn, it.PersonID, f)
	}
	if f.Archived {
		return skip("the doc was archived")
	}
	if reason := momentGone(a, f); reason != "" {
		return skip(reason)
	}
	if !a.Conditions.holds(f) {
		return skip("its conditions no longer hold")
	}
	if a.RunOn == RunOnComputer && a.RunAs != it.PersonID {
		return skip("its computer was picked by someone else")
	}
	return samePerson(a.RunOn, it.PersonID, f)
}

func samePerson(runOn RunOn, personID string, f Facts) verdict {
	if runOn == RunOnDeveloper && f.Developer != personID {
		return skip("the developer changed")
	}
	if runOn == RunOnTester && f.Tester != personID {
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
