package docs

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

type SearchResult struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Rank  float64 `json:"rank"`
}

// FolderRepo persists a project's doc folders; mutations carry events for the outbox write.
type FolderRepo interface {
	// ListFolders returns a project's folders, the default first, then the rest in creation order.
	ListFolders(ctx context.Context, projectID string) ([]*Folder, error)
	GetFolder(ctx context.Context, id string) (*Folder, error)
	DefaultFolder(ctx context.Context, projectID string) (*Folder, error)
	// CreateFolder returns ErrConflict when the project already has a folder of that name, ignoring case.
	CreateFolder(ctx context.Context, f *Folder, evts ...eventbus.OutboxEvent) error
	RenameFolder(ctx context.Context, f *Folder, evts ...eventbus.OutboxEvent) error
	// DeleteFolder moves the folder's docs into toFolderID and deletes the folder in one transaction.
	DeleteFolder(ctx context.Context, id, toFolderID string, evts ...eventbus.OutboxEvent) error
	SetDocFolder(ctx context.Context, docID, folderID string, evts ...eventbus.OutboxEvent) error
}

// WatcherRepo persists who watches a doc; a stopped watcher's row is kept so their own edits don't re-add them.
type WatcherRepo interface {
	// ListWatchers returns the people currently watching the doc, earliest first.
	ListWatchers(ctx context.Context, docID string) ([]*Watcher, error)
	SetWatching(ctx context.Context, docID, userID string, watching bool, at time.Time, evts ...eventbus.OutboxEvent) error
}

// ClarificationRepo persists a doc's clarification rounds and their questions; deleting the doc deletes them.
type ClarificationRepo interface {
	// ListClarificationRounds returns the doc's rounds oldest first, without their questions.
	ListClarificationRounds(ctx context.Context, docID string) ([]*ClarificationRound, error)
	// ListClarificationQuestions returns every question of the doc by round, then position.
	ListClarificationQuestions(ctx context.Context, docID string) ([]*ClarificationQuestion, error)
	GetClarificationQuestion(ctx context.Context, id string) (*ClarificationQuestion, error)
	// CreateClarificationRound returns ErrConflict when the doc already has a round of that number.
	CreateClarificationRound(ctx context.Context, r *ClarificationRound, evts ...eventbus.OutboxEvent) error
	// SaveClarification updates the given rounds and inserts the new questions in one transaction.
	SaveClarification(ctx context.Context, rounds []*ClarificationRound, questions []*ClarificationQuestion, evts ...eventbus.OutboxEvent) error
	DeleteClarificationRound(ctx context.Context, docID string, round int, evts ...eventbus.OutboxEvent) error
	// SaveClarificationAnswer stores q's answer, writing roundAnswered too only when no question of q's round stays pending.
	SaveClarificationAnswer(ctx context.Context, q *ClarificationQuestion, roundAnswered *eventbus.OutboxEvent, evts ...eventbus.OutboxEvent) error
}

// Repo is the consumer-side persistence contract for docs; a save also makes its creator or editor a watcher (ADR 0101).
type Repo interface {
	FolderRepo
	WatcherRepo
	ClarificationRepo
	Create(ctx context.Context, d *Doc, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Doc, error)
	List(ctx context.Context) ([]*Doc, error)
	// ListByProject returns the docs belonging to one project (ticket 10), mirroring tickets.Repo.ListByProject.
	ListByProject(ctx context.Context, projectID string) ([]*Doc, error)
	Update(ctx context.Context, d *Doc, editorID string, evts ...eventbus.OutboxEvent) error
	SetArchived(ctx context.Context, id string, archived bool, evts ...eventbus.OutboxEvent) error
	SetLocked(ctx context.Context, id string, locked bool, evts ...eventbus.OutboxEvent) error
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	// Page reads one window of the docs f and scope keep, oldest first, and how many they keep in all; f.Query is not
	// read here.
	Page(ctx context.Context, f DocFilter, scope DocScope, w paging.Window) ([]*Doc, int, error)
	// SearchIDs ranks every doc f.Query matches that f and scope keep, best first; an archived doc never matches.
	SearchIDs(ctx context.Context, f DocFilter, scope DocScope) ([]string, error)
	// ListByIDs reads the docs ids names, in the order of ids.
	ListByIDs(ctx context.Context, ids []string) ([]*Doc, error)
	ListVersions(ctx context.Context, docID string) ([]*DocVersion, error)
	GetVersion(ctx context.Context, docID string, version int) (*DocVersion, error)
	// CommitBody writes a converged collaboration state to the canonical doc without appending a version row.
	CommitBody(ctx context.Context, d *Doc, editorID string, evts ...eventbus.OutboxEvent) error
	// CreateNamedVersion appends a milestone version, capturing the doc's current state under a name and author.
	CreateNamedVersion(ctx context.Context, v *DocVersion) error
}
