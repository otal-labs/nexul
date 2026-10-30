package colors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValid(t *testing.T) {
	tests := []struct {
		name string
		c    Color
		want bool
	}{
		{"cyan is valid", Cyan, true},
		{"empty is not valid", Color(""), false},
		{"unknown hue is not valid", Color("magenta"), false},
		{"a hex color is valid", Color("#3B82f6"), true},
		{"a short hex is not valid", Color("#38f"), false},
		{"a hex without its hash is not valid", Color("3b82f6"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Valid(tt.c))
		})
	}
}

func TestAll(t *testing.T) {
	all := All()
	assert.Len(t, all, 5)
	for _, c := range all {
		assert.True(t, Valid(c))
	}
}
