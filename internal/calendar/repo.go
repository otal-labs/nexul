package calendar

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

// Repo stores meetings with their invitees; every write carries its events for the outbox.
type Repo interface {
	ReminderRepo
	// Get returns one meeting with its invitees, or ErrNotFound.
	Get(ctx context.Context, id string) (*Meeting, error)
	// ListVisible returns the meetings in a workspace userID organises or is invited to that may take place in the
	// range, each with its invitees.
	ListVisible(ctx context.Context, workspaceID, userID string, f RangeFilter) ([]*Meeting, error)
	// PageVisible reads one window of those meetings, by start then id, and how many there are.
	PageVisible(ctx context.Context, workspaceID, userID string, f RangeFilter, w paging.Window) ([]*Meeting, int, error)
	// Replacements returns the meetings standing in for occurrences of the given series.
	Replacements(ctx context.Context, seriesIDs []string) ([]*Meeting, error)
	// Save inserts or updates each meeting with its invitees, keeping a reply already stored, in one transaction.
	Save(ctx context.Context, meetings []*Meeting, evts ...eventbus.OutboxEvent) error
	// SaveReply stores an invitee's reply; ErrNotFound when they are not invited.
	SaveReply(ctx context.Context, meetingID, userID string, reply Reply, at time.Time, evts ...eventbus.OutboxEvent) error
}

// ReminderRepo is what the reminder loop reads and marks.
type ReminderRepo interface {
	// NextReminder is the earliest reminder any meeting has planned; ok is false when none has one.
	NextReminder(ctx context.Context) (at time.Time, ok bool, err error)
	// RemindersDue returns the meetings whose planned reminder is at or before now, with their invitees.
	RemindersDue(ctx context.Context, now time.Time) ([]*Meeting, error)
	// MarkReminded moves a meeting's planned reminder on from the one it sent, with the reminder's event; a meeting
	// changed meanwhile, whose plan is no longer sent, is left alone and sends nothing.
	MarkReminded(ctx context.Context, m *Meeting, sent time.Time, evts ...eventbus.OutboxEvent) error
	// DueTickets returns the tickets due on a date in [fromDate, toDate] whose developer has not been reminded of
	// that date, leaving out a ticket in a done column or with no developer.
	DueTickets(ctx context.Context, fromDate, toDate string) ([]DueTicket, error)
	// MarkTicketReminded records that the ticket's developer heard of dueOn, with the reminder's event.
	MarkTicketReminded(ctx context.Context, ticketID, dueOn string, at time.Time, evts ...eventbus.OutboxEvent) error
}
