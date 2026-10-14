package deploy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// deployToGate holds Deploy to on the machines listed by id and answers every other machine as not found.
type deployToGate struct {
	allowGate
	deployable map[string]bool
}

func (g deployToGate) RequireMachine(_ context.Context, id string, _ permissions.Action) error {
	if g.deployable[id] {
		return nil
	}
	return fmt.Errorf("%w: machine %s", apperrs.ErrNotFound, id)
}

// TestImport_TakesDeployToOnTheMachineBeforeReadingIt: importing from a machine the caller can't deploy to is refused
// before the machine is discovered or a stack is written.
func TestImport_TakesDeployToOnTheMachineBeforeReadingIt(t *testing.T) {
	s, stacks, _ := newImportTestService([]DiscoveredContainer{{Name: "redis", Image: "redis:7", Status: "running"}})
	s.SetGate(deployToGate{deployable: map[string]bool{"m-1": true}})
	s.SetMachineLookup(fakeMachineLookup{names: map[string]string{"m-1": "prod", "m-2": "staging"}})

	_, err := s.Import(context.Background(), "m-2", ImportRequest{ProjectID: "proj-1", Standalone: []string{"redis"}})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = stacks.GetBySlugAndMachine(context.Background(), "redis", "staging")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "nothing was imported")

	_, err = s.Import(context.Background(), "m-1", ImportRequest{ProjectID: "proj-1", Standalone: []string{"redis"}})
	require.NoError(t, err)
}

// TestDeployTargets_ListsOnlyTheMachinesTheCallerMayDeployTo: the picker's choices are the machines Deploy to answers yes
// for, and a machine the caller can't see is left out rather than failing the list.
func TestDeployTargets_ListsOnlyTheMachinesTheCallerMayDeployTo(t *testing.T) {
	s, _, _ := newImportTestService(nil)
	s.SetGate(deployToGate{deployable: map[string]bool{"m-1": true}})
	s.SetMachineLookup(fakeMachineLookup{names: map[string]string{"m-1": "prod", "m-2": "staging"}})

	got, err := s.DeployTargets(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []MachineRef{{ID: "m-1", Name: "prod"}}, got)
}
