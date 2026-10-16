package plays

import (
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

// Why a queued item waits; a decided item's reason is a sentence for the ticket instead. A gap in its person's computer
// setup waits under the harness refusal's own reason (unpaired, expired_token, no_default, no_default_computer,
// setup_required) until it is fixed.
const (
	ReasonOffline    = "offline"
	ReasonTicketBusy = "ticket busy"
	ReasonPaused     = "paused"
	// ReasonNoFreeSlot: its person already has as many runs going as their queue lets go at once.
	ReasonNoFreeSlot = "no free slot"
)

// QueueItem is one match of an auto play's moment on a ticket or doc, or an automation's runPlay call (ADR 0132),
// queued on the person it runs on and kept once decided, so the target shows it and the daily cap counts it.
type QueueItem struct {
	ID           string      `json:"id"`
	WorkspaceID  string      `json:"workspace_id"`
	ProjectID    string      `json:"project_id"`
	TargetType   TargetType  `json:"target_type" enum:"ticket,doc,stack"`
	TargetID     string      `json:"target_id"`
	PlayID       string      `json:"play_id"`
	PlayLabel    string      `json:"play_label"`
	AutoPlayID   string      `json:"auto_play_id" jsonschema:"The auto play whose moment matched; empty for a run an automation queued."`
	AutomationID string      `json:"automation_id" jsonschema:"The automation whose runPlay queued the run; empty for an auto play's."`
	PersonID     string      `json:"person_id" jsonschema:"Whose computer the run lands on; empty when there was nobody to run it on."`
	RunOn        RunOn       `json:"run_on" enum:"developer,tester,causer,computer"`
	Moment       Moment      `json:"moment" enum:"ticket.unblocked,ticket.entered_stage,ticket.created,ticket.developer_set,ticket.tester_set,ticket.test_failed,doc.created,doc.changed,stack.service_crashed,stack.service_recovered,stack.error_rate_crossed,automation"`
	Priority     Level       `json:"priority" enum:"high,normal,low"`
	Status       QueueStatus `json:"status" enum:"queued,dispatching,started,skipped,didnt_run,cancelled"`
	Reason       string      `json:"reason" jsonschema:"While queued: offline, ticket busy, paused, no free slot, a gap in its person's computer setup (unpaired, expired_token, no_default, no_default_computer, setup_required), or empty until first tried. Once skipped or didn't run: why."`
	TrailID      string      `json:"trail_id" jsonschema:"The run's trail once started, or the failed trail of a run that didn't run."`
	Via          Via         `json:"via" enum:"web,mcp,automation"`
	QueuedAt     time.Time   `json:"queued_at"`
	DecidedAt    *time.Time  `json:"decided_at"`
	NotBefore    time.Time   `json:"not_before" jsonschema:"The item is not tried before this; an offline computer or a setup gap is tried again a minute later."`
	ComputerID   string      `json:"computer_id" jsonschema:"The computer it runs on: the one its auto play names for run_on computer, else the one its person picked when several are paired and none is the default; empty until picked."`
	Service      string      `json:"service" jsonschema:"For a stack moment, the service that went down or came back; empty otherwise."`
	OutageID     string      `json:"outage_id" jsonschema:"For a stack moment, the outage it opened or closed; empty otherwise."`
	// HarnessProjectID is the T3 project picked with ComputerID; like a trail's, it stays off the wire.
	HarnessProjectID string `json:"-"`
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
	TargetType  TargetType `json:"target_type" enum:"ticket,doc,stack"`
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
	q := &Queue{Items: r.seenItems(ctx, workspaceID, items)}
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
	if !r.perm.HasPermission(ctx, actor, play.WorkspaceID, permissions.PlaysRead, resourceTypePlay, play.ID) {
		return nil, fmt.Errorf("get play %s: %w", playID, apperrs.ErrNotFound)
	}
	if !r.perm.HasPermission(ctx, actor, play.WorkspaceID, permissions.AutoplaysRead, "", "") {
		return nil, fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.AutoplaysRead)
	}
	projectIDs, all, err := r.perm.CallerProjects(ctx, permissions.Member)
	if err != nil {
		return nil, fmt.Errorf("list the projects the caller may open: %w", err)
	}
	items, err := r.queue.ListQueuedByPlay(ctx, play.ID, ProjectScope{All: all, ProjectIDs: projectIDs})
	if err != nil {
		return nil, fmt.Errorf("list the queued runs of play %s: %w", play.ID, err)
	}
	return items, nil
}

