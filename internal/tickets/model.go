package tickets

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusClosed     Status = "closed"
)

type Ticket struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id"`
	CategoryID string `json:"category_id"`
	TypeID     string `json:"type_id"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Status     Status `json:"status"`
	// Position: status/category moves append to the end; SetPosition reorders directly (ADR 0002).
	Position int `json:"position"`
	// Number combines with the project's Prefix to render PREFIX-NUMBER; ID remains the real primary key.
	Number int    `json:"number"`
	DocID  string `json:"doc_id"`
	// Developer and Tester are member logins, both optional; Reporter is set once at creation and never edited.
	Developer  string     `json:"developer"`
	Tester     string     `json:"tester"`
	Reporter   Reporter   `json:"reporter"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Labels     []string   `json:"labels"`
}

// MarshalJSON adds the deprecated assignee field, always equal to developer, so the published payload stays additive (ADR 0044).
func (t Ticket) MarshalJSON() ([]byte, error) {
	type plain Ticket
	return json.Marshal(struct {
		plain
		Assignee string `json:"assignee"`
	}{plain(t), t.Developer})
}

// CanTransition allows any status pair; a same-status move is a no-op, not an error.
func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	return to != ""
}

// PRRef mirrors the git provider's PR reference so the composition root can adapt without a direct import.
type PRRef struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	SHA    string `json:"sha"`
}

// PRState starts open; git.pr_merged/pr_closed flip it, and completion evaluation reads these.
type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateMerged PRState = "merged"
	PRStateClosed PRState = "closed"
)

// PRLink is a linked PR together with its lifecycle state.
type PRLink struct {
	PRRef
	State PRState `json:"state"`
}

// BranchLink records a branch associated with a ticket; a branch or PR may be linked to multiple tickets.
type BranchLink struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// DevStatusCounts is the board's dev-status badge tally, from one batched call per card.
type DevStatusCounts struct {
	Open   int `json:"open"`
	Merged int `json:"merged"`
}

// LabelColor is the label_colors side table's row shape; ticket_labels itself carries no color.
type LabelColor struct {
	Label string       `json:"label"`
	Color colors.Color `json:"color"`
}

// Actor records who performed a transition for provenance: a human user, a named automation, or a play run.
type Actor struct {
	Kind           string `json:"kind"`
	AutomationID   string `json:"automation_id,omitempty"`
	AutomationName string `json:"automation_name,omitempty"`
	PlayLabel      string `json:"play_label,omitempty"`
	TrailID        string `json:"trail_id,omitempty"`
	// UserID is the person behind a user or play move, empty for an automation's.
	UserID string `json:"user_id,omitempty"`
}

// Actor kinds; a play started through MCP carries the ":mcp" provenance suffix (ADR 0049), as executions do.
const (
	ActorKindUser       = "user"
	ActorKindUserMCP    = ActorKindUser + ":mcp"
	ActorKindAutomation = "automation"
	ActorKindPlay       = "play"
)

// Reporter is who filed a ticket: a person (user), Nexul for a person via MCP (user:mcp), or an automation.
type Reporter struct {
	Kind           string `json:"kind"`
	Login          string `json:"login,omitempty"`
	AutomationID   string `json:"automation_id,omitempty"`
	AutomationName string `json:"automation_name,omitempty"`
}

// Role names one of the two editable people on a ticket.
type Role string

const (
	RoleDeveloper Role = "developer"
	RoleTester    Role = "tester"
)

// BugTypeName is the ticket type that must carry a found-in link; matched by name, so renaming the type away drops the rule.
const BugTypeName = "bug"

// IsBugType reports whether a ticket type name is the bug type, ignoring case and surrounding space.
func IsBugType(name string) bool { return strings.EqualFold(strings.TrimSpace(name), BugTypeName) }

// LinkKind names a ticket-to-ticket link: found_in points a bug at its origin, blocked_by at a ticket that must reach done first.
type LinkKind string

const (
	LinkFoundIn   LinkKind = "found_in"
	LinkBlockedBy LinkKind = "blocked_by"
)

// TicketLink is a directed link record; an empty TargetID on a found_in link marks the origin unknown.
type TicketLink struct {
	TicketID  string    `json:"ticket_id"`
	Kind      LinkKind  `json:"kind"`
	TargetID  string    `json:"target_id"`
	CreatedAt time.Time `json:"created_at"`
}

// LinkedTicket is the ticket at the other end of a link; Done reads its status stage, never the column name.
type LinkedTicket struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Prefix    string `json:"prefix"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Status    Status `json:"status"`
	Done      bool   `json:"done"`
}

// LinkEnd is one stored link seen from a ticket; Ticket is nil for a found-in whose origin is unknown.
type LinkEnd struct {
	Kind   LinkKind
	Ticket *LinkedTicket
}

// LinkSet is both directions of a ticket's links; Blocked holds while any blocker is outside a done-stage status.
type LinkSet struct {
	FoundIn       *LinkedTicket  `json:"found_in"`
	OriginUnknown bool           `json:"origin_unknown"`
	BugsFound     []LinkedTicket `json:"bugs_found"`
	BlockedBy     []LinkedTicket `json:"blocked_by"`
	Blocks        []LinkedTicket `json:"blocks"`
	Blocked       bool           `json:"blocked"`
}
