package tickets

// Topics published by the tickets domain.
const (
	TopicCreated          = "ticket.created"
	TopicUpdated          = "ticket.updated"
	TopicStatusChanged    = "ticket.status_changed"
	TopicDeveloperChanged = "ticket.developer_changed"
	TopicTesterChanged    = "ticket.tester_changed"
	TopicFinished         = "ticket.finished"
	TopicDeleted          = "ticket.deleted"
)

// Topics returns every topic the tickets domain publishes.
func Topics() []string {
	return []string{TopicCreated, TopicUpdated, TopicStatusChanged, TopicDeveloperChanged, TopicTesterChanged, TopicFinished, TopicDeleted}
}

// CreatedEvent field names are part of the published contract (ADR 0044) and are additive-only.
type CreatedEvent struct {
	Ticket Ticket `json:"ticket"`
}

// UpdatedEvent is the payload for ticket.updated: a title/body edit; Ticket reflects the post-edit state.
type UpdatedEvent struct {
	Ticket Ticket `json:"ticket"`
}

// StatusChangedEvent's RunID links an automation-performed transition to its run history entry.
type StatusChangedEvent struct {
	Ticket Ticket `json:"ticket"`
	From   Status `json:"from"`
	To     Status `json:"to"`
	Actor  Actor  `json:"actor,omitempty"`
	RunID  string `json:"run_id,omitempty"`
}

// PersonChangedEvent is the payload for ticket.developer_changed and ticket.tester_changed; empty From/To mean nobody.
type PersonChangedEvent struct {
	Ticket Ticket `json:"ticket"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// personTopic maps a role to the topic its change publishes on.
func personTopic(role Role) string {
	if role == RoleTester {
		return TopicTesterChanged
	}
	return TopicDeveloperChanged
}

// FinishedEvent publishes exactly once per ticket, once a PR merges and none remain open.
type FinishedEvent struct {
	Ticket Ticket `json:"ticket"`
}

// DeletedEvent's ticket is already gone (hard delete), so consumers get identity only.
type DeletedEvent struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
