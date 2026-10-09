package access

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Repo is the consumer-side persistence contract for permission overwrites.
type Repo interface {
	// Get returns the overwrite row for (resourceType, resourceID, userID), or ErrNotFound when absent.
	Get(ctx context.Context, resourceType, resourceID, userID string) (*Overwrite, error)
	// GetMany returns userID's overwrite on each of resourceIDs that has one, keyed by resource id, in one read.
	GetMany(ctx context.Context, resourceType string, resourceIDs []string, userID string) (map[string]*Overwrite, error)
	// ListByResource returns every overwrite on one resource instance (e.g. every per-user grant on a document).
	ListByResource(ctx context.Context, resourceType, resourceID string) ([]*Overwrite, error)
	// Set upserts the allow/deny sets for (resourceType, resourceID, userID); both empty deletes the row.
	Set(ctx context.Context, resourceType, resourceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error
	// DeleteByResource removes every overwrite on one resource instance (doc delete/restore path).
	DeleteByResource(ctx context.Context, resourceType, resourceID string) error
	// HasAllowAny gates the user directory for permissions:write holders.
	HasAllowAny(ctx context.Context, resourceType, userID string, action permissions.Action) (bool, error)
}

// Users lets access use the auth user store without importing auth (ADR 0017).
type Users interface {
	ListUsers(ctx context.Context) ([]*User, error)
}

// RoleResolver feeds HasPermission's Owner-bypass and role-set layers without importing roles/tenancy (ADR 0017).
type RoleResolver interface {
	// MemberRole returns ErrNotFound if userID isn't a member of workspaceID.
	MemberRole(ctx context.Context, workspaceID, userID string) (RoleInfo, error)
}

// DocScope is where a doc lives: its project and that project's workspace, both empty for a doc in no project.
type DocScope struct {
	WorkspaceID string
	ProjectID   string
}

// DocWorkspaceResolver resolves docs' projects and workspaces so HasPermission applies to docs too.
type DocWorkspaceResolver interface {
	// DocScopes returns the scope of each doc in docIDs it knows, in one read; an unknown doc is left out.
	DocScopes(ctx context.Context, docIDs []string) (map[string]DocScope, error)
}

// Scopes resolves where a checked entity lives straight from storage, never through a gated use-case (ADR 0017).
type Scopes interface {
	// WorkspaceIDForProject returns ErrNotFound for an unknown project.
	WorkspaceIDForProject(ctx context.Context, projectID string) (string, error)
	// ProjectIDs lists the projects in workspaceID.
	ProjectIDs(ctx context.Context, workspaceID string) ([]string, error)
	// UnrestrictedWorkspaceIDsForUser lists the workspaces userID is a member of and not a Restricted member of.
	UnrestrictedWorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error)
	// WorkspaceIDsForUser lists every workspace userID is a member of, a Restricted membership included.
	WorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error)
}

// PlayWorkspaceResolver resolves a play's own workspace so HasPermission/canManage apply to plays too (ADR 0017).
type PlayWorkspaceResolver interface {
	// WorkspaceIDForPlay returns "" if it can't be resolved (unknown play).
	WorkspaceIDForPlay(ctx context.Context, playID string) (string, error)
}
