package chat

import "time"

// Topics published by the chat domain.
const (
	TopicConversationCreated = "chat.conversation.created"
	TopicConversationUpdated = "chat.conversation.updated"
	TopicConversationDeleted = "chat.conversation.deleted"
	// TopicConversationMembersChanged is a private channel's members changing, or a channel switching private or public.
	TopicConversationMembersChanged = "chat.conversation.members_changed"
	TopicMessageCreated             = "chat.message.created"
	TopicMessageUpdated             = "chat.message.updated"
	TopicMessageDeleted             = "chat.message.deleted"
)

// Topics returns every topic the chat domain publishes.
func Topics() []string {
	return []string{
		TopicConversationCreated, TopicConversationUpdated, TopicConversationDeleted, TopicConversationMembersChanged,
		TopicMessageCreated, TopicMessageUpdated, TopicMessageDeleted,
	}
}

// ConversationCreatedEvent field names are part of the published contract (ADR 0044) and are additive-only.
type ConversationCreatedEvent struct {
	Conversation Conversation `json:"conversation"`
}

// ConversationUpdatedEvent is the payload for chat.conversation.updated: a channel's new name beside the one it replaced.
type ConversationUpdatedEvent struct {
	ConversationID string `json:"conversation_id"`
	WorkspaceID    string `json:"workspace_id"`
	Kind           Kind   `json:"kind"`
	Name           string `json:"name"`
	PreviousName   string `json:"previous_name"`
	ActorID        string `json:"actor_id,omitempty"`
}

// ConversationDeletedEvent names what chat.conversation.deleted removed, since nothing is left to fetch.
type ConversationDeletedEvent struct {
	ConversationID string `json:"conversation_id"`
	WorkspaceID    string `json:"workspace_id"`
	Kind           Kind   `json:"kind"`
	Name           string `json:"name"`
	ActorID        string `json:"actor_id,omitempty"`
	// Private and MemberIDs say who could read a private channel, so its deletion reaches only them and the Owner.
	Private   bool     `json:"private,omitempty"`
	MemberIDs []string `json:"member_ids,omitempty"`
}

// ConversationMembersChangedEvent carries ids only; a switch to private names everyone not kept as removed.
type ConversationMembersChangedEvent struct {
	ConversationID string   `json:"conversation_id"`
	WorkspaceID    string   `json:"workspace_id"`
	Private        bool     `json:"private"`
	AddedUserIDs   []string `json:"added_user_ids"`
	RemovedUserIDs []string `json:"removed_user_ids"`
	ActorID        string   `json:"actor_id,omitempty"`
}

// MessageCreatedEvent is the payload for chat.message.created.
type MessageCreatedEvent struct {
	Message Message `json:"message"`
	// MembersOnly marks a DM or private channel's message, which never reaches integrations or automations.
	MembersOnly bool `json:"members_only,omitempty"`
}

// MessageUpdatedEvent is the payload for chat.message.updated: Message reflects the post-edit state.
type MessageUpdatedEvent struct {
	Message     Message `json:"message"`
	MembersOnly bool    `json:"members_only,omitempty"`
}

// MessageDeletedEvent's body is already cleared (soft delete), so consumers get identity + timing only.
type MessageDeletedEvent struct {
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id"`
	DeletedAt      time.Time `json:"deleted_at"`
	MembersOnly    bool      `json:"members_only,omitempty"`
}
