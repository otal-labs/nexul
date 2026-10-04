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
		{"session token", "auth ses_" + token[4:], "auth " + Placeholder},
		{"query token of any shape", `ws connection error {"url":"wss://x/ws/events?token=nxr_abc"}`, `ws connection error {"url":"wss://x/ws/events?token=` + Placeholder + `"}`},
		{"query token beside other params", "/logs?tail=100&token=abc&follow=1", "/logs?tail=100&token=" + Placeholder + "&follow=1"},
		{"a name ending in token is not the param", "/x?csrftoken=abc", "/x?csrftoken=abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Tokens(tt.in))
		})
	}
}
