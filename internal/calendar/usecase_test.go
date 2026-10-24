package calendar

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

func TestCreate_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		actor   string
		ws      string
		details func(*Details)
		want    error
	}{
		{"no workspace", "alice", " ", nil, apperrs.ErrInvalid},
		{"without calendar:write", "bob", "ws-1", nil, apperrs.ErrForbidden},
		{"outside the workspace", "mallory", "ws-1", nil, apperrs.ErrNotFound},
		{"no title", "alice", "ws-1", func(d *Details) { d.Title = ptr("  ") }, apperrs.ErrInvalid},
		{"a title too long", "alice", "ws-1", func(d *Details) { d.Title = ptr(string(make([]rune, 201))) }, apperrs.ErrInvalid},
		{"ending before it starts", "alice", "ws-1", func(d *Details) { d.EndsAt = ptr(at("2026-11-02T09:00:00Z")) }, apperrs.ErrInvalid},
		{"lasting over a week", "alice", "ws-1", func(d *Details) { d.EndsAt = ptr(at("2026-11-12T10:00:00Z")) }, apperrs.ErrInvalid},
		{"an unknown timezone", "alice", "ws-1", func(d *Details) { d.Timezone = ptr("Mars/Olympus") }, apperrs.ErrInvalid},
		{"an unknown repeat", "alice", "ws-1", func(d *Details) { d.Repeats = ptr(Repeat("yearly")) }, apperrs.ErrInvalid},
		{"a repeat end before the start", "alice", "ws-1", func(d *Details) {
			d.Repeats, d.RepeatUntil = ptr(RepeatWeekly), ptr(at("2026-10-01T00:00:00Z"))
		}, apperrs.ErrInvalid},
		{"a reminder a month ahead", "alice", "ws-1", func(d *Details) { d.RemindMinutes = ptr(31 * 24 * 60) }, apperrs.ErrInvalid},
		{"a link that is not http", "alice", "ws-1", func(d *Details) { d.Link = ptr("javascript:alert(1)") }, apperrs.ErrInvalid},
		{"both a channel and a link", "alice", "ws-1", func(d *Details) {
			d.VoiceChannelID, d.Link = ptr("vc-open"), ptr("https://meet.example.com/x")
		}, apperrs.ErrInvalid},
		{"an invitee outside the workspace", "alice", "ws-1", func(d *Details) { d.InviteeIDs = ptr([]string{"mallory"}) }, apperrs.ErrInvalid},
		{"a channel nobody can see", "alice", "ws-1", func(d *Details) { d.VoiceChannelID = ptr("vc-missing") }, apperrs.ErrInvalid},
		{"a text channel", "alice", "ws-1", func(d *Details) { d.VoiceChannelID = ptr("ch-text") }, apperrs.ErrInvalid},
		{"another workspace's channel", "alice", "ws-1", func(d *Details) { d.VoiceChannelID = ptr("vc-other") }, apperrs.ErrInvalid},
		{"a private channel missing an invitee", "alice", "ws-1", func(d *Details) {
			d.VoiceChannelID, d.InviteeIDs = ptr("vc-private"), ptr([]string{"bob", "dave"})
		}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness()
			d := standup()
			if tt.details != nil {
				tt.details(&d)
			}
			_, err := h.svc.Create(as(t, tt.actor), tt.ws, d)
			require.ErrorIs(t, err, tt.want)
			assert.Zero(t, h.repo.saveCalls, "nothing is written")
		})
	}
}

func TestCreate_WithoutAPerson_IsRefused(t *testing.T) {
	h := newHarness()
	_, err := h.svc.Create(t.Context(), "ws-1", standup())
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a call with no actor is nobody in the workspace")
}

func TestCreate_InvitesAndPlansTheReminder(t *testing.T) {
	h := newHarness()
	d := standup()
	d.InviteeIDs = ptr([]string{"bob", "alice", "bob", " "})
	d.VoiceChannelID = ptr("vc-private")

	m, err := h.svc.Create(as(t, "alice"), "ws-1", d)
	require.NoError(t, err)

	assert.Equal(t, "alice", m.OrganiserID)
	assert.Equal(t, []Invitee{{UserID: "bob", Reply: ReplyPending}}, m.Invitees, "the organiser and repeats drop out")
	assert.Equal(t, "UTC", m.Timezone)
	assert.Equal(t, DefaultRemind, m.RemindMinutes)
	assert.Equal(t, at("2026-11-02T09:50:00Z"), *m.NextRemindAt)

	ev := h.repo.lastEvent(t)
	assert.Equal(t, TopicMeetingCreated, ev.Topic)
	payload := ev.Payload.(MeetingEvent)
	assert.Equal(t, []string{"alice", "bob"}, payload.UserIDs)
	assert.Equal(t, []string{"bob"}, payload.InvitedIDs)
	assert.Equal(t, "alice", payload.ActorID)
}

