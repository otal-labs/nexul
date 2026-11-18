package issuesync

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// The topics the sync consumes, published by gitprovider, tickets and chat; named here so the domain stays apart from
// them (ADR 0017).
const (
	TopicIssue        = "git.issue"
	TopicIssueComment = "git.issue_comment"

	TopicTicketUpdated          = "ticket.updated"
	TopicTicketStatusChanged    = "ticket.status_changed"
	TopicTicketDeveloperChanged = "ticket.developer_changed"
	TopicTicketLabelsChanged    = "ticket.labels_changed"

	TopicMessageCreated = "chat.message.created"
	TopicMessageUpdated = "chat.message.updated"
	TopicMessageDeleted = "chat.message.deleted"
)

// authorPerson is chat's author kind for a message a person wrote.
const authorPerson = "user"

type sender struct {
	Login string `json:"login"`
}

// issueEvent mirrors the git.issue payload.
type issueEvent struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Action string `json:"action"`
	Issue  Issue  `json:"issue"`
	Sender sender `json:"sender"`
}

// commentEvent mirrors the git.issue_comment payload.
type commentEvent struct {
	Owner       string  `json:"owner"`
	Repo        string  `json:"repo"`
	IssueNumber int     `json:"issue_number"`
	Action      string  `json:"action"`
	Comment     Comment `json:"comment"`
	Sender      sender  `json:"sender"`
}

// HandleIssueEvent is registered on git.issue.
func (s *Service) HandleIssueEvent(ctx context.Context, ev eventbus.Event) error {
	var e issueEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicIssue, err))
	}
	return s.HandleIssue(ctx, e.Owner, e.Repo, e.Action, e.Issue, e.Sender.Login)
}

// HandleCommentEvent is registered on git.issue_comment.
func (s *Service) HandleCommentEvent(ctx context.Context, ev eventbus.Event) error {
	var e commentEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicIssueComment, err))
	}
	return s.HandleComment(ctx, e.Owner, e.Repo, e.IssueNumber, e.Action, e.Comment, e.Sender.Login)
}

// ticketEvent mirrors what the sync reads of the ticket events it consumes.
type ticketEvent struct {
	Ticket struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
	} `json:"ticket"`
	// Added is set on ticket.labels_changed only.
	Added []string `json:"added"`
}

// messageEvent mirrors chat.message.created and chat.message.updated.
type messageEvent struct {
	Message struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversation_id"`
		AuthorKind     string `json:"author_kind"`
	} `json:"message"`
}

// messageDeletedEvent mirrors chat.message.deleted, which no longer says who wrote it.
type messageDeletedEvent struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
}

// HandleTicketEvent is registered on the ticket topics whose change the issue takes.
func (s *Service) HandleTicketEvent(ctx context.Context, ev eventbus.Event) error {
	var e ticketEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	return s.TicketChanged(ctx, e.Ticket.ID, e.Ticket.ProjectID, e.Added)
}

// HandleMessageEvent is registered on chat.message.created and chat.message.updated; only a person's message goes on.
func (s *Service) HandleMessageEvent(ctx context.Context, ev eventbus.Event) error {
	var e messageEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if e.Message.AuthorKind != authorPerson {
		return nil
	}
	return s.MessageChanged(ctx, e.Message.ConversationID, e.Message.ID)
}

// HandleMessageDeletedEvent is registered on chat.message.deleted; the push reads whether a person wrote it.
func (s *Service) HandleMessageDeletedEvent(ctx context.Context, ev eventbus.Event) error {
	var e messageDeletedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicMessageDeleted, err))
	}
	return s.MessageChanged(ctx, e.ConversationID, e.MessageID)
}
