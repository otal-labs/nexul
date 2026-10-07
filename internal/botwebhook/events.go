package botwebhook

// Topics published by the botwebhook domain.
const (
	TopicCreated  = "botwebhook.created"
	TopicUpdated  = "botwebhook.updated"
	TopicDeleted  = "botwebhook.deleted"
	TopicRestored = "botwebhook.restored"
)

// Topics returns every topic the botwebhook domain publishes.
func Topics() []string {
	return []string{TopicCreated, TopicUpdated, TopicDeleted, TopicRestored}
}

// Event is every botwebhook.* payload; it never carries the token or the URL (ADR 0044: fields are additive-only).
type Event struct {
	BotwebhookID   string `json:"botwebhook_id"`
	ConversationID string `json:"conversation_id"`
	WorkspaceID    string `json:"workspace_id"`
	Name           string `json:"name"`
	ActorID        string `json:"actor_id,omitempty"`
	// Changes is set on botwebhook.updated only: what the change touched.
	Changes     []Change `json:"changes,omitempty"`
	MembersOnly bool     `json:"members_only,omitempty"`
}
