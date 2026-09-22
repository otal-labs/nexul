package deploy

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type Repo interface {
	// Create persists a deploy and its events in one transaction, so deploy.requested is never lost on a crash.
	Create(ctx context.Context, d *Deploy, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Deploy, error)
	List(ctx context.Context) ([]*Deploy, error)
	ListByService(ctx context.Context, service string) ([]*Deploy, error)
	ListByStackID(ctx context.Context, stackID string) ([]*Deploy, error)
	ListByStatus(ctx context.Context, status Status) ([]*Deploy, error)
	HasActive(ctx context.Context, stackID string) (bool, error)
	LastHealthy(ctx context.Context, stackID string) (*Deploy, error)
	// UpdateStatus sets the status and writes evts in the same transaction.
	UpdateStatus(ctx context.Context, id string, status Status, evts ...eventbus.OutboxEvent) error
	// SetAddress records the container's address on its docker network, reported by the runner at start.
	SetAddress(ctx context.Context, id, address string) error
	// AppendLogLines stores the lines in order under one deploy and writes evts in the same transaction;
	// Seq is assigned by the store.
	AppendLogLines(ctx context.Context, id string, lines []LogLine, evts ...eventbus.OutboxEvent) error
	// ListLogLines returns a deploy's log ordered by ts then seq; a deploy with no output yields an empty slice.
	ListLogLines(ctx context.Context, id string) ([]LogLine, error)
	// CancelRequested enqueues a cancel_requested row; no local row, so this just makes the enqueue transactional.
	CancelRequested(ctx context.Context, evt eventbus.OutboxEvent) error
}

// StackRepo is the persistence contract for stacks; mutations write events atomically.
type StackRepo interface {
	Create(ctx context.Context, s *Stack, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Stack, error)
	// GetBySlugAndMachine looks up a stack by its unique (machine, slug) pair.
	GetBySlugAndMachine(ctx context.Context, slug, machine string) (*Stack, error)
	// GetByName returns the first stack with this name, regardless of machine (dns exposures key by bare name).
	GetByName(ctx context.Context, name string) (*Stack, error)
	// ListByProject lists a project's stacks; an empty projectID lists every project's.
	ListByProject(ctx context.Context, projectID string) ([]*Stack, error)
	// ListByBuildRepo returns every base stack whose build source references this repository, for the push consumer.
	ListByBuildRepo(ctx context.Context, owner, name string) ([]*Stack, error)
	// ListByDerivedFrom returns a base stack's branch deployments, oldest first.
	ListByDerivedFrom(ctx context.Context, baseStackID string) ([]*Stack, error)
	Update(ctx context.Context, s *Stack, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
}

// ServiceRepo is the persistence contract for observed containers.
type ServiceRepo interface {
	Create(ctx context.Context, svc *Container) error
	// Upsert replaces the row matching (stack_id, name), or inserts one if none exists; used by observation reports.
	Upsert(ctx context.Context, svc *Container) error
	ListByStack(ctx context.Context, stackID string) ([]*Container, error)
	Get(ctx context.Context, id string) (*Container, error)
	DeleteByStack(ctx context.Context, stackID string) error
}
