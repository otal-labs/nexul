package labels

import (
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
