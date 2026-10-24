package calendar

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Topics published by the calendar domain.
const (
	TopicMeetingCreated   = "meeting.created"
	TopicMeetingUpdated   = "meeting.updated"
	TopicMeetingCancelled = "meeting.cancelled"
	TopicMeetingRestored  = "meeting.restored"
	TopicMeetingReplied   = "meeting.replied"
	TopicReminderDue      = "meeting.reminder_due"
	TopicDueDateReminder  = "due_date.reminder_due"
)

// Topics returns every topic the calendar domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicMeetingCreated, Payload: MeetingEvent{}},
		{Name: TopicMeetingUpdated, Payload: MeetingEvent{}},
		{Name: TopicMeetingCancelled, Payload: MeetingEvent{}},
		{Name: TopicMeetingRestored, Payload: MeetingEvent{}},
		{Name: TopicMeetingReplied, Payload: ReplyEvent{}},
		{Name: TopicReminderDue, Payload: ReminderEvent{}},
		{Name: TopicDueDateReminder, Payload: DueDateReminderEvent{}},
	}
}

// MeetingRef is a meeting event's identity, everything but its notes and invitees.
type MeetingRef struct {
	ID             string     `json:"id"`
	WorkspaceID    string     `json:"workspace_id"`
	OrganiserID    string     `json:"organiser_id"`
	Title          string     `json:"title"`
	StartsAt       time.Time  `json:"starts_at"`
	EndsAt         time.Time  `json:"ends_at"`
	Timezone       string     `json:"timezone"`
	Repeats        string     `json:"repeats" enum:"none,daily,weekly,monthly"`
	SeriesID       string     `json:"series_id,omitempty" jsonschema:"The series this meeting is one changed occurrence of."`
	VoiceChannelID string     `json:"voice_channel_id,omitempty"`
	Link           string     `json:"link,omitempty"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
}

func refOf(m *Meeting) MeetingRef {
	return MeetingRef{
		ID: m.ID, WorkspaceID: m.WorkspaceID, OrganiserID: m.OrganiserID, Title: m.Title, StartsAt: m.StartsAt,
		EndsAt: m.EndsAt, Timezone: m.Timezone, Repeats: string(m.Repeats), SeriesID: m.SeriesID,
		VoiceChannelID: m.VoiceChannelID, Link: m.Link, CancelledAt: m.CancelledAt,
	}
}

// MeetingEvent is the payload for a meeting created, changed, cancelled or restored.
type MeetingEvent struct {
	Meeting MeetingRef `json:"meeting"`
	ActorID string     `json:"actor_id"`
	// UserIDs is everyone whose calendar shows the change, people just uninvited included.
	UserIDs []string `json:"user_ids"`
	// InvitedIDs is who the change invited for the first time.
	InvitedIDs []string `json:"invited_ids"`
}

// ReplyEvent is the payload for meeting.replied.
type ReplyEvent struct {
	MeetingID   string   `json:"meeting_id"`
	WorkspaceID string   `json:"workspace_id"`
	ActorID     string   `json:"actor_id"`
	Reply       string   `json:"reply" enum:"accepted,declined"`
	UserIDs     []string `json:"user_ids"`
}

// ReminderEvent is the payload for meeting.reminder_due: one occurrence is about to start for UserIDs.
type ReminderEvent struct {
	Meeting  MeetingRef `json:"meeting"`
	StartsAt time.Time  `json:"occurrence_starts_at" jsonschema:"When this occurrence starts."`
	UserIDs  []string   `json:"user_ids"`
}

// DueDateReminderEvent is the payload for due_date.reminder_due: a ticket is due tomorrow for its developer.
type DueDateReminderEvent struct {
	TicketID    string   `json:"ticket_id"`
	ProjectID   string   `json:"project_id"`
	WorkspaceID string   `json:"workspace_id"`
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	DueOn       string   `json:"due_on" jsonschema:"The due date as YYYY-MM-DD."`
	UserIDs     []string `json:"user_ids"`
}

func outboxEvent(topic string, payload any) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: payload}
}
