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
	Description       string   `json:"description" jsonschema:"The role's one-line description as it now stands, at most 140 characters; empty when it has none."`
	WelcomeStepsAdded []string `json:"welcome_steps_added" jsonschema:"The Welcome wizard steps this save ticked that the role did not tick before; each is given again to everyone holding the role."`
	ShowLockedActions bool     `json:"show_locked_actions" jsonschema:"Whether the role, as saved, shows its holders an action they can't take dimmed with the reason; false when the role went."`
}
