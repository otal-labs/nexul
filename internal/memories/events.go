package memories

import "time"

// Topics published by the memories domain.
const (
	TopicCreated = "memory.created"
	TopicUpdated = "memory.updated"
	TopicDeleted = "memory.deleted"
)

// Topics returns every topic the memories domain publishes.
func Topics() []string {
	return []string{TopicCreated, TopicUpdated, TopicDeleted}
}

// MemoryRef is a memory event's identity payload, everything but the body (ADR 0044: additive-only, kept lean).
type MemoryRef struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspace_id"`
	ProjectID      string    `json:"project_id"`
	Title          string    `json:"title"`
	WhenToUse      string    `json:"when_to_use"`
	AlwaysIncluded bool      `json:"always_included"`
	Version        int       `json:"version"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreatedEvent is the payload for memory.created.
type CreatedEvent struct {
	Memory   MemoryRef `json:"memory"`
	AuthorID string    `json:"author_id"`
}

// UpdatedEvent is the payload for memory.updated; AuthorVia is "mcp" when the save came from an MCP tool call
// during an Agent turn (ADR 0049), so the inbox notification can say "Agent via <user>".
type UpdatedEvent struct {
	Memory    MemoryRef `json:"memory"`
	AuthorID  string    `json:"author_id"`
	AuthorVia string    `json:"author_via,omitempty"`
}

// DeletedEvent is the memory.deleted payload; the memory is already gone by publish time.
type DeletedEvent struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	AuthorID string `json:"author_id"`
}

func toRef(m *Memory) MemoryRef {
	return MemoryRef{
		ID:             m.ID,
		WorkspaceID:    m.WorkspaceID,
		ProjectID:      m.ProjectID,
		Title:          m.Title,
		WhenToUse:      m.WhenToUse,
		AlwaysIncluded: m.AlwaysIncluded,
		Version:        m.Version,
		UpdatedAt:      m.UpdatedAt,
	}
}
