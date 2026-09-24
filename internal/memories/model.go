package memories

import "time"

// KindInterview marks a project's interview memory: one per project, always included, capped (ADR 0065).
const KindInterview = "interview"

// MaxInterviewChars caps the interview memory and the Interview template, measured as exported markdown.
const MaxInterviewChars = 8_000

// Memory is an agent-facing note (ADR 0056): own entity, never a doc. ProjectID empty means the memory belongs
// to the whole workspace instead of one project (ADR 0059).
type Memory struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id,omitempty"`
	// Kind is empty for an ordinary memory; a special kind is unique per project (KindInterview).
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	WhenToUse string `json:"when_to_use"`
	Body      string `json:"body"`
	// AlwaysIncluded marks a memory for automatic inlining into every turn (ticket 27); this ticket only stores the flag.
	AlwaysIncluded bool `json:"always_included"`
	// Version is the current pointer into memory_versions; every save appends a row and bumps this (ticket 17).
	Version   int       `json:"version"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoryItem is one memory's entry in the agent's turn index: enough to decide relevance, no body — except an
// always-included memory, which carries its markdown Body too, since the turn inlines it in full (ticket 27).
type MemoryItem struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	WhenToUse      string `json:"when_to_use"`
	AlwaysIncluded bool   `json:"always_included"`
	Kind           string `json:"kind,omitempty"`
	Body           string `json:"body,omitempty"`
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

// InterviewTemplate is the markdown a project's interview memory is copied from once, at creation.
type InterviewTemplate struct {
	WorkspaceID string `json:"workspace_id"`
	Body        string `json:"body"`
	// DefaultBody is the seeded categories, so an editor can reset to them.
	DefaultBody string    `json:"default_body"`
	UpdatedBy   string    `json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}
