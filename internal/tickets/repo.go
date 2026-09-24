package tickets

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type SearchResult struct {
	ID    string
	Title string
	Rank  float64
}

// StatusStore is workspace's status columns, used to validate transitions without importing workspace (ADR 0017).
type StatusStore interface {
	Exists(ctx context.Context, id string) (bool, error)
}

// UserLogins resolves a user id to the member login tickets store for people, without importing auth (ADR 0017).
type UserLogins interface {
	LoginForUserID(ctx context.Context, userID string) (string, error)
}

// TypeTemplates is workspace's ticket types, read for a type's body template without importing workspace (ADR 0017).
type TypeTemplates interface {
	BodyTemplate(ctx context.Context, typeID string) (string, error)
}

// LinkRepo persists found-in and blocked-by links; writes carry outbox events.
type LinkRepo interface {
	// ListLinkEnds returns the links a ticket holds and the links other tickets hold against it.
	ListLinkEnds(ctx context.Context, id string) (from, to []LinkEnd, err error)
	// BlockerIDs returns the ids a ticket is directly blocked by, for the cycle walk.
	BlockerIDs(ctx context.Context, id string) ([]string, error)
	// PutLink inserts a link; a found_in link replaces any found-in the ticket already holds.
	PutLink(ctx context.Context, link TicketLink, evts ...eventbus.OutboxEvent) error
	// DeleteLink returns false when no such link existed; a found_in link is matched by ticket alone.
	DeleteLink(ctx context.Context, link TicketLink, evts ...eventbus.OutboxEvent) (bool, error)
	// UnclearedBlockers maps each blocked ticket id to its blockers not yet in a done-stage status.
	UnclearedBlockers(ctx context.Context) (map[string][]LinkedTicket, error)
}

// Repo is the consumer-side persistence contract for tickets; mutations carry outbox events.
type Repo interface {
	LinkRepo
	Create(ctx context.Context, t *Ticket, evts ...eventbus.OutboxEvent) error
	GetByID(ctx context.Context, id string) (*Ticket, error)
	List(ctx context.Context) ([]*Ticket, error)
	ListByDoc(ctx context.Context, docID string) ([]*Ticket, error)
	ListByProject(ctx context.Context, projectID string) ([]*Ticket, error)
	// UpdateStatus appends the ticket to the end of the new status's manual order, not its old position.
	UpdateStatus(ctx context.Context, id string, status Status, evts ...eventbus.OutboxEvent) error
	// UpdateType changes a ticket's type id in place; the ticket keeps its identity.
	UpdateType(ctx context.Context, id, typeID string) error
	// UpdatePerson sets the ticket's developer or tester, enqueueing evts in the same transaction; empty clears it.
	UpdatePerson(ctx context.Context, id string, role Role, login string, evts ...eventbus.OutboxEvent) error
	// UpdateTicket edits a ticket's title and body in place, keeping its identity; the ticket must exist.
	UpdateTicket(ctx context.Context, id, title, body string, evts ...eventbus.OutboxEvent) error
	// SetPosition orders a ticket within its current (status, category) pair (ADR 0002).
	SetPosition(ctx context.Context, id string, position int) error
	// AddLabel attaches a cross-cutting tag; a duplicate is a no-op.
	AddLabel(ctx context.Context, id, label string) error
	// RemoveLabel detaches a tag; a missing label is a no-op.
	RemoveLabel(ctx context.Context, id, label string) error
	// ListLabels returns a ticket's labels.
	ListLabels(ctx context.Context, id string) ([]string, error)
	// ListAllLabels returns the distinct labels across all tickets, ordered, for the board filter bar.
	ListAllLabels(ctx context.Context) ([]string, error)
	// SetLabelColor works even for a label no ticket has used yet, with no separate registration step.
	SetLabelColor(ctx context.Context, projectID, label string, color colors.Color) error
	// LabelColors batches lookups in one query for the board, which renders many labels per screen.
	LabelColors(ctx context.Context, projectID string, labels []string) (map[string]colors.Color, error)
	Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	LinkPR(ctx context.Context, id string, ref PRRef, state PRState) error
	ListPRLinks(ctx context.Context, id string) ([]PRLink, error)
	// ListPRLinksBatch is the dev-status batch endpoint's data source, keyed by ticket id.
	ListPRLinksBatch(ctx context.Context, ids []string) (map[string][]PRLink, error)
	// MarkPRState returns the ids of tickets affected by the PR identity, for the completion fan-out (ADR 0021).
	MarkPRState(ctx context.Context, owner, repo string, number int, state PRState) ([]string, error)
	// SetFinishedAt returns false if already set, keeping ticket.finished exactly-once.
	SetFinishedAt(ctx context.Context, id string, at time.Time, evts ...eventbus.OutboxEvent) (bool, error)
	LinkBranch(ctx context.Context, id string, link BranchLink) error
	ListBranchLinks(ctx context.Context, id string) ([]BranchLink, error)
}
