package memories

import "time"

// Topics published by the memories domain.
const (
	TopicCreated = "memory.created"
	TopicUpdated = "memory.updated"
	TopicDeleted = "memory.deleted"

	TopicInterviewTemplateUpdated = "interview_template.updated"

	TopicAnswerSaved   = "interview_answer.saved"
	TopicAnswerCleared = "interview_answer.cleared"

	TopicSourceAdded   = "interview_source.added"
	TopicSourceChanged = "interview_source.changed"
	TopicSourceRemoved = "interview_source.removed"

	TopicDraftSaved     = "interview_draft.saved"
	TopicDraftDismissed = "interview_draft.dismissed"
)

// Topics returns every topic the memories domain publishes.
func Topics() []string {
	return []string{
		TopicCreated, TopicUpdated, TopicDeleted, TopicInterviewTemplateUpdated, TopicAnswerSaved, TopicAnswerCleared,
		TopicSourceAdded, TopicSourceChanged, TopicSourceRemoved, TopicDraftSaved, TopicDraftDismissed,
	}
}

// MemoryRef is a memory event's identity payload, everything but the body (ADR 0044: additive-only, kept lean).
type MemoryRef struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspace_id"`
	ProjectID      string    `json:"project_id"`
	Kind           string    `json:"kind,omitempty"`
	Title          string    `json:"title"`
	WhenToUse      string    `json:"when_to_use"`
	AlwaysIncluded bool      `json:"always_included"`
	Footer         bool      `json:"footer"`
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
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	AuthorID    string `json:"author_id"`
}

// InterviewTemplateUpdatedEvent is the interview_template.updated payload; the body stays out, like MemoryRef.
type InterviewTemplateUpdatedEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	AuthorID    string    `json:"author_id"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AnswerEvent is the interview_answer.saved and interview_answer.cleared payload: which question, never the answer.
type AnswerEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
	Round       int       `json:"round"`
	Question    string    `json:"question"`
	AuthorID    string    `json:"author_id"`
	At          time.Time `json:"at"`
}

// SourceEvent is the interview_source.added, .changed, and .removed payload: never the ref's content or a text body.
type SourceEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
	SourceID    string    `json:"source_id"`
	Kind        string    `json:"kind"`
	Stance      string    `json:"stance"`
	AuthorID    string    `json:"author_id"`
	At          time.Time `json:"at"`
}

// DraftEvent is the interview_draft.saved and interview_draft.dismissed payload: which question, never the draft.
type DraftEvent struct {
	WorkspaceID string    `json:"workspace_id"`
	ProjectID   string    `json:"project_id"`
	DraftID     string    `json:"draft_id"`
	Question    string    `json:"question"`
	AuthorID    string    `json:"author_id"`
	At          time.Time `json:"at"`
}

func toRef(m *Memory) MemoryRef {
	return MemoryRef{
		ID:             m.ID,
		WorkspaceID:    m.WorkspaceID,
		ProjectID:      m.ProjectID,
		Kind:           m.Kind,
		Title:          m.Title,
		WhenToUse:      m.WhenToUse,
		AlwaysIncluded: m.AlwaysIncluded,
		Footer:         m.Footer,
		Version:        m.Version,
		UpdatedAt:      m.UpdatedAt,
	}
}
