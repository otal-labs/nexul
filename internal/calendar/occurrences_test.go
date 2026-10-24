package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptr[T any](v T) *T { return &v }

func newTestMeeting(opts ...func(*Meeting)) *Meeting {
	m := &Meeting{
		ID: "m-1", WorkspaceID: "ws-1", OrganiserID: "alice", Title: "Standup", Timezone: "UTC", Repeats: RepeatNone,
		StartsAt: at("2026-11-02T10:00:00Z"), EndsAt: at("2026-11-02T10:30:00Z"), RemindMinutes: DefaultRemind,
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

func repeating(r Repeat) func(*Meeting) { return func(m *Meeting) { m.Repeats = r } }

func starts(ocs []Occurrence) []string {
	out := make([]string, len(ocs))
	for i, o := range ocs {
		out[i] = o.StartsAt.Format(time.RFC3339)
	}
	return out
}

func TestOccurrences(t *testing.T) {
	tests := []struct {
		name     string
		meeting  *Meeting
		from, to string
		replaced map[int64]bool
		want     []string
	}{
		{"a single meeting outside the range", newTestMeeting(), "2026-11-03T00:00:00Z", "2026-11-04T00:00:00Z", nil, []string{}},
		{"a single meeting ending as the range starts", newTestMeeting(), "2026-11-02T10:30:00Z", "2026-11-03T00:00:00Z", nil, []string{}},
		{"a single meeting still running when the range starts", newTestMeeting(), "2026-11-02T10:15:00Z", "2026-11-03T00:00:00Z", nil, []string{"2026-11-02T10:00:00Z"}},
		{
			"a daily meeting each day of the range", newTestMeeting(repeating(RepeatDaily)),
			"2026-11-03T00:00:00Z", "2026-11-06T00:00:00Z", nil,
			[]string{"2026-11-03T10:00:00Z", "2026-11-04T10:00:00Z", "2026-11-05T10:00:00Z"},
		},
		{
			"a daily meeting begun two years ago jumps to the range", newTestMeeting(repeating(RepeatDaily), func(m *Meeting) {
				m.StartsAt, m.EndsAt = at("2024-11-02T10:00:00Z"), at("2024-11-02T10:30:00Z")
			}),
			"2026-11-03T00:00:00Z", "2026-11-04T00:00:00Z", nil, []string{"2026-11-03T10:00:00Z"},
		},
		{
			"a weekly London meeting keeps 10:00 across the clocks going back", newTestMeeting(repeating(RepeatWeekly), func(m *Meeting) {
				m.Timezone = "Europe/London"
				m.StartsAt, m.EndsAt = at("2026-10-19T09:00:00Z"), at("2026-10-19T09:30:00Z")
			}),
			"2026-10-19T00:00:00Z", "2026-11-03T00:00:00Z", nil,
			[]string{"2026-10-19T09:00:00Z", "2026-10-26T10:00:00Z", "2026-11-02T10:00:00Z"},
		},
		{
			"a monthly meeting on the 31st skips the months without one", newTestMeeting(repeating(RepeatMonthly), func(m *Meeting) {
				m.StartsAt, m.EndsAt = at("2026-01-31T10:00:00Z"), at("2026-01-31T11:00:00Z")
			}),
			"2026-01-01T00:00:00Z", "2026-06-01T00:00:00Z", nil,
			[]string{"2026-01-31T10:00:00Z", "2026-03-31T10:00:00Z", "2026-05-31T10:00:00Z"},
		},
		{
			"a series stops at its end", newTestMeeting(repeating(RepeatDaily), func(m *Meeting) { m.RepeatUntil = ptr(at("2026-11-03T10:00:00Z")) }),
			"2026-11-01T00:00:00Z", "2026-11-10T00:00:00Z", nil, []string{"2026-11-02T10:00:00Z", "2026-11-03T10:00:00Z"},
		},
		{
			"a replaced slot is left out", newTestMeeting(repeating(RepeatDaily)),
			"2026-11-02T00:00:00Z", "2026-11-05T00:00:00Z", map[int64]bool{at("2026-11-03T10:00:00Z").Unix(): true},
			[]string{"2026-11-02T10:00:00Z", "2026-11-04T10:00:00Z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.meeting.Occurrences(at(tt.from), at(tt.to), tt.replaced)
			assert.Equal(t, tt.want, starts(got))
		})
	}
}

func TestOccurrences_WithNoEnd_GivesTheNextOneOnly(t *testing.T) {
	m := newTestMeeting(repeating(RepeatWeekly))
	got := m.Occurrences(at("2026-11-05T00:00:00Z"), time.Time{}, nil)
	assert.Equal(t, []string{"2026-11-09T10:00:00Z"}, starts(got))
}

func TestHasSlot(t *testing.T) {
	m := newTestMeeting(repeating(RepeatWeekly))
	assert.True(t, m.HasSlot(at("2026-11-09T10:00:00Z"), nil))
	assert.False(t, m.HasSlot(at("2026-11-10T10:00:00Z"), nil), "not a weekday the series falls on")
	assert.False(t, m.HasSlot(at("2026-11-09T10:00:00Z"), map[int64]bool{at("2026-11-09T10:00:00Z").Unix(): true}), "replaced")
}

func TestPlanReminder(t *testing.T) {
	tests := []struct {
		name           string
		meeting        *Meeting
		after          string
		wantRemind     string
		wantOccurrence string
	}{
		{"before a single meeting", newTestMeeting(), "2026-11-01T00:00:00Z", "2026-11-02T09:50:00Z", "2026-11-02T10:00:00Z"},
		{"once a single meeting started", newTestMeeting(), "2026-11-02T10:00:00Z", "", ""},
		{"a cancelled meeting", newTestMeeting(func(m *Meeting) { m.CancelledAt = ptr(at("2026-10-01T00:00:00Z")) }), "2026-11-01T00:00:00Z", "", ""},
		{"the occurrence after the one just reminded of", newTestMeeting(repeating(RepeatDaily)), "2026-11-02T10:00:00Z", "2026-11-03T09:50:00Z", "2026-11-03T10:00:00Z"},
		{"a reminder an hour ahead", newTestMeeting(func(m *Meeting) { m.RemindMinutes = 60 }), "2026-11-01T00:00:00Z", "2026-11-02T09:00:00Z", "2026-11-02T10:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.meeting.planReminder(at(tt.after), nil)
			if tt.wantRemind == "" {
				assert.Nil(t, tt.meeting.NextRemindAt)
				assert.Nil(t, tt.meeting.NextOccurrence)
				return
			}
			require.NotNil(t, tt.meeting.NextRemindAt)
			assert.Equal(t, at(tt.wantRemind), tt.meeting.NextRemindAt.UTC())
			assert.Equal(t, at(tt.wantOccurrence), tt.meeting.NextOccurrence.UTC())
		})
	}
}

func TestDueReminderAt(t *testing.T) {
	tests := []struct {
		name, dueOn, timezone string
		wantRemind, wantEnd   string
		wantOK                bool
	}{
		{"not a date", "next week", "UTC", "", "", false},
		{"in UTC", "2026-11-05", "UTC", "2026-11-04T09:00:00Z", "2026-11-06T00:00:00Z", true},
		{"no timezone reported reads as UTC", "2026-11-05", "", "2026-11-04T09:00:00Z", "2026-11-06T00:00:00Z", true},
		{"an unknown timezone reads as UTC", "2026-11-05", "Mars/Olympus", "2026-11-04T09:00:00Z", "2026-11-06T00:00:00Z", true},
		{"in the developer's own timezone", "2026-11-05", "America/New_York", "2026-11-04T14:00:00Z", "2026-11-06T05:00:00Z", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remind, end, ok := dueReminderAt(tt.dueOn, tt.timezone)
			require.Equal(t, tt.wantOK, ok)
			if !ok {
				return
			}
			assert.Equal(t, at(tt.wantRemind), remind.UTC())
			assert.Equal(t, at(tt.wantEnd), end.UTC())
		})
	}
}
