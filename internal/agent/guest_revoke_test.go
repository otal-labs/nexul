package agent

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
