package tenancy

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Repo is the consumer-side persistence contract for workspaces; implemented in internal/platform/storage.
type Repo interface {
	Create(ctx context.Context, w *Workspace) error
	Update(ctx context.Context, w *Workspace, events ...eventbus.OutboxEvent) error
	Get(ctx context.Context, id string) (*Workspace, error)
	// GetBySlug returns apperrs.ErrNotFound when no workspace holds slug.
	GetBySlug(ctx context.Context, slug string) (*Workspace, error)
	// ListForUser returns the workspaces the given user is a member of, via workspace_members.
	ListForUser(ctx context.Context, userID string) ([]*Workspace, error)
	// ListWithRoles returns every workspace on the instance with its roles, Owner first; Roles are filled, CanManageMembers is not.
	ListWithRoles(ctx context.Context) ([]*TeamWorkspace, error)
}

// MemberRepo is the consumer-side persistence contract for workspace membership.
type MemberRepo interface {
	// AddMember adds the membership and the roles it holds.
	AddMember(ctx context.Context, m *Member, events ...eventbus.OutboxEvent) error
	// RoleIDFor reads the membership and returns its first role, "" for none; apperrs.ErrNotFound for a non-member.
	RoleIDFor(ctx context.Context, workspaceID, userID string) (string, error)
	// ListByWorkspace returns every member of workspaceID with their roles (Membership invites' roster).
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Member, error)
	RemoveMember(ctx context.Context, workspaceID, userID string, events ...eventbus.OutboxEvent) error
	// ChangeRoles gives userID the roles in add and takes away those in remove, in one write, or apperrs.ErrNotFound.
	ChangeRoles(ctx context.Context, workspaceID, userID string, add, remove []string, events ...eventbus.OutboxEvent) error
	// ListAllMemberships returns every membership on the instance with its workspace and roles.
	ListAllMemberships(ctx context.Context) ([]*TeamMembership, error)
	// Member returns userID's membership of workspaceID with their roles, or apperrs.ErrNotFound.
	Member(ctx context.Context, workspaceID, userID string) (*Member, error)
	// ProjectMemberIDs lists the members of workspaceID who are in projectID, in membership order, in one read.
	ProjectMemberIDs(ctx context.Context, workspaceID, projectID string) ([]string, error)
	// ProjectRoleIDs lists the roles projectID's rules let in, in the order they were listed, in one read.
	ProjectRoleIDs(ctx context.Context, projectID string) ([]string, error)
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
	GetByAcceptanceHash(ctx context.Context, acceptanceHash string, now time.Time) (*Invitation, error)
	List(ctx context.Context, actorID string, now time.Time) ([]*Invitation, error)
	Revoke(ctx context.Context, actorID, invitationID string, now time.Time, events ...eventbus.OutboxEvent) error
	Redeem(ctx context.Context, acceptanceHash string, identity InvitationIdentity, now time.Time, events ...eventbus.OutboxEvent) (*InvitationAdmission, error)
}

// RoleGate creates a new workspace's Owner role without tenancy importing roles (ADR 0017).
type RoleGate interface {
	// CreateOwnerRole creates workspaceID's Owner role and returns its id.
	CreateOwnerRole(ctx context.Context, workspaceID string) (roleID string, err error)
}

// PermissionGate answers instance-level checks without tenancy importing access (ADR 0017): whether userID holds
// action in any workspace they belong to (ADR 0087).
type PermissionGate interface {
	HoldsAnywhere(ctx context.Context, userID string, action permissions.Action) (bool, error)
}

// RoleNameGate resolves a role's display name without tenancy importing roles (ADR 0017).
type RoleNameGate interface {
	// RoleName returns the display name of roleID in workspaceID.
	RoleName(ctx context.Context, workspaceID, roleID string) (string, error)
	// IsOwnerRole reports whether roleID is workspaceID's protected singleton Owner role.
	IsOwnerRole(ctx context.Context, workspaceID, roleID string) (bool, error)
	// RolePermissions returns the actions roleID grants in workspaceID.
	RolePermissions(ctx context.Context, workspaceID, roleID string) (permissions.Set, error)
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

// AccountGate reads registered accounts without tenancy importing auth (ADR 0017).
type AccountGate interface {
	ListAccounts(ctx context.Context) ([]*TeamAccount, error)
	Account(ctx context.Context, userID string) (*TeamAccount, error)
	// Presence is who holds a live socket now and each account's latest session activity; absent means offline or signed out.
	Presence(ctx context.Context) (online map[string]bool, lastSeen map[string]time.Time, err error)
}

// WorkspacePermissionGate resolves permissions without tenancy importing access (ADR 0017).
type WorkspacePermissionGate interface {
	// WorkspacePermissions returns the workspace-scoped action names userID holds in workspaceID.
	WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string
}

// ProjectGate reads where a project lives and what someone holds inside it, without tenancy importing workspace or
// access (ADR 0017).
type ProjectGate interface {
	// ProjectWorkspace returns the workspace projectID belongs to, or apperrs.ErrNotFound.
	ProjectWorkspace(ctx context.Context, projectID string) (string, error)
	// ProjectPermissions is every action userID holds inside projectID; opens is false when it is hidden from them.
	ProjectPermissions(ctx context.Context, userID, projectID string) (actions []string, opens bool)
	// WorkspaceProjects lists the ids of workspaceID's projects.
	WorkspaceProjects(ctx context.Context, workspaceID string) ([]string, error)
	// ProjectActions is each project of workspaceID userID is in, with the project-area actions they hold there.
	ProjectActions(ctx context.Context, userID, workspaceID string) ([]MeProject, error)
}

// ChannelGate creates a new workspace's #general channel without tenancy importing chat (ADR 0017).
type ChannelGate interface {
	// CreateGeneralChannel idempotently ensures workspaceID has its #general channel, attributed to creatorUserID.
	CreateGeneralChannel(ctx context.Context, workspaceID, creatorUserID string) error
}

// DefaultsGate seeds a new workspace's template roles, default plays and default automations without tenancy importing
// any of them (ADR 0017).
type DefaultsGate interface {
	// SeedWorkspaceDefaults creates workspaceID's template roles, default plays and default automations; idempotent.
	SeedWorkspaceDefaults(ctx context.Context, workspaceID string) error
}
