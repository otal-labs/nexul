package access

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Repo is the consumer-side persistence contract for permission overwrites.
type Repo interface {
	// Get returns the rule naming userID on (resourceType, resourceID), or ErrNotFound when absent.
	Get(ctx context.Context, resourceType, resourceID, userID string) (*Overwrite, error)
	// GetRole returns the rule naming roleID on (resourceType, resourceID), or ErrNotFound when absent.
	GetRole(ctx context.Context, resourceType, resourceID, roleID string) (*Overwrite, error)
	// Rules returns the rules naming userID or a role they hold on each of resourceIDs that has any, keyed by
	// resource id, in one read.
	Rules(ctx context.Context, resourceType string, resourceIDs []string, userID string) (map[string]*Rules, error)
	// RulesOfType returns every rule on a resourceType naming userID or a role they hold, keyed by resource id, in one
	// read.
	RulesOfType(ctx context.Context, resourceType, userID string) (map[string]*Rules, error)
	// DocRules returns the rules naming userID or a role they hold on each of docIDs and its folder, by doc id, in one read.
	DocRules(ctx context.Context, docIDs []string, userID string) (map[string]*DocRules, error)
	// DocRuleTurns decides in one read the folders and docs in ruled whose rules for userID turn readable's answer.
	DocRuleTurns(ctx context.Context, userID string, action permissions.Action, ruled, readable []string) (permissions.DocReach, error)
	// StackRuleTurns decides in one read the stacks of the projects in ruled whose rules for userID or a role they hold in
	// its workspace turn readable's answer for action, as RequireStack decides it: allowed outside readable, denied in it.
	StackRuleTurns(ctx context.Context, userID string, action permissions.Action, ruled, readable []string) (allowed, denied []string, err error)
	// OwnRules returns the rules naming userID on each of projectIDs that has one, keyed by project id, in one read.
	OwnRules(ctx context.Context, userID string, projectIDs []string) (map[string]*Overwrite, error)
	// RoleRules returns the rules for roleIDs on each of resourceIDs that has any, keyed by resource id in role order,
	// in one read.
	RoleRules(ctx context.Context, resourceType string, resourceIDs, roleIDs []string) (map[string][]*Overwrite, error)
	// ListByResource returns every rule on one resource instance, people's and roles'.
	ListByResource(ctx context.Context, resourceType, resourceID string) ([]*Overwrite, error)
	// Set upserts the allow/deny sets of userID's rule on (resourceType, resourceID); both empty deletes the row.
	Set(ctx context.Context, resourceType, resourceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error
	// SetRole is Set for the rule naming roleID.
	SetRole(ctx context.Context, resourceType, resourceID, roleID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error
	// DeleteByResource removes every overwrite on one resource instance (doc delete/restore path).
	DeleteByResource(ctx context.Context, resourceType, resourceID string) error
	// HasAllowAny reports whether a rule on a resourceType naming userID or a role they hold allows action; it gates
	// the user directory for permissions:write holders.
	HasAllowAny(ctx context.Context, resourceType, userID string, action permissions.Action) (bool, error)
	// Grid reads what workspaceID's Access page is built from in the same statements at any size.
	Grid(ctx context.Context, workspaceID string) (*GridSource, error)
	// SaveGrid stores rules, deleting each whose sets are both empty, and with finishedBy set marks workspaceID's access
	// setup finished by them, with events, in one transaction of the same statements at any size.
	SaveGrid(ctx context.Context, workspaceID string, rules []*Overwrite, finishedBy string, events ...eventbus.OutboxEvent) error
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

// DocScope is where a doc or folder lives: its workspace, its project, and a doc's folder; empty outside any project.
type DocScope struct {
	WorkspaceID string
	ProjectID   string
	FolderID    string
}

// DocWorkspaceResolver resolves docs' projects and workspaces so HasPermission applies to docs too.
type DocWorkspaceResolver interface {
	// DocScopes returns the scope of each doc in docIDs it knows, in one read; an unknown doc is left out.
	DocScopes(ctx context.Context, docIDs []string) (map[string]DocScope, error)
	// FolderScope returns where a doc folder lives; ErrNotFound for an unknown folder.
	FolderScope(ctx context.Context, folderID string) (DocScope, error)
}

// Scopes resolves where a checked entity lives straight from storage, never through a gated use-case (ADR 0017).
type Scopes interface {
	// WorkspaceIDForProject returns ErrNotFound for an unknown project.
	WorkspaceIDForProject(ctx context.Context, projectID string) (string, error)
	// ProjectIDs lists the projects in workspaceID.
	ProjectIDs(ctx context.Context, workspaceID string) ([]string, error)
	// WorkspaceIDsForUser lists every workspace userID is a member of.
	WorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error)
}

// StackScope is where a stack lives: its project and that project's workspace, both empty for an instance stack, and the
// base stack a branch deployment follows, and the name of the machine it deploys to.
type StackScope struct {
	WorkspaceID string
	ProjectID   string
	BaseID      string
	Machine     string
}

// StackScopeResolver resolves where a stack lives straight from storage, so its rules are written and explained there.
type StackScopeResolver interface {
	// StackScope returns ErrNotFound for an unknown stack.
	StackScope(ctx context.Context, stackID string) (StackScope, error)
}

// RoleWorkspaceResolver resolves a role's workspace, so a rule names only a role of its entity's workspace (ADR 0017).
type RoleWorkspaceResolver interface {
	// WorkspaceIDForRole returns ErrNotFound for an unknown role.
	WorkspaceIDForRole(ctx context.Context, roleID string) (string, error)
	// HeldRole returns what roleID allows and whether it is the Owner role; ErrNotFound for an unknown role.
	HeldRole(ctx context.Context, roleID string) (HeldRole, error)
}

// PlayWorkspaceResolver resolves a play's own workspace so HasPermission/canManage apply to plays too (ADR 0017).
type PlayWorkspaceResolver interface {
	// WorkspaceIDForPlay returns "" if it can't be resolved (unknown play).
	WorkspaceIDForPlay(ctx context.Context, playID string) (string, error)
}

// ChannelScope is where a channel lives and, before its rules, who reads it: every member, or a private one's members.
type ChannelScope struct {
	WorkspaceID string
	Private     bool
	MemberIDs   []string
}

// ChannelWorkspaceResolver resolves a channel's scope so a rule on it is written and explained there (ADR 0017).
type ChannelWorkspaceResolver interface {
	// ChannelScope returns ErrNotFound for an unknown conversation or one that is not a text or voice channel.
	ChannelScope(ctx context.Context, channelID string) (ChannelScope, error)
}

// Names resolves what a source names for the sentence a person reads, read only on the explain and refusal paths.
type Names interface {
	// Name returns the name of the workspace, project, doc, play, role, or person with id: a title, a label, a login.
	Name(ctx context.Context, entityType, id string) (string, error)
}
