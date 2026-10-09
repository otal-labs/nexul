package tenancy

import "github.com/otal-labs/nexul/internal/platform/eventbus"

const (
	TopicInvitationCreated      = "invitation.created"
	TopicInvitationRevoked      = "invitation.revoked"
	TopicInvitationRedeemed     = "invitation.redeemed"
	TopicInvitationDeleted      = "invitation.deleted"
	TopicAccountAdmitted        = "account.admitted"
	TopicWorkspaceMemberAdded   = "workspace.member.added"
	TopicWorkspaceMemberRemoved = "workspace.member.removed"
	TopicWorkspaceMemberUpdated = "workspace.member.updated"
	TopicWorkspaceUpdated       = "workspace.updated"
	// TopicProjectAccessChanged is the access domain's grant topic, shared so a Project access change rides it (ADR 0097).
	TopicProjectAccessChanged = "access.grant.changed"
)

// Topics declares each topic with its payload; workspace.member.added carries an InvitationEvent when an
// invitation admits the member, a MemberEvent otherwise.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicInvitationCreated, Payload: InvitationEvent{}},
		{Name: TopicInvitationRevoked, Payload: InvitationEvent{}},
		{Name: TopicInvitationRedeemed, Payload: InvitationEvent{}},
		{Name: TopicInvitationDeleted, Payload: InvitationEvent{}},
		{Name: TopicAccountAdmitted, Payload: InvitationEvent{}},
		{Name: TopicWorkspaceMemberAdded, Payload: InvitationEvent{}},
		{Name: TopicWorkspaceMemberAdded, Payload: MemberEvent{}},
		{Name: TopicWorkspaceMemberRemoved, Payload: MemberEvent{}},
		{Name: TopicWorkspaceMemberUpdated, Payload: MemberEvent{}},
		{Name: TopicWorkspaceUpdated, Payload: WorkspaceEvent{}},
		{Name: TopicProjectAccessChanged, Payload: ProjectAccessEvent{}},
	}
}

// ProjectAccessEvent is access.grant.changed for one person's Project access; it reaches them and the workspace's
// holders of members:write.
type ProjectAccessEvent struct {
	ResourceType string `json:"resource_type" enum:"doc,play,project"`
	ResourceID   string `json:"resource_id"`
	UserID       string `json:"user_id"`
	ActorID      string `json:"actor_id,omitempty"`
}

type InvitationEvent struct {
	InvitationID string `json:"invitation_id"`
	ActorID      string `json:"actor_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	WorkspaceID  string `json:"workspace_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// MemberEvent is the payload of a membership change made outside an invitation; updated covers a role or overrides change.
type MemberEvent struct {
	UserID      string   `json:"user_id"`
	WorkspaceID string   `json:"workspace_id"`
	ActorID     string   `json:"actor_id,omitempty"`
	ProjectIDs  []string `json:"project_ids,omitempty" jsonschema:"On workspace.member.updated, the workspace's projects, whose access the change can move."`
}

// WorkspaceEvent is the payload of a rename or slug change; it carries the new name and slug so a client open on the workspace can move its URL.
type WorkspaceEvent struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ActorID     string `json:"actor_id,omitempty"`
}
