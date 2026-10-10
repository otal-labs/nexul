package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/storage"
)

// automationScope holds each event to its workspace's automations (ADR 0095); a topic without a rule reaches none.
type automationScope struct {
	lookup *storage.EventWorkspacesRepo
}

// scopeRule resolves one topic's payload to its workspaces; every marks an instance-level event.
type scopeRule func(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) (ids []string, every bool, err error)

// Workspaces implements automations.EventScope.
func (s automationScope) Workspaces(ctx context.Context, topic string, payload []byte) ([]string, bool, error) {
	rule, ok := scopeRules[topic]
	if !ok {
		return nil, false, nil
	}
	ids, every, err := rule(ctx, s.lookup, payload)
	if err != nil {
		return nil, false, fmt.Errorf("resolve workspace of %s: %w", topic, err)
	}
	return ids, every, nil
}

var scopeRules = map[string]scopeRule{
	"invitation.created": instanceScope, "invitation.revoked": instanceScope, "invitation.redeemed": instanceScope,
	"invitation.deleted": instanceScope, "account.admitted": instanceScope, "account.disabled": instanceScope,
	"account.reactivated": instanceScope, "account.removed": instanceScope, "account.restored": instanceScope,
	"account.profile_updated": instanceScope, "account.presence_changed": instanceScope,
	"session.created": instanceScope, "session.revoked": instanceScope,
	"identity.linked": instanceScope, "identity.unlinked": instanceScope,
	"personal_access_token.minted": instanceScope, "personal_access_token.revoked": instanceScope,
	"computer.paired": instanceScope, "computer.setup_confirmed": instanceScope,
	"computer.setup_unconfirmed": instanceScope, "computer.tunnel_created": instanceScope,
	"computer.tunnel_removed": instanceScope, "computer.tunnel_status_changed": instanceScope,
	"computer.setup_turn_changed": instanceScope, "computer.setup_finished": instanceScope,
	"computer.setup_turn_activity": instanceScope, "computer.harness_switched": instanceScope,
	"runner.connected": instanceScope, "runner.disconnected": instanceScope, "runner.heartbeat": instanceScope,
	"instance.upgrade_requested": instanceScope, "instance.upgrade_changed": instanceScope,
	"topology.updated": workspaceScope, "notification.created": notificationScope,
	"dns.record_changed": instanceScope, "dns.tunnel_changed": instanceScope, "dns.gateway_changed": instanceScope,
	"dns.exposure_changed": instanceScope, "instance_template.updated": instanceScope,

	"workspace.member.added": workspaceScope, "workspace.member.removed": workspaceScope,
	"workspace.member.updated": workspaceScope, "workspace.updated": workspaceScope, "role.updated": workspaceScope,
	"interview_template.updated": workspaceScope, "memory.deleted": workspaceScope, "play.deleted": workspaceScope,
	"interview_answer.saved": workspaceScope, "interview_answer.cleared": workspaceScope,
	"interview_source.added": workspaceScope, "interview_source.changed": workspaceScope,
	"interview_source.removed": workspaceScope, "interview_draft.saved": workspaceScope, "interview_draft.dismissed": workspaceScope,
	"play.run_started": workspaceScope, "play.run_waiting": workspaceScope, "play.run_finished": workspaceScope,
	"chat.conversation.updated": workspaceScope, "chat.conversation.deleted": workspaceScope, "chat.conversation.members_changed": workspaceScope,
	"botwebhook.created": workspaceScope, "botwebhook.updated": workspaceScope, "botwebhook.deleted": workspaceScope, "botwebhook.restored": workspaceScope,
	"play.created": nestedWorkspaceScope("play"), "play.updated": nestedWorkspaceScope("play"),
	"auto_play.created": nestedWorkspaceScope("auto_play"), "auto_play.updated": nestedWorkspaceScope("auto_play"), "auto_play.deleted": workspaceScope, "auto_play.limits_updated": workspaceScope,
	"memory.created": nestedWorkspaceScope("memory"), "memory.updated": nestedWorkspaceScope("memory"),
	"chat.conversation.created": nestedWorkspaceScope("conversation"),
	"chat.message.created":      conversationScope, "chat.message.updated": conversationScope,
	"chat.message.deleted": conversationScope, "chat.message.reactions_changed": conversationScope,
	"voice.occupancy.changed": conversationScope,

	"ticket.created": nestedProjectScope("ticket"), "ticket.updated": nestedProjectScope("ticket"),
	"ticket.status_changed": nestedProjectScope("ticket"), "ticket.finished": nestedProjectScope("ticket"),
	"ticket.assignee_changed": nestedProjectScope("ticket"), "ticket.developer_changed": nestedProjectScope("ticket"),
	"ticket.tester_changed": nestedProjectScope("ticket"), "ticket.test_passed": nestedProjectScope("ticket"),
	"ticket.test_failed": nestedProjectScope("ticket"), "ticket.deleted": projectScope, "doc.deleted": projectScope,
	"ticket.unblocked": projectScope, "doc.settled": nestedProjectScope("doc"),
	"doc.created": nestedProjectScope("doc"), "doc.updated": nestedProjectScope("doc"), "doc.moved": nestedProjectScope("doc"),
	"doc.folder.created": nestedProjectScope("folder"), "doc.folder.updated": nestedProjectScope("folder"),
	"doc.folder.deleted":                    nestedProjectScope("folder"),
	"doc.watchers.changed":                  nestedProjectScope("doc"),
	"doc.clarification.round_started":       nestedProjectScope("doc"),
	"doc.clarification.round_posted":        nestedProjectScope("doc"),
	"doc.clarification.round_ended":         nestedProjectScope("doc"),
	"doc.clarification.round_answered":      nestedProjectScope("doc"),
	"doc.clarification.answer_saved":        nestedProjectScope("doc"),
	"doc.clarification.answer_cleared":      nestedProjectScope("doc"),
	"doc.clarification.anything_else_saved": nestedProjectScope("doc"),
	"doc.clarification.closed":              nestedProjectScope("doc"),
	"ticket.link_created":                   linkScope, "ticket.link_deleted": linkScope, "ticket.category_changed": ticketIDScope,
	"category.created": boardScope("category"), "category.updated": boardScope("category"),
	"category.deleted": boardScope("category"), "ticket_type.created": boardScope("ticket_type"),
	"ticket_type.updated": boardScope("ticket_type"), "ticket_type.deleted": boardScope("ticket_type"),
	"status.created": boardScope("status"), "status.updated": boardScope("status"), "status.deleted": boardScope("status"),

	"service.created": stackScope, "service.updated": stackScope, "service.deleted": stackScope,
	"deploy.requested": deployScope, "deploy.status_changed": deployScope, "deploy.cancel_requested": deployScope,
	"deploy.updated": deployScope, "deploy.build_started": deployScope, "deploy.build_progress": deployScope,
	"deploy.build_completed": deployScope, "deploy.deploy_progress": deployScope, "deploy.log": deployScope,

	"git.pr_opened": pullRequestScope, "git.pr_merged": pullRequestScope, "git.pr_closed": pullRequestScope,
	"git.pr_review_submitted": repoScope, "git.pr_comment": pullRequestScope, "git.push": repoScope,
	"git.branch_deleted": repoScope, "git.provider_event": providerEventScope, "review.status_changed": reviewScope,

	"notification.push_requested": pushScope, "access.grant.changed": grantScope,
}

