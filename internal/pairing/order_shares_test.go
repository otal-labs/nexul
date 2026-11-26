package pairing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// newOrderShareFixture is u1 with their desk, and a client's PC its owner shares with u1 under held.
func newOrderShareFixture(t *testing.T, held permissions.Set) (*placementFixture, *fakeBus) {
	t.Helper()
	bus := &fakeBus{}
	f := newPlacementFixture(t)
	f.svc.bus = bus
	f.add(t, "desk", nil)
	f.add(t, "client-pc", func(c *Computer) { c.UserID = "u-client" })
	f.svc.computerAccess = fakeGate{owner: "u-client", held: map[string]permissions.Set{"u1": held}}
	return f, bus
}

func landsOn(t *testing.T, f *placementFixture) string {
	t.Helper()
	target, err := f.svc.PreviewTarget(t.Context(), "u1", "proj-1", false)
	require.NoError(t, err)
	return target.Computer.ID
}

func TestAddToOrder_ASharedComputerFirstInTheOrderTakesTheRun(t *testing.T) {
	t.Parallel()
	f, bus := newOrderShareFixture(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersAgents))
	assert.Equal(t, "desk", landsOn(t, f), "shared alone, it takes none of their runs")

	require.NoError(t, f.svc.AddToOrder(t.Context(), "u1", "client-pc"))
	assert.Equal(t, "desk", landsOn(t, f), "added last, it waits behind the desk")

	require.NoError(t, f.svc.MoveComputer(t.Context(), "u1", "client-pc", 0))
	assert.Equal(t, "client-pc", landsOn(t, f))
	one := PlacementChangedEvent{UserID: "u1", MembersOnly: true}
	assert.Equal(t, []PlacementChangedEvent{one, one}, bus.placements, "adding and moving each wake the waiting runs")

	require.NoError(t, f.svc.RemoveFromOrder(t.Context(), "u1", "client-pc"))
	assert.Equal(t, "desk", landsOn(t, f))
	require.ErrorIs(t, f.svc.RemoveFromOrder(t.Context(), "u1", "client-pc"), apperrs.ErrNotFound)
}

func TestAddToOrder_NeedsRunAgentsAndSomeoneElsesComputer(t *testing.T) {
	t.Parallel()
	f, _ := newOrderShareFixture(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersCommands))

	require.ErrorIs(t, f.svc.AddToOrder(t.Context(), "u1", "client-pc"), apperrs.ErrNotFound, "commands alone never put it in an order")
	require.ErrorIs(t, f.svc.AddToOrder(t.Context(), "u1", "desk"), apperrs.ErrInvalid, "their own is in the order already")
	assert.Empty(t, f.repo.orderShares["u1"])
}

func TestReorderComputers_NamesTheSharedOnesInTheOrderToo(t *testing.T) {
	t.Parallel()
	f, _ := newOrderShareFixture(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersAgents))
	require.NoError(t, f.svc.AddToOrder(t.Context(), "u1", "client-pc"))

	require.ErrorIs(t, f.svc.ReorderComputers(t.Context(), "u1", []string{"desk"}), apperrs.ErrInvalid)
	require.NoError(t, f.svc.ReorderComputers(t.Context(), "u1", []string{"client-pc", "desk"}))
	assert.Equal(t, "client-pc", landsOn(t, f))
}

func TestHandleComputerGrantChanged_TakingRunAgentsAwayTakesItOutOfTheOrder(t *testing.T) {
	t.Parallel()
	f, bus := newOrderShareFixture(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersAgents))
	require.NoError(t, f.svc.AddToOrder(t.Context(), "u1", "client-pc"))
	require.NoError(t, f.svc.MoveComputer(t.Context(), "u1", "client-pc", 0))
	bus.placements = nil

	f.svc.computerAccess = fakeGate{owner: "u-client", held: map[string]permissions.Set{"u1": permissions.SetOf(permissions.ComputersSee, permissions.ComputersCommands)}}
	assert.Equal(t, "desk", landsOn(t, f), "a share that lost Run agents is passed by at once")
	payload, err := json.Marshal(map[string]any{"computer_id": "client-pc", "user_id": "u1", "permissions": []string{"computers:see", "computers:run_commands"}})
	require.NoError(t, err)
	require.NoError(t, f.svc.HandleComputerGrantChanged(t.Context(), eventbus.Event{Topic: "computer.grant_changed", Payload: payload}))

	assert.Empty(t, f.repo.orderShares["u1"], "and leaves the order for good")
	assert.Equal(t, []PlacementChangedEvent{{UserID: "u1", MembersOnly: true}}, bus.placements)
}
