package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDockerfileExpose(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []int
	}{
		{"single expose", "FROM go\nEXPOSE 8080\n", []int{8080}},
		{"multiple ports on one line", "FROM go\nEXPOSE 80 443\n", []int{80, 443}},
		{"protocol suffix stripped", "FROM go\nEXPOSE 8080/tcp\n", []int{8080}},
		{"multiple EXPOSE lines", "FROM go\nEXPOSE 8080\nRUN echo hi\nEXPOSE 9090/udp\n", []int{8080, 9090}},
		{"lowercase instruction", "from go\nexpose 3000\n", []int{3000}},
		{"no expose", "FROM go\nRUN echo hi\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDockerfileExpose([]byte(tt.body))
			assert.Equal(t, tt.want, got)
		})
	}
}
