// Package calendar is the workspace calendar (ADR 0168): meetings, who is invited and their replies, and the
// reminders sent before a meeting starts and the day before a ticket is due.
package calendar

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Repeat is how often a meeting comes round.
type Repeat string

// The repeats a meeting may take; a monthly one skips a month without its day.
const (
	RepeatNone    Repeat = "none"
	RepeatDaily   Repeat = "daily"
	RepeatWeekly  Repeat = "weekly"
	RepeatMonthly Repeat = "monthly"
)

// Reply is an invitee's answer to a meeting.
type Reply string

// The replies; everyone starts pending.
const (
	ReplyPending  Reply = "pending"
	ReplyAccepted Reply = "accepted"
	ReplyDeclined Reply = "declined"
)

// Scope is which occurrences of a repeating meeting a change covers.
type Scope string

// The scopes: one occurrence alone, or it and every later one.
const (
	ScopeThis      Scope = "this"
	ScopeFollowing Scope = "following"
)

// Limits on what a meeting holds.
const (
	maxTitle       = 200
	maxNotes       = 10000
	maxLink        = 2000
	maxInvitees    = 200
	maxDuration    = 7 * 24 * time.Hour
	maxRemind      = 7 * 24 * 60
	DefaultRemind  = 10
	maxRangeLength = 62 * 24 * time.Hour
)

// Invitee is one person invited to a meeting besides its organiser.
type Invitee struct {
	UserID    string     `json:"user_id"`
	Reply     Reply      `json:"reply"`
	RepliedAt *time.Time `json:"replied_at,omitempty"`
}

// Meeting is one meeting, or one repeating series of them, in one workspace. A meeting with a SeriesID is a single
// occurrence of that series changed on its own, standing in for the occurrence that started at ReplacesAt.
type Meeting struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	OrganiserID string    `json:"organiser_id"`
	Title       string    `json:"title"`
	Notes       string    `json:"notes"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	// Timezone is the IANA zone the meeting repeats in, so a weekly 10:00 stays 10:00 across a clock change.
	Timezone       string     `json:"timezone"`
	VoiceChannelID string     `json:"voice_channel_id"`
	Link           string     `json:"link"`
	Repeats        Repeat     `json:"repeats"`
	RepeatUntil    *time.Time `json:"repeat_until,omitempty"`
	RemindMinutes  int        `json:"remind_minutes"`
	SeriesID       string     `json:"series_id,omitempty"`
	ReplacesAt     *time.Time `json:"replaces_at,omitempty"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
	Invitees       []Invitee  `json:"invitees"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	// NextRemindAt and NextOccurrence are the reminder the loop sends next and the occurrence it is for.
	NextRemindAt   *time.Time `json:"-"`
	NextOccurrence *time.Time `json:"-"`
}

// Cancelled says the meeting is cancelled.
func (m *Meeting) Cancelled() bool { return m.CancelledAt != nil }

// Repeating says the meeting is a series.
func (m *Meeting) Repeating() bool { return m.Repeats != RepeatNone }

// Invited says userID is one of the meeting's invitees.
func (m *Meeting) Invited(userID string) bool {
	return slices.ContainsFunc(m.Invitees, func(i Invitee) bool { return i.UserID == userID })
}

// Sees says userID may see the meeting: its organiser and the people invited.
func (m *Meeting) Sees(userID string) bool { return m.OrganiserID == userID || m.Invited(userID) }

// Attendees are who a reminder goes to: the organiser and every invitee who has not declined.
func (m *Meeting) Attendees() []string {
	out := []string{m.OrganiserID}
	for _, i := range m.Invitees {
		if i.Reply != ReplyDeclined {
			out = append(out, i.UserID)
		}
	}
	return out
}

// Audience is everyone who sees the meeting.
func (m *Meeting) Audience() []string {
	out := []string{m.OrganiserID}
	for _, i := range m.Invitees {
		out = append(out, i.UserID)
	}
	return out
}

// InviteeIDs lists the invitees' user ids.
func (m *Meeting) InviteeIDs() []string {
	out := make([]string, len(m.Invitees))
	for i, inv := range m.Invitees {
		out[i] = inv.UserID
	}
	return out
}

// Occurrence is one time a meeting takes place: Slot is where its series put it, which names it in a change.
type Occurrence struct {
	MeetingID string    `json:"meeting_id"`
	Slot      time.Time `json:"slot"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
}

