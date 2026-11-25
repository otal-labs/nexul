package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func grantEvent(t *testing.T, permissions ...string) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"computer_id": "c-laptop", "user_id": "u-bob", "permissions": permissions})
	require.NoError(t, err)
	return eventbus.Event{Topic: "computer.grant_changed", Payload: raw}
}

func TestHandleComputerGrantChanged_InterruptsOnlyTheTurnsTheShareNoLongerAllows(t *testing.T) {
	bobs, alices := &fakeHarness{interruptCh: make(chan struct{})}, &fakeHarness{interruptCh: make(chan struct{})}
	s := &Service{log: slog.New(slog.DiscardHandler), active: map[string][]*activeTurn{
		"conv-bob":   {{client: bobs, computerID: "c-laptop", ownerID: "u-alice", viaUserID: "u-bob"}},
		"conv-alice": {{client: alices, computerID: "c-laptop", ownerID: "u-alice", viaUserID: "u-alice"}},
	}}

	require.NoError(t, s.HandleComputerGrantChanged(t.Context(), grantEvent(t, "computers:see", "computers:run_agents")))
	assert.Len(t, s.guestTurns("c-laptop", "u-bob"), 1, "a share that still gives Run agents stops nothing")

	require.NoError(t, s.HandleComputerGrantChanged(t.Context(), grantEvent(t, "computers:see", "computers:run_commands")))

	select {
	case <-bobs.interruptCh:
	default:
		t.Fatal("bob's turn on alice's computer was not interrupted")
	}
	select {
	case <-alices.interruptCh:
		t.Fatal("alice's own turn was interrupted")
	default:
	}
}

func TestRunTurn_OnSomeoneElsesComputer_RunsOnItsOwnInstanceAndSaysWhose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		conv := newFakeConversations(Conversation{ID: "conv-1"})
		client := &fakeHarness{startResult: harness.StartResult{SessionID: "th-1", Updates: updatesChan(
			harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}},
		)}}
		target := testTarget()
		target.Computer.UserID, target.Computer.Name, target.Provider = "u-alice", "Laptop", "claudeAgent"
		targets := &fakeTargets{target: target, guestInstance: "nexul-guest-1"}
		svc := NewService(Config{Conversations: conv, Targets: targets, Harnesses: registryOf(client), Live: &fakeLive{}})

		require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-bob", "@Agent hi", true)))
		synctest.Wait()

		client.mu.Lock()
		assert.Equal(t, "nexul-guest-1", client.lastTarget.Provider, "the turn starts on the run's own instance")
		client.mu.Unlock()
		assert.EqualValues(t, 1, targets.guestClosed.Load(), "and it goes when the turn ends")
		_, systemPosts := conv.snapshot()
		require.NotEmpty(t, systemPosts)
		assert.Contains(t, systemPosts[0].body, "Run on someone else's Laptop")
	})
}

func TestRunTurn_ARefusedSharedRunStartsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		conv := newFakeConversations(Conversation{ID: "conv-1"})
		client := &fakeHarness{}
		target := testTarget()
		target.Computer.UserID = "u-alice"
		svc := NewService(Config{Conversations: conv, Targets: &fakeTargets{target: target, guestErr: apperrs.ErrConflict}, Harnesses: registryOf(client), Live: &fakeLive{}})

		require.NoError(t, svc.HandleMessageCreated(context.Background(), messageCreatedEvent(t, "conv-1", "u-bob", "@Agent hi", true)))
		synctest.Wait()

		client.mu.Lock()
		assert.Empty(t, client.lastTarget.Provider, "no turn started")
		client.mu.Unlock()
		_, systemPosts := conv.snapshot()
		require.Len(t, systemPosts, 1)
		assert.Contains(t, systemPosts[0].body, "Agent turn failed to start")
	})
}
