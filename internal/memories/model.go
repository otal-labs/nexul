package memories

import "time"

// KindInterview marks a project's interview memory: one per project, always included, capped (ADR 0065).
const KindInterview = "interview"

// KindDecisionsLog marks a project's decisions log: one per project, found with memory_list, never named every turn.
const KindDecisionsLog = "decisions_log"

// MaxInterviewChars caps the interview memory, measured as exported markdown; the template has its own limit.
const MaxInterviewChars = 8_000

// Memory is an agent-facing note (ADR 0056): own entity, never a doc, always in one project (ADR 0099).
type Memory struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	// Kind is empty for an ordinary memory; a special kind is unique per project (KindInterview).
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	WhenToUse string `json:"when_to_use"`
	Body      string `json:"body"`
	// AlwaysIncluded marks a memory every agent turn in its project names for the agent to read first (ADR 0111).
	AlwaysIncluded bool `json:"always_included"`
	// Footer marks a memory a play run names last, to read once the work is done (ADR 0112).
	Footer bool `json:"footer"`
	// Version is the current pointer into memory_versions; every save appends a row and bumps this (ticket 17).
	Version   int       `json:"version"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoryItem is one memory as an agent turn names it for the agent to read itself, never its body (ADR 0111).
type MemoryItem struct {
	ID             string
	Title          string
	AlwaysIncluded bool
	Kind           string
}

// MemoryVersion is one historical snapshot; every save appends a row, revert included, mirroring doc_versions
// but with an author on every row rather than only on named milestones (ticket 17).
type MemoryVersion struct {
	ID             string `json:"id"`
	MemoryID       string `json:"memory_id"`
	Version        int    `json:"version"`
	Title          string `json:"title"`
	WhenToUse      string `json:"when_to_use"`
	Body           string `json:"body"`
	AlwaysIncluded bool   `json:"always_included"`
	AuthorID       string `json:"author_id"`
	// AuthorVia is "mcp" when the save came from an MCP tool call, empty for the browser (ADR 0049).
	AuthorVia string    `json:"author_via,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// InterviewTemplate is the markdown list of questions a project's interview asks.
type InterviewTemplate struct {
	WorkspaceID string     `json:"workspace_id"`
	Body        string     `json:"body"`
	Questions   []Question `json:"questions"`
	// DefaultBody is the instance's Interview template, what an unedited workspace follows and a reset returns to.
	DefaultBody string `json:"default_body"`
	// Edited is true once the workspace saved its own; until then Body follows the instance's live.
	Edited    bool      `json:"edited"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InterviewAnswer is one stored answer on the Interview page, matched to its question by text; round 0 is the
// template's questions, 1 and up the follow-up run's rounds. Kept apart from the interview memory row.
type InterviewAnswer struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Round       int    `json:"round"`
	Question    string `json:"question"`
	// Options, MultiSelect, and Why are the follow-up as the run asked it; empty in round 0.
	Options     []AnswerOption `json:"options,omitempty"`
	MultiSelect bool           `json:"multi_select,omitempty"`
	Why         string         `json:"why,omitempty"`
	Selected    []string       `json:"selected"`
	Text        string         `json:"text"`
	Skipped     bool           `json:"skipped"`
	AnsweredBy  string         `json:"answered_by"`
	AnsweredAt  time.Time      `json:"answered_at"`
}

// AnswerOption is one choice a follow-up offered.
type AnswerOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}