func TestCreate_AStorageFailureIsWrapped(t *testing.T) {
	h := newHarness()
	h.repo.saveErr = errors.New("disk full")
	_, err := h.svc.Create(as(t, "alice"), "ws-1", standup())
	require.ErrorContains(t, err, "create meeting: disk full")
}

func mustCreate(t *testing.T, h *harness, d Details) *Meeting {
	t.Helper()
	m, err := h.svc.Create(as(t, "alice"), "ws-1", d)
	require.NoError(t, err)
	return m
}

func TestGet_ShowsTheMeetingOnlyToThePeopleInIt(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, standup())

	for _, who := range []string{"alice", "bob"} {
		got, err := h.svc.Get(as(t, who), m.ID)
		require.NoError(t, err, who)
		assert.Equal(t, m.ID, got.ID)
	}
	_, err := h.svc.Get(as(t, "dave"), m.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a member not invited")
	_, err = h.svc.Get(as(t, "mallory"), m.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "someone outside the workspace")
	_, err = h.svc.Get(as(t, "alice"), "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = h.svc.Get(as(t, "alice"), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestUpdate_WhoMayChangeAMeeting(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		want  error
	}{
		{"its organiser with calendar:write", "alice", nil},
		{"an invitee holding calendar:delete", "carol", nil},
		{"an invitee without it", "bob", apperrs.ErrForbidden},
		{"a member not invited", "dave", apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness()
			d := standup()
			d.InviteeIDs = ptr([]string{"bob", "carol"})
			m := mustCreate(t, h, d)
			_, err := h.svc.Update(as(t, tt.actor), m.ID, Change{Details: Details{Title: ptr("Retro")}})
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestUpdate_AnOmittedFieldKeepsItsValue(t *testing.T) {
	h := newHarness()
	d := standup()
	d.Notes, d.Link = ptr("Bring updates"), ptr("https://meet.example.com/standup")
	m := mustCreate(t, h, d)

	got, err := h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{Title: ptr("Retro")}})
	require.NoError(t, err)

	assert.Equal(t, "Retro", got.Title)
	assert.Equal(t, "Bring updates", got.Notes)
	assert.Equal(t, "https://meet.example.com/standup", got.Link)
	assert.Equal(t, m.StartsAt, got.StartsAt)
	assert.Equal(t, []string{"bob"}, got.InviteeIDs())
	assert.Equal(t, TopicMeetingUpdated, h.repo.lastEvent(t).Topic)
}

func TestUpdate_InviteeChangesKeepReplies(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, standup())
	_, err := h.svc.Reply(as(t, "bob"), m.ID, ReplyAccepted)
	require.NoError(t, err)

	got, err := h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{AddInviteeIDs: []string{"dave", "erin"}, RemoveInviteeIDs: []string{"erin"}}})
	require.NoError(t, err)

	assert.Equal(t, []string{"bob", "dave"}, got.InviteeIDs())
	assert.Equal(t, ReplyAccepted, got.Invitees[0].Reply)
	assert.Equal(t, ReplyPending, got.Invitees[1].Reply)
	payload := h.repo.lastEvent(t).Payload.(MeetingEvent)
	assert.Equal(t, []string{"dave"}, payload.InvitedIDs)

	got, err = h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{InviteeIDs: ptr([]string{"dave"})}})
	require.NoError(t, err)
	assert.Equal(t, []string{"dave"}, got.InviteeIDs())
	assert.ElementsMatch(t, []string{"alice", "bob", "dave"}, h.repo.lastEvent(t).Payload.(MeetingEvent).UserIDs, "bob, just uninvited, still hears of it")
}

func TestUpdate_CancelAndRestore(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, standup())

	got, err := h.svc.Update(as(t, "alice"), m.ID, Change{Cancelled: ptr(true)})
	require.NoError(t, err)
	assert.True(t, got.Cancelled())
	assert.Nil(t, got.NextRemindAt, "a cancelled meeting reminds nobody")
	assert.Equal(t, TopicMeetingCancelled, h.repo.lastEvent(t).Topic)

	got, err = h.svc.Update(as(t, "alice"), m.ID, Change{Cancelled: ptr(true)})
	require.NoError(t, err)
	assert.Equal(t, TopicMeetingUpdated, h.repo.lastEvent(t).Topic, "cancelling twice is no second cancellation")
	assert.True(t, got.Cancelled())

	got, err = h.svc.Update(as(t, "alice"), m.ID, Change{Cancelled: ptr(false)})
	require.NoError(t, err)
	assert.False(t, got.Cancelled())
	assert.NotNil(t, got.NextRemindAt)
	assert.Equal(t, TopicMeetingRestored, h.repo.lastEvent(t).Topic)
}

