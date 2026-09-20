package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMentions(t *testing.T) {
	t.Run("no mentions", func(t *testing.T) {
		assert.Nil(t, ParseMentions("just a plain message"))
	})
	t.Run("user mention", func(t *testing.T) {
		got := ParseMentions("hey @onik97 can you look at this")
		assert.Equal(t, []Mention{{Kind: MentionUser, Handle: "onik97"}}, got)
	})
	t.Run("agent mention is case-insensitive but stores the canonical handle", func(t *testing.T) {
		got := ParseMentions("@agent please summarize this thread")
		assert.Equal(t, []Mention{{Kind: MentionAgent, Handle: AgentHandle}}, got)
	})
	t.Run("mixed mentions preserve first-seen order and dedupe", func(t *testing.T) {
		got := ParseMentions("@onik97 @Agent thoughts? cc @onik97 again")
		assert.Equal(t, []Mention{
			{Kind: MentionUser, Handle: "onik97"},
			{Kind: MentionAgent, Handle: AgentHandle},
		}, got)
	})
	t.Run("hyphenated login", func(t *testing.T) {
		got := ParseMentions("@onik-97 take a look")
		assert.Equal(t, []Mention{{Kind: MentionUser, Handle: "onik-97"}}, got)
	})
}