// Details are a meeting's own fields as a create sets them or a change patches them; nil keeps a patched field.
type Details struct {
	Title          *string
	Notes          *string
	StartsAt       *time.Time
	EndsAt         *time.Time
	Timezone       *string
	VoiceChannelID *string
	Link           *string
	Repeats        *Repeat
	// RepeatUntil set to the zero time clears the end, so the series repeats for good.
	RepeatUntil   *time.Time
	RemindMinutes *int
	// InviteeIDs replaces the invitees; AddInviteeIDs and RemoveInviteeIDs apply after it, in that order.
	InviteeIDs       *[]string
	AddInviteeIDs    []string
	RemoveInviteeIDs []string
}

// Change is a patch to a meeting: its details, whether it is cancelled, and which occurrences it covers.
type Change struct {
	Details
	// Cancelled true cancels and false restores; nil leaves it.
	Cancelled *bool
	// Slot names one occurrence of a repeating meeting, and Scope whether the change covers it alone or it and every
	// later one; without a Slot the change covers the whole meeting.
	Slot  *time.Time
	Scope Scope
}

// RangeFilter keeps the meetings that may take place between From and To; zero values leave a side open.
type RangeFilter struct {
	From time.Time
	To   time.Time
}

// DueTicket is a ticket due soon whose developer has not been reminded of its date.
type DueTicket struct {
	TicketID    string
	ProjectID   string
	WorkspaceID string
	Key         string
	Title       string
	DueOn       string
	DeveloperID string
	Timezone    string
}

// apply writes d's set fields onto m, the invitees last; it validates nothing.
func (d Details) apply(m *Meeting) {
	setIf(&m.Title, d.Title)
	setIf(&m.Notes, d.Notes)
	setIf(&m.StartsAt, d.StartsAt)
	setIf(&m.EndsAt, d.EndsAt)
	setIf(&m.Timezone, d.Timezone)
	setIf(&m.VoiceChannelID, d.VoiceChannelID)
	setIf(&m.Link, d.Link)
	setIf(&m.Repeats, d.Repeats)
	setIf(&m.RemindMinutes, d.RemindMinutes)
	if d.RepeatUntil != nil {
		m.RepeatUntil = nil
		if !d.RepeatUntil.IsZero() {
			until := d.RepeatUntil.UTC()
			m.RepeatUntil = &until
		}
	}
	if d.InviteeIDs != nil {
		m.Invitees = keepReplies(m.Invitees, *d.InviteeIDs)
	}
	if len(d.AddInviteeIDs) > 0 {
		m.Invitees = keepReplies(m.Invitees, append(m.InviteeIDs(), d.AddInviteeIDs...))
	}
	if len(d.RemoveInviteeIDs) > 0 {
		m.Invitees = slices.DeleteFunc(m.Invitees, func(i Invitee) bool { return slices.Contains(d.RemoveInviteeIDs, i.UserID) })
	}
}

func setIf[T any](dst *T, v *T) {
	if v != nil {
		*dst = *v
	}
}

// keepReplies is ids as invitees, in order and once each, keeping the reply of anyone already invited.
func keepReplies(current []Invitee, ids []string) []Invitee {
	out := make([]Invitee, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || slices.ContainsFunc(out, func(i Invitee) bool { return i.UserID == id }) {
			continue
		}
		at := slices.IndexFunc(current, func(i Invitee) bool { return i.UserID == id })
		if at >= 0 {
			out = append(out, current[at])
			continue
		}
		out = append(out, Invitee{UserID: id, Reply: ReplyPending})
	}
	return out
}

