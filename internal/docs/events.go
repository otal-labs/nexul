package docs

// Topics published by the docs domain. Consumers: mcp (re-index).
const (
	TopicCreated = "doc.created"
	TopicUpdated = "doc.updated"
	TopicDeleted = "doc.deleted"
)

// Topics returns every topic the docs domain publishes.
func Topics() []string {
	return []string{TopicCreated, TopicUpdated, TopicDeleted}
}

// CreatedEvent is the payload for doc.created; field names are part of the event contract (ADR 0044) and additive-only.
type CreatedEvent struct {
	Doc Doc `json:"doc"`
}

// UpdatedEvent is the payload for doc.updated.
type UpdatedEvent struct {
	Doc Doc `json:"doc"`
}

// DeletedEvent is the doc.deleted payload; the doc is already gone by publish time, so consumers get identity only.
type DeletedEvent struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
