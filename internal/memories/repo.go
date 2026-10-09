package memories

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

// Repo is the consumer-side persistence contract for memories; mutations carry events for the outbox write.
// Create and Update each append a memory_versions row in the same transaction, tagged with authorVia (ticket 17).
type Repo interface {
	TemplateRepo
	AnswerRepo
	SourceRepo
	DraftRepo
	Create(ctx context.Context, m *Memory, authorVia string, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Memory, error)
	// GetByProjectKind returns a project's memory of a special kind, or ErrNotFound.
	GetByProjectKind(ctx context.Context, projectID, kind string) (*Memory, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Memory, error)
	ListByProject(ctx context.Context, projectID string) ([]*Memory, error)
	// PageByProject reads one window of a project's memories, oldest first, and how many it has.
	PageByProject(ctx context.Context, projectID string, w paging.Window) ([]*Memory, int, error)
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
	// DeleteInterviewTemplate drops the workspace's own template so it follows the instance's again.
	DeleteInterviewTemplate(ctx context.Context, workspaceID string, evts ...eventbus.OutboxEvent) error
}

// AnswerRepo stores a project's interview answers, one row per project, round, and question.
type AnswerRepo interface {
	ListAnswers(ctx context.Context, projectID string) ([]*InterviewAnswer, error)
	// UpsertAnswer saves a round-0 answer, replacing the stored one, and returns the row as stored.
	UpsertAnswer(ctx context.Context, a *InterviewAnswer, evts ...eventbus.OutboxEvent) (*InterviewAnswer, error)
	// UpdateAnswer replaces the answer of a question already stored, keeping its options and why; ErrNotFound if none.
	UpdateAnswer(ctx context.Context, a *InterviewAnswer, evts ...eventbus.OutboxEvent) (*InterviewAnswer, error)
	// DeleteAnswer removes a stored answer; ErrNotFound if none.
	DeleteAnswer(ctx context.Context, projectID string, round int, question string, evts ...eventbus.OutboxEvent) error
	// LastRound is the highest round stored for the project, 0 when it has none.
	LastRound(ctx context.Context, projectID string) (int, error)
	// InsertRound writes a follow-up round's questions and answers in one transaction.
	InsertRound(ctx context.Context, answers []*InterviewAnswer, evts ...eventbus.OutboxEvent) error
}

// SourceRepo stores a project's interview sources as given; refs are resolved by the use-case.
type SourceRepo interface {
	ListSources(ctx context.Context, projectID string) ([]*InterviewSource, error)
	GetSource(ctx context.Context, id string) (*InterviewSource, error)
	CountSources(ctx context.Context, projectID string) (int, error)
	// InsertSource returns ErrConflict when the project already points at that kind and ref.
	InsertSource(ctx context.Context, src *InterviewSource, evts ...eventbus.OutboxEvent) error
	// UpdateSource stores the stance and label; ErrNotFound if the source is gone.
	UpdateSource(ctx context.Context, src *InterviewSource, evts ...eventbus.OutboxEvent) error
	// DeleteSource returns ErrNotFound if there is no such source.
	DeleteSource(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
}

// DraftRepo stores a project's interview drafts, one per question.
type DraftRepo interface {
	ListDrafts(ctx context.Context, projectID string) ([]*InterviewDraft, error)
	GetDraft(ctx context.Context, id string) (*InterviewDraft, error)
	// SaveDrafts upserts each draft by project and question in one transaction.
	SaveDrafts(ctx context.Context, drafts []*InterviewDraft, evts ...eventbus.OutboxEvent) error
	// DeleteDraft returns ErrNotFound if there is no such draft.
	DeleteDraft(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
}