func weeklyStandup() Details {
	d := standup()
	d.Repeats = ptr(RepeatWeekly)
	return d
}

func TestUpdate_OneOccurrence(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, weeklyStandup())
	slot := at("2026-11-09T10:00:00Z")

	one, err := h.svc.Update(as(t, "alice"), m.ID, Change{
		Details: Details{StartsAt: ptr(at("2026-11-09T14:00:00Z")), EndsAt: ptr(at("2026-11-09T14:30:00Z"))},
		Slot:    &slot, Scope: ScopeThis,
	})
	require.NoError(t, err)

	assert.NotEqual(t, m.ID, one.ID)
	assert.Equal(t, m.ID, one.SeriesID)
	assert.Equal(t, slot, *one.ReplacesAt)
	assert.Equal(t, RepeatNone, one.Repeats)
	assert.Equal(t, []string{"bob"}, one.InviteeIDs())

	_, occurrences, err := h.svc.Calendar(as(t, "bob"), "ws-1", at("2026-11-01T00:00:00Z"), at("2026-11-20T00:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-11-02T10:00:00Z", "2026-11-09T14:00:00Z", "2026-11-16T10:00:00Z"}, starts(occurrences))

	_, err = h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{Title: ptr("x")}, Slot: &slot, Scope: ScopeThis})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "the slot is already replaced")
}

func TestUpdate_OneOccurrenceRefusals(t *testing.T) {
	tests := []struct {
		name   string
		change Change
	}{
		{"a slot the series doesn't have", Change{Slot: ptr(at("2026-11-10T10:00:00Z")), Scope: ScopeThis}},
		{"no scope", Change{Slot: ptr(at("2026-11-09T10:00:00Z"))}},
		{"one occurrence made to repeat", Change{Details: Details{Repeats: ptr(RepeatDaily)}, Slot: ptr(at("2026-11-09T10:00:00Z")), Scope: ScopeThis}},
		{"one occurrence given a repeat end", Change{Details: Details{RepeatUntil: ptr(time.Time{})}, Slot: ptr(at("2026-11-09T10:00:00Z")), Scope: ScopeThis}},
		{"an invalid change to one occurrence", Change{Details: Details{Title: ptr("")}, Slot: ptr(at("2026-11-09T10:00:00Z")), Scope: ScopeThis}},
		{"an invalid change from one on", Change{Details: Details{Title: ptr("")}, Slot: ptr(at("2026-11-09T10:00:00Z")), Scope: ScopeFollowing}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness()
			m := mustCreate(t, h, weeklyStandup())
			_, err := h.svc.Update(as(t, "alice"), m.ID, tt.change)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestUpdate_CancelOneOccurrence(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, weeklyStandup())
	slot := at("2026-11-02T10:00:00Z")

	one, err := h.svc.Update(as(t, "alice"), m.ID, Change{Cancelled: ptr(true), Slot: &slot, Scope: ScopeThis})
	require.NoError(t, err)

	assert.True(t, one.Cancelled())
	assert.Equal(t, TopicMeetingCancelled, h.repo.lastEvent(t).Topic)
	series, err := h.repo.Get(t.Context(), m.ID)
	require.NoError(t, err)
	assert.Equal(t, at("2026-11-09T09:50:00Z"), *series.NextRemindAt, "the series reminds of the next occurrence instead")
}

func TestUpdate_ThisAndTheFollowingOnes(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, weeklyStandup())
	later := at("2026-11-23T10:00:00Z")
	_, err := h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{Notes: ptr("Demo day")}, Slot: &later, Scope: ScopeThis})
	require.NoError(t, err)
	slot := at("2026-11-16T10:00:00Z")

	rest, err := h.svc.Update(as(t, "alice"), m.ID, Change{
		Details: Details{StartsAt: ptr(at("2026-11-16T11:00:00Z")), EndsAt: ptr(at("2026-11-16T11:30:00Z"))},
		Slot:    &slot, Scope: ScopeFollowing,
	})
	require.NoError(t, err)

	assert.NotEqual(t, m.ID, rest.ID)
	assert.Equal(t, RepeatWeekly, rest.Repeats)
	assert.Empty(t, rest.SeriesID)
	series, err := h.repo.Get(t.Context(), m.ID)
	require.NoError(t, err)
	assert.Equal(t, at("2026-11-16T09:59:59Z"), *series.RepeatUntil)
	assert.Equal(t, []string{TopicMeetingCreated, TopicMeetingUpdated, TopicMeetingUpdated, TopicMeetingCreated}, h.repo.topics(),
		"created, the one occurrence changed, the series ended, the new series begun")

	_, occurrences, err := h.svc.Calendar(as(t, "alice"), "ws-1", at("2026-11-01T00:00:00Z"), at("2026-12-01T00:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-11-02T10:00:00Z", "2026-11-09T10:00:00Z", "2026-11-16T11:00:00Z", "2026-11-23T10:00:00Z", "2026-11-30T11:00:00Z"}, starts(occurrences),
		"the occurrence changed on its own keeps its time, and its slot moved with the series so it shows once")
}

