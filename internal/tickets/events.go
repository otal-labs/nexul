package tickets

import "github.com/otal-labs/nexul/internal/platform/eventbus"

// Topics published by the tickets domain.
const (
	TopicCreated          = "ticket.created"
	TopicUpdated          = "ticket.updated"
	TopicStatusChanged    = "ticket.status_changed"
	TopicAssigneeChanged  = "ticket.assignee_changed" // deprecated alias of TopicDeveloperChanged, still published (ADR 0044)
	TopicDeveloperChanged = "ticket.developer_changed"
	TopicTesterChanged    = "ticket.tester_changed"
	TopicFinished         = "ticket.finished"
	TopicDeleted          = "ticket.deleted"
	TopicLinkCreated      = "ticket.link_created"
	TopicLinkDeleted      = "ticket.link_deleted"
	TopicTestPassed       = "ticket.test_passed"
	TopicTestFailed       = "ticket.test_failed"
)

// Topics returns every topic the tickets domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicCreated, Payload: CreatedEvent{}},
		{Name: TopicUpdated, Payload: UpdatedEvent{}},
		{Name: TopicStatusChanged, Payload: StatusChangedEvent{}},
		{Name: TopicAssigneeChanged, Payload: AssigneeChangedEvent{}, Description: "Deprecated in favour of ticket.developer_changed; still published with the same payload whenever the developer changes."},
		{Name: TopicDeveloperChanged, Payload: PersonChangedEvent{}},
		{Name: TopicTesterChanged, Payload: PersonChangedEvent{}},
		{Name: TopicFinished, Payload: FinishedEvent{}},
		{Name: TopicDeleted, Payload: DeletedEvent{}},
		{Name: TopicLinkCreated, Payload: LinkEvent{}, Description: "A found-in or blocked-by link was added; ticket_id is found in or blocked by target_id, and an empty target_id on found_in marks the origin unknown."},
		{Name: TopicLinkDeleted, Payload: LinkEvent{}, Description: "A found-in or blocked-by link was removed, or replaced by a new found-in."},
		{Name: TopicTestPassed, Payload: TestedEvent{}, Description: "A ticket passed testing and moved to a done-stage column; tester is the login of whoever passed it."},
		{Name: TopicTestFailed, Payload: TestedEvent{}, Description: "A ticket failed testing and moved back to a progress-stage column; report is the bug report posted to its thread."},
	}
}

// WireShape is the type a ticket encodes as, which its event schemas describe.
func (Ticket) WireShape() any { return ticketJSON{} }

// CreatedEvent field names are part of the published contract (ADR 0044) and are additive-only.
type CreatedEvent struct {
	Ticket Ticket `json:"ticket"`
	// MentionedUserIDs are the people the body @-mentions, each told once in their inbox.
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty" jsonschema:"People the body @-mentions, by user id."`
}

// UpdatedEvent is the payload for ticket.updated: a title, body, or source doc edit; Ticket reflects the post-edit state.
type UpdatedEvent struct {
	Ticket Ticket `json:"ticket"`
	// ActorID is the user who made the edit; empty for an automation.
	ActorID string `json:"actor_id,omitempty"`
	// MentionedUserIDs are the people the new body @-mentions that the previous one did not.
	MentionedUserIDs []string `json:"mentioned_user_ids,omitempty" jsonschema:"People this edit newly @-mentions, by user id."`
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

// AssigneeChangedEvent is ticket.assignee_changed's unchanged payload, published beside ticket.developer_changed.
type AssigneeChangedEvent struct {
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
	ID        string `json:"id"`
	Title     string `json:"title"`
	ProjectID string `json:"project_id"`
}

// LinkEvent is the payload for ticket.link_created and ticket.link_deleted.
type LinkEvent struct {
	Link TicketLink `json:"link"`
}

// TestedEvent is the payload for ticket.test_passed and ticket.test_failed; Report is a failure's thread message.
type TestedEvent struct {
	Ticket Ticket `json:"ticket"`
	Tester string `json:"tester"`
	Report string `json:"report,omitempty"`
}
