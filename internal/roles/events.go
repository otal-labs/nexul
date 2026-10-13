package roles

import "github.com/otal-labs/nexul/internal/platform/eventbus"

const TopicUpdated = "role.updated"

func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicUpdated, Payload: RoleEvent{}}}
}

// RoleEvent is the payload of a role changing or going; every member holding it refetches what they hold.
type RoleEvent struct {
	RoleID            string   `json:"role_id"`
	WorkspaceID       string   `json:"workspace_id"`
	ActorID           string   `json:"actor_id,omitempty"`
	WelcomeStepsAdded []string `json:"welcome_steps_added" jsonschema:"The Welcome wizard steps this save ticked that the role did not tick before; each is given again to everyone holding the role."`
}
