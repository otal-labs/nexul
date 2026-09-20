package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEnvFileKeys(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"basic key=value", "FOO=bar\nBAZ=qux\n", []string{"FOO", "BAZ"}},
		{"skips comments and blanks", "# comment\nFOO=bar\n\nBAZ=qux\n", []string{"FOO", "BAZ"}},
		{"strips export prefix", "export FOO=bar\n", []string{"FOO"}},
		{"key with no value", "FOO=\n", []string{"FOO"}},
		{"line without equals is skipped", "FOO=bar\nNOTAKEYVALUE\n", []string{"FOO"}},
		{"empty file", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEnvFileKeys([]byte(tt.body))
			assert.Equal(t, tt.want, got)
		})
	}
}
