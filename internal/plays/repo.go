package plays

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Repo is the consumer-side persistence contract for plays; implemented in internal/platform/storage.
type Repo interface {
	Create(ctx context.Context, p *Play, evts ...eventbus.OutboxEvent) error
	// Get returns a play by id regardless of workspace; the use-case checks WorkspaceID so a lookup
	// across workspaces reports not found instead of leaking another workspace's play.
	Get(ctx context.Context, id string) (*Play, error)
	List(ctx context.Context, workspaceID string) ([]*Play, error)
	Update(ctx context.Context, p *Play, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	// DecisionsCheckEnabled and SetDecisionsCheckEnabled hold the built-in decisions check's per-workspace switch.
	DecisionsCheckEnabled(ctx context.Context, workspaceID string) (bool, error)
	SetDecisionsCheckEnabled(ctx context.Context, workspaceID string, enabled bool) error
	AutoPlayRepo
}

// AutoPlayRepo persists a play's auto plays, which go with their play, and the workspace's daily cap on them.
type AutoPlayRepo interface {
	CreateAutoPlay(ctx context.Context, a *AutoPlay, evts ...eventbus.OutboxEvent) error
	GetAutoPlay(ctx context.Context, id string) (*AutoPlay, error)
	// ListAutoPlays returns the auto plays of every play named in one read, each play's oldest first.
	ListAutoPlays(ctx context.Context, playIDs []string) ([]*AutoPlay, error)
	UpdateAutoPlay(ctx context.Context, a *AutoPlay, evts ...eventbus.OutboxEvent) error
	DeleteAutoPlay(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	AutoPlayDailyCap(ctx context.Context, workspaceID string) (int, error)
	SetAutoPlayDailyCap(ctx context.Context, workspaceID string, limit int, evts ...eventbus.OutboxEvent) error
	// ListEnabledAutoPlays returns a workspace's switched-on auto plays waiting for moment, oldest first.
	ListEnabledAutoPlays(ctx context.Context, workspaceID string, moment Moment) ([]*AutoPlay, error)
}

// QueueRepo persists the run queue (ADR 0132): one row per match of an auto play, kept once decided.
type QueueRepo interface {
	// EnqueueRun inserts it with evts, or writes nothing and reports false while the same auto play already waits on the target.
	EnqueueRun(ctx context.Context, it *QueueItem, evts ...eventbus.OutboxEvent) (bool, error)
	GetQueueItem(ctx context.Context, id string) (*QueueItem, error)
	// MoveQueueItem saves it's status, reason, trail, and times if the row is still in status from; ErrConflict otherwise.
	MoveQueueItem(ctx context.Context, it *QueueItem, from QueueStatus, evts ...eventbus.OutboxEvent) error
	// ListQueueByTarget returns a target's items, newest first.
	ListQueueByTarget(ctx context.Context, targetType TargetType, targetID string) ([]*QueueItem, error)
	// Resume restarts the target's count of automatic runs from at.
	Resume(ctx context.Context, targetType TargetType, targetID, by string, at time.Time, evts ...eventbus.OutboxEvent) error
	QueueDispatchRepo
}

// QueueDispatchRepo is what the queue's dispatcher and its limits read.
type QueueDispatchRepo interface {
	// QueuedPeople returns everyone with a queued item due by now.
	QueuedPeople(ctx context.Context, now time.Time) ([]string, error)
	// ListDue returns a person's queued items due by now, highest priority then oldest first.
	ListDue(ctx context.Context, personID string, now time.Time) ([]*QueueItem, error)
	// NextNotBefore is the earliest not_before of a queued item still held back after now; false when there is none.
	NextNotBefore(ctx context.Context, after time.Time) (time.Time, bool, error)
	// CountActiveRuns counts a person's starting, running, and waiting trails, pressed or automatic.
	CountActiveRuns(ctx context.Context, personID string) (int, error)
	// CountAutoRuns counts the target's started items since since or its last resume, whichever is later, with the oldest's time.
	CountAutoRuns(ctx context.Context, targetType TargetType, targetID string, since time.Time) (int, time.Time, error)
	// QueuedSince reports whether autoPlayID queued, or started, a run on the target at or after since.
	QueuedSince(ctx context.Context, autoPlayID string, targetType TargetType, targetID string, since time.Time) (bool, error)
	// RecoverDispatching settles items a crash left dispatching: started when their trail exists, queued again otherwise.
	RecoverDispatching(ctx context.Context) error
}

// TrailRepo is the consumer-side persistence contract for trails; implemented in internal/platform/storage.
type TrailRepo interface {
	// CreateTrail and UpdateTrail write the given outbox events in the trail's own transaction (ADR 0044).
	CreateTrail(ctx context.Context, t *Trail, evts ...eventbus.OutboxEvent) error
	GetTrail(ctx context.Context, id string) (*Trail, error)
	// UpdateTrail persists the run's moving parts: state, session, activity, and the terminal fields.
	UpdateTrail(ctx context.Context, t *Trail, evts ...eventbus.OutboxEvent) error
	// ListTrailsByTarget returns newest first.
	ListTrailsByTarget(ctx context.Context, targetType TargetType, targetID string) ([]*Trail, error)
	// ListActiveTrailsByTargets returns the starting, running, or waiting trails on any of the given targets.
	ListActiveTrailsByTargets(ctx context.Context, targetType TargetType, targetIDs []string) ([]*Trail, error)
	// ListRunningTrails returns the starting or running trails across every workspace.
	ListRunningTrails(ctx context.Context) ([]*Trail, error)
	// LatestTrailForChoices returns the starter's newest trail of one play in one project, ErrNotFound when none.
	LatestTrailForChoices(ctx context.Context, starterID, playID, projectID string) (*Trail, error)
	// LatestTrailInConversation returns the newest trail that ran in a conversation, ErrNotFound when none.
	LatestTrailInConversation(ctx context.Context, conversationID string) (*Trail, error)
}

// PermissionGate is the consumer-side slice of access's HasPermission check (ADR 0017); resourceType and
// resourceID are empty for a workspace-wide check and name a play or doc for a per-resource overwrite.
type PermissionGate interface {
	HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action, resourceType, resourceID string) bool
}
