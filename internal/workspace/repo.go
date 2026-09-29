package workspace

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Gate is the permission check project, board-setting, and ticket-move use-cases ask (the access domain, ADR 0042).
type Gate interface {
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// TicketProjects names the project a ticket sits in, so moving it is checked there too (ADR 0017).
type TicketProjects interface {
	ProjectOfTicket(ctx context.Context, ticketID string) (string, error)
}

// WorkspaceGate lets workspace check existence without importing tenancy (ADR 0017).
type WorkspaceGate interface {
	WorkspaceExists(ctx context.Context, id string) (bool, error)
}

// Repo is the consumer-side persistence contract for workspace's projects.
type Repo interface {
	Create(ctx context.Context, p *Project) error
	Get(ctx context.Context, id string) (*Project, error)
	List(ctx context.Context, workspaceID string) ([]*Project, error)
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, ids []string) error
	CountTickets(ctx context.Context, projectID string) (int, error)
	CountRepos(ctx context.Context, projectID string) (int, error)
	CountServices(ctx context.Context, projectID string) (int, error)
	AddRepo(ctx context.Context, projectID string, r RepoRef) error
	RemoveRepo(ctx context.Context, owner, name string) error
	ListRepos(ctx context.Context, projectID string) ([]RepoRef, error)
	// GetRepoByFullName looks up a RepoRef by owner/name regardless of project.
	GetRepoByFullName(ctx context.Context, owner, name string) (RepoRef, error)
	MoveTicket(ctx context.Context, ticketID, projectID string) error
}

// NotificationRepo is the consumer-side persistence contract for the workspace notifications capability.
type NotificationRepo interface {
	CreateMany(ctx context.Context, ns []*Notification, evts ...eventbus.OutboxEvent) error
	// List, UnreadCount, and MarkAllRead span every workspace when workspaceID is empty.
	List(ctx context.Context, userID, workspaceID string, limit int) ([]*Notification, error)
	UnreadCount(ctx context.Context, userID, workspaceID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID, workspaceID string) error
}

// ProjectReader resolves a notification subject's project to the workspace the notification belongs to.
type ProjectReader interface {
	Get(ctx context.Context, id string) (*Project, error)
}

// CategoryRepo mutations carry events to write to the transactional outbox in the same transaction.
type CategoryRepo interface {
	Create(ctx context.Context, c *Category, evts ...eventbus.OutboxEvent) error
	Get(ctx context.Context, id string) (*Category, error)
	ListByProject(ctx context.Context, projectID string) ([]*Category, error)
	List(ctx context.Context) ([]*Category, error)
	Update(ctx context.Context, c *Category, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	Reorder(ctx context.Context, projectID string, ids []string) error
	CountTickets(ctx context.Context, categoryID string) (int, error)
	// SetTicketCategory never changes ticket identity; an empty categoryID uncategorizes.
	SetTicketCategory(ctx context.Context, ticketID, categoryID string, evts ...eventbus.OutboxEvent) error
}

// TicketTypeRepo is project-scoped; there's no unscoped "all types" concept.
type TicketTypeRepo interface {
	Create(ctx context.Context, t *TicketType, evts ...eventbus.OutboxEvent) error
	Get(ctx context.Context, id string) (*TicketType, error)
	ListByProject(ctx context.Context, projectID string) ([]*TicketType, error)
	Update(ctx context.Context, t *TicketType, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	Reorder(ctx context.Context, projectID string, ids []string) error
	CountTickets(ctx context.Context, typeID string) (int, error)
}

// StatusRepo is project-scoped; there's no unscoped "all statuses" concept.
type StatusRepo interface {
	Create(ctx context.Context, s *Status, evts ...eventbus.OutboxEvent) error
	Get(ctx context.Context, id string) (*Status, error)
	ListByProject(ctx context.Context, projectID string) ([]*Status, error)
	Update(ctx context.Context, s *Status, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	Reorder(ctx context.Context, projectID string, ids []string) error
	CountTickets(ctx context.Context, statusID string) (int, error)
}

// UserStore resolves notification recipients via auth's UsersRepo, adapted at the composition root (ADR 0017).
type UserStore interface {
	GetUserByLogin(ctx context.Context, login string) (*User, error)
	// LoginForUserID reverses GetUserByLogin, for fan-out paths (memories) that already hold a user id.
	LoginForUserID(ctx context.Context, userID string) (string, error)
}

// WorkspaceMemberStore resolves a workspace's member user ids, adapted at the composition root (ADR 0017)
// onto tenancy's membership store; memories are workspace-scoped, unlike docs' every-registered-user fan-out.
type WorkspaceMemberStore interface {
	ListMemberUserIDs(ctx context.Context, workspaceID string) ([]string, error)
}

// PermissionChecker adapts access (ADR 0017), so a notice only reaches someone who may open its subject.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action) bool
	CanReadDoc(ctx context.Context, userID, docID string) bool
}

// User is the minimal user projection the generation rules resolve against.
type User struct {
	ID    string
	Login string
}
