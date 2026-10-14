package roles

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo is the consumer-side persistence contract for roles.
type Repo interface {
	Create(ctx context.Context, r *Role) error
	Get(ctx context.Context, id string) (*Role, error)
	List(ctx context.Context, workspaceID string) ([]*Role, error)
	Update(ctx context.Context, r *Role, events ...eventbus.OutboxEvent) error
	// Delete removes the role and, by cascade, every holding of it and every rule naming it.
	Delete(ctx context.Context, id string, events ...eventbus.OutboxEvent) error
}

// MemberGate answers whether someone belongs to a workspace without roles importing tenancy (ADR 0017).
type MemberGate interface {
	// RequireMember returns apperrs.ErrNotFound unless userID is a member of workspaceID.
	RequireMember(ctx context.Context, workspaceID, userID string) error
	// HolderEvents returns a member update for each member of workspaceID holding roleID, written with the role's change.
	HolderEvents(ctx context.Context, workspaceID, roleID, actorID string) ([]eventbus.OutboxEvent, error)
}

// PermissionGate reads what a user holds in a workspace without roles importing access (ADR 0017), so a role never
// grants more than whoever writes it holds (ADR 0088).
type PermissionGate interface {
	// WorkspacePermissions returns every action userID holds in workspaceID; an Owner's is every action.
	WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string
	// GrantablePermissions is WorkspacePermissions without the instance-area actions userID does not hold at instance
	// level, which an Owner of the workspace does not hold by being its Owner (ADR 0151).
	GrantablePermissions(ctx context.Context, userID, workspaceID string) []string
}
