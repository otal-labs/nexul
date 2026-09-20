package tenancy

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo is the consumer-side persistence contract for workspaces; implemented in internal/platform/storage.
type Repo interface {
	Create(ctx context.Context, w *Workspace) error
	Update(ctx context.Context, w *Workspace) error
	Get(ctx context.Context, id string) (*Workspace, error)
	// ListForUser returns the workspaces the given user is a member of, via workspace_members.
	ListForUser(ctx context.Context, userID string) ([]*Workspace, error)
}

// MemberRepo is the consumer-side persistence contract for workspace membership.
type MemberRepo interface {
	AddMember(ctx context.Context, m *Member) error
	// RoleIDFor returns the role id userID holds in workspaceID, or apperrs.ErrNotFound if userID isn't a member.
	RoleIDFor(ctx context.Context, workspaceID, userID string) (string, error)
	// ListByWorkspace returns every member of workspaceID (Membership invites' roster).
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Member, error)
	RemoveMember(ctx context.Context, workspaceID, userID string) error
	SetRole(ctx context.Context, workspaceID, userID, roleID string) error
}

// InviteRepo is the consumer-side persistence contract for pending workspace invites.
type InviteRepo interface {
	// Upsert inserts a pending invite, or updates its role_id if one already exists for (WorkspaceID, Login).
	Upsert(ctx context.Context, inv *Invite) error
	Delete(ctx context.Context, workspaceID, login string) error
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Invite, error)
	ListByLogin(ctx context.Context, login string) ([]*Invite, error)
}

type InvitationRepo interface {
	Create(ctx context.Context, invitation *Invitation, tokenHash string, events ...eventbus.OutboxEvent) error
	GetByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*Invitation, error)
	List(ctx context.Context, actorID string, now time.Time) ([]*Invitation, error)
	Revoke(ctx context.Context, actorID, invitationID string, now time.Time, events ...eventbus.OutboxEvent) error
	Redeem(ctx context.Context, acceptanceHash string, identity InvitationIdentity, now time.Time, events ...eventbus.OutboxEvent) (*InvitationAdmission, error)
}

// RoleGate creates a new workspace's Owner role without tenancy importing roles (ADR 0017).
type RoleGate interface {
	// CreateOwnerRole creates workspaceID's Owner role and returns its id.
	CreateOwnerRole(ctx context.Context, workspaceID string) (roleID string, err error)
}

// PermissionGate checks can_create_workspace without tenancy importing auth (ADR 0017).
type PermissionGate interface {
	CanCreateWorkspace(ctx context.Context, userID string) (bool, error)
}

// RoleNameGate resolves a role's display name without tenancy importing roles (ADR 0017).
type RoleNameGate interface {
	// RoleName returns the display name of roleID in workspaceID.
	RoleName(ctx context.Context, workspaceID, roleID string) (string, error)
	// IsOwnerRole reports whether roleID is workspaceID's protected singleton Owner role.
	IsOwnerRole(ctx context.Context, workspaceID, roleID string) (bool, error)
}

// AllowlistGate checks the sign-in allowlist without tenancy importing auth (ADR 0017).
type AllowlistGate interface {
	IsAllowlisted(ctx context.Context, login string) (bool, error)
}

// UserLookupGate resolves users without tenancy importing auth (ADR 0017).
type UserLookupGate interface {
	UserIDForLogin(ctx context.Context, login string) (userID string, found bool, err error)
	LoginForUserID(ctx context.Context, userID string) (string, error)
}

// WorkspacePermissionGate resolves permissions without tenancy importing access (ADR 0017).
type WorkspacePermissionGate interface {
	// WorkspacePermissions returns the workspace-scoped action names userID holds in workspaceID.
	WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string
}

// ChannelGate creates a new workspace's #general channel without tenancy importing chat (ADR 0017).
type ChannelGate interface {
	// CreateGeneralChannel idempotently ensures workspaceID has its #general channel, attributed to creatorUserID.
	CreateGeneralChannel(ctx context.Context, workspaceID, creatorUserID string) error
}

// PlaysGate seeds a new workspace's two default plays without tenancy importing plays (ADR 0017).
type PlaysGate interface {
	// SeedDefaultPlays creates "Fix with AI" and "To tickets via AI" for workspaceID (ticket 02).
	SeedDefaultPlays(ctx context.Context, workspaceID string) error
}
