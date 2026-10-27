package calendar

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// maxListedOccurrences bounds the occurrences one listed meeting carries.
const maxListedOccurrences = 50

// meetingResult is a meeting as the model reads it, with the occurrences the call asked about.
type meetingResult struct {
	ID             string       `json:"id"`
	WorkspaceID    string       `json:"workspace_id"`
	OrganiserID    string       `json:"organiser_id"`
	Title          string       `json:"title"`
	Notes          string       `json:"notes,omitempty"`
	StartsAt       time.Time    `json:"starts_at"`
	EndsAt         time.Time    `json:"ends_at"`
	Timezone       string       `json:"timezone"`
	VoiceChannelID string       `json:"voice_channel_id,omitempty"`
	Link           string       `json:"link,omitempty"`
	Repeats        Repeat       `json:"repeats"`
	RepeatUntil    *time.Time   `json:"repeat_until,omitempty"`
	RemindMinutes  int          `json:"remind_minutes"`
	SeriesID       string       `json:"series_id,omitempty"`
	ReplacesAt     *time.Time   `json:"replaces_at,omitempty"`
	Cancelled      bool         `json:"cancelled"`
	Invitees       []Invitee    `json:"invitees"`
	Occurrences    []Occurrence `json:"occurrences"`
}

func toMeetingResult(m *Meeting, occurrences []Occurrence) meetingResult {
	if len(occurrences) > maxListedOccurrences {
		occurrences = occurrences[:maxListedOccurrences]
	}
	return meetingResult{
		ID: m.ID, WorkspaceID: m.WorkspaceID, OrganiserID: m.OrganiserID, Title: m.Title, Notes: m.Notes,
		StartsAt: m.StartsAt, EndsAt: m.EndsAt, Timezone: m.Timezone, VoiceChannelID: m.VoiceChannelID, Link: m.Link,
		Repeats: m.Repeats, RepeatUntil: m.RepeatUntil, RemindMinutes: m.RemindMinutes, SeriesID: m.SeriesID,
		ReplacesAt: m.ReplacesAt, Cancelled: m.Cancelled(), Invitees: m.Invitees, Occurrences: occurrences,
	}
}

type meetingListIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace, from workspace_list."`
	From        string `json:"from,omitempty" jsonschema:"Keep only meetings that take place after this RFC 3339 time."`
	To          string `json:"to,omitempty" jsonschema:"Keep only meetings that take place before this RFC 3339 time. With from, each meeting lists its occurrences between the two."`
	mcptool.PageArgs
}

type meetingFields struct {
	Title          *string `json:"title,omitempty" jsonschema:"The meeting's title, at most 200 characters."`
	Notes          *string `json:"notes,omitempty" jsonschema:"Notes as markdown, at most 10000 characters."`
	StartsAt       *string `json:"starts_at,omitempty" jsonschema:"When it starts, an RFC 3339 time such as 2026-11-03T10:00:00Z."`
	EndsAt         *string `json:"ends_at,omitempty" jsonschema:"When it ends, an RFC 3339 time after starts_at and at most 7 days later."`
	Timezone       *string `json:"timezone,omitempty" jsonschema:"The IANA timezone the meeting repeats in, so a weekly 10:00 stays 10:00 across clock changes, such as Europe/London. Defaults to UTC."`
	VoiceChannelID *string `json:"voice_channel_id,omitempty" jsonschema:"A voice channel the meeting takes place in, from conversation_list; a private one must hold every invitee. An empty string clears it."`
	Link           *string `json:"link,omitempty" jsonschema:"An http or https address the meeting takes place at instead of a voice channel. An empty string clears it."`
	Repeats        *string `json:"repeats,omitempty" jsonschema:"none, daily, weekly or monthly; a monthly meeting skips a month without its day. Defaults to none."`
	RepeatUntil    *string `json:"repeat_until,omitempty" jsonschema:"The RFC 3339 time a repeating meeting stops repeating at; an empty string has it repeat for good."`
	RemindMinutes  *int    `json:"remind_minutes,omitempty" jsonschema:"How many minutes before each occurrence the reminder goes, 0 to 10080. Defaults to 10."`
}

type meetingCreateIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace, from workspace_list."`
	meetingFields
	InviteeIDs []string `json:"invitee_ids,omitempty" jsonschema:"The account ids of the workspace members to invite, from account_list or mention_search; the organiser, you, is always in it."`
}

type meetingUpdateIn struct {
	ID string `json:"id" jsonschema:"The meeting id, from meeting_list."`
	meetingFields
	AddInviteeIDs    []string `json:"add_invitee_ids,omitempty" jsonschema:"Account ids of workspace members to invite."`
	RemoveInviteeIDs []string `json:"remove_invitee_ids,omitempty" jsonschema:"Account ids of invitees to uninvite."`
	Cancelled        *bool    `json:"cancelled,omitempty" jsonschema:"true cancels the meeting and tells its invitees; false restores a cancelled one."`
	Slot             string   `json:"slot,omitempty" jsonschema:"For a repeating meeting, the slot of the occurrence to change, as meeting_list's occurrences give it. Omit to change every occurrence."`
	Scope            string   `json:"scope,omitempty" jsonschema:"With slot: this changes that occurrence alone, following changes it and every later one."`
	Reply            string   `json:"reply,omitempty" jsonschema:"Your own answer as an invitee, accepted or declined; it takes no other field in the same call."`
}

// MCPTools returns the calendar tools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{meetingListTool(s), meetingCreateTool(s), meetingUpdateTool(s)}
}

