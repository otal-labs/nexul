package plays

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const (
	// scheduleFallback bounds the scheduler's sleep, so a restored workspace or a moved clock is seen within the hour.
	scheduleFallback = time.Hour
	// sweepCatchUp is how late a missed firing may still sweep when its schedule names no stop time.
	sweepCatchUp = time.Hour
	// sweepLookBack bounds how far back the scheduler looks for a missed firing.
	sweepLookBack = 7 * 24 * time.Hour
)

// Sweep is one firing of a schedule auto play (ADR 0179): the tickets it queued, worked through in its order, closed
// with a digest once nothing of it waits or works.
type Sweep struct {
	ID          string      `json:"id"`
	WorkspaceID string      `json:"workspace_id"`
	AutoPlayID  string      `json:"auto_play_id"`
	PlayID      string      `json:"play_id"`
	PlayLabel   string      `json:"play_label"`
	FiredAt     time.Time   `json:"fired_at"`
	StartBy     *time.Time  `json:"start_by" jsonschema:"No run starts after this; null for none."`
	Matched     int         `json:"matched" jsonschema:"Tickets matched."`
	OverCap     int         `json:"over_cap" jsonschema:"Matches left past max_runs."`
	Counts      SweepCounts `json:"counts" jsonschema:"How its runs ended, once finished."`
	FinishedAt  *time.Time  `json:"finished_at" jsonschema:"Null while it goes."`
}

// SweepCounts is how a sweep's tickets ended.
type SweepCounts struct {
	Finished int `json:"finished" `
	Asked    int `json:"asked" `
	Failed   int `json:"failed" `
	Skipped  int `json:"skipped" `
	DidntRun int `json:"didnt_run" `
}

// SweepEvent is the auto_play.sweep_started payload, and with People and Counts the auto_play.sweep_finished one.
type SweepEvent struct {
	Sweep Sweep `json:"sweep"`
	// People are those its runs went out as, who hear of it once it finishes.
	People []string `json:"people" jsonschema:"User ids of the people its runs went out as."`
}

// SweepDigest is a sweep with each of its tickets, as the morning read shows them to someone who may see the play.
type SweepDigest struct {
	Sweep
	Lines []SweepLine `json:"lines"`
}

// SweepLine is one ticket of a sweep and how it went.
type SweepLine struct {
	QueueItemID string      `json:"queue_item_id"`
	TicketID    string      `json:"ticket_id"`
	Key         string      `json:"key"`
	Title       string      `json:"title"`
	Status      QueueStatus `json:"status" enum:"queued,dispatching,started,skipped,didnt_run,cancelled"`
	Reason      string      `json:"reason"`
	TrailID     string      `json:"trail_id"`
	// Outcome is how a started run went: running, finished, asked, failed; empty for a ticket that never started.
	Outcome  string `json:"outcome" enum:",running,finished,asked,failed"`
	Question string `json:"question"`
	// Stage is where the ticket stands now, such as review for one a run handed back with a question.
	Stage Stage          `json:"stage" enum:",backlog,progress,review,testing,done"`
	Bugs  []LinkedTicket `json:"bugs" jsonschema:"Bugs found in the ticket while the sweep went."`
}

// TicketFacts is a ticket's id beside its facts, for a sweep reading every ticket of a workspace.
type TicketFacts struct {
	ID string
	Facts
}

// SweepRepo persists sweeps and what the dispatcher reads about them; implemented in internal/platform/storage.
type SweepRepo interface {
	// CreateSweep writes s once; false when its firing was already written.
	CreateSweep(ctx context.Context, s *Sweep, evts ...eventbus.OutboxEvent) (bool, error)
	GetSweep(ctx context.Context, id string) (*Sweep, error)
	ListOpenSweeps(ctx context.Context) ([]*Sweep, error)
	// FinishSweep stamps s finished with its counts; false when it already was.
	FinishSweep(ctx context.Context, s *Sweep, evts ...eventbus.OutboxEvent) (bool, error)
	// ListQueueBySweep returns a sweep's items in its order.
	ListQueueBySweep(ctx context.Context, sweepID string) ([]*QueueItem, error)
	// SweepBusy reports whether anything of the sweep still waits, starts, or works; a run waiting on a question does not.
	SweepBusy(ctx context.Context, sweepID string) (bool, error)
	// BusyTargets names those of targetIDs with a run going or waiting, or an item queued.
	BusyTargets(ctx context.Context, targetType TargetType, targetIDs []string) (map[string]bool, error)
	// SweepWorkingOn reports whether a sweep's run is starting or working on the target.
	SweepWorkingOn(ctx context.Context, targetType TargetType, targetID string) (bool, error)
	// LockHeld reports whether an item other than exceptID holding lockKey is starting or working.
	LockHeld(ctx context.Context, lockKey, exceptID string) (bool, error)
	// NextStartBy is the earliest start_by of a queued item after after; false when there is none.
	NextStartBy(ctx context.Context, after time.Time) (time.Time, bool, error)
}

