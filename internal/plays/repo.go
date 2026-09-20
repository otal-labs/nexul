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
	// ListActiveTrailsByTargets returns the starting or running trails on any of the given targets.
	ListActiveTrailsByTargets(ctx context.Context, targetType TargetType, targetIDs []string) ([]*Trail, error)
	// LatestTrailForChoices returns the starter's newest trail of one play in one project, ErrNotFound when none.
	LatestTrailForChoices(ctx context.Context, starterID, playID, projectID string) (*Trail, error)
}

// PermissionGate is the consumer-side slice of access's HasPermission check (ADR 0017); resourceType and
// resourceID are empty for a workspace-wide check and name a play or doc for a per-resource overwrite.
type PermissionGate interface {
	HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action, resourceType, resourceID string) bool
}
