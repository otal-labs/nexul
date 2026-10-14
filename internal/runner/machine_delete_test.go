package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type fakeViewers []string

func (f fakeViewers) MachineViewers(context.Context, string) ([]string, error) { return f, nil }

// viewerGate is ownerGate that also names who sees a machine, as the access service does.
type viewerGate struct {
	ownerGate
	fakeViewers
}

// newDeletable is a service over machine "edge-box" with runner "builder" on it, the shape DeleteMachine reads.
func newDeletable(live *fakeDispatch) (*Service, *fakeRunnerRepo) {
	repo := newFakeRunnerRepo()
	repo.enrolled("r-1", "builder", "edge-box")
	return NewService(repo, live).WithMachines(repo.machines).WithGate(viewerGate{fakeViewers: fakeViewers{"u-alice", "u-bob"}}), repo
}

// A machine that something still depends on must stay, and the refusal must name each dependent.
func TestDeleteMachine_RefusedWhileInUse_NamesWhatUsesIt(t *testing.T) {
	tests := []struct {
		name       string
		dependents MachineDependents
		live       []RunnerStatus
		want       []string
	}{
		{"a stack", MachineDependents{Stacks: []string{"api"}}, nil, []string{"stack api"}},
		{"a gateway", MachineDependents{Gateways: []string{"web"}}, nil, []string{"gateway on network web"}},
		{"a connected runner", MachineDependents{}, []RunnerStatus{{RunnerID: "r-1", Name: "builder"}}, []string{"connected runner builder"}},
		{"all three in order", MachineDependents{Stacks: []string{"api", "web"}, Gateways: []string{"edge"}},
			[]RunnerStatus{{RunnerID: "r-1", Name: "builder"}},
			[]string{"stack api", "stack web", "gateway on network edge", "connected runner builder"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			live := &fakeDispatch{runners: tt.live}
			svc, repo := newDeletable(live)
			repo.machines.dependents = tt.dependents

			err := svc.DeleteMachine(context.Background(), "m-edge-box")

			var inUse *apperrs.InUseError
			require.ErrorAs(t, err, &inUse)
			assert.ErrorIs(t, err, apperrs.ErrConflict)
			assert.Equal(t, tt.want, inUse.Users)
			assert.Equal(t, `machine "edge-box"`, inUse.Thing)
			assert.Len(t, repo.machines.machines, 1, "the machine stays")
			assert.Empty(t, repo.machines.deleted, "no event leaves")
			assert.Empty(t, live.uninstalled, "no runner is told to go")
		})
	}
}

// A free machine goes with an event addressed to the people who saw it, and its runners are told to uninstall.
func TestDeleteMachine_Free_WritesTheEventForItsViewersAndRetiresItsRunners(t *testing.T) {
	live := &fakeDispatch{}
	svc, repo := newDeletable(live)

	require.NoError(t, svc.DeleteMachine(context.Background(), "m-edge-box"))

	assert.Empty(t, repo.machines.machines)
	require.Len(t, repo.machines.deleted, 1)
	assert.Equal(t, TopicMachineDeleted, repo.machines.deleted[0].Topic)
	assert.Equal(t, MachineDeletedEvent{MachineID: "m-edge-box", Name: "edge-box", UserIDs: []string{"u-alice", "u-bob"}}, repo.machines.deleted[0].Payload)
	assert.Equal(t, []string{"r-1"}, live.uninstalled)
}

func TestDeleteMachine_Unknown_IsNotFound(t *testing.T) {
	svc, _ := newDeletable(&fakeDispatch{})
	assert.ErrorIs(t, svc.DeleteMachine(context.Background(), "m-nope"), apperrs.ErrNotFound)
}
