package dns

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputerLabel_SlugsTheNameAndAddsEightRandomCharacters(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"spaces and case", "Onik Laptop", "onik-laptop"},
		{"punctuation runs collapse", "  Work -- PC!! ", "work-pc"},
		{"nothing usable", "💻 ☕", "computer"},
		{"too long for a DNS label", strings.Repeat("a", 80), strings.Repeat("a", computerLabelMax)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computerLabel(tt.in)
			assert.Regexp(t, "^"+tt.want+"-[a-z2-7]{8}$", got)
			assert.LessOrEqual(t, len(got), 63)
		})
	}
	assert.NotEqual(t, computerLabel("same"), computerLabel("same"))
}
