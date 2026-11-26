package pairing

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestNormalizeLabels(t *testing.T) {
	t.Parallel()
	tooMany := make([]string, MaxLabels+1)
	for i := range tooMany {
		tooMany[i] = "l" + strings.Repeat("x", i)
	}
	tests := []struct {
		name string
		in   []string
		want []string
		err  string
	}{
		{name: "lowercased, trimmed, blanks and repeats dropped", in: []string{" GPU ", "gpu", "", "windows-vm"}, want: []string{"gpu", "windows-vm"}},
		{name: "none clears them", in: nil, want: []string{}},
		{name: "a space inside", in: []string{"staging keys"}, err: `label "staging keys"`},
		{name: "an underscore", in: []string{"staging_keys"}, err: "lowercase letters, digits and dashes"},
		{name: "longer than 32", in: []string{strings.Repeat("a", 33)}, err: "up to 32 characters"},
		{name: "more than 20", in: tooMany, err: "up to 20 labels"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeLabels(tt.in)
			if tt.err != "" {
				require.ErrorIs(t, err, apperrs.ErrInvalid)
				assert.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