// KickSchedules wakes the scheduler to read the schedules again; it never blocks.
func (r *Runner) KickSchedules() {
	select {
	case r.scheduleKick <- struct{}{}:
	default:
	}
}

// ScheduleTopics are the changes that can move a schedule's next firing; HandleScheduleChanged is subscribed to each.
var ScheduleTopics = []string{TopicAutoPlayCreated, TopicAutoPlayUpdated, TopicAutoPlayDeleted}

// HandleScheduleChanged wakes the scheduler after an auto play changed.
func (r *Runner) HandleScheduleChanged(context.Context, eventbus.Event) error {
	r.KickSchedules()
	return nil
}

// RunSchedules fires schedule auto plays until ctx ends: it sleeps to the next firing, waking early on a kick.
func (r *Runner) RunSchedules(ctx context.Context) {
	if r.queue == nil || r.facts == nil {
		return
	}
	for {
		wait, err := r.sweepDue(ctx)
		if err != nil {
			r.log.Error("plays: fire the schedules", "error", err)
			wait = queueRetry
		}
		timer := time.NewTimer(min(wait, scheduleFallback))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		case <-r.scheduleKick:
			timer.Stop()
		}
	}
}

// sweepDue fires every schedule whose firing has come and says how long until the next one.
func (r *Runner) sweepDue(ctx context.Context) (time.Duration, error) {
	list, err := r.plays.ListEnabledScheduleAutoPlays(ctx)
	if err != nil {
		return 0, fmt.Errorf("list the schedule auto plays: %w", err)
	}
	autoPlayIDs := make([]string, 0, len(list))
	for _, a := range list {
		autoPlayIDs = append(autoPlayIDs, a.ID)
	}
	last, err := r.plays.LastSweeps(ctx, autoPlayIDs)
	if err != nil {
		return 0, fmt.Errorf("read the last sweeps: %w", err)
	}
	now := r.now().UTC()
	next := now.Add(scheduleFallback)
	for _, a := range list {
		due, upcoming, ok := firing(a, last[a.ID], now)
		if !ok {
			continue
		}
		if !due.IsZero() {
			if err := r.Sweep(ctx, a, due); err != nil {
				return 0, err
			}
		}
		if !upcoming.IsZero() && upcoming.Before(next) {
			next = upcoming
		}
	}
	return max(next.Sub(r.now().UTC()), 0), nil
}

// firing is a's missed firing still worth sweeping, zero for none, and its next one after now; false when a's schedule
// no longer reads. A change to a starts it over from the change, and only the latest missed firing is caught up.
func firing(a *AutoPlay, last *Sweep, now time.Time) (due, upcoming time.Time, ok bool) {
	if a.Schedule == nil {
		return time.Time{}, time.Time{}, false
	}
	spec, err := parseCron(a.Schedule.Cron)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	loc, err := time.LoadLocation(a.Schedule.Timezone)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	since := a.UpdatedAt
	if floor := now.Add(-sweepLookBack); since.Before(floor) {
		since = floor
	}
	if last != nil && last.FiredAt.After(since) {
		since = last.FiredAt
	}
	for t := spec.next(since.In(loc)); !t.IsZero() && !t.After(now); t = spec.next(t) {
		due = t
	}
	upcoming = spec.next(now.In(loc))
	if due.IsZero() {
		return due, upcoming, true
	}
	stop := startBy(a.Schedule, due)
	if now.Sub(due) > sweepCatchUp && (stop == nil || !now.Before(*stop)) {
		return time.Time{}, upcoming, true
	}
	return due.UTC(), upcoming, true
}

