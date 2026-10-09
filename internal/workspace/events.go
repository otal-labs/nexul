package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicNotificationCreated is published by the workspace capability after it fans out new notifications (ws-26).
const TopicNotificationCreated = "notification.created"

// TopicNotificationPushRequested names the new rows and their recipients for the push sender; never bridged to the browser.
const TopicNotificationPushRequested = "notification.push_requested"

// Category topics (ws-15); web refetches the board's swimlanes and filter bar via WS push when categories change.
const (
	TopicCategoryCreated = "category.created"
	TopicCategoryUpdated = "category.updated"
	TopicCategoryDeleted = "category.deleted"
)

// TicketType topics (ws-15) refetch the filter bar's type dimension via WS push.
const (
	TopicTicketTypeCreated = "ticket_type.created"
	TopicTicketTypeUpdated = "ticket_type.updated"
	TopicTicketTypeDeleted = "ticket_type.deleted"
)

// Status topics update every swimlane via WS push when a status column changes.
const (
	TopicStatusCreated = "status.created"
	TopicStatusUpdated = "status.updated"
	TopicStatusDeleted = "status.deleted"
)

// TopicTicketCategoryChanged re-renders the board's swimlane via WS push.
const TopicTicketCategoryChanged = "ticket.category_changed"

// Topics returns every topic the workspace domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicNotificationCreated, Payload: NotificationCreatedEvent{}},
		{Name: TopicNotificationPushRequested, Payload: NotificationPushRequestedEvent{}},
		{Name: TopicCategoryCreated, Payload: CategoryEvent{}},
		{Name: TopicCategoryUpdated, Payload: CategoryEvent{}},
		{Name: TopicCategoryDeleted, Payload: CategoryEvent{}},
		{Name: TopicTicketTypeCreated, Payload: TicketTypeEvent{}},
		{Name: TopicTicketTypeUpdated, Payload: TicketTypeEvent{}},
		{Name: TopicTicketTypeDeleted, Payload: TicketTypeEvent{}},
		{Name: TopicStatusCreated, Payload: StatusEvent{}},
		{Name: TopicStatusUpdated, Payload: StatusEvent{}},
		{Name: TopicStatusDeleted, Payload: StatusEvent{}},
		{Name: TopicTicketCategoryChanged, Payload: TicketCategoryChangedEvent{}},
	}
}

// CategoryEvent is the payload for category.created/updated/deleted.
type CategoryEvent struct {
	Category Category `json:"category"`
}

// TicketTypeEvent is the payload for ticket_type.created/updated/deleted.
type TicketTypeEvent struct {
	TicketType TicketType `json:"ticket_type"`
}

// StatusEvent is the payload for status.created/updated/deleted.
type StatusEvent struct {
	Status Status `json:"status"`
	// PreviousKind is set on status.updated, so a reader can tell whether the column's stage moved.
	PreviousKind StatusKind `json:"previous_kind,omitempty" jsonschema:"On status.updated, the column's stage before the change."`
}

// TicketCategoryChangedEvent is the payload for ticket.category_changed.
type TicketCategoryChangedEvent struct {
	TicketID   string `json:"ticket_id"`
	CategoryID string `json:"category_id"`
	ProjectID  string `json:"project_id,omitempty" jsonschema:"The ticket's project."`
}

// NotificationCreatedEvent names the new notices' recipients and place; the live socket delivers it to the recipients alone.
type NotificationCreatedEvent struct {
	UserIDs     []string `json:"user_ids" jsonschema:"The people whose inbox gained or lifted a notice."`
	WorkspaceID string   `json:"workspace_id" jsonschema:"The workspace whose inbox holds the notices; empty for a subject that belongs to no workspace."`
	ProjectID   string   `json:"project_id,omitempty" jsonschema:"The subject's project, when it has one."`
}

// NotificationPushRequestedEvent is the payload for notification.push_requested: ids only, the phone fetches the content.
type NotificationPushRequestedEvent struct {
	Notifications []NotificationPushItem `json:"notifications"`
}

// NotificationPushItem is one new inbox row; WorkspaceID is empty when the subject is not workspace-scoped.
type NotificationPushItem struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

