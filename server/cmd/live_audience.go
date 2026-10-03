package main

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/templates"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/workspace"
)

// liveAudience checks a live frame against the same read its entity's own fetch runs, so a socket never receives
// what its person could not load.
type liveAudience struct {
	access  *access.Service
	tickets *tickets.Service
	chat    *chat.Service
	deploy  *deploy.Service
}

// liveRule answers for one topic whether the actor on ctx may receive a frame carrying raw.
type liveRule func(ctx context.Context, a liveAudience, raw json.RawMessage) bool

// Presence and account frames name nobody but an id; the page that receives them refetches through its own checks.
const topicPresenceChanged = "account.presence_changed"

// topicTopologyCanvas is the frame the canvas consumer pushes with the whole stored canvas.
const topicTopologyCanvas = "topology"

// liveRules names the read each pushed topic takes; a topic without a rule reaches nobody.
var liveRules = map[string]liveRule{
	topicPresenceChanged:               everyone,
	workspace.TopicNotificationCreated: everyone,
	auth.TopicAccountAdmitted:          everyone,
	auth.TopicAccountDisabled:          everyone,
	auth.TopicAccountReactivated:       everyone,
	auth.TopicAccountRemoved:           everyone,
	auth.TopicAccountRestored:          everyone,
	auth.TopicProfileUpdated:           everyone,
	templates.TopicUpdated:             everyone,

	auth.TopicTokenMinted:            ownFrame,
	auth.TopicTokenRevoked:           ownFrame,
	auth.TopicSessionCreated:         ownFrame,
	auth.TopicSessionRevoked:         ownFrame,
	pairing.TopicComputerPaired:      ownFrame,
	pairing.TopicSetupConfirmed:      ownFrame,
	pairing.TopicSetupUnconfirmed:    ownFrame,
	pairing.TopicTunnelCreated:       ownFrame,
	pairing.TopicTunnelRemoved:       ownFrame,
	pairing.TopicTunnelStatusChanged: ownFrame,
	pairing.TopicSetupTurnChanged:    ownFrame,
	pairing.TopicSetupFinished:       ownFrame,
	pairing.TopicSetupTurnActivity:   ownFrame,
	access.TopicGrantChanged:         grantFrame,

	tenancy.TopicWorkspaceMemberAdded:   memberFrame,
	tenancy.TopicWorkspaceMemberRemoved: memberFrame,
	tenancy.TopicWorkspaceMemberUpdated: memberFrame,
	tenancy.TopicWorkspaceUpdated:       workspaceFrame,
	roles.TopicUpdated:                  workspaceFrame,

	tickets.TopicCreated:                 ticketFrame,
	tickets.TopicUpdated:                 ticketFrame,
	tickets.TopicStatusChanged:           ticketFrame,
	tickets.TopicAssigneeChanged:         ticketFrame,
	tickets.TopicDeveloperChanged:        ticketFrame,
	tickets.TopicTesterChanged:           ticketFrame,
	tickets.TopicFinished:                ticketFrame,
	tickets.TopicLinkCreated:             ticketLinkFrame,
	tickets.TopicLinkDeleted:             ticketLinkFrame,
	workspace.TopicTicketCategoryChanged: ticketIDFrame,

	workspace.TopicCategoryCreated:   boardFrame("category"),
	workspace.TopicCategoryUpdated:   boardFrame("category"),
	workspace.TopicCategoryDeleted:   boardFrame("category"),
	workspace.TopicTicketTypeCreated: boardFrame("ticket_type"),
	workspace.TopicTicketTypeUpdated: boardFrame("ticket_type"),
	workspace.TopicTicketTypeDeleted: boardFrame("ticket_type"),
	workspace.TopicStatusCreated:     boardFrame("status"),
	workspace.TopicStatusUpdated:     boardFrame("status"),
	workspace.TopicStatusDeleted:     boardFrame("status"),

	docs.TopicCreated:         docFrame,
	docs.TopicUpdated:         docFrame,
	docs.TopicMoved:           docFrame,
	docs.TopicFolderCreated:   docFolderFrame,
	docs.TopicFolderUpdated:   docFolderFrame,
	docs.TopicFolderDeleted:   docFolderFrame,
	docs.TopicWatchersChanged: docFrame,

	memories.TopicCreated: memoryFrame,
	memories.TopicUpdated: memoryFrame,
	memories.TopicDeleted: memoryDeletedFrame,

	chat.TopicConversationCreated:        conversationFrame,
	chat.TopicConversationUpdated:        conversationFrame,
	chat.TopicConversationDeleted:        conversationDeletedFrame,
	chat.TopicConversationMembersChanged: membersChangedFrame,
	chat.TopicMessageCreated:             conversationFrame,
	chat.TopicMessageUpdated:             conversationFrame,
	chat.TopicMessageDeleted:             conversationFrame,
	chat.TopicMessageReactionsChanged:    conversationFrame,
	voice.TopicOccupancyChanged:          conversationFrame,
	agent.TopicAgentStream:               conversationFrame,
	plays.TopicPlayRun:                   playRunFrame,

	deploy.TopicStackCreated:         stackFrame,
	deploy.TopicStackUpdated:         stackFrame,
	deploy.TopicStackDeleted:         anywhere(permissions.StacksRead),
	deploy.TopicDeployUpdated:        deployFrame,
	runner.TopicDeployBuildStarted:   deployFrame,
	runner.TopicDeployBuildProgress:  deployFrame,
	runner.TopicDeployBuildCompleted: deployFrame,
	runner.TopicDeployDeployProgress: deployFrame,
	runner.TopicDeployStatusChanged:  deployFrame,
	runner.TopicRunnerConnected:      anywhere(permissions.RunnersRead),
	runner.TopicRunnerDisconnected:   anywhere(permissions.RunnersRead),
	topicTopologyCanvas:              anywhere(permissions.TopologyRead),
	dns.TopicRecordChanged:           anywhere(permissions.DNSRead),
	dns.TopicTunnelChanged:           anywhere(permissions.DNSRead),
	dns.TopicGatewayChanged:          anywhere(permissions.DNSRead),
	dns.TopicExposureChanged:         anywhere(permissions.DNSRead),
}

