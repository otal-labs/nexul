package access

import "github.com/otal-labs/nexul/internal/platform/eventbus"

const TopicGrantChanged = "access.grant.changed"

func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicGrantChanged, Payload: GrantEvent{}}}
}

// GrantEvent is the payload of a rule on a doc, folder, play, project, channel, or instance area changing; it reaches the
// person it names, or a role's holders, whose open reads refetch, and an instance area's also the instance Owner. A save
// of a workspace's Access page is one event on the workspace, which reaches every member.
type GrantEvent struct {
	ResourceType string `json:"resource_type" enum:"doc,folder,play,project,channel,workspace,instance_area" jsonschema:"What the rule sits on; a folder's rules reach every doc in it; workspace is a save of the Access page, which may change rules on many of its projects and channels; instance_area is an instance area, keyed by resource_id (dns, connector:github)."`
	ResourceID   string `json:"resource_id"`
	Target       string `json:"target" enum:"person,role" jsonschema:"Whom the rule names: a person, as user_id, or a role, as role_id."`
	UserID       string `json:"user_id"`
	RoleID       string `json:"role_id,omitempty" jsonschema:"The role a rule for a role names; user_id is then empty."`
	ActorID      string `json:"actor_id,omitempty"`
}
