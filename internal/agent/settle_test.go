package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type settleRecorder struct {
	mu       sync.Mutex
	sessions []string
}

func (r *settleRecorder) settle(_ context.Context, t harness.Target) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions = append(r.sessions, t.SessionID)
	return nil
}

func (r *settleRecorder) settled() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.sessions...)
}

func statusChangedEvent(t *testing.T, ticketID string) eventbus.Event {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"ticket": map[string]any{"id": ticketID}, "from": "st-1", "to": "st-2"})
	require.NoError(t, err)
	return eventbus.Event{Topic: "ticket.status_changed", Payload: payload}
}

// settleService runs one turn per updates channel, in order, all on the ticket thread's session thread-1.
func settleService(tickets *fakeTickets, rec *settleRecorder, turns ...<-chan harness.Update) *Service {
	var mu sync.Mutex
	client := &harnesstest.Client{
		StartTurnFn: func(context.Context, harness.Target, string, harness.TurnPrompts) (harness.StartResult, error) {
			mu.Lock()
			defer mu.Unlock()
			updates := turns[0]
			turns = turns[1:]
			return harness.StartResult{SessionID: "thread-1", Updates: updates}, nil
		},
		SettleFn: rec.settle,
	}
	return NewService(Config{
		Conversations: newFakeConversations(Conversation{ID: "conv-1", IsTicketThread: true, TicketID: "t-1"}),
		Targets:       &fakeTargets{target: testTarget()},
		Harnesses:     harnesstest.Registry(client),
		Tickets:       tickets,
		Live:          &fakeLive{},
	})
}

// The agent moves its own ticket to done mid-turn: T3 refuses to settle a running thread, so it waits for the turn's end.
func TestSettle_TicketDoneMidTurn_SettlesOnceTheTurnEnds(t *testing.T) {
	updates := make(chan harness.Update)
	tickets := &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Title: "Pick a strategy"}}
	rec := &settleRecorder{}
	svc := settleService(tickets, rec, updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}), updates)
	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent plan"})
	ended := make(chan struct{})
	go func() {
		svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})
		close(ended)
	}()
	updates <- harness.Update{Snapshot: &harness.Snapshot{MessageID: "m-1", Text: "working", Streaming: true}}

	tickets.ticket.Done = true
	require.NoError(t, svc.HandleTicketStatusChanged(t.Context(), statusChangedEvent(t, "t-1")))
	assert.Empty(t, rec.settled(), "the thread is busy with the second turn")

	updates <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
	close(updates)
	<-ended
	assert.Equal(t, []string{"thread-1"}, rec.settled())

	require.NoError(t, svc.HandleTicketStatusChanged(t.Context(), statusChangedEvent(t, "t-1")))
	assert.Len(t, rec.settled(), 1, "a settled thread is forgotten, so a later move settles nothing")
}

// A person moves the card to done after the turn: the status change settles the thread the last turn ran on.
func TestSettle_TicketDoneAfterTheTurn_SettlesOnTheStatusChange(t *testing.T) {
	tickets := &fakeTickets{ticket: Ticket{ProjectID: "proj-1", Title: "Pick a strategy"}}
	rec := &settleRecorder{}
	svc := settleService(tickets, rec, updatesChan(harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}))

	svc.RunTurn(t.Context(), TurnRequest{ConversationID: "conv-1", ViaUserID: "u-1", RequestBody: "@Agent go"})
	assert.Empty(t, rec.settled(), "a ticket still open keeps its thread active")

	tickets.ticket.Done = true
	require.NoError(t, svc.HandleTicketStatusChanged(t.Context(), statusChangedEvent(t, "t-1")))
	assert.Equal(t, []string{"thread-1"}, rec.settled())
}

func TestHandleTicketStatusChanged_BadPayload_IsFatal(t *testing.T) {
	svc := settleService(&fakeTickets{}, &settleRecorder{})
	err := svc.HandleTicketStatusChanged(t.Context(), eventbus.Event{Topic: "ticket.status_changed", Payload: []byte(`{"ticket":{}}`)})
	assert.ErrorIs(t, err, apperrs.ErrFatal)
}
