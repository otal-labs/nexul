package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/calendar"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

func meetingAt(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func newTestMeeting(id string, start string, opts ...func(*calendar.Meeting)) *calendar.Meeting {
	st := meetingAt(start)
	m := &calendar.Meeting{
		ID: id, WorkspaceID: "workspace-default", OrganiserID: "u-alice", Title: "Standup", Timezone: "UTC",
		StartsAt: st, EndsAt: st.Add(30 * time.Minute), Repeats: calendar.RepeatNone, RemindMinutes: 10,
		Invitees: []calendar.Invitee{{UserID: "u-bob", Reply: calendar.ReplyPending}}, CreatedAt: st, UpdatedAt: st,
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

// TestMeetingsRepo_SaveKeepsRepliesAndReadsInvitees saves a meeting, records a reply, saves it again with another
// invitee: the reply survives, the event lands in the outbox, and every field reads back as written.
func TestMeetingsRepo_SaveKeepsRepliesAndReadsInvitees(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	remind := meetingAt("2026-11-02T09:50:00Z")
	m := newTestMeeting("m-1", "2026-11-02T10:00:00Z", func(m *calendar.Meeting) {
		m.Repeats, m.Link, m.Notes, m.NextRemindAt, m.NextOccurrence = calendar.RepeatWeekly, "https://meet.example.com/x", "Notes", &remind, &m.StartsAt
	})
	require.NoError(t, s.Meetings.Save(ctx, []*calendar.Meeting{m}, eventbus.OutboxEvent{ID: "evt-1", Topic: calendar.TopicMeetingCreated, Payload: calendar.MeetingEvent{}}))
	assert.Equal(t, 1, count(t, s.db, `SELECT COUNT(*) FROM outbox WHERE id = 'evt-1'`))

	repliedAt := meetingAt("2026-11-01T12:00:00Z")
	require.NoError(t, s.Meetings.SaveReply(ctx, "m-1", "u-bob", calendar.ReplyAccepted, repliedAt))
	require.ErrorIs(t, s.Meetings.SaveReply(ctx, "m-1", "u-nobody", calendar.ReplyAccepted, repliedAt), apperrs.ErrNotFound)

	m.Title = "Sync"
	m.Invitees = []calendar.Invitee{{UserID: "u-bob", Reply: calendar.ReplyPending}, {UserID: "u-carol", Reply: calendar.ReplyPending}}
	require.NoError(t, s.Meetings.Save(ctx, []*calendar.Meeting{m}))

	got, err := s.Meetings.Get(ctx, "m-1")
	require.NoError(t, err)
	assert.Equal(t, "Sync", got.Title)
	assert.Equal(t, "https://meet.example.com/x", got.Link)
	assert.Equal(t, calendar.RepeatWeekly, got.Repeats)
	assert.Equal(t, remind, *got.NextRemindAt)
	require.Len(t, got.Invitees, 2)
	assert.Equal(t, calendar.ReplyAccepted, got.Invitees[0].Reply, "a stored reply outlives the save")
	assert.Equal(t, repliedAt, *got.Invitees[0].RepliedAt)

	m.Invitees = m.Invitees[1:]
	require.NoError(t, s.Meetings.Save(ctx, []*calendar.Meeting{m}))
	got, err = s.Meetings.Get(ctx, "m-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"u-carol"}, got.InviteeIDs(), "an uninvited person goes")

	_, err = s.Meetings.Get(ctx, "m-gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestMeetingsRepo_ListsWhatAPersonSeesInARange(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	until := meetingAt("2026-10-20T10:00:00Z")
	require.NoError(t, s.Meetings.Save(ctx, []*calendar.Meeting{
		newTestMeeting("m-in", "2026-11-03T10:00:00Z"),
		newTestMeeting("m-before", "2026-10-20T10:00:00Z"),
		newTestMeeting("m-after", "2026-12-03T10:00:00Z"),
		newTestMeeting("m-series", "2026-09-01T10:00:00Z", func(m *calendar.Meeting) { m.Repeats = calendar.RepeatWeekly }),
		newTestMeeting("m-ended", "2026-09-01T10:00:00Z", func(m *calendar.Meeting) { m.Repeats, m.RepeatUntil = calendar.RepeatWeekly, &until }),
		newTestMeeting("m-other", "2026-11-04T10:00:00Z", func(m *calendar.Meeting) { m.Invitees = nil }),
		newTestMeeting("m-dave", "2026-11-04T11:00:00Z", func(m *calendar.Meeting) { m.OrganiserID = "u-dave"; m.Invitees = nil }),
	}))
	from, to := meetingAt("2026-11-01T00:00:00Z"), meetingAt("2026-11-30T00:00:00Z")

	ids := func(ms []*calendar.Meeting) []string {
		out := make([]string, len(ms))
		for i, m := range ms {
			out[i] = m.ID
		}
		return out
	}
	bob, err := s.Meetings.ListVisible(ctx, "workspace-default", "u-bob", calendar.RangeFilter{From: from, To: to})
	require.NoError(t, err)
	assert.Equal(t, []string{"m-series", "m-in"}, ids(bob), "invited, in the range, or a series still going")
	alice, err := s.Meetings.ListVisible(ctx, "workspace-default", "u-alice", calendar.RangeFilter{From: from, To: to})
	require.NoError(t, err)
	assert.Equal(t, []string{"m-series", "m-in", "m-other"}, ids(alice), "the organiser sees what they invited nobody to")

	page, total, err := s.Meetings.PageVisible(ctx, "workspace-default", "u-bob", calendar.RangeFilter{}, paging.Window{Limit: 2, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Equal(t, []string{"m-series", "m-before"}, ids(page), "by start, then id")
	page, total, err = s.Meetings.PageVisible(ctx, "workspace-default", "u-bob", calendar.RangeFilter{From: from}, paging.Window{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, []string{"m-series", "m-in", "m-after"}, ids(page))
	page, total, err = s.Meetings.PageVisible(ctx, "workspace-default", "u-bob", calendar.RangeFilter{To: from}, paging.Window{Limit: 10, Offset: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Empty(t, page)
}

func TestMeetingsRepo_ReplacementsAndReminders(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	slot := meetingAt("2026-11-09T10:00:00Z")
	early, late := meetingAt("2026-11-02T09:50:00Z"), meetingAt("2026-11-09T09:50:00Z")
	require.NoError(t, s.Meetings.Save(ctx, []*calendar.Meeting{
		newTestMeeting("m-series", "2026-11-02T10:00:00Z", func(m *calendar.Meeting) {
			m.Repeats, m.NextRemindAt, m.NextOccurrence = calendar.RepeatWeekly, &early, &m.StartsAt
		}),
		newTestMeeting("m-one", "2026-11-09T12:00:00Z", func(m *calendar.Meeting) { m.SeriesID, m.ReplacesAt, m.NextRemindAt = "m-series", &slot, &late }),
	}))

	rs, err := s.Meetings.Replacements(ctx, []string{"m-series"})
	require.NoError(t, err)
	require.Len(t, rs, 1)
	assert.Equal(t, slot, *rs[0].ReplacesAt)
	none, err := s.Meetings.Replacements(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, none)

	next, ok, err := s.Meetings.NextReminder(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, early, next)

	due, err := s.Meetings.RemindersDue(ctx, early)
	require.NoError(t, err)
	require.Len(t, due, 1)
	series := due[0]
	series.NextRemindAt = &late
	ev := eventbus.OutboxEvent{ID: "evt-remind", Topic: calendar.TopicReminderDue, Payload: calendar.ReminderEvent{}}
	require.NoError(t, s.Meetings.MarkReminded(ctx, series, early, ev))
	assert.Equal(t, 1, count(t, s.db, `SELECT COUNT(*) FROM outbox WHERE id = 'evt-remind'`))

	again := eventbus.OutboxEvent{ID: "evt-again", Topic: calendar.TopicReminderDue, Payload: calendar.ReminderEvent{}}
	require.NoError(t, s.Meetings.MarkReminded(ctx, series, early, again))
	assert.Zero(t, count(t, s.db, `SELECT COUNT(*) FROM outbox WHERE id = 'evt-again'`), "a plan already moved on sends nothing")

	series.NextRemindAt, series.NextOccurrence = nil, nil
	require.NoError(t, s.Meetings.MarkReminded(ctx, series, late))
	require.NoError(t, s.Meetings.MarkReminded(ctx, due[0], late))
	_, ok, err = s.Meetings.NextReminder(ctx)
	require.NoError(t, err)
	assert.True(t, ok, "the one occurrence still has its own")
}

func TestMeetingsRepo_DueTickets(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, err := s.db.Exec(`
INSERT INTO users (id, login, created_at, updated_at, timezone) VALUES ('u-dev', 'dev', 1, 1, 'Europe/London');
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-due', 'Due', 'DUE', 1, 'workspace-default', 1, 1);
INSERT INTO statuses (id, project_id, name, position, kind, created_at, updated_at) VALUES
    ('s-open', 'p-due', 'Open', 1, 'backlog', 1, 1), ('s-done', 'p-due', 'Done', 2, 'done', 1, 1);
INSERT INTO tickets (id, project_id, title, status, number, developer, due_on, created_at, updated_at) VALUES
    ('t-due', 'p-due', 'Ship it', 's-open', 7, 'dev', '2026-11-03', 1, 1),
    ('t-done', 'p-due', 'Shipped', 's-done', 8, 'dev', '2026-11-03', 1, 1),
    ('t-nobody', 'p-due', 'Unowned', 's-open', 9, '', '2026-11-03', 1, 1),
    ('t-later', 'p-due', 'Later', 's-open', 10, 'dev', '2026-12-03', 1, 1),
    ('t-none', 'p-due', 'Whenever', 's-open', 11, 'dev', '', 1, 1);`)
	require.NoError(t, err)

	due, err := s.Meetings.DueTickets(ctx, "2026-11-01", "2026-11-04")
	require.NoError(t, err)
	assert.Equal(t, []calendar.DueTicket{{
		TicketID: "t-due", ProjectID: "p-due", WorkspaceID: "workspace-default", Key: "DUE-7", Title: "Ship it", DueOn: "2026-11-03",
		DeveloperID: "u-dev", Timezone: "Europe/London",
	}}, due, "not done, has a developer, due in the range")

	ev := eventbus.OutboxEvent{ID: "evt-due", Topic: calendar.TopicDueDateReminder, Payload: calendar.DueDateReminderEvent{}}
	require.NoError(t, s.Meetings.MarkTicketReminded(ctx, "t-due", "2026-11-03", meetingAt("2026-11-02T09:00:00Z"), ev))
	due, err = s.Meetings.DueTickets(ctx, "2026-11-01", "2026-11-04")
	require.NoError(t, err)
	assert.Empty(t, due, "reminded of that date")

	_, err = s.db.Exec(`UPDATE tickets SET due_on = '2026-11-04' WHERE id = 't-due'`)
	require.NoError(t, err)
	due, err = s.Meetings.DueTickets(ctx, "2026-11-01", "2026-11-04")
	require.NoError(t, err)
	assert.Len(t, due, 1, "a new date is a new reminder")
}

func TestUsersRepo_SetTimezone(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.db.Exec(`INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-tz', 'tz', 1, 1)`)
	require.NoError(t, err)

	require.NoError(t, s.Users.SetTimezone(t.Context(), "u-tz", "Asia/Tokyo"))
	got, err := s.Users.GetUserByID(t.Context(), "u-tz")
	require.NoError(t, err)
	assert.Equal(t, "Asia/Tokyo", got.Timezone)
	require.ErrorIs(t, s.Users.SetTimezone(t.Context(), "u-gone", "Asia/Tokyo"), apperrs.ErrNotFound)
}