// normalize trims m's text and drops its organiser from its invitees.
func (m *Meeting) normalize() {
	m.Title = strings.TrimSpace(m.Title)
	m.Notes = strings.TrimSpace(m.Notes)
	m.Timezone = strings.TrimSpace(m.Timezone)
	m.VoiceChannelID = strings.TrimSpace(m.VoiceChannelID)
	m.Link = strings.TrimSpace(m.Link)
	m.StartsAt = m.StartsAt.UTC().Truncate(time.Second)
	m.EndsAt = m.EndsAt.UTC().Truncate(time.Second)
	if m.Timezone == "" {
		m.Timezone = "UTC"
	}
	if m.Repeats == "" {
		m.Repeats = RepeatNone
	}
	m.Invitees = slices.DeleteFunc(m.Invitees, func(i Invitee) bool { return i.UserID == m.OrganiserID })
}

// validate checks a meeting's own fields; whether its invitees and voice channel may be named is the use-case's.
func (m *Meeting) validate() error {
	if m.Title == "" {
		return fmt.Errorf("%w: a meeting needs a title", apperrs.ErrInvalid)
	}
	if len([]rune(m.Title)) > maxTitle {
		return fmt.Errorf("%w: a meeting's title is at most %d characters", apperrs.ErrInvalid, maxTitle)
	}
	if len([]rune(m.Notes)) > maxNotes {
		return fmt.Errorf("%w: a meeting's notes are at most %d characters", apperrs.ErrInvalid, maxNotes)
	}
	if len(m.Invitees) > maxInvitees {
		return fmt.Errorf("%w: a meeting invites at most %d people", apperrs.ErrInvalid, maxInvitees)
	}
	if err := m.validateTimes(); err != nil {
		return err
	}
	if m.VoiceChannelID != "" && m.Link != "" {
		return fmt.Errorf("%w: a meeting takes place in a voice channel or at a link, not both", apperrs.ErrInvalid)
	}
	return validLink(m.Link)
}

func (m *Meeting) validateTimes() error {
	if m.StartsAt.IsZero() || !m.EndsAt.After(m.StartsAt) {
		return fmt.Errorf("%w: a meeting ends after it starts", apperrs.ErrInvalid)
	}
	if m.EndsAt.Sub(m.StartsAt) > maxDuration {
		return fmt.Errorf("%w: a meeting lasts at most 7 days", apperrs.ErrInvalid)
	}
	if _, err := time.LoadLocation(m.Timezone); err != nil {
		return fmt.Errorf("%w: timezone %q is not an IANA timezone such as Europe/London", apperrs.ErrInvalid, m.Timezone)
	}
	if !slices.Contains([]Repeat{RepeatNone, RepeatDaily, RepeatWeekly, RepeatMonthly}, m.Repeats) {
		return fmt.Errorf("%w: repeats is none, daily, weekly or monthly", apperrs.ErrInvalid)
	}
	if m.RepeatUntil != nil && m.RepeatUntil.Before(m.StartsAt) {
		return fmt.Errorf("%w: a meeting repeats until a time after it starts", apperrs.ErrInvalid)
	}
	if m.RemindMinutes < 0 || m.RemindMinutes > maxRemind {
		return fmt.Errorf("%w: the reminder is 0 to %d minutes before the meeting", apperrs.ErrInvalid, maxRemind)
	}
	return nil
}

func validLink(link string) error {
	if link == "" {
		return nil
	}
	u, err := url.Parse(link)
	if len(link) > maxLink || err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("%w: a meeting's link is an http or https address such as https://meet.example.com/standup", apperrs.ErrInvalid)
	}
	return nil
}

// ValidReply checks an invitee's answer.
func ValidReply(r Reply) error {
	if r != ReplyAccepted && r != ReplyDeclined {
		return fmt.Errorf("%w: a reply is accepted or declined", apperrs.ErrInvalid)
	}
	return nil
}

// ValidRange checks a calendar range: both ends, in order, at most 62 days apart.
func ValidRange(from, to time.Time) error {
	if from.IsZero() || to.IsZero() || !to.After(from) {
		return fmt.Errorf("%w: a range needs from before to, as RFC 3339 times", apperrs.ErrInvalid)
	}
	if to.Sub(from) > maxRangeLength {
		return fmt.Errorf("%w: a range spans at most 62 days", apperrs.ErrInvalid)
	}
	return nil
}
