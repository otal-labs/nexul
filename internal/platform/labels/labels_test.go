package labels

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestNormalize(t *testing.T) {
	t.Parallel()
	tooMany := make([]string, Max+1)
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
			got, err := Normalize(tt.in)
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

func TestComputerRule_LeftOut(t *testing.T) {
	t.Parallel()
	has := func(held ...string) func(string) bool {
		return func(l string) bool { return slices.Contains(held, l) }
	}
	rule := ComputerRule{LabelsIn: []string{"gpu", "linux"}, LabelsOut: []string{"flaky"}, IDs: []string{"a", "b"}}
	tests := []struct {
		name string
		rule ComputerRule
		id   string
		held []string
		want bool
	}{
		{"an empty rule leaves nothing out", ComputerRule{}, "z", nil, false},
		{"one of labels_in is enough", rule, "a", []string{"linux"}, false},
		{"none of labels_in leaves it out", rule, "a", []string{"docker"}, true},
		{"a labels_out label leaves it out", rule, "a", []string{"gpu", "flaky"}, true},
		{"an id outside ids leaves it out", rule, "c", []string{"gpu"}, true},
		{"labels_out alone", ComputerRule{LabelsOut: []string{"flaky"}}, "z", []string{"flaky"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.rule.LeftOut(tt.id, has(tt.held...)))
		})
	}
}

func TestComputerRule_Normalize(t *testing.T) {
	t.Parallel()
	got, err := ComputerRule{LabelsIn: []string{" GPU ", "gpu"}, IDs: []string{" a ", "a", ""}}.Normalize()
	require.NoError(t, err)
	assert.Equal(t, ComputerRule{LabelsIn: []string{"gpu"}, LabelsOut: []string{}, IDs: []string{"a"}}, got)

	_, err = ComputerRule{LabelsOut: []string{"bad label"}}.Normalize()
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