// allows is the hub's Audience.
func (a liveAudience) allows(ctx context.Context, topic string, payload any) bool {
	rule, ok := liveRules[topic]
	if !ok {
		return false
	}
	raw, ok := payload.(json.RawMessage)
	if !ok {
		b, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		raw = b
	}
	return rule(ctx, a, raw)
}

func decode(raw json.RawMessage, v any) bool {
	return json.Unmarshal(raw, v) == nil
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
}

func everyone(context.Context, liveAudience, json.RawMessage) bool { return true }

func ownFrame(ctx context.Context, _ liveAudience, raw json.RawMessage) bool {
	var p struct {
		UserID string `json:"user_id"`
	}
	return decode(raw, &p) && p.UserID != "" && p.UserID == actorID(ctx)
}

// grantFrame reaches the person whose grant changed and, for Project access, whoever manages the project's workspace.
func grantFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}
	if ownFrame(ctx, a, raw) {
		return true
	}
	return decode(raw, &p) && p.ResourceType == "project" && a.access.RequireProject(ctx, p.ResourceID, permissions.MembersWrite) == nil
}

func memberFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		UserID      string `json:"user_id"`
		WorkspaceID string `json:"workspace_id"`
	}
	if !decode(raw, &p) {
		return false
	}
	return p.UserID == actorID(ctx) || a.access.Require(ctx, p.WorkspaceID, permissions.Member) == nil
}

// workspaceFrame carries the workspace's new name and slug, which only its members may see.
func workspaceFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		WorkspaceID string `json:"workspace_id"`
	}
	return decode(raw, &p) && a.access.Require(ctx, p.WorkspaceID, permissions.Member) == nil
}

func anywhere(action permissions.Action) liveRule {
	return func(ctx context.Context, a liveAudience, _ json.RawMessage) bool {
		return a.access.RequireAnywhere(ctx, action) == nil
	}
}

func ticketFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		Ticket struct {
			ProjectID string `json:"project_id"`
		} `json:"ticket"`
	}
	return decode(raw, &p) && a.access.RequireProject(ctx, p.Ticket.ProjectID, permissions.TicketsRead) == nil
}

func ticketLinkFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		Link struct {
			TicketID string `json:"ticket_id"`
		} `json:"link"`
	}
	return decode(raw, &p) && readsTicket(ctx, a, p.Link.TicketID)
}

func ticketIDFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		TicketID string `json:"ticket_id"`
	}
	return decode(raw, &p) && readsTicket(ctx, a, p.TicketID)
}

