package plays

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// QueueStatus is where a matched auto run stands; queued and dispatching wait, the rest are decided.
type QueueStatus string

const (
	QueueQueued      QueueStatus = "queued"
	QueueDispatching QueueStatus = "dispatching"
	QueueStarted     QueueStatus = "started"
	QueueSkipped     QueueStatus = "skipped"
	QueueDidntRun    QueueStatus = "didnt_run"
	QueueCancelled   QueueStatus = "cancelled"
)

// Why a queued item waits; a decided item's reason is a sentence for the ticket instead.
const (
	ReasonOffline    = "offline"
	ReasonTicketBusy = "ticket busy"
	ReasonPaused     = "paused"
)

// QueueItem is one match of an auto play's moment on a ticket or doc (ADR 0132), queued on the person it runs on and
// kept once decided, so the target shows it and the daily cap counts it.
type QueueItem struct {
	ID          string      `json:"id"`
	WorkspaceID string      `json:"workspace_id"`
	ProjectID   string      `json:"project_id"`
	TargetType  TargetType  `json:"target_type" enum:"ticket,doc"`
	TargetID    string      `json:"target_id"`
	PlayID      string      `json:"play_id"`
	PlayLabel   string      `json:"play_label"`
	AutoPlayID  string      `json:"auto_play_id" jsonschema:"The auto play whose moment matched."`
	PersonID    string      `json:"person_id" jsonschema:"Whose computer the run lands on; empty when there was nobody to run it on."`
	RunOn       RunOn       `json:"run_on" enum:"developer,tester,causer"`
	Moment      Moment      `json:"moment" enum:"ticket.unblocked,ticket.entered_stage,ticket.created,ticket.developer_set,ticket.tester_set,ticket.test_failed,doc.created,doc.changed"`
	Priority    Level       `json:"priority" enum:"high,normal,low"`
	Status      QueueStatus `json:"status" enum:"queued,dispatching,started,skipped,didnt_run,cancelled"`
	Reason      string      `json:"reason" jsonschema:"While queued: offline, ticket busy, paused, or empty while it waits for a free slot. Once skipped or didn't run: why."`
	TrailID     string      `json:"trail_id" jsonschema:"The run's trail once started, or the failed trail of a run that didn't run."`
	Via         Via         `json:"via" enum:"web,mcp"`
	QueuedAt    time.Time   `json:"queued_at"`
	DecidedAt   *time.Time  `json:"decided_at"`
	NotBefore   time.Time   `json:"not_before" jsonschema:"The item is not tried before this; an offline computer is tried again a minute later."`
}

// Queue is a target's auto runs as its page shows them: every item newest first, and whether the daily cap paused it.
type Queue struct {
	Items    []*QueueItem `json:"items"`
	Paused   bool         `json:"paused"`
	AutoRuns int          `json:"auto_runs"`
	DailyCap int          `json:"daily_cap"`
	// PausedUntil is when the oldest counted run leaves the rolling day, so a page can look again then; null unless paused.
	PausedUntil *time.Time `json:"paused_until"`
}

// QueueResumedEvent is the play.queue_resumed payload: the target's daily count of automatic runs starts again from ResumedAt.
type QueueResumedEvent struct {
	WorkspaceID string     `json:"workspace_id"`
	ProjectID   string     `json:"project_id"`
	TargetType  TargetType `json:"target_type" enum:"ticket,doc"`
	TargetID    string     `json:"target_id"`
	ResumedBy   string     `json:"resumed_by"`
	ResumedAt   time.Time  `json:"resumed_at"`
}

// GetQueue returns a target's queue, under the gate of its trails.
func (r *Runner) GetQueue(ctx context.Context, targetType TargetType, targetID string) (*Queue, error) {
	_, workspaceID, err := r.queueTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	targetID = strings.TrimSpace(targetID)
	items, err := r.queue.ListQueueByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("list the queue of %s %s: %w", targetType, targetID, err)
	}
	q := &Queue{Items: items}
	var frees time.Time
	q.AutoRuns, q.DailyCap, frees, err = r.autoRunsToday(ctx, workspaceID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	q.Paused = q.AutoRuns >= q.DailyCap
	if q.Paused {
		q.PausedUntil = &frees
	}
	return q, nil
}