func decodeScope(raw json.RawMessage, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}

// one turns a single lookup into a rule result: a gone entity, or an empty id, belongs to no workspace.
func one(id string, err error) ([]string, bool, error) {
	if err != nil || id == "" {
		return nil, false, err
	}
	return []string{id}, false, nil
}

func instanceScope(context.Context, *storage.EventWorkspacesRepo, json.RawMessage) ([]string, bool, error) {
	return nil, true, nil
}

func workspaceScope(_ context.Context, _ *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	return one(p.WorkspaceID, nil)
}

func nestedWorkspaceScope(key string) scopeRule {
	return func(_ context.Context, _ *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
		var p map[string]struct {
			WorkspaceID string `json:"workspace_id"`
		}
		if err := decodeScope(raw, &p); err != nil {
			return nil, false, err
		}
		return one(p[key].WorkspaceID, nil)
	}
}

func projectScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		ProjectID string `json:"project_id"`
	}
	if err := decodeScope(raw, &p); err != nil || p.ProjectID == "" {
		return nil, false, err
	}
	return one(l.OfProject(ctx, p.ProjectID))
}

func nestedProjectScope(key string) scopeRule {
	return func(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
		var p map[string]struct {
			ProjectID string `json:"project_id"`
		}
		if err := decodeScope(raw, &p); err != nil || p[key].ProjectID == "" {
			return nil, false, err
		}
		return one(l.OfProject(ctx, p[key].ProjectID))
	}
}

