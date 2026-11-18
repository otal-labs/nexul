package richtext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForOutside(t *testing.T) {
	logins := func(userID, login string) string {
		if userID == "u-alice" || login == "alice" {
			return "alice-gh"
		}
		return ""
	}
	tests := []struct {
		name, in, want string
	}{
		{"a person with a login there is mentioned by it", "Ask [@alice](/people/u-alice) first", "Ask @alice-gh first"},
		{"a person without one links to their page", "Ask [@bob](/people/u-bob)", "Ask [@bob](https://nexul.example.com/people/u-bob)"},
		{"a bare mention maps the same way", "@alice can you look", "@alice-gh can you look"},
		{"a bare mention nobody holds pings nobody", "thanks @bob and @Agent", "thanks `@bob` and `@Agent`"},
		{"an address is no mention", "mail bob@example.com", "mail bob@example.com"},
		{"a ticket and a doc link point at the instance", "See [NEXUL-1](/tickets/t-1) and [Plan](/docs/d-1)",
			"See [NEXUL-1](https://nexul.example.com/tickets/t-1) and [Plan](https://nexul.example.com/docs/d-1)"},
		{"an attachment image points at the instance", "![shot.png](/api/attachments/a-1)", "![shot.png](https://nexul.example.com/api/attachments/a-1)"},
		{"an absolute link stays", "[docs](https://example.com/x)", "[docs](https://example.com/x)"},
		{"fenced code is left alone", "```\n@bob [x](/tickets/t-1)\n```\n@bob", "```\n@bob [x](/tickets/t-1)\n```\n`@bob`"},
		{"a code span keeps its mention", "run `@bob` now", "run `@bob` now"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ForOutside(tt.in, "https://nexul.example.com/", logins))
		})
	}
}