// QueuedForPlay returns what waits for a play, in queue order, where the caller may open the project (autoplays:read).
func (r *Runner) QueuedForPlay(ctx context.Context, playID string) ([]*QueueItem, error) {
	playID = strings.TrimSpace(playID)
	if playID == "" {
		return nil, fmt.Errorf("%w: play id is required", apperrs.ErrInvalid)
	}
	if r.queue == nil {
		return nil, fmt.Errorf("%w: auto plays are not running on this server", apperrs.ErrNotFound)
	}
	play, err := r.plays.Get(ctx, playID)
	if err != nil {
		return nil, fmt.Errorf("get play %s: %w", playID, err)
	}
	actor := actorID(ctx)
	if !r.perm.HasPermission(ctx, actor, play.WorkspaceID, permissions.AutoplaysRead, "", "") {
		return nil, fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.AutoplaysRead)
	}
	items, err := r.queue.ListQueuedByPlay(ctx, play.ID)
	if err != nil {
		return nil, fmt.Errorf("list the queued runs of play %s: %w", play.ID, err)
	}
	visible := make([]*QueueItem, 0, len(items))
	for _, it := range items {
		if r.perm.HasPermission(ctx, actor, it.WorkspaceID, permissions.Member, resourceTypeProject, it.ProjectID) {
			visible = append(visible, it)
		}
	}
	return visible, nil
}

// CancelQueued drops a queued item before it starts: the person it runs on or an autoplays:write holder may.
func (r *Runner) CancelQueued(ctx context.Context, id string) (*QueueItem, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: queue item id is required", apperrs.ErrInvalid)
	}
	it, err := r.queue.GetQueueItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get queue item %s: %w", id, err)
	}
	if _, _, err := r.queueTarget(ctx, it.TargetType, it.TargetID); err != nil {
		return nil, err
	}
	actor := actorID(ctx)
	if actor != it.PersonID && !r.perm.HasPermission(ctx, actor, it.WorkspaceID, permissions.AutoplaysWrite, "", "") {
		return nil, fmt.Errorf("%w: only the person it runs on or a %s holder can cancel a queued run", apperrs.ErrForbidden, permissions.AutoplaysWrite)
	}
	if it.Status != QueueQueued {
		return nil, fmt.Errorf("%w: the run is %s, not queued", apperrs.ErrConflict, it.Status)
	}
	if err := r.decide(ctx, it, QueueQueued, QueueCancelled, "cancelled by "+r.login(ctx, actor)); err != nil {
		return nil, err
	}
	r.Kick()
	return it, nil
}

// ResumeAutoPlays restarts a paused target's daily count of automatic runs from now; an autoplays:write holder or the
// ticket's developer may.
func (r *Runner) ResumeAutoPlays(ctx context.Context, targetType TargetType, targetID string) (*Queue, error) {
	f, workspaceID, err := r.queueTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	targetID = strings.TrimSpace(targetID)
	actor := actorID(ctx)
	if (actor == "" || actor != f.Developer) && !r.perm.HasPermission(ctx, actor, workspaceID, permissions.AutoplaysWrite, "", "") {
		return nil, fmt.Errorf("%w: only the ticket's developer or a %s holder can resume auto plays", apperrs.ErrForbidden, permissions.AutoplaysWrite)
	}
	count, limit, _, err := r.autoRunsToday(ctx, workspaceID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if count < limit {
		return nil, fmt.Errorf("%w: auto plays are not paused here; %d of %d automatic runs today", apperrs.ErrConflict, count, limit)
	}
	now := r.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicQueueResumed, Payload: QueueResumedEvent{
		WorkspaceID: workspaceID, ProjectID: f.ProjectID, TargetType: targetType, TargetID: targetID, ResumedBy: actor, ResumedAt: now,
	}}
	if err := r.queue.Resume(ctx, targetType, targetID, actor, now, evt); err != nil {
		return nil, fmt.Errorf("resume auto plays on %s %s: %w", targetType, targetID, err)
	}
	r.Kick()
	return r.GetQueue(ctx, targetType, targetID)
}

