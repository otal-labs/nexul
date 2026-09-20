// Package mentions implements the @-mention/chip resolution seam.
package mentions

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Kind labels a mention reference's target type; values are the wire contract.
type Kind string

const (
	KindTicket Kind = "ticket"
	KindDoc    Kind = "doc"
)

// Ref identifies one mention reference found in a document body; Type is one of KindTicket/KindDoc.
type Ref struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Chip is the resolved view of a reference at render time; unopenable targets stay visible but inert.
type Chip struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status,omitempty"`       // status id, tickets only
	StatusLabel string `json:"status_label,omitempty"` // configurable column name
	CanOpen     bool   `json:"can_open"`

	// Back the chip template's tokens (spec.md 6); Prefix+Number stay split (ADR 0004) for independent substitution.
	ProjectPrefix string `json:"project_prefix,omitempty"`
	ProjectNumber int    `json:"project_number,omitempty"`
	TypeLabel     string `json:"type_label,omitempty"`
	AssigneeLabel string `json:"assignee_label,omitempty"`
	// DueLabel backs {ticket.Due}; always empty until tickets gain a due-date field, renders as "" not an error.
	DueLabel string `json:"due_label,omitempty"`
}

// SearchResult is one autocomplete entry for the mention picker; docs the actor cannot open are excluded.
type SearchResult struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	StatusLabel string `json:"status_label,omitempty"`
	CanOpen     bool   `json:"can_open"`
}

// Ticket is the slice of a ticket record the resolver needs (ADR 0017 seam onto tickets).
type Ticket struct {
	ID     string
	Title  string
	Status string
	// Back the chip template's tokens (spec.md 6): Number+ProjectID resolve PREFIX-NUMBER (ADR 0004).
	Number    int
	ProjectID string
	TypeID    string
	Assignee  string
}

// Project is the slice of a project record the resolver needs, its Prefix for the PREFIX-NUMBER id (ADR 0004).
type Project struct {
	ID     string
	Prefix string
}

// TicketType is the slice of a ticket-type record the resolver needs (ADR 0017 seam onto workspace).
type TicketType struct {
	ID   string
	Name string
}

// Doc is the slice of a doc record the resolver needs (ADR 0017 seam onto docs).
type Doc struct {
	ID    string
	Title string
}

// Status is the slice of a status column the resolver needs; statuses are configurable.
type Status struct {
	ID   string
	Name string
}

// SearchHit is one full-text hit from either the tickets or docs search.
type SearchHit struct {
	ID    string
	Title string
}

// TicketSource is the consumer-side slice of the tickets persistence layer the resolver needs.
type TicketSource interface {
	GetByID(ctx context.Context, id string) (*Ticket, error)
	Search(ctx context.Context, query string, limit int) ([]SearchHit, error)
	// GetByKey resolves a ticket by its PREFIX-NUMBER display id (ADR 0004).
	GetByKey(ctx context.Context, prefix string, number int) (*Ticket, error)
}

// DocSource is the consumer-side slice of the docs persistence layer the resolver needs.
type DocSource interface {
	GetByID(ctx context.Context, id string) (*Doc, error)
	Search(ctx context.Context, query string, limit int) ([]SearchHit, error)
}

// StatusSource reads a workspace status column by id for its display name.
type StatusSource interface {
	Get(ctx context.Context, id string) (*Status, error)
}

// ProjectSource is the consumer-side slice of workspace's projects layer needed to render {ticket.Project}.
type ProjectSource interface {
	Get(ctx context.Context, id string) (*Project, error)
}

// TicketTypeSource is workspace's ticket-types layer needed to render {ticket.Type}.
type TicketTypeSource interface {
	Get(ctx context.Context, id string) (*TicketType, error)
}

// AccessChecker reports whether a user may act on a document; tickets are workspace-level in v1, no check needed.
type AccessChecker interface {
	Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error)
}
