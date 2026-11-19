package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func backgroundFrames(live *fakeLive) []bool {
	var out []bool
	for _, f := range live.snapshot() {
		if f.Background != nil {
			out = append(out, *f.Background)
		}
	}
	return out
}

func TestOnSessionUpdate_BackgroundWork_ShowsOnTheTicketUntilItEndsOrTheComputerDrops(t *testing.T) {
	t.Parallel()
	conv := newFakeConversations(Conversation{ID: "conv-1", ThreadID: "th-1", IsTicketThread: true, TicketID: "t-1"})
	conv.seen["conv-1"] = "run-7"
	live := &fakeLive{}
	started := time.Date(2026, 11, 19, 9, 30, 0, 0, time.UTC)
	now := started
	svc := NewService(Config{Conversations: conv, Live: live, Now: func() time.Time { return now }})
	monitoring := harness.SessionUpdate{SessionID: "th-1", Latest: "run-7", Background: true}

	svc.OnSessionUpdate("u-1", "c-2", monitoring)
	now = started.Add(time.Minute)
	svc.OnSessionUpdate("u-1", "c-2", monitoring)
	assert.Equal(t, map[string]time.Time{"t-1": started}, svc.BackgroundTickets([]string{"t-1", "t-9"}), "a repeat report keeps when the work started")
	assert.Empty(t, svc.BackgroundTickets([]string{"t-9"}), "only the tickets asked about")

	dropped, err := json.Marshal(presenceChanged{ComputerID: "c-2", State: "connecting"})
	require.NoError(t, err)
	require.NoError(t, svc.HandlePresenceChanged(t.Context(), eventbus.Event{Topic: "computer.presence_changed", Payload: dropped}))
	assert.Empty(t, svc.BackgroundTickets([]string{"t-1"}), "a computer no longer heard says nothing runs")

	svc.OnSessionUpdate("u-1", "c-2", monitoring)
	svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Latest: "run-7"})
	svc.OnSessionUpdate("u-1", "c-2", monitoring)
	svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Gone: true})

	assert.Equal(t, []bool{true, false, true, false, true, false}, backgroundFrames(live), "one frame per change, none for a repeat")
	assert.Empty(t, svc.BackgroundTickets([]string{"t-1"}))
}

func TestOnSessionUpdate_BackgroundWorkOnAThreadNotOnATicket_IsNotShown(t *testing.T) {
	t.Parallel()
	live := &fakeLive{}
	svc := NewService(Config{Conversations: newFakeConversations(Conversation{ID: "conv-1", ThreadID: "th-1"}), Live: live})

	svc.OnSessionUpdate("u-1", "c-2", harness.SessionUpdate{SessionID: "th-1", Background: true})

	assert.Empty(t, backgroundFrames(live))
}

func TestHandlePresenceChanged_BadPayload_IsFatal(t *testing.T) {
	t.Parallel()
	svc := NewService(Config{Live: &fakeLive{}})
	err := svc.HandlePresenceChanged(t.Context(), eventbus.Event{Topic: "computer.presence_changed", Payload: []byte("{")})
	require.Error(t, err)
}