// queueTarget reads a ticket or doc for the queue's surfaces and checks the caller may read its trails.
func (r *Runner) queueTarget(ctx context.Context, targetType TargetType, targetID string) (Facts, string, error) {
	targetID = strings.TrimSpace(targetID)
	if (targetType != TargetTicket && targetType != TargetDoc) || targetID == "" {
		return Facts{}, "", fmt.Errorf("%w: target type (ticket or doc) and target id are required", apperrs.ErrInvalid)
	}
	if r.queue == nil || r.facts == nil {
		return Facts{}, "", fmt.Errorf("%w: auto plays are not running on this server", apperrs.ErrNotFound)
	}
	f, err := r.facts.Facts(ctx, targetType, targetID)
	if err != nil {
		return Facts{}, "", fmt.Errorf("get %s %s: %w", targetType, targetID, err)
	}
	workspaceID, err := r.projects.WorkspaceForProject(ctx, f.ProjectID)
	if err != nil {
		return Facts{}, "", fmt.Errorf("resolve workspace for project %s: %w", f.ProjectID, err)
	}
	if !r.perm.HasPermission(ctx, actorID(ctx), workspaceID, permissions.Member, resourceTypeProject, f.ProjectID) {
		return Facts{}, "", fmt.Errorf("get %s %s: %w", targetType, targetID, apperrs.ErrNotFound)
	}
	if err := r.requireTrailAccess(ctx, workspaceID, targetType, targetID); err != nil {
		return Facts{}, "", err
	}
	return f, workspaceID, nil
}

// autoRunsToday counts the target's automatic runs in the rolling day, or since its resume, against the workspace's
// cap, and says when the oldest of them leaves the count.
func (r *Runner) autoRunsToday(ctx context.Context, workspaceID string, targetType TargetType, targetID string) (count, limit int, frees time.Time, err error) {
	limit, err = r.plays.AutoPlayDailyCap(ctx, workspaceID)
	if err != nil {
		return 0, 0, time.Time{}, fmt.Errorf("get the auto play daily cap of workspace %s: %w", workspaceID, err)
	}
	count, oldest, err := r.queue.CountAutoRuns(ctx, targetType, targetID, r.now().UTC().Add(-autoRunWindow))
	if err != nil {
		return 0, 0, time.Time{}, fmt.Errorf("count the automatic runs on %s %s: %w", targetType, targetID, err)
	}
	return count, limit, oldest.Add(autoRunWindow), nil
}

// decide moves it from one status to another with reason, publishing the change; ErrConflict when it already moved on.
func (r *Runner) decide(ctx context.Context, it *QueueItem, from, to QueueStatus, reason string) error {
	return r.move(ctx, it, from, to, reason, true)
}

// move is decide with the event optional, for the dispatcher's own steps nobody needs to hear about.
func (r *Runner) move(ctx context.Context, it *QueueItem, from, to QueueStatus, reason string, announce bool) error {
	prev := *it
	it.Status, it.Reason, it.DecidedAt = to, reason, nil
	if to != QueueQueued {
		now := r.now().UTC()
		it.DecidedAt = &now
	}
	var evts []eventbus.OutboxEvent
	if announce {
		evts = append(evts, r.queueEvent(TopicQueueUpdated, it))
	}
	err := r.queue.MoveQueueItem(ctx, it, from, evts...)
	if err != nil {
		*it = prev
		return fmt.Errorf("move queue item %s to %s: %w", it.ID, to, err)
	}
	return nil
}

func (r *Runner) queueEvent(topic string, it *QueueItem) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: *it}
}