func meetingListTool(s *Service) mcptool.Tool {
	return mcptool.New("meeting_list", "List meetings",
		"Lists the meetings on your workspace calendar that you organise or are invited to, by start time, each with its invitees and their replies. "+
			"A repeating meeting is one item; with both from and to its occurrences lists every time it takes place between them, otherwise only the next one, and each occurrence's slot is what meeting_update takes to change it alone. "+
			"A cancelled meeting stays listed with cancelled true. Tickets that are due show through ticket_list's due filters, not here.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in meetingListIn) (any, error) {
			f, err := rangeFilter(in.From, in.To)
			if err != nil {
				return nil, err
			}
			ms, total, err := s.Page(ctx, in.WorkspaceID, f, in.Window())
			if err != nil {
				return nil, err
			}
			from, to := f.From, f.To
			if from.IsZero() || to.IsZero() {
				from, to = s.now(), time.Time{}
			}
			byMeeting, err := s.Expand(ctx, ms, from, to)
			if err != nil {
				return nil, err
			}
			items := make([]meetingResult, len(ms))
			for i, m := range ms {
				items[i] = toMeetingResult(m, byMeeting[m.ID])
			}
			return mcptool.PageOf(items, total, in.Window()), nil
		})
}

func meetingCreateTool(s *Service) mcptool.Tool {
	return mcptool.New("meeting_create", "Create meeting",
		"Puts a meeting on a workspace's calendar with you as its organiser, invites the members you name, and tells each of them. "+
			"It needs a title, starts_at and ends_at; it takes place in a voice channel or at a link, and may repeat daily, weekly or monthly. "+
			"Everyone who has not declined gets a reminder before each occurrence. "+
			"Returns the meeting with its next occurrence; change or cancel it later with meeting_update.",
		mcptool.Hints{Additive: true, Local: true},
		func(ctx context.Context, in meetingCreateIn) (any, error) {
			d, err := in.details()
			if err != nil {
				return nil, err
			}
			ids := in.InviteeIDs
			d.InviteeIDs = &ids
			m, err := s.Create(ctx, in.WorkspaceID, d)
			if err != nil {
				return nil, err
			}
			return s.result(ctx, m)
		})
}

func meetingUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("meeting_update", "Update meeting",
		"Changes a meeting, cancels or restores it, or records your reply to it. "+
			"Changing or cancelling needs you to be the organiser, or to hold calendar:delete; replying needs you to be invited. "+
			"On a repeating meeting, slot and scope this change one occurrence alone and scope following changes it and every later one, which ends the series there and returns the new series that carries on; without a slot the change covers every occurrence. "+
			"Cancelling tells the invitees, and a cancelled meeting stays on the calendar until cancelled false restores it. "+
			"Returns the meeting the change landed on.",
		mcptool.Hints{Local: true},
		func(ctx context.Context, in meetingUpdateIn) (any, error) {
			if in.Reply != "" {
				return s.replyOnly(ctx, in)
			}
			d, err := in.details()
			if err != nil {
				return nil, err
			}
			d.AddInviteeIDs, d.RemoveInviteeIDs = in.AddInviteeIDs, in.RemoveInviteeIDs
			c := Change{Details: d, Cancelled: in.Cancelled, Scope: Scope(in.Scope)}
			if in.Slot != "" {
				slot, err := parseTime(in.Slot, "slot")
				if err != nil {
					return nil, err
				}
				c.Slot = &slot
			}
			m, err := s.Update(ctx, in.ID, c)
			if err != nil {
				return nil, err
			}
			return s.result(ctx, m)
		})
}

func (s *Service) replyOnly(ctx context.Context, in meetingUpdateIn) (any, error) {
	if in.meetingFields != (meetingFields{}) || in.Cancelled != nil || in.Slot != "" || len(in.AddInviteeIDs)+len(in.RemoveInviteeIDs) > 0 {
		return nil, fmt.Errorf("%w: reply takes no other field; send the change in its own call", apperrs.ErrInvalid)
	}
	m, err := s.Reply(ctx, in.ID, Reply(in.Reply))
	if err != nil {
		return nil, err
	}
	return s.result(ctx, m)
}

// result is m with its next occurrence.
func (s *Service) result(ctx context.Context, m *Meeting) (meetingResult, error) {
	byMeeting, err := s.Expand(ctx, []*Meeting{m}, s.now(), time.Time{})
	if err != nil {
		return meetingResult{}, err
	}
	return toMeetingResult(m, byMeeting[m.ID]), nil
}

func (f meetingFields) details() (Details, error) {
	d := Details{Title: f.Title, Notes: f.Notes, Timezone: f.Timezone, VoiceChannelID: f.VoiceChannelID, Link: f.Link, RemindMinutes: f.RemindMinutes}
	for _, t := range []struct {
		v    *string
		dst  **time.Time
		name string
	}{{f.StartsAt, &d.StartsAt, "starts_at"}, {f.EndsAt, &d.EndsAt, "ends_at"}} {
		if t.v == nil {
			continue
		}
		at, err := parseTime(*t.v, t.name)
		if err != nil {
			return Details{}, err
		}
		*t.dst = &at
	}
	if f.Repeats != nil {
		r := Repeat(*f.Repeats)
		d.Repeats = &r
	}
	until, err := ParseRepeatUntil(f.RepeatUntil)
	if err != nil {
		return Details{}, err
	}
	d.RepeatUntil = until
	return d, nil
}

func rangeFilter(from, to string) (RangeFilter, error) {
	var f RangeFilter
	var err error
	if from != "" {
		if f.From, err = parseTime(from, "from"); err != nil {
			return f, err
		}
	}
	if to != "" {
		if f.To, err = parseTime(to, "to"); err != nil {
			return f, err
		}
	}
	return f, nil
}
