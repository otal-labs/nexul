package botwebhook

import "github.com/otal-labs/nexul/internal/platform/eventbus"

// Topics published by the botwebhook domain.
const (
	TopicCreated  = "botwebhook.created"
	TopicUpdated  = "botwebhook.updated"
	TopicDeleted  = "botwebhook.deleted"
	TopicRestored = "botwebhook.restored"
)

// Topics returns every topic the botwebhook domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicCreated, Payload: Event{}},
		{Name: TopicUpdated, Payload: Event{}},
		{Name: TopicDeleted, Payload: Event{}},
		{Name: TopicRestored, Payload: Event{}},
	}
}

// Event is every botwebhook.* payload; it never carries the token or the URL (ADR 0044: fields are additive-only).
type Event struct {
	BotwebhookID   string `json:"botwebhook_id"`
	ConversationID string `json:"conversation_id"`
	WorkspaceID    string `json:"workspace_id"`
	Name           string `json:"name" jsonschema:"The bot's name after the change."`
	ActorID        string `json:"actor_id,omitempty"`
	// Changes is set on botwebhook.updated only.
	Changes     []Change `json:"changes,omitempty" enum:"renamed,avatar,regenerated" jsonschema:"What changed; regenerated means the old URL stopped working."`
	MembersOnly bool     `json:"members_only,omitempty" jsonschema:"Set on a bot of a DM or private channel, whose events are never delivered to integrations or automations."`
}
