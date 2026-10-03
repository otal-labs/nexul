package memories

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo is the consumer-side persistence contract for memories; mutations carry events for the outbox write.
// Create and Update each append a memory_versions row in the same transaction, tagged with authorVia (ticket 17).
type Repo interface {
	TemplateRepo
	AnswerRepo
	Create(ctx context.Context, m *Memory, authorVia string, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Memory, error)
	// GetByProjectKind returns a project's memory of a special kind, or ErrNotFound.
	GetByProjectKind(ctx context.Context, projectID, kind string) (*Memory, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*Memory, error)
	ListByProject(ctx context.Context, projectID string) ([]*Memory, error)
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
