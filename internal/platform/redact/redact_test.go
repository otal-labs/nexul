package redact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokens(t *testing.T) {
	t.Parallel()
	token := "dep_" + "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO-_"
	tests := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"no token", "wrote the MCP entry", "wrote the MCP entry"},
		{"token alone", token, Placeholder},
		{"token in a config line", `headers = { Authorization = "Bearer ` + token + `" }`, `headers = { Authorization = "Bearer ` + Placeholder + `" }`},
		{"two tokens", token + " and " + token, Placeholder + " and " + Placeholder},
		{"prefix with a short body is not a token", "dep_short", "dep_short"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Tokens(tt.in))
		})
	}
}
