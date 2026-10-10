package plays

import (
	"context"

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
	SetAutoPlayDailyCap(ctx context.Context, workspaceID string, limit int) error
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
