package calendar

import (
	"context"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/wake"
)

func TestSendReminders_TellsEveryAttendeeWhenItFallsDue(t *testing.T) {
	h := newHarness()
	d := standup()
	d.InviteeIDs = ptr([]string{"bob", "dave"})
	d.Repeats = ptr(RepeatDaily)
	m := mustCreate(t, h, d)
	_, err := h.svc.Reply(as(t, "dave"), m.ID, ReplyDeclined)
	require.NoError(t, err)

	h.now = at("2026-11-02T09:49:00Z")
	wait, err := h.svc.SendReminders(t.Context())
	require.NoError(t, err)
	assert.Equal(t, time.Minute, wait, "nothing due yet; the next reminder is a minute away")
	assert.NotContains(t, h.repo.topics(), TopicReminderDue)

	h.now = at("2026-11-02T09:50:00Z")
	wait, err = h.svc.SendReminders(t.Context())
	require.NoError(t, err)

	ev := h.repo.lastEvent(t)
	require.Equal(t, TopicReminderDue, ev.Topic)
	payload := ev.Payload.(ReminderEvent)
	assert.Equal(t, []string{"alice", "bob"}, payload.UserIDs, "dave declined")
	assert.Equal(t, at("2026-11-02T10:00:00Z"), payload.StartsAt)
	assert.Equal(t, 14*time.Hour+10*time.Minute, wait, "midnight, when more due dates come into reach, falls before the next occurrence")

	stored, err := h.repo.Get(t.Context(), m.ID)
	require.NoError(t, err)
	assert.Equal(t, at("2026-11-03T09:50:00Z"), *stored.NextRemindAt)

	_, err = h.svc.SendReminders(t.Context())
	require.NoError(t, err)
	assert.Len(t, filterTopics(h.repo.topics(), TopicReminderDue), 1, "a reminder goes once")
}

func filterTopics(topics []string, want string) []string {
	var out []string
	for _, tp := range topics {
		if tp == want {
			out = append(out, tp)
		}
	}
	return out
}

func TestSendReminders_LongStartedSendsNothing(t *testing.T) {
	h := newHarness()
	mustCreate(t, h, standup())

	h.now = at("2026-11-02T10:06:00Z")
	_, err := h.svc.SendReminders(t.Context())
	require.NoError(t, err)

	assert.NotContains(t, h.repo.topics(), TopicReminderDue)
	next, ok, err := h.repo.NextReminder(t.Context())
	require.NoError(t, err)
	assert.False(t, ok, "the meeting has nothing left to remind of, got %v", next)
}

func TestSendReminders_ADueTicket(t *testing.T) {
	h := newHarness()
	h.repo.due = []DueTicket{
		{TicketID: "t-1", ProjectID: "p-1", WorkspaceID: "ws-1", Key: "REF-1", Title: "Ship it", DueOn: "2026-11-03", DeveloperID: "bob", Timezone: "UTC"},
		{TicketID: "t-old", ProjectID: "p-1", WorkspaceID: "ws-1", Key: "REF-2", Title: "Late", DueOn: "2026-11-01", DeveloperID: "bob", Timezone: "UTC"},
	}

	h.now = at("2026-11-02T08:00:00Z")
	wait, err := h.svc.SendReminders(t.Context())
	require.NoError(t, err)
	assert.Equal(t, time.Hour, wait, "09:00 the day before")
	assert.Empty(t, filterTopics(h.repo.topics(), TopicDueDateReminder))
	assert.Equal(t, "", h.repo.reminded["t-1"])

	h.now = at("2026-11-02T09:00:00Z")
	_, err = h.svc.SendReminders(t.Context())
	require.NoError(t, err)

	ev := h.repo.lastEvent(t)
	require.Equal(t, TopicDueDateReminder, ev.Topic)
	assert.Equal(t, DueDateReminderEvent{
		TicketID: "t-1", ProjectID: "p-1", WorkspaceID: "ws-1", Key: "REF-1", Title: "Ship it", DueOn: "2026-11-03", UserIDs: []string{"bob"},
	}, ev.Payload)
	assert.Equal(t, "2026-11-03", h.repo.reminded["t-1"])
	assert.Equal(t, "2026-11-01", h.repo.reminded["t-old"], "a date already past is passed over without a reminder")
	assert.Len(t, filterTopics(h.repo.topics(), TopicDueDateReminder), 1)
}

func TestSendReminders_WithNothingPlannedWaitsForTheNextDay(t *testing.T) {
	h := newHarness()
	h.now = at("2026-11-02T20:00:00Z")
	wait, err := h.svc.SendReminders(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 4*time.Hour, wait)
}

func TestRunReminderLoop_SendsWhenTheReminderFallsDueAndStopsWithItsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness()
		h.svc.now = time.Now
		d := standup()
		start := time.Now().Add(time.Hour)
		d.StartsAt, d.EndsAt = &start, ptr(start.Add(30*time.Minute))
		mustCreate(t, h, d)

		var commits wake.Broadcast
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			h.svc.RunReminderLoop(ctx, commits.Next, slog.New(slog.DiscardHandler))
			close(done)
		}()

		synctest.Wait()
		assert.NotContains(t, h.repo.topics(), TopicReminderDue)
		time.Sleep(50 * time.Minute)
		synctest.Wait()
		assert.Contains(t, h.repo.topics(), TopicReminderDue)

		cancel()
		<-done
	})
}
