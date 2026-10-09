package chat

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

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
	// TopicMessageReactionsChanged is its own topic so a reaction never reads as an edit to a consumer of updated.
	TopicMessageReactionsChanged = "chat.message.reactions_changed"
)

// Topics returns every topic the chat domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicConversationCreated, Payload: ConversationCreatedEvent{}},
		{Name: TopicConversationUpdated, Payload: ConversationUpdatedEvent{}},
		{Name: TopicConversationDeleted, Payload: ConversationDeletedEvent{}},
		{Name: TopicConversationMembersChanged, Payload: ConversationMembersChangedEvent{}},
		{Name: TopicMessageCreated, Payload: MessageCreatedEvent{}},
		{Name: TopicMessageUpdated, Payload: MessageUpdatedEvent{}},
		{Name: TopicMessageDeleted, Payload: MessageDeletedEvent{}},
		{Name: TopicMessageReactionsChanged, Payload: MessageReactionsChangedEvent{}},
	}
}

// MessageReactionsChangedEvent is one person adding or removing one emoji, a delta so concurrent reactions never overwrite each other.
type MessageReactionsChangedEvent struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	UserID         string `json:"user_id" jsonschema:"The person who added or removed the reaction."`
	Emoji          string `json:"emoji" jsonschema:"The emoji itself, such as 👍."`
	Reacted        bool   `json:"reacted" jsonschema:"True when the reaction was added, false when it was removed."`
	MembersOnly    bool   `json:"members_only,omitempty" jsonschema:"Set on a DM or private channel's message, which is never delivered to integrations or automations."`
}

// ConversationCreatedEvent field names are part of the published contract (ADR 0044) and are additive-only.
type ConversationCreatedEvent struct {
	Conversation Conversation `json:"conversation"`
	MembersOnly  bool         `json:"members_only,omitempty" jsonschema:"Set on a DM's or a private channel's creation, which is never delivered to integrations or automations."`
}

// ConversationUpdatedEvent is the payload for chat.conversation.updated: a channel's new name beside the one it replaced.
type ConversationUpdatedEvent struct {
	ConversationID string `json:"conversation_id"`
	WorkspaceID    string `json:"workspace_id"`
	Kind           Kind   `json:"kind" enum:"channel,voice_channel"`
	Name           string `json:"name"`
	PreviousName   string `json:"previous_name"`
	ActorID        string `json:"actor_id,omitempty"`
	MembersOnly    bool   `json:"members_only,omitempty" jsonschema:"Set on a private channel's event, which is never delivered to integrations or automations."`
}

// ConversationDeletedEvent names what chat.conversation.deleted removed, since nothing is left to fetch.
type ConversationDeletedEvent struct {
	ConversationID string `json:"conversation_id"`
	WorkspaceID    string `json:"workspace_id"`
	Kind           Kind   `json:"kind" enum:"channel,voice_channel"`
	Name           string `json:"name"`
	ActorID        string `json:"actor_id,omitempty"`
	// Private and MemberIDs say who could read a private channel, so its deletion reaches only them and the Owner.
	Private     bool     `json:"private,omitempty"`
	MemberIDs   []string `json:"member_ids,omitempty"`
	MembersOnly bool     `json:"members_only,omitempty" jsonschema:"Set on a private channel's event, which is never delivered to integrations or automations."`
}

// ConversationMembersChangedEvent carries ids only; a switch to private names everyone not kept as removed.
type ConversationMembersChangedEvent struct {
	ConversationID string   `json:"conversation_id"`
	WorkspaceID    string   `json:"workspace_id"`
	Private        bool     `json:"private"`
	AddedUserIDs   []string `json:"added_user_ids"`
	RemovedUserIDs []string `json:"removed_user_ids"`
	ActorID        string   `json:"actor_id,omitempty"`
	MembersOnly    bool     `json:"members_only,omitempty" jsonschema:"Set on a private channel's change, a switch to private included, which is never delivered to integrations or automations."`
}

// MessageCreatedEvent is the payload for chat.message.created.
type MessageCreatedEvent struct {
	Message     Message `json:"message" jsonschema:"The message as chat stores it. Its attachment_id is set only on a note: an Agent message on a ticket's thread, posted on the author_id person's behalf, whose markdown file is that attachment of the conversation. Its handoffs are set only on an Agent reply that handed work to other agents: each one's id, driver, model, title, prompt, state (running, done, failed, interrupted or left_running), final reply and steps. A message with author_kind bot has the bot's id as author_id, the name it showed as author_name, the bot's own name as via when the post overrode it, as author_avatar_url the sender's avatar_url or else a path on this instance to the bot's current avatar, and its embeds as the sender posted them less their color."`
	MembersOnly bool    `json:"members_only,omitempty" jsonschema:"Set on a DM or private channel's message, which is never delivered to integrations or automations."`
}

// MessageUpdatedEvent is the payload for chat.message.updated: Message reflects the post-edit state.
type MessageUpdatedEvent struct {
	Message     Message `json:"message" jsonschema:"The message after the change: an edited body, or a note whose markdown file changed, which moves its updated_at and leaves its body as it was."`
	MembersOnly bool    `json:"members_only,omitempty" jsonschema:"Set on a DM or private channel's message, which is never delivered to integrations or automations."`
}

// MessageDeletedEvent's body is already cleared (soft delete), so consumers get identity + timing only.
type MessageDeletedEvent struct {
	ConversationID string    `json:"conversation_id"`
	MessageID      string    `json:"message_id"`
	DeletedAt      time.Time `json:"deleted_at"`
	MembersOnly    bool      `json:"members_only,omitempty" jsonschema:"Set on a DM or private channel's message, which is never delivered to integrations or automations."`
}
