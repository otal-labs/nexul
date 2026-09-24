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
func Topics() []string {
	return []string{
		TopicNotificationCreated,
		TopicCategoryCreated, TopicCategoryUpdated, TopicCategoryDeleted,
		TopicTicketTypeCreated, TopicTicketTypeUpdated, TopicTicketTypeDeleted,
		TopicStatusCreated, TopicStatusUpdated, TopicStatusDeleted,
		TopicTicketCategoryChanged,
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
}

// TicketCategoryChangedEvent is the payload for ticket.category_changed.
type TicketCategoryChangedEvent struct {
	TicketID   string `json:"ticket_id"`
	CategoryID string `json:"category_id"`
}

// NotificationCreatedEvent is empty because the topic broadcasts to every browser: a recipient or subject field would leak one user's inbox, so consumers refetch their own.
type NotificationCreatedEvent struct{}

// ticketCreatedEvent is declared consumer-side so this package stays decoupled from tickets (ADR 0017).
type ticketCreatedEvent struct {
	Ticket ticketRef `json:"ticket"`
}

// ticketStatusChangedEvent mirrors ticket.status_changed.
type ticketStatusChangedEvent struct {
	Ticket ticketRef `json:"ticket"`
}

// ticketRef is the slice of a ticket the generation rules need.
type ticketRef struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Developer string `json:"developer"`
	Tester    string `json:"tester"`
}

// docEvent mirrors the docs domain's doc.created / doc.updated payloads.
type docEvent struct {
	Doc docRef `json:"doc"`
}

// docRef is the slice of a doc the generation rules need.
type docRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
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
	return svc.onTicketCreated(CtxWithEventKey(ctx, ev.ID), e.Ticket)
}

// HandleTicketStatusChanged notifies the developer, the tester, and @-mentioned users of that ticket.
func HandleTicketStatusChanged(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e ticketStatusChangedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse ticket.status_changed: %w", err))
	}
	if e.Ticket.ID == "" {
		return apperrs.Fatal(fmt.Errorf("ticket.status_changed missing ticket id"))
	}
	return svc.onTicketStatusChanged(CtxWithEventKey(ctx, ev.ID), e.Ticket)
}

// HandleDocCreated notifies all members for now; ws-22 will narrow to granted users.
func HandleDocCreated(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e docEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse doc.created: %w", err))
	}
	if e.Doc.ID == "" {
		return apperrs.Fatal(fmt.Errorf("doc.created missing doc id"))
	}
	return svc.onDocActivity(CtxWithEventKey(ctx, ev.ID), e.Doc, KindDocCreated)
}

// HandleDocUpdated fans out doc.updated to every user who can access the doc (v1: all members; ws-22 will narrow).
func HandleDocUpdated(ctx context.Context, svc *NotificationService, ev eventbus.Event) error {
	var e docEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse doc.updated: %w", err))
	}
	if e.Doc.ID == "" {
		return apperrs.Fatal(fmt.Errorf("doc.updated missing doc id"))
	}
	return svc.onDocActivity(CtxWithEventKey(ctx, ev.ID), e.Doc, KindDocUpdated)
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
