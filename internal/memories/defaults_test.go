package memories

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestDefaults_ParseWithUniqueNamesAndSurviveASaveInTheEditor(t *testing.T) {
	t.Parallel()
	keys, names := map[string]bool{}, map[string]bool{}
	for _, d := range Defaults() {
		t.Run(d.Key, func(t *testing.T) {
			f, err := ParseSkillFile(d.Text)
			require.NoError(t, err)
			assert.False(t, keys[d.Key], "keys are unique")
			assert.False(t, names[f.Name], "titles are unique")
			keys[d.Key], names[f.Name] = true, true

			stored, err := richtext.Normalize(f.Body)
			require.NoError(t, err)
			saved, err := richtext.ToMarkdown(stored)
			require.NoError(t, err)
			again, err := richtext.Normalize(saved)
			require.NoError(t, err)
			assert.JSONEq(t, stored, again, "the editor's markdown converts back to the same document")
		})
	}
	assert.Len(t, keys, len(DefaultKeys))
}

func TestParseSkillFile_NamesWhatIsWrong(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want string
	}{
		{"no frontmatter", "Just a body.", "opens with frontmatter"},
		{"frontmatter never closed", "---\nname: A\ndescription: B\n\nBody", "opens with frontmatter"},
		{"no name", "---\ndescription: Use when testing.\n---\nBody", "needs a name"},
		{"no description", "---\nname: Testing\n---\nBody", "needs a description"},
		{"empty frontmatter", "---\n---\nBody", "needs a name"},
		{"unknown key", "---\nname: Testing\ndescription: Use when testing.\nalways: true\n---\nBody", "field always not found"},
		{"flag that is not a boolean", "---\nname: Testing\ndescription: Use when testing.\nfooter: yes please\n---\nBody", "true or false"},
		{"key given twice", "---\nname: Testing\nname: Again\ndescription: Use when testing.\n---\nBody", "already defined"},
		{"colon in an unquoted value", "---\nname: Testing\ndescription: Use when: testing\n---\nBody", "in quotes"},
		{"description too long for a when-to-use line", "---\nname: Testing\ndescription: " + strings.Repeat("x", 201) + "\n---\nBody", "when-to-use line"},
		{"empty body", "---\nname: Testing\ndescription: Use when testing.\n---\n\n  \n", "needs a body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckSkillFile(tt.text)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParseSkillFile_FooterWithoutAlwaysIncludedAndWindowsLineEndings_Parse(t *testing.T) {
	t.Parallel()
	f, err := ParseSkillFile("\ufeff---\r\nname: Wrap up\r\ndescription: \"Use when: done\"\r\nfooter: true\r\n---\r\n\r\nMove the ticket.\r\n")
	require.NoError(t, err)
	assert.Equal(t, SkillFile{Name: "Wrap up", Description: "Use when: done", Footer: true, Body: "Move the ticket."}, f)
}

func TestSeeds_SkipATextThatDoesNotParse(t *testing.T) {
	t.Parallel()
	seeds := Seeds(t.Context(), []string{"not a skill file", Defaults()[0].Text})
	require.Len(t, seeds, 1)
	assert.Equal(t, "Writing a doc", seeds[0].Title)
	assert.True(t, richtext.IsStructured(seeds[0].Body), "a seed is stored as rich text, not legacy markdown")
}