func readsTicket(ctx context.Context, a liveAudience, id string) bool {
	_, err := a.tickets.Get(ctx, id)
	return err == nil
}

// boardFrame reads a project's columns, types, or categories, which every member of its workspace sees; a delete
// carries only the id, so it goes to everyone.
func boardFrame(key string) liveRule {
	return func(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
		var p map[string]struct {
			ProjectID string `json:"project_id"`
		}
		if !decode(raw, &p) {
			return false
		}
		projectID := p[key].ProjectID
		return projectID == "" || a.access.RequireProject(ctx, projectID, permissions.Member) == nil
	}
}

func docFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		Doc struct {
			ID string `json:"id"`
		} `json:"doc"`
	}
	return decode(raw, &p) && readsDoc(ctx, a, p.Doc.ID)
}

// docFolderFrame carries a folder's name, which a reader of the project's docs sees in the Docs list.
func docFolderFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		Folder struct {
			ProjectID string `json:"project_id"`
		} `json:"folder"`
	}
	return decode(raw, &p) && a.access.RequireProject(ctx, p.Folder.ProjectID, permissions.DocsRead) == nil
}

func readsDoc(ctx context.Context, a liveAudience, id string) bool {
	ok, err := a.access.Can(ctx, actorID(ctx), id, permissions.DocsRead)
	return err == nil && ok
}

func memoryFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		Memory struct {
			ProjectID string `json:"project_id"`
		} `json:"memory"`
	}
	return decode(raw, &p) && a.access.RequireProject(ctx, p.Memory.ProjectID, permissions.MemoriesRead) == nil
}

// memoryDeletedFrame reads the project the memory lived in; a frame from before it carried one reaches nobody.
func memoryDeletedFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		ProjectID string `json:"project_id"`
	}
	return decode(raw, &p) && p.ProjectID != "" && a.access.RequireProject(ctx, p.ProjectID, permissions.MemoriesRead) == nil
}

// conversationFrame covers every chat-shaped payload: a conversation, a message, or a bare conversation id.
func conversationFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		ConversationID string `json:"conversation_id"`
		Conversation   struct {
			ID string `json:"id"`
		} `json:"conversation"`
		Message struct {
			ConversationID string `json:"conversation_id"`
		} `json:"message"`
	}
	if !decode(raw, &p) {
		return false
	}
	id := p.ConversationID + p.Conversation.ID + p.Message.ConversationID
	_, err := a.chat.GetConversation(ctx, id)
	return err == nil
}

// conversationDeletedFrame reaches whoever could read the channel before it went: a private one's members and the Owner.
func conversationDeletedFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p chat.ConversationDeletedEvent
	return decode(raw, &p) && a.chat.ReadsDeleted(ctx, p)
}

// membersChangedFrame reaches the channel's readers and the people it removed, who can no longer read it.
func membersChangedFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p chat.ConversationMembersChangedEvent
	if !decode(raw, &p) {
		return false
	}
	return slices.Contains(p.RemovedUserIDs, actorID(ctx)) || conversationFrame(ctx, a, raw)
}

func playRunFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		TargetType plays.TargetType `json:"target_type"`
		TargetID   string           `json:"target_id"`
	}
	if !decode(raw, &p) {
		return false
	}
	if p.TargetType == plays.TargetDoc {
		return readsDoc(ctx, a, p.TargetID)
	}
	if p.TargetType == plays.TargetInterview {
		return a.access.RequireProject(ctx, p.TargetID, permissions.MemoriesRead) == nil
	}
	return readsTicket(ctx, a, p.TargetID)
}

func stackFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		Stack struct {
			ProjectID string `json:"project_id"`
		} `json:"stack"`
	}
	if !decode(raw, &p) {
		return false
	}
	if p.Stack.ProjectID == "" {
		return a.access.RequireAnywhere(ctx, permissions.StacksRead) == nil
	}
	return a.access.RequireProject(ctx, p.Stack.ProjectID, permissions.StacksRead) == nil
}

func deployFrame(ctx context.Context, a liveAudience, raw json.RawMessage) bool {
	var p struct {
		ID string `json:"id"`
	}
	if !decode(raw, &p) {
		return false
	}
	_, err := a.deploy.Get(ctx, p.ID)
	return err == nil
}
