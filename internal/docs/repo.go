package docs

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type SearchResult struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Rank  float64 `json:"rank"`
}

// Repo is the consumer-side persistence contract for docs; mutations carry events for the outbox write.
type Repo interface {
	Create(ctx context.Context, d *Doc, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Doc, error)
	List(ctx context.Context) ([]*Doc, error)
	// ListByProject returns the docs belonging to one project (ticket 10), mirroring tickets.Repo.ListByProject.
	ListByProject(ctx context.Context, projectID string) ([]*Doc, error)
	Update(ctx context.Context, d *Doc, evts ...eventbus.OutboxEvent) error
	SetArchived(ctx context.Context, id string, archived bool, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	ListVersions(ctx context.Context, docID string) ([]*DocVersion, error)
	GetVersion(ctx context.Context, docID string, version int) (*DocVersion, error)
	// CommitBody writes a converged collaboration state to the canonical doc without appending a version row.
	CommitBody(ctx context.Context, d *Doc, evts ...eventbus.OutboxEvent) error
	// CreateNamedVersion appends a milestone version, capturing the doc's current state under a name and author.
	CreateNamedVersion(ctx context.Context, v *DocVersion) error
}