// CancelQueued drops a queued item before it starts: the person it runs on or an autoplays:write holder may.
func (r *Runner) CancelQueued(ctx context.Context, id string) (*QueueItem, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: queue item id is required", apperrs.ErrInvalid)
	}
	it, err := r.queuedItem(ctx, id)
	if err != nil {
		return nil, err
	}
	actor := actorID(ctx)
	if actor != it.PersonID && !r.perm.HasPermission(ctx, actor, it.WorkspaceID, permissions.AutoplaysWrite, "", "") {
		return nil, fmt.Errorf("%w: only the person it runs on or an %s holder can cancel a queued run", apperrs.ErrForbidden, permissions.AutoplaysWrite)
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

// PickComputer runs a waiting item on computerID and the T3 project on it, replacing an earlier pick, for the person it
// lands on only, and saves the pick as their link for its project, so their later runs there need no pick; their
// defaults stay as they are. Once the dispatcher has claimed the run, the pick is refused, never dropped.
func (r *Runner) PickComputer(ctx context.Context, id, computerID, harnessProjectID string) (*QueueItem, error) {
	id, computerID, harnessProjectID = strings.TrimSpace(id), strings.TrimSpace(computerID), strings.TrimSpace(harnessProjectID)
	if id == "" || computerID == "" || harnessProjectID == "" {
		return nil, fmt.Errorf("%w: a queued run's id, a computer, and a T3 project on it are required", apperrs.ErrInvalid)
	}
	it, err := r.queuedItem(ctx, id)
	if err != nil {
		return nil, err
	}
	if actorID(ctx) != it.PersonID {
		return nil, fmt.Errorf("%w: only the person a queued run lands on can pick its computer", apperrs.ErrForbidden)
	}
	if it.Status != QueueQueued {
		return nil, pickRefused(it.Status)
	}
	it.ComputerID, it.HarnessProjectID, it.Reason, it.NotBefore = computerID, harnessProjectID, "", r.now().UTC()
	err = r.queue.PickQueueItem(ctx, it, r.queueEvent(TopicQueueUpdated, it))
	if errors.Is(err, apperrs.ErrConflict) {
		return nil, pickRefused(QueueDispatching)
	}
	if err != nil {
		return nil, fmt.Errorf("pick a computer for queue item %s: %w", id, err)
	}
	r.rememberPick(ctx, it)
	r.Kick()
	return it, nil
}

// pickRefused says why a run that left the queue takes no pick; a run that left it mid-pick was claimed to start.
func pickRefused(status QueueStatus) error {
	if status == QueueStarted {
		return fmt.Errorf("%w: the run already started, so its computer can no longer change", apperrs.ErrConflict)
	}
	if status == QueueSkipped || status == QueueDidntRun || status == QueueCancelled {
		return fmt.Errorf("%w: the run is %s, not waiting", apperrs.ErrConflict, status)
	}
	return fmt.Errorf("%w: the run is already starting, so its computer can no longer change", apperrs.ErrConflict)
}

// rememberPick saves a picked item's location as its person's project link and makes their held runs due, so those
// resolve through it; a failed save leaves the pick on this item alone.
func (r *Runner) rememberPick(ctx context.Context, it *QueueItem) {
	if r.checkouts == nil {
		return
	}
	if err := r.checkouts.SaveLink(ctx, it.PersonID, it.ProjectID, it.ComputerID, it.HarnessProjectID); err != nil {
		r.log.Warn("plays: save a picked computer as the project link failed", "queue_item", it.ID, "project", it.ProjectID, "error", err)
		return
	}
	if err := r.queue.WakePerson(ctx, it.PersonID, r.now().UTC()); err != nil {
		r.log.Warn("plays: wake the queue after a pick failed", "queue_item", it.ID, "person", it.PersonID, "error", err)
	}
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
		return nil, fmt.Errorf("%w: only the ticket's developer or an %s holder can resume auto plays", apperrs.ErrForbidden, permissions.AutoplaysWrite)
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

// queuedItem reads a queue item whose target and play the caller can see; any other is not found.
func (r *Runner) queuedItem(ctx context.Context, id string) (*QueueItem, error) {
	it, err := r.queue.GetQueueItem(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get queue item %s: %w", id, err)
	}
	_, workspaceID, err := r.queueTarget(ctx, it.TargetType, it.TargetID)
	if err != nil {
		return nil, err
	}
	if len(r.seenItems(ctx, workspaceID, []*QueueItem{it})) == 0 {
		return nil, fmt.Errorf("get queue item %s: %w", id, apperrs.ErrNotFound)
	}
	return it, nil
}

// seenItems keeps the items of plays the caller can see, from one read of the rules on them.
func (r *Runner) seenItems(ctx context.Context, workspaceID string, items []*QueueItem) []*QueueItem {
	playIDs := make([]string, 0, len(items))
	for _, it := range items {
		playIDs = append(playIDs, it.PlayID)
	}
	sees := r.perm.PlaysWith(ctx, actorID(ctx), workspaceID, permissions.PlaysRead, playIDs)
	return slices.DeleteFunc(slices.Clone(items), func(it *QueueItem) bool { return !sees[it.PlayID] })
}

// queueTarget reads a ticket or doc for the queue's surfaces and checks the caller may read its trails.
func (r *Runner) queueTarget(ctx context.Context, targetType TargetType, targetID string) (Facts, string, error) {
	targetID = strings.TrimSpace(targetID)
	if (targetType != TargetTicket && targetType != TargetDoc && targetType != TargetStack) || targetID == "" {
		return Facts{}, "", fmt.Errorf("%w: target type (ticket, doc, or stack) and target id are required", apperrs.ErrInvalid)
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
	if err := r.requireThread(ctx, workspaceID, targetType, targetID); err != nil {
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
