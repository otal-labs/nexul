package richtext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForOutside(t *testing.T) {
	people := func(userID, login string) (Outsider, bool) {
		switch {
		case userID == "u-alice" || login == "alice":
			return Outsider{Login: "alice-gh", Name: "Alice"}, true
		case userID == "u-sam" || login == "sam":
			return Outsider{Name: "Sam Lee"}, true
		}
		return Outsider{}, false
	}
	tests := []struct {
		name, in, want string
	}{
		{"an escaped backtick opens no code span", "press the \\` key, then ask [@sam@example.com](/people/u-sam) about [T-1](/tickets/t-1)",
			"press the \\` key, then ask [Sam Lee](https://nexul.example.com/people/u-sam) about [T-1](https://nexul.example.com/tickets/t-1)"},
		{"a double-backtick span holds a single backtick", "``a ` @sam`` and @sam", "``a ` @sam`` and **Sam Lee**"},
		{"an unclosed backtick is text", "a ` b @sam", "a ` b **Sam Lee**"},
		{"a person with a login there is mentioned by it", "Ask [@alice](/people/u-alice) first", "Ask @alice-gh first"},
		{"a person without one shows by name, linked to their page", "ask [@sam@example.com](/people/u-sam)", "ask [Sam Lee](https://nexul.example.com/people/u-sam)"},
		{"a person link nobody resolves names nobody", "ask [@gone](/people/u-gone)", "ask [a Nexul member](https://nexul.example.com/people/u-gone)"},
		{"a bare mention maps the same way", "@alice can you look", "@alice-gh can you look"},
		{"a bare mention of a member without a login there shows their name", "thanks @sam", "thanks **Sam Lee**"},
		{"a handle nobody in Nexul holds stays, as written on GitHub", "cc @octocat and @acme/core", "cc @octocat and @acme/core"},
		{"an address is no mention", "mail bob@example.com", "mail bob@example.com"},
		{"a ticket and a doc link point at the instance", "See [NEXUL-1](/tickets/t-1) and [Plan](/docs/d-1)",
			"See [NEXUL-1](https://nexul.example.com/tickets/t-1) and [Plan](https://nexul.example.com/docs/d-1)"},
		{"an attachment image points at the instance", "![shot.png](/api/attachments/a-1)", "![shot.png](https://nexul.example.com/api/attachments/a-1)"},
		{"an absolute link stays", "[docs](https://example.com/x)", "[docs](https://example.com/x)"},
		{"fenced code is left alone", "```\n@sam [x](/tickets/t-1)\n```\n@sam", "```\n@sam [x](/tickets/t-1)\n```\n**Sam Lee**"},
		{"a code span is left alone", "run `ssh @sam now` then ask @sam", "run `ssh @sam now` then ask **Sam Lee**"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ForOutside(tt.in, "https://nexul.example.com/", people, true))
		})
	}
}

func TestForOutside_ABodyLeavesBareHandles(t *testing.T) {
	people := func(string, string) (Outsider, bool) { return Outsider{Login: "sam-gh"}, true }
	assert.Equal(t, "@sam wrote this on GitHub, [x](https://nexul.example.com/tickets/t-1)",
		ForOutside("@sam wrote this on GitHub, [x](/tickets/t-1)", "https://nexul.example.com", people, false),
		"a body's mentions are links, so a plain @handle in it is someone's on GitHub")
}
