package memories

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo is the consumer-side persistence contract for memories; mutations carry events for the outbox write.
// Create and Update each append a memory_versions row in the same transaction, tagged with authorVia (ticket 17).
type Repo interface {
	TemplateRepo
	Create(ctx context.Context, m *Memory, authorVia string, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Memory, error)
	// GetByProjectKind returns a project's memory of a special kind, or ErrNotFound.
	GetByProjectKind(ctx context.Context, projectID, kind string) (*Memory, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Memory, error)
	// ListByProject returns the project's own memories plus its workspace's workspace-scoped ones,
	// workspace-scoped first (ADR 0059).
	ListByProject(ctx context.Context, projectID, workspaceID string) ([]*Memory, error)
	// ListWorkspaceScoped returns only a workspace's workspace-scoped memories (ADR 0059).
	ListWorkspaceScoped(ctx context.Context, workspaceID string) ([]*Memory, error)
	Update(ctx context.Context, m *Memory, authorVia string, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	// ListVersions returns a memory's version history, newest first.
	ListVersions(ctx context.Context, memoryID string) ([]*MemoryVersion, error)
	// GetVersion returns one historical version of a memory.
	GetVersion(ctx context.Context, memoryID string, version int) (*MemoryVersion, error)
}

// TemplateRepo stores each workspace's Interview template; ErrNotFound means the workspace never saved one.
type TemplateRepo interface {
	GetInterviewTemplate(ctx context.Context, workspaceID string) (*InterviewTemplate, error)
	SaveInterviewTemplate(ctx context.Context, t *InterviewTemplate, evts ...eventbus.OutboxEvent) error
}