func TestUpdate_FollowingFromTheFirstChangesTheWholeSeries(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, weeklyStandup())
	got, err := h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{Title: ptr("Sync")}, Slot: ptr(m.StartsAt), Scope: ScopeFollowing})
	require.NoError(t, err)
	assert.Equal(t, m.ID, got.ID)
	assert.Equal(t, "Sync", got.Title)
}

func TestUpdate_MovingASeriesMovesItsChangedOccurrences(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, weeklyStandup())
	slot := at("2026-11-09T10:00:00Z")
	_, err := h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{Notes: ptr("Short one")}, Slot: &slot, Scope: ScopeThis})
	require.NoError(t, err)

	_, err = h.svc.Update(as(t, "alice"), m.ID, Change{Details: Details{StartsAt: ptr(at("2026-11-02T12:00:00Z")), EndsAt: ptr(at("2026-11-02T12:30:00Z"))}})
	require.NoError(t, err)

	_, occurrences, err := h.svc.Calendar(as(t, "alice"), "ws-1", at("2026-11-01T00:00:00Z"), at("2026-11-17T00:00:00Z"))
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-11-02T12:00:00Z", "2026-11-09T10:00:00Z", "2026-11-16T12:00:00Z"}, starts(occurrences),
		"the changed occurrence keeps its own time and its slot moves with the series")
}

func TestReply(t *testing.T) {
	h := newHarness()
	m := mustCreate(t, h, standup())

	_, err := h.svc.Reply(as(t, "bob"), m.ID, "maybe")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = h.svc.Reply(as(t, "alice"), m.ID, ReplyAccepted)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "the organiser is not an invitee")
	_, err = h.svc.Reply(as(t, "dave"), m.ID, ReplyAccepted)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	got, err := h.svc.Reply(as(t, "bob"), m.ID, ReplyDeclined)
	require.NoError(t, err)
	assert.Equal(t, ReplyDeclined, got.Invitees[0].Reply)
	assert.Equal(t, TopicMeetingReplied, h.repo.lastEvent(t).Topic)
	assert.Equal(t, []string{"alice"}, got.Attendees(), "who declined hears no reminder")
}

func TestCalendar(t *testing.T) {
	h := newHarness()
	_, _, err := h.svc.Calendar(as(t, "bob"), "ws-1", at("2026-11-02T00:00:00Z"), at("2026-11-01T00:00:00Z"))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, _, err = h.svc.Calendar(as(t, "bob"), "ws-1", at("2026-11-01T00:00:00Z"), at("2027-01-15T00:00:00Z"))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "over 62 days")
	_, _, err = h.svc.Calendar(as(t, "mallory"), "ws-1", at("2026-11-01T00:00:00Z"), at("2026-11-08T00:00:00Z"))
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	mustCreate(t, h, weeklyStandup())
	d := standup()
	d.Title, d.StartsAt, d.EndsAt = ptr("Planning"), ptr(at("2026-11-03T09:00:00Z")), ptr(at("2026-11-03T10:00:00Z"))
	mustCreate(t, h, d)
	d.Title, d.StartsAt, d.EndsAt = ptr("Next month"), ptr(at("2026-12-03T09:00:00Z")), ptr(at("2026-12-03T10:00:00Z"))
	mustCreate(t, h, d)

	ms, occurrences, err := h.svc.Calendar(as(t, "bob"), "ws-1", at("2026-11-01T00:00:00Z"), at("2026-11-10T00:00:00Z"))
	require.NoError(t, err)
	assert.Len(t, ms, 2, "the meeting outside the range is left out")
	assert.Equal(t, []string{"2026-11-02T10:00:00Z", "2026-11-03T09:00:00Z", "2026-11-09T10:00:00Z"}, starts(occurrences))

	ms, _, err = h.svc.Calendar(as(t, "dave"), "ws-1", at("2026-11-01T00:00:00Z"), at("2026-11-10T00:00:00Z"))
	require.NoError(t, err)
	assert.Empty(t, ms, "dave is invited to none")
}

func TestPage(t *testing.T) {
	h := newHarness()
	mustCreate(t, h, standup())
	_, _, err := h.svc.Page(as(t, "bob"), "ws-1", RangeFilter{From: at("2026-11-02T00:00:00Z"), To: at("2026-11-01T00:00:00Z")}, paging.Window{Limit: 10})
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	ms, total, err := h.svc.Page(as(t, "bob"), "ws-1", RangeFilter{}, paging.Window{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, ms, 1)
}
