package roles

import "context"

// Repo is the consumer-side persistence contract for roles.
type Repo interface {
	Create(ctx context.Context, r *Role) error
	Get(ctx context.Context, id string) (*Role, error)
	List(ctx context.Context, workspaceID string) ([]*Role, error)
	Update(ctx context.Context, r *Role) error
	Delete(ctx context.Context, id string) error
}

// MemberGate lets Create/Update/Delete check ActionManageRoles/Owner without roles importing tenancy (ADR 0017).
type MemberGate interface {
	// MemberRoleID returns the role id userID holds in workspaceID.
	MemberRoleID(ctx context.Context, workspaceID, userID string) (string, error)
}