// ticketCreatedEvent is declared consumer-side so this package stays decoupled from tickets (ADR 0017).
type ticketCreatedEvent struct {
	Ticket           ticketRef `json:"ticket"`
	MentionedUserIDs []string  `json:"mentioned_user_ids"`
}

// ticketUpdatedEvent mirrors ticket.updated: a title, body, or source doc edit and the people it newly mentions.
type ticketUpdatedEvent struct {
	Ticket           ticketRef `json:"ticket"`
	ActorID          string    `json:"actor_id"`
	MentionedUserIDs []string  `json:"mentioned_user_ids"`
}

// ticketStatusChangedEvent mirrors ticket.status_changed.
type ticketStatusChangedEvent struct {
	Ticket ticketRef `json:"ticket"`
	Actor  struct {
		UserID string `json:"user_id"`
	} `json:"actor"`
}

// ticketRef is the slice of a ticket the generation rules need.
type ticketRef struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Developer string `json:"developer"`
	Tester    string `json:"tester"`
	Reporter  struct {
		Login string `json:"login"`
	} `json:"reporter"`
}

// docEvent mirrors the docs domain's doc.created / doc.updated payloads.
type docEvent struct {
	Doc              docRef   `json:"doc"`
	ActorID          string   `json:"actor_id"`
	MentionedUserIDs []string `json:"mentioned_user_ids"`
	LockChanged      bool     `json:"lock_changed"`
}

// docRef is the slice of a doc the generation rules need.
type docRef struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}

// docQuestionsPostedEvent mirrors the docs domain's doc.clarification.round_posted payload.
type docQuestionsPostedEvent struct {
	Doc           docRef `json:"doc"`
	StartedBy     string `json:"started_by"`
	QuestionCount int    `json:"question_count"`
	NoGaps        bool   `json:"no_gaps"`
}

// docRoundAnsweredEvent mirrors the docs domain's doc.clarification.round_answered payload.
type docRoundAnsweredEvent struct {
	Doc       docRef `json:"doc"`
	StartedBy string `json:"started_by"`
	ActorID   string `json:"actor_id"`
}

// memoryUpdatedEvent mirrors the memories domain's memory.updated payload.
type memoryUpdatedEvent struct {
	Memory    memoryRef `json:"memory"`
	AuthorID  string    `json:"author_id"`
	AuthorVia string    `json:"author_via"`
}

// memoryRef is the slice of a memory the generation rules need.
type memoryRef struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	Version     int    `json:"version"`
}

