package botwebhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// TestExecute_AllowedMentions reads allowed_mentions as Discord does: absent or parse users mentions every member
// named, a users list only those, by id or login, and anything else nobody; roles and everyone mention no one.
func TestExecute_AllowedMentions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		allowed *AllowedMentions
		want    []string
		asked   int
	}{
		{"absent mentions every member", "@alice @bob", nil, []string{"alice", "bob"}, 1},
		{"parse users", "@alice", &AllowedMentions{Parse: []string{"users", "roles"}}, []string{"alice", "bob"}, 1},
		{"users by id and login", "@alice @bob", &AllowedMentions{Users: []string{"u-alice", "BOB"}}, []string{"alice", "bob"}, 1},
		{"users list leaves the rest out", "@alice @bob", &AllowedMentions{Users: []string{"u-bob"}}, []string{"bob"}, 1},
		{"parse empty mentions nobody", "@alice", &AllowedMentions{Parse: []string{}}, nil, 0},
		{"roles and everyone mention nobody", "@alice @everyone", &AllowedMentions{Parse: []string{"roles", "everyone"}}, nil, 0},
		{"no @ asks nobody", "build passed", nil, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			s := newTestService(repo)
			b := mustCreate(t, s, "CI")
			_, err := s.Execute(t.Context(), b, Payload{Content: tt.content, AllowedMentions: tt.allowed})
			require.NoError(t, err)
			posts := s.poster.(*fakePoster).posts
			require.Len(t, posts, 1)
			assert.Equal(t, tt.want, posts[0].Mentionable)
			assert.Equal(t, tt.asked, s.people.(*fakePeople).asked)
		})
	}
}

// TestExecute_ShowsTheBotUnlessThePostOverrides: a post's username and avatar_url win; without them the bot's name, and
// its own avatar only when it has one; a bot deleted mid-post answers like any refused URL.
func TestExecute_ShowsTheBotUnlessThePostOverrides(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	b := mustCreate(t, s, "CI")

	m, err := s.Execute(t.Context(), b, Payload{Content: "hi", Username: "Builds", AvatarURL: "https://example.com/ci.png"})
	require.NoError(t, err)
	assert.Equal(t, MessageAuthor{ID: b.ID, Username: "Builds", Avatar: "https://example.com/ci.png", Bot: true}, m.Author)

	m, err = s.Execute(t.Context(), b, Payload{Content: "hi"})
	require.NoError(t, err)
	assert.Equal(t, MessageAuthor{ID: b.ID, Username: "CI", Bot: true}, m.Author, "no avatar shows the glyph")

	b.Avatar = "data:image/png;base64,iVBORw0KGgo="
	m, err = s.Execute(t.Context(), b, Payload{Content: "hi"})
	require.NoError(t, err)
	assert.Equal(t, "/api/botwebhooks/"+b.ID+"/avatar?v=1791321060", m.Author.Avatar, "versioned by the bot's last change")
	assert.Equal(t, "POST /api/botwebhooks/"+b.ID, s.poster.(*fakePoster).posts[2].Audit)

	s.poster.(*fakePoster).err = apperrs.ErrNotFound
	_, err = s.Execute(t.Context(), b, Payload{Content: "hi"})
	require.ErrorIs(t, err, ErrUnknownWebhook)
}
