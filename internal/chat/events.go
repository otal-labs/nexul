package chat

import "time"

// Topics published by the chat domain.
const (
	TopicConversationCreated = "chat.conversation.created"
	TopicMessageCreated      = "chat.message.created"
	TopicMessageUpdated      = "chat.message.updated"
	TopicMessageDeleted      = "chat.message.deleted"
)

// Topics returns every topic the chat domain publishes.
func Topics() []string {
	return []string{TopicConversationCreated, TopicMessageCreated, TopicMessageUpdated, TopicMessageDeleted}
}

// ConversationCreatedEvent field names are part of the published contract (ADR 0044) and are additive-only.
type ConversationCreatedEvent struct {
	Conversation Conversation `json:"conversation"`
}

// MessageCreatedEvent is the payload for chat.message.created.
type MessageCreatedEvent struct {
	Message Message `json:"message"`
}

// MessageUpdatedEvent is the payload for chat.message.updated: Message reflects the post-edit state.
type MessageUpdatedEvent struct {
	Message Message `json:"message"`
}

// MessageDeletedEvent's body is already cleared (soft delete), so consumers get identity + timing only.
type MessageDeletedEvent struct {
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id"`
	DeletedAt      time.Time `json:"deleted_at"`
}