// playRunFinishedEvent mirrors the plays domain's play.run_finished payload.
type playRunFinishedEvent struct {
	TrailID     string `json:"trail_id"`
	PlayLabel   string `json:"play_label"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	TargetTitle string `json:"target_title"`
	StarterID   string `json:"starter_id"`
	WorkspaceID string `json:"workspace_id"`
	Outcome     string `json:"outcome"`
}

// HandlePlayRunFinished notifies the run's starter once per terminal outcome, linking to the target.
func HandlePlayRunFinished(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e playRunFinishedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse play.run_finished: %w", err))
	}
	if e.TrailID == "" || e.StarterID == "" {
		return apperrs.Fatal(fmt.Errorf("play.run_finished missing trail id or starter"))
	}
	return svc.onPlayRunFinished(CtxWithEventKey(ctx, ev.ID), e)
}

// HandlePlayRunWaiting tells the starter their run stopped on a question, linking to the target.
func HandlePlayRunWaiting(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e playRunFinishedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse play.run_waiting: %w", err))
	}
	if e.TrailID == "" || e.StarterID == "" {
		return apperrs.Fatal(fmt.Errorf("play.run_waiting missing trail id or starter"))
	}
	return svc.onPlayRunWaiting(CtxWithEventKey(ctx, ev.ID), e)
}

// HandleTicketCreated notifies the developer, the tester, and every user @-mentioned in the title or body.
func HandleTicketCreated(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e ticketCreatedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse ticket.created: %w", err))
	}
	if e.Ticket.ID == "" {
		return apperrs.Fatal(fmt.Errorf("ticket.created missing ticket id"))
	}
	return svc.onTicketCreated(CtxWithEventKey(ctx, ev.ID), e.Ticket, e.MentionedUserIDs)
}

// HandleTicketUpdated tells each person an edit newly @-mentions, never the editor.
func HandleTicketUpdated(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e ticketUpdatedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse ticket.updated: %w", err))
	}
	if e.Ticket.ID == "" {
		return apperrs.Fatal(fmt.Errorf("ticket.updated missing ticket id"))
	}
	return svc.onTicketUpdated(CtxWithEventKey(ctx, ev.ID), e)
}

// HandleTicketStatusChanged notifies the developer, the tester, and @-mentioned users of that ticket, never the mover.
func HandleTicketStatusChanged(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e ticketStatusChangedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse ticket.status_changed: %w", err))
	}
	if e.Ticket.ID == "" {
		return apperrs.Fatal(fmt.Errorf("ticket.status_changed missing ticket id"))
	}
	return svc.onTicketStatusChanged(CtxWithEventKey(ctx, ev.ID), e.Ticket, e.Actor.UserID)
}

// HandleDocCreated tells the people the new doc @-mentions; nobody else is watching it yet but its creator.
func HandleDocCreated(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e docEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse doc.created: %w", err))
	}
	if e.Doc.ID == "" {
		return apperrs.Fatal(fmt.Errorf("doc.created missing doc id"))
	}
	return svc.onDocActivity(CtxWithEventKey(ctx, ev.ID), e, KindDocCreated)
}

// HandleDocUpdated notifies the doc's watchers except the editor; people the edit newly @-mentions get a mention instead.
// A lock or unlock is no edit and tells nobody.
func HandleDocUpdated(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e docEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse doc.updated: %w", err))
	}
	if e.Doc.ID == "" {
		return apperrs.Fatal(fmt.Errorf("doc.updated missing doc id"))
	}
	if e.LockChanged {
		return nil
	}
	return svc.onDocActivity(CtxWithEventKey(ctx, ev.ID), e, KindDocUpdated)
}

// HandleDocQuestionsPosted tells the doc's watchers a round asked new questions, never the round's starter; a round
// that found no gaps asked nothing and tells nobody.
func HandleDocQuestionsPosted(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e docQuestionsPostedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse doc.clarification.round_posted: %w", err))
	}
	if e.Doc.ID == "" {
		return apperrs.Fatal(fmt.Errorf("doc.clarification.round_posted missing doc id"))
	}
	return svc.onDocQuestionsPosted(CtxWithEventKey(ctx, ev.ID), e)
}

// HandleDocRoundAnswered tells the round's starter its last question was answered, unless they answered it themself.
func HandleDocRoundAnswered(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e docRoundAnsweredEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse doc.clarification.round_answered: %w", err))
	}
	if e.Doc.ID == "" || e.StartedBy == "" {
		return apperrs.Fatal(fmt.Errorf("doc.clarification.round_answered missing doc id or starter"))
	}
	return svc.onDocRoundAnswered(CtxWithEventKey(ctx, ev.ID), e)
}

// HandleMemoryUpdated notifies every workspace member who holds memories:read, except the author.
func HandleMemoryUpdated(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e memoryUpdatedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse memory.updated: %w", err))
	}
	if e.Memory.ID == "" {
		return apperrs.Fatal(fmt.Errorf("memory.updated missing memory id"))
	}
	return svc.onMemoryUpdated(CtxWithEventKey(ctx, ev.ID), e.Memory, e.AuthorID, e.AuthorVia)
}

// mentionRe matches @-mention tokens (GitHub-style usernames) anywhere in a ticket's title or body.
var mentionRe = regexp.MustCompile(`@([a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?)`)

// extractMentions lowercases logins since GitHub usernames are case-insensitive.
func extractMentions(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range mentionRe.FindAllStringSubmatch(text, -1) {
		login := strings.ToLower(strings.TrimSpace(m[1]))
		if login == "" || seen[login] {
			continue
		}
		seen[login] = true
		out = append(out, login)
	}
	return out
}
