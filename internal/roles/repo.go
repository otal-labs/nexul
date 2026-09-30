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
	Delete(ctx context.Context, id string) error
}

// MemberGate lets Create/Update/Delete check ActionManageRoles/Owner without roles importing tenancy (ADR 0017).
type MemberGate interface {
	// MemberRoleID returns the role id userID holds in workspaceID.
	MemberRoleID(ctx context.Context, workspaceID, userID string) (string, error)
}

// PermissionGate reads what a user holds in a workspace without roles importing access (ADR 0017), so a role never
// grants more than whoever writes it holds (ADR 0088).
type PermissionGate interface {
	// WorkspacePermissions returns every action userID holds in workspaceID; an Owner's is every action.
	WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string
}
