package richtext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeadingAbove(t *testing.T) {
	body := "Intro mentions nothing.\n\n# Setup\n\nInstall it.\n\n## Run the **SQLite** migrations\n\nThe migrator runs on boot.\n\n```sh\n# not a heading\nmake reindex\n```\n"
	tests := []struct {
		name  string
		body  string
		words []string
		want  string
	}{
		{"a match under a heading", body, []string{"migrator"}, "Run the SQLite migrations"},
		{"a word typed in part", body, []string{"insta"}, "Setup"},
		{"a match in the heading itself", body, []string{"setup"}, "Setup"},
		{"a match before any heading", body, []string{"intro"}, ""},
		{"a comment in a code block is not a heading", body, []string{"reindex"}, "Run the SQLite migrations"},
		{"a word only inside another word", body, []string{"grat"}, ""},
		{"no words", body, nil, ""},
		{"a hashtag is not a heading", "#tag\n\nsearch me", []string{"search"}, ""},
		{"a closed heading drops its closing hashes", "### Notes ###\n\nsearch me", []string{"search"}, "Notes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HeadingAbove(tt.body, tt.words))
		})
	}
}