// boardScope reads a column, type, or category's project; a delete carries only the id, so it reaches every workspace.
func boardScope(key string) scopeRule {
	return func(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
		var p map[string]struct {
			ProjectID string `json:"project_id"`
		}
		if err := decodeScope(raw, &p); err != nil {
			return nil, false, err
		}
		if p[key].ProjectID == "" {
			return nil, true, nil
		}
		return one(l.OfProject(ctx, p[key].ProjectID))
	}
}

func ticketIDScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		TicketID string `json:"ticket_id"`
	}
	if err := decodeScope(raw, &p); err != nil || p.TicketID == "" {
		return nil, false, err
	}
	return one(l.OfTicket(ctx, p.TicketID))
}

func linkScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		Link json.RawMessage `json:"link"`
	}
	if err := decodeScope(raw, &p); err != nil || p.Link == nil {
		return nil, false, err
	}
	return ticketIDScope(ctx, l, p.Link)
}

func conversationScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		ConversationID string `json:"conversation_id"`
		Message        struct {
			ConversationID string `json:"conversation_id"`
		} `json:"message"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	id := p.ConversationID + p.Message.ConversationID
	if id == "" {
		return nil, false, nil
	}
	return one(l.OfConversation(ctx, id))
}

// stackScope reads a stack's project; an instance stack belongs to no project and so to every workspace.
func stackScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		ProjectID string `json:"project_id"`
		Stack     struct {
			ProjectID string `json:"project_id"`
		} `json:"stack"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	projectID := p.ProjectID + p.Stack.ProjectID
	if projectID == "" {
		return nil, true, nil
	}
	return one(l.OfProject(ctx, projectID))
}

// deployScope reads the deploy's stack; an instance stack's deploys reach every workspace.
func deployScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	id, err := l.OfDeploy(ctx, p.ID)
	if err != nil {
		return nil, false, err
	}
	if id == "" {
		return nil, true, nil
	}
	return []string{id}, false, nil
}

func repoScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	return one(l.OfRepo(ctx, p.Owner, p.Repo))
}

// pullRequestScope reaches every workspace among the PR's linked tickets, or its repository's when it links none.
func pullRequestScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		PR struct {
			LinkedTicketIDs []string `json:"linked_ticket_ids"`
		} `json:"pr"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	var ids []string
	for _, ticketID := range p.PR.LinkedTicketIDs {
		id, err := l.OfTicket(ctx, ticketID)
		if err != nil {
			return nil, false, err
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		return ids, false, nil
	}
	return repoScope(ctx, l, raw)
}

func providerEventScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		Repository *struct {
			Owner string `json:"owner"`
			Name  string `json:"name"`
		} `json:"repository"`
	}
	if err := decodeScope(raw, &p); err != nil || p.Repository == nil {
		return nil, false, err
	}
	return one(l.OfRepo(ctx, p.Repository.Owner, p.Repository.Name))
}

func reviewScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		Repo string `json:"repo"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	owner, name, ok := strings.Cut(p.Repo, "/")
	if !ok {
		return nil, false, nil
	}
	return one(l.OfRepo(ctx, owner, name))
}

// pushScope reaches the workspaces of the notices being pushed; one with no workspace is the person's own, instance-wide.
func pushScope(_ context.Context, _ *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		Notifications []struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"notifications"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	var ids []string
	for _, n := range p.Notifications {
		if n.WorkspaceID == "" {
			return nil, true, nil
		}
		ids = append(ids, n.WorkspaceID)
	}
	return ids, false, nil
}

// notificationScope reaches the notices' workspace; a notice with none is the person's own, instance-wide.
func notificationScope(_ context.Context, _ *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	if p.WorkspaceID == "" {
		return nil, true, nil
	}
	return []string{p.WorkspaceID}, false, nil
}

// grantScope reads where the changed grant applies: a workspace, or the doc, play, or project it was set on.
func grantScope(ctx context.Context, l *storage.EventWorkspacesRepo, raw json.RawMessage) ([]string, bool, error) {
	var p struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}
	if err := decodeScope(raw, &p); err != nil {
		return nil, false, err
	}
	switch p.ResourceType {
	case "workspace":
		return one(p.ResourceID, nil)
	case "doc":
		return one(l.OfDoc(ctx, p.ResourceID))
	case "play":
		return one(l.OfPlay(ctx, p.ResourceID))
	case "project":
		return one(l.OfProject(ctx, p.ResourceID))
	}
	return nil, true, nil
}
