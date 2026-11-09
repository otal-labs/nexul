package main

import (
	"context"
	"encoding/json"
	"maps"
	"slices"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/calendar"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/templates"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/voice"
	"github.com/otal-labs/nexul/internal/welcome"
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

// liveCheck answers whether the person on ctx may receive the frame it was prepared for.
type liveCheck func(ctx context.Context) bool

// liveRule names for one topic how a frame is checked: it decodes the payload raw once and returns the check each
// person's socket then runs, so what a frame costs does not grow with the people connected.
type liveRule func(a liveAudience, raw json.RawMessage) liveCheck

// Presence and account frames name nobody but an id; the page that receives them refetches through its own checks.
const topicPresenceChanged = "account.presence_changed"

// topicTopologyCanvas is the frame the canvas consumer pushes with one workspace's whole canvas.
const topicTopologyCanvas = "topology"

// directLiveTopics are pushed straight to the hub, not bridged from the bus.
var directLiveTopics = []string{topicPresenceChanged, topicTopologyCanvas, agent.TopicAgentStream, plays.TopicPlayRun}

// bridgedLiveTopics are the bus topics the browser socket bridges onto: every rule's topic that is not direct.
var bridgedLiveTopics = func() []string {
	topics := slices.Sorted(maps.Keys(liveRules))
	return slices.DeleteFunc(topics, func(t string) bool { return slices.Contains(directLiveTopics, t) })
}()

// liveRules names the read each pushed topic takes; a topic without a rule reaches nobody.
var liveRules = map[string]liveRule{
	topicPresenceChanged:               everyone,
	workspace.TopicNotificationCreated: recipientFrame,
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
	pairing.TopicTunnelRemoved:       ownFrame,
	pairing.TopicSetupTurnChanged:    ownFrame,
	pairing.TopicSetupFinished:       ownFrame,
	pairing.TopicSetupTurnActivity:   ownFrame,
	pairing.TopicHarnessSwitched:     ownFrame,
	runner.TopicPersonalChanged:      ownFrame,
	pairing.TopicFactsChanged:        ownFrame,
	pairing.TopicPairFailed:          ownFrame,
	pairing.TopicPresenceChanged:     ownFrame,
	access.TopicGrantChanged:         grantFrame,
	access.TopicComputerGrantChanged: computerGrantFrame,

	tenancy.TopicWorkspaceMemberAdded:   memberFrame,
	tenancy.TopicWorkspaceMemberRemoved: memberFrame,
	tenancy.TopicWorkspaceMemberUpdated: memberFrame,
	tenancy.TopicWorkspaceUpdated:       workspaceFrame,
	tenancy.TopicWorkspaceArchived:      archiveFrame,
	tenancy.TopicWorkspaceRestored:      archiveFrame,
	roles.TopicUpdated:                  workspaceFrame,
	welcome.TopicProgressChanged:        welcomeFrame,

	tickets.TopicCreated:                 ticketFrame,
	tickets.TopicUpdated:                 ticketFrame,
	tickets.TopicStatusChanged:           ticketFrame,
	tickets.TopicAssigneeChanged:         ticketFrame,
	tickets.TopicDeveloperChanged:        ticketFrame,
	tickets.TopicTesterChanged:           ticketFrame,
	tickets.TopicDueDateChanged:          ticketFrame,
	tickets.TopicFinished:                ticketFrame,
	tickets.TopicDeleted:                 ticketDeletedFrame,
	tickets.TopicLinkCreated:             ticketLinkFrame,
	tickets.TopicLinkDeleted:             ticketLinkFrame,
	workspace.TopicTicketCategoryChanged: ticketIDFrame,
	workspace.TopicTicketProjectChanged:  ticketProjectChangedFrame,

	workspace.TopicCategoryCreated:     boardFrame("category"),
	workspace.TopicCategoryUpdated:     boardFrame("category"),
	workspace.TopicCategoryDeleted:     boardFrame("category"),
	workspace.TopicTicketTypeCreated:   boardFrame("ticket_type"),
	workspace.TopicTicketTypeUpdated:   boardFrame("ticket_type"),
	workspace.TopicTicketTypeDeleted:   boardFrame("ticket_type"),
	workspace.TopicStatusCreated:       boardFrame("status"),
	workspace.TopicStatusUpdated:       boardFrame("status"),
	workspace.TopicStatusDeleted:       boardFrame("status"),
	workspace.TopicProjectSetupChanged: projectFrame,

	docs.TopicCreated:                        docFrame,
	docs.TopicUpdated:                        docFrame,
	docs.TopicMoved:                          docFrame,
	docs.TopicDeleted:                        docDeletedFrame,
	docs.TopicFolderCreated:                  docFolderFrame,
	docs.TopicFolderUpdated:                  docFolderFrame,
	docs.TopicFolderDeleted:                  docFolderFrame,
	docs.TopicWatchersChanged:                docFrame,
	docs.TopicClarificationRoundStarted:      docFrame,
	docs.TopicClarificationRoundPosted:       docFrame,
	docs.TopicClarificationRoundEnded:        docFrame,
	docs.TopicClarificationRoundAnswered:     docFrame,
	docs.TopicClarificationAnswerSaved:       docFrame,
	docs.TopicClarificationAnswerCleared:     docFrame,
	docs.TopicClarificationAnythingElseSaved: docFrame,
	docs.TopicClarificationClosed:            docFrame,

	memories.TopicCreated:        memoryFrame,
	memories.TopicUpdated:        memoryFrame,
	memories.TopicDeleted:        memoryProjectFrame,
	memories.TopicAnswerSaved:    memoryProjectFrame,
	memories.TopicAnswerCleared:  memoryProjectFrame,
	memories.TopicSourceAdded:    memoryProjectFrame,
	memories.TopicSourceChanged:  memoryProjectFrame,
	memories.TopicSourceRemoved:  memoryProjectFrame,
	memories.TopicDraftSaved:     memoryProjectFrame,
	memories.TopicDraftDismissed: memoryProjectFrame,

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
	plays.TopicCreated:                   playFrame,
	plays.TopicUpdated:                   playFrame,
	plays.TopicDeleted:                   playFrame,
	plays.TopicAutoPlayCreated:           autoPlayFrame,
	plays.TopicAutoPlayUpdated:           autoPlayFrame,
	plays.TopicAutoPlayDeleted:           autoPlayFrame,
	plays.TopicAutoPlayLimitsUpdated:     autoPlayFrame,
	plays.TopicQueued:                    playRunFrame,
	plays.TopicQueueUpdated:              playRunFrame,
	plays.TopicQueueResumed:              playRunFrame,
	plays.TopicSweepStarted:              sweepFrame,
	plays.TopicSweepFinished:             sweepFrame,
	botwebhook.TopicCreated:              botwebhookFrame,
	botwebhook.TopicUpdated:              botwebhookFrame,
	botwebhook.TopicDeleted:              botwebhookFrame,
	botwebhook.TopicRestored:             botwebhookFrame,
	calendar.TopicMeetingCreated:         recipientFrame,
	calendar.TopicMeetingUpdated:         recipientFrame,
	calendar.TopicMeetingCancelled:       recipientFrame,
	calendar.TopicMeetingRestored:        recipientFrame,
	calendar.TopicMeetingReplied:         recipientFrame,

	deploy.TopicStackCreated:           stackFrame,
	deploy.TopicStackUpdated:           stackFrame,
	deploy.TopicStackDeleted:           anywhere(permissions.StacksRead),
	deploy.TopicContainerStopped:       outageFrame,
	deploy.TopicContainerRecovered:     outageFrame,
	deploy.TopicErrorRateCrossed:       outageFrame,
	deploy.TopicErrorRateRecovered:     outageFrame,
	deploy.TopicDeployUpdated:          deployFrame,
	runner.TopicDeployBuildStarted:     deployFrame,
	runner.TopicDeployBuildProgress:    deployFrame,
	runner.TopicDeployBuildCompleted:   deployFrame,
	runner.TopicDeployDeployProgress:   deployFrame,
	runner.TopicDeployStatusChanged:    deployFrame,
	runner.TopicRunnerConnected:        runnerFrame,
	runner.TopicRunnerDisconnected:     runnerFrame,
	runner.TopicMachineDeleted:         recipientFrame,
	runner.TopicInstanceUpgradeChanged: anywhere(permissions.InstanceRead),
	topicTopologyCanvas:                topologyCanvasFrame,
	dns.TopicRecordChanged:             anywhere(permissions.DNSRead),
	dns.TopicTunnelChanged:             anywhere(permissions.DNSRead),
	dns.TopicGatewayChanged:            anywhere(permissions.DNSRead),
	dns.TopicExposureChanged:           anywhere(permissions.DNSRead),

	repository.TopicInstallationAssigned:   installationFrame,
	repository.TopicInstallationUnassigned: installationFrame,
}

// frame prepares a frame once and returns the check each person's socket runs on it.
func (a liveAudience) frame(topic string, payload any) liveCheck {
	rule, ok := liveRules[topic]
	if !ok {
		return never
	}
	raw, ok := payload.(json.RawMessage)
	if !ok {
		b, err := json.Marshal(payload)
		if err != nil {
			return never
		}
		raw = b
	}
	return rule(a, raw)
}

// allows is frame's check for one person.
func (a liveAudience) allows(ctx context.Context, topic string, payload any) bool {
	return a.frame(topic, payload)(ctx)
}

func decode(raw json.RawMessage, v any) bool {
	return json.Unmarshal(raw, v) == nil
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
}

func never(context.Context) bool  { return false }
func always(context.Context) bool { return true }

// prepared is a rule that decodes its payload into P once per frame and builds from it the check each person runs.
func prepared[P any](prepare func(a liveAudience, p P) liveCheck) liveRule {
	return func(a liveAudience, raw json.RawMessage) liveCheck {
		var p P
		if !decode(raw, &p) {
			return never
		}
		return prepare(a, p)
	}
}

// perPerson is a rule that decodes its payload into P once per frame and asks check about each person.
func perPerson[P any](check func(ctx context.Context, a liveAudience, p P) bool) liveRule {
	return prepared(func(a liveAudience, p P) liveCheck {
		return func(ctx context.Context) bool { return check(ctx, a, p) }
	})
}

// readsRow reads the row a frame names for the first person whose check reaches it and answers everyone from that
// read; a row that cannot be read reaches nobody.
func readsRow(load func(context.Context, string) (func(context.Context) error, error), id string) liveCheck {
	var (
		check  func(context.Context) error
		loaded bool
	)
	return func(ctx context.Context) bool {
		if !loaded {
			check, _ = load(ctx, id) // a failed load leaves check nil
			loaded = true
		}
		return check != nil && check(ctx) == nil
	}
}

type userPayload struct {
	UserID string `json:"user_id"`
}

type workspacePayload struct {
	WorkspaceID string `json:"workspace_id"`
}

type projectPayload struct {
	ProjectID string `json:"project_id"`
}

func everyone(liveAudience, json.RawMessage) liveCheck { return always }

var ownFrame = perPerson(func(ctx context.Context, _ liveAudience, p userPayload) bool {
	return p.UserID != "" && p.UserID == actorID(ctx)
})

// computerGrantFrame reaches a computer's owner and the person its rule names, and nobody else (ADR 0148).
var computerGrantFrame = perPerson(func(ctx context.Context, _ liveAudience, p struct {
	OwnerID string `json:"owner_id"`
	UserID  string `json:"user_id"`
}) bool {
	me := actorID(ctx)
	return me != "" && (p.OwnerID == me || p.UserID == me)
})

// recipientFrame reaches only the people whose inbox the frame names, or who saw the machine it says went.
var recipientFrame = perPerson(func(ctx context.Context, _ liveAudience, p struct {
	UserIDs []string `json:"user_ids"`
}) bool {
	return slices.Contains(p.UserIDs, actorID(ctx))
})

// manageAction is what writing the rules on each entity takes there.
var manageAction = map[string]permissions.Action{
	"doc": permissions.PermissionsWrite, "folder": permissions.PermissionsWrite, "play": permissions.PlaysWrite, "stack": permissions.StacksWrite,
}

// grantFrame reaches the person whose rule changed, or the holders of the role it names, and whoever manages the rules
// it changed: a doc's or a play's, a machine's Manage holders and the instance Owner, or for Project access the project's workspace.
var grantFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	UserID       string `json:"user_id"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	RoleID       string `json:"role_id"`
}) bool {
	if p.UserID != "" && p.UserID == actorID(ctx) {
		return true
	}
	if holds, err := a.access.HoldsRole(ctx, actorID(ctx), p.RoleID); holds && err == nil {
		return true
	}
	switch p.ResourceType {
	case "project":
		return a.access.RequireProject(ctx, p.ResourceID, permissions.ProjectsAccess) == nil
	case "doc", "folder", "play", "stack":
		manages, _, err := a.access.Decide(ctx, actorID(ctx), p.ResourceType, p.ResourceID, manageAction[p.ResourceType])
		return err == nil && manages
	case "channel":
		return a.access.ManagesRules(ctx, actorID(ctx), p.ResourceType, p.ResourceID)
	case "workspace":
		return a.access.Require(ctx, p.ResourceID, permissions.Member) == nil
	case "instance_area":
		owner, err := a.access.IsInstanceOwner(ctx, actorID(ctx))
		return err == nil && owner
	case "machine":
		return a.access.RequireMachine(ctx, p.ResourceID, permissions.MachinesWrite) == nil
	}
	return false
})

var memberFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
}) bool {
	return p.UserID == actorID(ctx) || a.access.Require(ctx, p.WorkspaceID, permissions.Member) == nil
})

// welcomeFrame reaches the person and whoever sees them on Team: accounts:read anywhere, or members:write in a
// workspace they are in.
var welcomeFrame = perPerson(func(ctx context.Context, a liveAudience, p userPayload) bool {
	if p.UserID == "" {
		return false
	}
	if p.UserID == actorID(ctx) || a.access.RequireAnywhere(ctx, permissions.AccountsRead) == nil {
		return true
	}
	workspaceIDs, err := a.access.WorkspacesOf(ctx, p.UserID)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(workspaceIDs, func(id string) bool { return a.access.Require(ctx, id, permissions.MembersWrite) == nil })
})

// workspaceFrame carries the workspace's new name and slug, which only its members may see.
var workspaceFrame = perPerson(func(ctx context.Context, a liveAudience, p workspacePayload) bool {
	return a.access.Require(ctx, p.WorkspaceID, permissions.Member) == nil
})

// archiveFrame reaches the members it names, whom an archived workspace no longer resolves, and whoever holds the restore.
func archiveFrame(a liveAudience, raw json.RawMessage) liveCheck {
	recipient, restorer := recipientFrame(a, raw), anywhere(permissions.InstanceWrite)(a, raw)
	return func(ctx context.Context) bool { return recipient(ctx) || restorer(ctx) }
}

// runnerFrame reaches whoever sees the machine the runner belongs to; a frame naming none reaches nobody.
var runnerFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	MachineID string `json:"machine_id"`
}) bool {
	return p.MachineID != "" && a.access.RequireMachine(ctx, p.MachineID, permissions.MachinesSee) == nil
})

var topologyCanvasFrame = perPerson(func(ctx context.Context, a liveAudience, p workspacePayload) bool {
	return p.WorkspaceID != "" && a.access.Require(ctx, p.WorkspaceID, permissions.TopologyRead) == nil
})

// installationFrame reaches who may create projects in the workspace: the installations card names those workspaces
// as using an account.
var installationFrame = perPerson(func(ctx context.Context, a liveAudience, p workspacePayload) bool {
	return p.WorkspaceID != "" && a.access.Require(ctx, p.WorkspaceID, permissions.ProjectsCreate) == nil
})

func anywhere(action permissions.Action) liveRule {
	return func(a liveAudience, _ json.RawMessage) liveCheck {
		return func(ctx context.Context) bool { return a.access.RequireAnywhere(ctx, action) == nil }
	}
}

var ticketFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	Ticket projectPayload `json:"ticket"`
}) bool {
	return a.access.RequireProject(ctx, p.Ticket.ProjectID, permissions.TicketsRead) == nil
})

var ticketLinkFrame = prepared(func(a liveAudience, p struct {
	Link struct {
		TicketID string `json:"ticket_id"`
	} `json:"link"`
}) liveCheck {
	return readsTicket(a, p.Link.TicketID)
})

var ticketIDFrame = prepared(func(a liveAudience, p struct {
	TicketID string `json:"ticket_id"`
}) liveCheck {
	return readsTicket(a, p.TicketID)
})

// ticketProjectChangedFrame reaches whoever reads tickets in the project the ticket left or the one it joined, since each board changes; the frame names ids only.
var ticketProjectChangedFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	ProjectID         string `json:"project_id"`
	PreviousProjectID string `json:"previous_project_id"`
}) bool {
	readsIn := func(projectID string) bool {
		return projectID != "" && a.access.RequireProject(ctx, projectID, permissions.TicketsRead) == nil
	}
	return readsIn(p.ProjectID) || readsIn(p.PreviousProjectID)
})

// ticketDeletedFrame carries the gone ticket's title, which a reader of its project's tickets saw on the board.
var ticketDeletedFrame = perPerson(func(ctx context.Context, a liveAudience, p projectPayload) bool {
	return p.ProjectID != "" && a.access.RequireProject(ctx, p.ProjectID, permissions.TicketsRead) == nil
})

func readsTicket(a liveAudience, id string) liveCheck {
	return readsRow(a.tickets.ReadCheck, id)
}

// boardFrame reads a project's columns, types, or categories, which every member of its workspace sees; a delete
// carries only the id, so it goes to everyone.
func boardFrame(key string) liveRule {
	return perPerson(func(ctx context.Context, a liveAudience, p map[string]projectPayload) bool {
		projectID := p[key].ProjectID
		return projectID == "" || a.access.RequireProject(ctx, projectID, permissions.Member) == nil
	})
}

// projectFrame reaches whoever may open the project, the people whose sidebar lists it.
var projectFrame = perPerson(func(ctx context.Context, a liveAudience, p projectPayload) bool {
	return p.ProjectID != "" && a.access.RequireProject(ctx, p.ProjectID, permissions.Member) == nil
})

var docFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	Doc struct {
		ID string `json:"id"`
	} `json:"doc"`
}) bool {
	return readsDoc(ctx, a, p.Doc.ID)
})

// docDeletedFrame carries the gone doc's title, which a reader of its project's docs saw in the Docs list.
var docDeletedFrame = perPerson(func(ctx context.Context, a liveAudience, p projectPayload) bool {
	return p.ProjectID != "" && a.access.RequireProject(ctx, p.ProjectID, permissions.DocsRead) == nil
})

// docFolderFrame carries a folder's name, which a reader of the project's docs sees in the Docs list.
var docFolderFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	Folder projectPayload `json:"folder"`
}) bool {
	return a.access.RequireProject(ctx, p.Folder.ProjectID, permissions.DocsRead) == nil
})

func readsDoc(ctx context.Context, a liveAudience, id string) bool {
	ok, err := a.access.Can(ctx, actorID(ctx), id, permissions.DocsRead)
	return err == nil && ok
}

var memoryFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	Memory projectPayload `json:"memory"`
}) bool {
	return a.access.RequireProject(ctx, p.Memory.ProjectID, permissions.MemoriesRead) == nil
})

// memoryProjectFrame reads a memories frame's top-level project; a frame without one reaches nobody.
var memoryProjectFrame = perPerson(func(ctx context.Context, a liveAudience, p projectPayload) bool {
	return p.ProjectID != "" && a.access.RequireProject(ctx, p.ProjectID, permissions.MemoriesRead) == nil
})

// conversationFrame covers every chat-shaped payload: a conversation, a message, or a bare conversation id.
var conversationFrame = prepared(func(a liveAudience, p struct {
	ConversationID string `json:"conversation_id"`
	Conversation   struct {
		ID string `json:"id"`
	} `json:"conversation"`
	Message struct {
		ConversationID string `json:"conversation_id"`
	} `json:"message"`
}) liveCheck {
	return readsRow(a.chat.ReadCheck, p.ConversationID+p.Conversation.ID+p.Message.ConversationID)
})

// conversationDeletedFrame reaches whoever could read the channel before it went: a private one's members and the Owner.
var conversationDeletedFrame = perPerson(func(ctx context.Context, a liveAudience, p chat.ConversationDeletedEvent) bool {
	return a.chat.ReadsDeleted(ctx, p)
})

// membersChangedFrame reaches the channel's readers and the people it removed, who can no longer read it.
func membersChangedFrame(a liveAudience, raw json.RawMessage) liveCheck {
	var p chat.ConversationMembersChangedEvent
	if !decode(raw, &p) {
		return never
	}
	reads := conversationFrame(a, raw)
	return func(ctx context.Context) bool { return slices.Contains(p.RemovedUserIDs, actorID(ctx)) || reads(ctx) }
}

// botwebhookFrame reaches whoever lists the bot's conversation's bots: botwebhook:read and a read of the conversation.
func botwebhookFrame(a liveAudience, raw json.RawMessage) liveCheck {
	var p workspacePayload
	if !decode(raw, &p) {
		return never
	}
	reads := conversationFrame(a, raw)
	return func(ctx context.Context) bool {
		return a.access.Require(ctx, p.WorkspaceID, permissions.BotwebhookRead) == nil && reads(ctx)
	}
}

// playRunFrame reaches whoever reads the run's target and its trail: the run's starter, or someone who can see its play.
var playRunFrame = prepared(func(a liveAudience, p struct {
	TargetType  plays.TargetType `json:"target_type"`
	TargetID    string           `json:"target_id"`
	PlayID      string           `json:"play_id"`
	StarterID   string           `json:"starter_id"`
	WorkspaceID string           `json:"workspace_id"`
}) liveCheck {
	readsTarget := playRunTarget(a, p.TargetType, p.TargetID)
	return func(ctx context.Context) bool {
		actor := actorID(ctx)
		started := actor != "" && actor == p.StarterID
		if p.PlayID != "" && !started && !a.access.HasPermission(ctx, actor, p.WorkspaceID, permissions.PlaysRead, "play", p.PlayID) {
			return false
		}
		return readsTarget(ctx)
	}
})

// playRunTarget is the check that a person reads what a run is on: a doc, an interview's project, a stack, or a ticket.
func playRunTarget(a liveAudience, targetType plays.TargetType, id string) liveCheck {
	if targetType == plays.TargetDoc {
		return func(ctx context.Context) bool { return readsDoc(ctx, a, id) }
	}
	if targetType == plays.TargetInterview {
		return func(ctx context.Context) bool {
			return a.access.RequireProject(ctx, id, permissions.MemoriesRead) == nil
		}
	}
	if targetType == plays.TargetStack {
		return func(ctx context.Context) bool {
			_, err := a.deploy.GetStack(ctx, id)
			return err == nil
		}
	}
	return readsTicket(a, id)
}

// playFrame reaches whoever can see the play (plays:read on it).
var playFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Play        struct {
		ID          string `json:"id"`
		WorkspaceID string `json:"workspace_id"`
	} `json:"play"`
}) bool {
	id, workspaceID := p.ID+p.Play.ID, p.WorkspaceID+p.Play.WorkspaceID
	if workspaceID == "" {
		return false
	}
	return a.access.HasPermission(ctx, actorID(ctx), workspaceID, permissions.PlaysRead, "play", id)
})

// autoPlayFrame reaches whoever reads the workspace's auto plays and can see the auto play's play; the workspace and
// play are top level on a delete, nested otherwise, and a limits frame names no play.
var autoPlayFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	WorkspaceID string `json:"workspace_id"`
	PlayID      string `json:"play_id"`
	AutoPlay    struct {
		WorkspaceID string `json:"workspace_id"`
		PlayID      string `json:"play_id"`
	} `json:"auto_play"`
}) bool {
	workspaceID, playID := p.WorkspaceID+p.AutoPlay.WorkspaceID, p.PlayID+p.AutoPlay.PlayID
	if workspaceID == "" || a.access.Require(ctx, workspaceID, permissions.AutoplaysRead) != nil {
		return false
	}
	return playID == "" || a.access.HasPermission(ctx, actorID(ctx), workspaceID, permissions.PlaysRead, "play", playID)
})

// stackFrame reaches who reads the stack: through its rules, a branch deployment's base stack's; an instance stack's
// readers see its machine and read dns (ADR 0151).
var stackFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	Stack struct {
		ID          string `json:"id"`
		ProjectID   string `json:"project_id"`
		DerivedFrom string `json:"derived_from"`
		Machine     string `json:"machine"`
	} `json:"stack"`
}) bool {
	if p.Stack.ProjectID == "" {
		return a.access.RequireAnywhere(ctx, permissions.StacksRead) == nil &&
			a.access.RequireAnywhere(ctx, permissions.DNSRead) == nil &&
			a.access.RequireMachineOfStack(ctx, p.Stack.Machine, permissions.MachinesSee) == nil
	}
	ruled := p.Stack.ID
	if p.Stack.DerivedFrom != "" {
		ruled = p.Stack.DerivedFrom
	}
	return a.access.RequireStack(ctx, p.Stack.ProjectID, ruled, permissions.StacksRead) == nil
})

// outageFrame reaches whoever reads the stack an outage or an error-rate episode is on, through the same read the stack's own fetch runs.
var outageFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	StackID string `json:"stack_id"`
}) bool {
	_, err := a.deploy.GetStack(ctx, p.StackID)
	return err == nil
})

var deployFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	ID string `json:"id"`
}) bool {
	_, err := a.deploy.Get(ctx, p.ID)
	return err == nil
})

// sweepFrame reaches whoever can see the sweep's play, as its digest does.
var sweepFrame = perPerson(func(ctx context.Context, a liveAudience, p struct {
	Sweep struct {
		PlayID      string `json:"play_id"`
		WorkspaceID string `json:"workspace_id"`
	} `json:"sweep"`
}) bool {
	return p.Sweep.WorkspaceID != "" && a.access.HasPermission(ctx, actorID(ctx), p.Sweep.WorkspaceID, permissions.PlaysRead, "play", p.Sweep.PlayID)
})
