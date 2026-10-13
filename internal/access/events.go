package access

import "github.com/otal-labs/nexul/internal/platform/eventbus"

const TopicGrantChanged = "access.grant.changed"

// TopicComputerGrantChanged is a computer rule changing or ending; it reaches the computer's owner and the person named.
const TopicComputerGrantChanged = "computer.grant_changed"

func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicGrantChanged, Payload: GrantEvent{}},
		{Name: TopicComputerGrantChanged, Payload: ComputerGrantEvent{}},
	}
}

// GrantEvent is the payload of a rule on a doc, folder, play, project, channel, stack, or instance area changing; it reaches the
// person it names, or a role's holders, whose open reads refetch, and an instance area's also the instance Owner. A save
// of a workspace's Access page is one event on the workspace, which reaches every member.
type GrantEvent struct {
	ResourceType string `json:"resource_type" enum:"doc,folder,play,project,channel,stack,workspace,instance_area" jsonschema:"What the rule sits on; a folder's rules reach every doc in it, a stack's its deploys and branch deployments; workspace is a save of the Access page, which may change rules on many of its projects and channels; instance_area is an instance area, keyed by resource_id (dns, connector:github)."`
	ResourceID   string `json:"resource_id"`
	Target       string `json:"target" enum:"person,role" jsonschema:"Whom the rule names: a person, as user_id, or a role, as role_id."`
	UserID       string `json:"user_id"`
	RoleID       string `json:"role_id,omitempty" jsonschema:"The role a rule for a role names; user_id is then empty."`
	ActorID      string `json:"actor_id,omitempty"`
}

// ComputerGrantEvent carries ids and what is held, never the computer's name, and reaches its owner and the person alone.
type ComputerGrantEvent struct {
	ComputerID  string   `json:"computer_id"`
	OwnerID     string   `json:"owner_id" jsonschema:"The computer's owner, the one person who writes its rules."`
	UserID      string   `json:"user_id" jsonschema:"The person the rule names."`
	Permissions []string `json:"permissions" jsonschema:"What the person now holds on the computer, of computers:see, computers:run_agents and computers:run_commands; empty once the rule ends."`
	ActorID     string   `json:"actor_id,omitempty" jsonschema:"Who changed the rule; absent when it ended because the two share no workspace or an account closed."`
	MembersOnly bool     `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}