// startBy is the first stop time after fired in the schedule's zone, nil when the schedule names none.
func startBy(s *Schedule, fired time.Time) *time.Time {
	minutes, ok := parseClock(s.StopAt)
	if !ok {
		return nil
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return nil
	}
	local := fired.In(loc)
	stop := time.Date(local.Year(), local.Month(), local.Day(), minutes/60, minutes%60, 0, 0, loc)
	if !stop.After(local) {
		stop = time.Date(local.Year(), local.Month(), local.Day()+1, minutes/60, minutes%60, 0, 0, loc)
	}
	stop = stop.UTC()
	return &stop
}

// Sweep fires a at fired: every ticket of its workspace its play and conditions take, highest priority then board order,
// is queued as one sweep up to its cap; a ticket already running or queued is skipped. A firing sweeps once.
func (r *Runner) Sweep(ctx context.Context, a *AutoPlay, fired time.Time) error {
	sweepID := ids.From(fmt.Sprintf("auto-play-sweep:%s:%d", a.ID, fired.Unix()))
	if _, err := r.queue.GetSweep(ctx, sweepID); err == nil || !errors.Is(err, apperrs.ErrNotFound) {
		return err
	}
	play, err := r.plays.Get(ctx, a.PlayID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get play %s: %w", a.PlayID, err)
	}
	matched, err := r.sweepMatches(ctx, a, play)
	if err != nil {
		return err
	}
	ticketIDs := make([]string, 0, len(matched))
	for _, t := range matched {
		ticketIDs = append(ticketIDs, t.ID)
	}
	busy, err := r.queue.BusyTargets(ctx, TargetTicket, ticketIDs)
	if err != nil {
		return fmt.Errorf("read which tickets have a run already: %w", err)
	}
	now := r.now().UTC()
	s := &Sweep{
		ID: sweepID, WorkspaceID: a.WorkspaceID, AutoPlayID: a.ID, PlayID: play.ID, PlayLabel: play.Label, FiredAt: fired,
		StartBy: startBy(a.Schedule, fired), Matched: len(matched),
	}
	people, err := r.queueSweep(ctx, a, play, s, matched, busy, now)
	if err != nil {
		return err
	}
	if _, err := r.queue.CreateSweep(ctx, s, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSweepStarted, Payload: SweepEvent{Sweep: *s, People: people}}); err != nil {
		return fmt.Errorf("create sweep %s: %w", s.ID, err)
	}
	r.Kick()
	return nil
}

// sweepMatches are the workspace's tickets a's play and conditions take, highest priority first, board order kept within one.
func (r *Runner) sweepMatches(ctx context.Context, a *AutoPlay, play *Play) ([]TicketFacts, error) {
	if !play.Enabled {
		return nil, nil
	}
	all, err := r.facts.WorkspaceTickets(ctx, a.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("read the tickets of workspace %s: %w", a.WorkspaceID, err)
	}
	matched := slices.DeleteFunc(all, func(t TicketFacts) bool {
		return slices.Contains(play.ExcludedProjectIDs, t.ProjectID) || !a.Conditions.holds(t.Facts)
	})
	slices.SortStableFunc(matched, func(x, y TicketFacts) int {
		return cmp.Compare(levelRank[a.Priority.level(y.Facts)], levelRank[a.Priority.level(x.Facts)])
	})
	return matched, nil
}

var levelRank = map[Level]int{LevelLow: 0, LevelNormal: 1, LevelHigh: 2}

