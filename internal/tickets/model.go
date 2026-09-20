package tickets

import (
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
	Number     int        `json:"number"`
	DocID      string     `json:"doc_id"`
	Assignee   string     `json:"assignee"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Labels     []string   `json:"labels"`
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
}

// Actor kinds; a play started through MCP carries the ":mcp" provenance suffix (ADR 0049), as executions do.
const (
	ActorKindUser       = "user"
	ActorKindAutomation = "automation"
	ActorKindPlay       = "play"
)
