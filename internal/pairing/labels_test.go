package pairing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestDerivedLabels(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{}, DerivedLabels(nil), "a computer that never reported has none")
	assert.Equal(t, []string{"docker", "linux"}, DerivedLabels(&Facts{OS: "linux", Docker: &DockerFacts{Available: true}}))
	assert.Equal(t, []string{"macos"}, DerivedLabels(&Facts{OS: "darwin", Docker: &DockerFacts{Available: false}}), "Docker that doesn't answer is no docker")
	assert.Equal(t, []string{"windows"}, DerivedLabels(&Facts{OS: "windows"}))
	assert.Equal(t, []string{}, DerivedLabels(&Facts{OS: "plan9"}))
}

func TestSetLabels_OnYourOwnComputerWakesWaitingRuns(t *testing.T) {
	t.Parallel()
	bus := &fakeBus{}
	f := newPlacementFixture(t)
	f.svc.bus = bus
	f.add(t, "desk", nil)

	got, err := f.svc.SetLabels(t.Context(), "u1", "desk", []string{"GPU", "staging"})
	require.NoError(t, err)
	assert.Equal(t, []string{"gpu", "staging"}, got.Labels)
	assert.Equal(t, []string{"gpu", "staging"}, f.repo.computers["desk"].Labels)
	assert.Equal(t, []PlacementChangedEvent{{UserID: "u1", MembersOnly: true}}, bus.placements)

	_, err = f.svc.SetLabels(t.Context(), "u2", "desk", []string{"mine"})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = f.svc.SetLabels(t.Context(), "u1", "desk", []string{"bad label"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, []string{"gpu", "staging"}, f.repo.computers["desk"].Labels, "a refused change keeps what was there")
}