// queueSweep writes s's items: busy tickets skipped, the rest queued up to the cap, and counts what the cap left over.
// It returns the people the queued runs go out as.
func (r *Runner) queueSweep(ctx context.Context, a *AutoPlay, play *Play, s *Sweep, matched []TicketFacts, busy map[string]bool, now time.Time) ([]string, error) {
	people := []string{}
	taken := 0
	for i, t := range matched {
		if taken >= a.Schedule.MaxRuns {
			if !busy[t.ID] {
				s.OverCap++
			}
			continue
		}
		it := &QueueItem{
			ID: ids.From("auto-play-sweep:" + s.ID + ":" + t.ID), WorkspaceID: a.WorkspaceID, ProjectID: t.ProjectID,
			TargetType: TargetTicket, TargetID: t.ID, PlayID: play.ID, PlayLabel: play.Label, AutoPlayID: a.ID,
			PersonID: a.person(t.Facts, moment{}), RunOn: a.RunOn, Moment: MomentSchedule, ComputerIDs: slices.Clone(a.ComputerIDs),
			ComputerID: a.ComputerID, Priority: a.Priority.level(t.Facts), Status: QueueQueued, Via: ViaWeb, QueuedAt: now,
			NotBefore: now, SweepID: s.ID, SweepPosition: i, StartBy: s.StartBy, LockKey: a.LockKey(),
		}
		if busy[t.ID] {
			it.Status, it.Reason, it.DecidedAt = QueueSkipped, "it already had a run going or waiting", &now
			if _, err := r.queue.EnqueueRun(ctx, it, r.queueEvent(TopicQueued, it)); err != nil {
				return nil, err
			}
			continue
		}
		taken++
		queued, err := r.offer(ctx, a, it)
		if err != nil {
			return nil, err
		}
		if queued && !slices.Contains(people, it.PersonID) {
			people = append(people, it.PersonID)
		}
	}
	return people, nil
}

