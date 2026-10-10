package plays

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// QueuePlayInput is an automation's runPlay call: a ticket play named by its label, the ticket, and whom it runs on.
type QueuePlayInput struct {
	Play     string
	TicketID string
	RunOn    RunOn
	Priority Level
}

// QueuePlay queues the ticket play an automation names by label, as its creator, through the auto play queue (ADR 0132).
func (r *Runner) QueuePlay(ctx context.Context, in QueuePlayInput) (*QueueItem, error) {
	actor, _ := identity.ActorFromCtx(ctx)
	if actor.Automation == nil {
		return nil, fmt.Errorf("%w: only an automation queues a play by name; a person presses it or calls play_run", apperrs.ErrForbidden)
	}
	if actor.ID == "" {
		return nil, fmt.Errorf("%w: a default automation has no creator to run plays as", apperrs.ErrForbidden)
	}
	if r.queue == nil || r.facts == nil {
		return nil, fmt.Errorf("%w: auto plays are not running on this server", apperrs.ErrNotFound)
	}
	in, err := in.normalize()
	if err != nil {
		return nil, err
	}
	play, err := r.plays.GetByLabel(ctx, actor.Automation.WorkspaceID, in.Play)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w: no play named %q in this workspace", apperrs.ErrNotFound, in.Play)
	}
	if err != nil {
		return nil, err
	}
	if play.Type != TypeTicket {
		return nil, fmt.Errorf("%w: %q is a %s play; runPlay starts ticket plays", apperrs.ErrInvalid, play.Label, play.Type)
	}
	f, err := r.facts.Facts(ctx, TargetTicket, in.TicketID)
	if err != nil {
		return nil, fmt.Errorf("get ticket %s: %w", in.TicketID, err)
	}
	if err := r.checkPlay(ctx, actor.ID, play, f.ProjectID); err != nil {
		return nil, err
	}
	return r.enqueueAutomationRun(ctx, actor.Automation.ID, play, f, in)
}

func (in QueuePlayInput) normalize() (QueuePlayInput, error) {
	in.Play, in.TicketID = strings.TrimSpace(in.Play), strings.TrimSpace(in.TicketID)
	if in.Play == "" || in.TicketID == "" {
		return in, fmt.Errorf("%w: play and ticket_id are required", apperrs.ErrInvalid)
	}
	if in.RunOn == "" {
		in.RunOn = RunOnDeveloper
	}
	if in.RunOn != RunOnDeveloper && in.RunOn != RunOnTester {
		return in, fmt.Errorf("%w: run_on must be developer or tester", apperrs.ErrInvalid)
	}
	if in.Priority == "" {
		in.Priority = LevelNormal
	}
	if !in.Priority.valid() {
		return in, fmt.Errorf("%w: priority must be high, normal, or low", apperrs.ErrInvalid)
	}
	return in, nil
}

// enqueueAutomationRun writes the run queued, held at the daily cap, or didn't run; a twin already waiting answers instead.
func (r *Runner) enqueueAutomationRun(ctx context.Context, automationID string, play *Play, f Facts, in QueuePlayInput) (*QueueItem, error) {
	now := r.now().UTC()
	it := &QueueItem{
		ID: ids.New(), WorkspaceID: play.WorkspaceID, ProjectID: f.ProjectID, TargetType: TargetTicket, TargetID: in.TicketID,
		PlayID: play.ID, PlayLabel: play.Label, AutomationID: automationID, PersonID: f.Developer, RunOn: in.RunOn,
		Moment: MomentAutomation, Priority: in.Priority, Status: QueueQueued, Via: ViaAutomation, QueuedAt: now, NotBefore: now,
	}
	if in.RunOn == RunOnTester {
		it.PersonID = f.Tester
	}
	if reason := r.whyNobody(ctx, it); reason != "" {
		r.failedTrail(ctx, it, reason)
		it.Status, it.Reason, it.DecidedAt = QueueDidntRun, reason, &now
		_, err := r.queue.EnqueueRun(ctx, it, r.queueEvent(TopicQueued, it))
		return it, err
	}
	count, limit, _, err := r.autoRunsToday(ctx, it.WorkspaceID, TargetTicket, it.TargetID)
	if err != nil {
		return nil, err
	}
	if count >= limit {
		it.Reason = ReasonPaused
	}
	queued, err := r.queue.EnqueueRun(ctx, it, r.queueEvent(TopicQueued, it))
	if err != nil {
		return nil, err
	}
	if !queued {
		return r.waitingRun(ctx, it)
	}
	r.Kick()
	return it, nil
}

// waitingRun is the run of it's automation and play already waiting on its ticket.
func (r *Runner) waitingRun(ctx context.Context, it *QueueItem) (*QueueItem, error) {
	items, err := r.queue.ListQueueByTarget(ctx, it.TargetType, it.TargetID)
	if err != nil {
		return nil, fmt.Errorf("list the queue of ticket %s: %w", it.TargetID, err)
	}
	for _, w := range items {
		if w.AutomationID == it.AutomationID && w.PlayID == it.PlayID && (w.Status == QueueQueued || w.Status == QueueDispatching) {
			return w, nil
		}
	}
	return nil, fmt.Errorf("%w: the run of %s on ticket %s moved on while queuing; try again", apperrs.ErrConflict, it.PlayLabel, it.TargetID)
}
