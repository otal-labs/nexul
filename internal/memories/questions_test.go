package memories

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestParseInterviewTemplate(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Question
	}{
		{"heading alone is free text", "## What do you use?", []Question{{Text: "What do you use?", Options: []Option{}}}},
		{"text under the heading is the hint", "##   Stack  \r\nName versions.\nOnly pinned ones.\n", []Question{
			{Text: "Stack", Hint: "Name versions.\nOnly pinned ones.", Options: []Option{}},
		}},
		{"dash bullets are single choice", "## Layout?\nPick one.\n- Layers\n- Feature folders", []Question{
			{Text: "Layout?", Hint: "Pick one.", Options: []Option{{Label: "Layers"}, {Label: "Feature folders"}}},
		}},
		{"box bullets are multi-select", "## Tests?\n- [ ] Unit\n- [ ] End-to-end", []Question{
			{Text: "Tests?", MultiSelect: true, Options: []Option{{Label: "Unit"}, {Label: "End-to-end"}}},
		}},
		{"text after the first colon is the description", "## Merge?\n- Squash: one commit: per PR\n- [x] odd", []Question{
			{Text: "Merge?", Options: []Option{{Label: "Squash", Description: "one commit: per PR"}, {Label: "[x] odd"}}},
		}},
		{"a line after the bullets continues the last option", "## Merge?\n- Squash: one commit\n  per pull request", []Question{
			{Text: "Merge?", Options: []Option{{Label: "Squash", Description: "one commit per pull request"}}},
		}},
		{"an empty bullet is skipped", "## Merge?\n-\n- : no label\n", []Question{{Text: "Merge?", Options: []Option{}}}},
		{"blank lines before the first heading are fine", "\n\n## A\n\n## B\n", []Question{
			{Text: "A", Options: []Option{}}, {Text: "B", Options: []Option{}},
		}},
		{"an edited pre-change template parses as free-text questions with a hint",
			"## Stack and versions\nLanguages, frameworks, and the versions this project pins.\n\n## Testing\nUnit or integration, and the coverage floor.\n",
			[]Question{
				{Text: "Stack and versions", Hint: "Languages, frameworks, and the versions this project pins.", Options: []Option{}},
				{Text: "Testing", Hint: "Unit or integration, and the coverage floor.", Options: []Option{}},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInterviewTemplate(tt.body)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseInterviewTemplate_Refusals_NameTheLine(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"text before the first heading", "\n# Interview\n## A", "line 2: text before the first ## question heading"},
		{"an empty heading", "## A\n##   \n", "line 2: an empty ## heading"},
		{"a bare heading mark", "## A\n##", "line 2: an empty ## heading"},
		{"no questions", "\n  \n", "has no questions"},
		{"a deeper heading is not a question", "### A", "line 1: text before the first ## question heading"},
		{"duplicate question text", "## Same\n\n##  Same \n", `line 3: the question "Same" is already asked on line 1`},
		{"single then multi", "## Mixed\n- One\n- [ ] Two", `line 3: "Mixed" mixes - [ ] multi-select options`},
		{"multi then single", "## Mixed\n- [ ] One\n- Two", `line 3: "Mixed" mixes`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseInterviewTemplate(tt.body)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParseInterviewTemplate_Refused_StillReturnsWhatItRead(t *testing.T) {
	got, err := ParseInterviewTemplate("Preamble\n## A\n## B")
	require.Error(t, err)
	assert.Len(t, got, 2)
	assert.Len(t, TemplateQuestions("Preamble\n## A"), 1)
}

func TestDefaultInterviewTemplate_IsTheTwelveQuestions(t *testing.T) {
	got, err := ParseInterviewTemplate(DefaultInterviewTemplate)
	require.NoError(t, err)
	require.Len(t, got, 12)
	assert.Equal(t, "What languages and frameworks does this project use?", got[0].Text)
	assert.Empty(t, got[0].Options)
	assert.Equal(t, "Which tests does a change need?", got[4].Text)
	assert.True(t, got[4].MultiSelect)
	assert.Equal(t, "And the coverage floor, if any.", got[4].Hint)
	assert.Len(t, got[4].Options, 4)
	assert.Equal(t, "Which words mean something specific here?", got[11].Text)
	assert.Empty(t, got[11].Options)
}

func TestCheckInterviewTemplate(t *testing.T) {
	long := "## A\n" + strings.Repeat("a", MaxInterviewTemplateChars)
	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"well past the memory's cap is fine", "## A\n" + strings.Repeat("a", MaxInterviewChars*3), ""},
		{"over the template limit", long, "over the 32000-character limit"},
		{"unparseable", "Intro\n## A", "line 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckInterviewTemplate(tt.body)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