// finishSweeps closes every open sweep nothing of which waits or works any more, counting how its runs went.
func (r *Runner) finishSweeps(ctx context.Context) error {
	open, err := r.queue.ListOpenSweeps(ctx)
	if err != nil {
		return fmt.Errorf("list the open sweeps: %w", err)
	}
	for _, s := range open {
		busy, err := r.queue.SweepBusy(ctx, s.ID)
		if err != nil {
			return fmt.Errorf("check sweep %s: %w", s.ID, err)
		}
		if busy {
			continue
		}
		if err := r.finishSweep(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) finishSweep(ctx context.Context, s *Sweep) error {
	items, err := r.queue.ListQueueBySweep(ctx, s.ID)
	if err != nil {
		return fmt.Errorf("list the items of sweep %s: %w", s.ID, err)
	}
	people := []string{}
	for _, it := range items {
		outcome, _, err := r.outcome(ctx, it)
		if err != nil {
			return err
		}
		s.Counts.add(it.Status, outcome)
		if it.PersonID != "" && !slices.Contains(people, it.PersonID) {
			people = append(people, it.PersonID)
		}
	}
	now := r.now().UTC()
	s.FinishedAt = &now
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSweepFinished, Payload: SweepEvent{Sweep: *s, People: people}}
	if _, err := r.queue.FinishSweep(ctx, s, evt); err != nil {
		return fmt.Errorf("finish sweep %s: %w", s.ID, err)
	}
	return nil
}

func (c *SweepCounts) add(status QueueStatus, outcome string) {
	switch {
	case outcome == "finished":
		c.Finished++
	case outcome == "asked":
		c.Asked++
	case outcome == "failed":
		c.Failed++
	case status == QueueDidntRun:
		c.DidntRun++
	case status == QueueSkipped || status == QueueCancelled:
		c.Skipped++
	}
}

// outcome reads how a started item's run went, with the question it stopped on; empty for an item that never started.
func (r *Runner) outcome(ctx context.Context, it *QueueItem) (string, string, error) {
	if it.Status != QueueStarted || it.TrailID == "" {
		return "", "", nil
	}
	t, err := r.trails.GetTrail(ctx, it.TrailID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("get trail %s: %w", it.TrailID, err)
	}
	question := ""
	if t.Question != nil && len(t.Question.Questions) > 0 {
		question = t.Question.Questions[0].Text
	}
	switch t.State {
	case TrailDone:
		return "finished", "", nil
	case TrailWaiting:
		return "asked", question, nil
	case TrailFailed, TrailInterrupted:
		return "failed", "", nil
	}
	return "running", "", nil
}

// LastSweeps returns the latest sweep of each of a play's schedule auto plays, keyed by auto play, to whoever may see
// the play.
func (r *Runner) LastSweeps(ctx context.Context, playID string) (map[string]*Sweep, error) {
	play, err := r.seenPlay(ctx, playID)
	if err != nil {
		return nil, err
	}
	list, err := r.plays.ListAutoPlays(ctx, []string{play.ID})
	if err != nil {
		return nil, fmt.Errorf("list auto plays of play %s: %w", play.ID, err)
	}
	autoPlayIDs := []string{}
	for _, a := range list {
		if a.Moment == MomentSchedule {
			autoPlayIDs = append(autoPlayIDs, a.ID)
		}
	}
	out, err := r.plays.LastSweeps(ctx, autoPlayIDs)
	if err != nil {
		return nil, fmt.Errorf("read the last sweeps of play %s: %w", play.ID, err)
	}
	return out, nil
}

// GetSweepDigest returns a sweep and its tickets to whoever may see its play, leaving out tickets in projects they
// cannot open.
func (r *Runner) GetSweepDigest(ctx context.Context, sweepID string) (*SweepDigest, error) {
	sweepID = strings.TrimSpace(sweepID)
	if sweepID == "" {
		return nil, fmt.Errorf("%w: sweep id is required", apperrs.ErrInvalid)
	}
	if r.queue == nil {
		return nil, fmt.Errorf("%w: auto plays are not running on this server", apperrs.ErrNotFound)
	}
	s, err := r.queue.GetSweep(ctx, sweepID)
	if err != nil {
		return nil, fmt.Errorf("get sweep %s: %w", sweepID, err)
	}
	if _, err := r.seenPlay(ctx, s.PlayID); err != nil {
		return nil, fmt.Errorf("get sweep %s: %w", sweepID, apperrs.ErrNotFound)
	}
	items, err := r.queue.ListQueueBySweep(ctx, s.ID)
	if err != nil {
		return nil, fmt.Errorf("list the items of sweep %s: %w", s.ID, err)
	}
	d := &SweepDigest{Sweep: *s, Lines: []SweepLine{}}
	actor := actorID(ctx)
	for _, it := range items {
		if !r.perm.HasPermission(ctx, actor, it.WorkspaceID, permissions.Member, resourceTypeProject, it.ProjectID) {
			continue
		}
		line, err := r.sweepLine(ctx, s, it)
		if err != nil {
			return nil, err
		}
		d.Lines = append(d.Lines, line)
	}
	return d, nil
}

func (r *Runner) sweepLine(ctx context.Context, s *Sweep, it *QueueItem) (SweepLine, error) {
	line := SweepLine{
		QueueItemID: it.ID, TicketID: it.TargetID, Key: it.TargetID, Status: it.Status, Reason: it.Reason, TrailID: it.TrailID,
		Bugs: []LinkedTicket{},
	}
	t, err := r.targets.GetTicket(ctx, it.TargetID)
	if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) {
		return line, nil
	}
	if err != nil {
		return line, fmt.Errorf("get ticket %s: %w", it.TargetID, err)
	}
	line.Key, line.Title, line.Stage = t.Key, t.Title, t.Stage
	if line.Outcome, line.Question, err = r.outcome(ctx, it); err != nil {
		return line, err
	}
	if r.links == nil || it.Status != QueueStarted {
		return line, nil
	}
	bugs, err := r.links.BugsFoundSince(ctx, it.TargetID, s.FiredAt)
	if err != nil {
		return line, fmt.Errorf("read the bugs found in ticket %s: %w", it.TargetID, err)
	}
	line.Bugs = append(line.Bugs, bugs...)
	return line, nil
}

// seenPlay reads a play the caller may see; any other is not found.
func (r *Runner) seenPlay(ctx context.Context, playID string) (*Play, error) {
	playID = strings.TrimSpace(playID)
	if playID == "" {
		return nil, fmt.Errorf("%w: play id is required", apperrs.ErrInvalid)
	}
	play, err := r.plays.Get(ctx, playID)
	if err != nil {
		return nil, fmt.Errorf("get play %s: %w", playID, err)
	}
	if !r.perm.HasPermission(ctx, actorID(ctx), play.WorkspaceID, permissions.PlaysRead, resourceTypePlay, play.ID) {
		return nil, fmt.Errorf("get play %s: %w", playID, apperrs.ErrNotFound)
	}
	return play, nil
}
