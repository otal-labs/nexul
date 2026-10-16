package skills

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestBuild_VersionFollowsTheContent(t *testing.T) {
	a := build("x", "---\nname: x\n---\n\nbody\n")
	b := build("x", "---\nname: x\n---\n\nbody\n")
	c := build("x", "---\nname: x\n---\n\nnew body\n")
	assert.Equal(t, a.Version, b.Version, "the same text keeps its version across releases")
	assert.NotEqual(t, a.Version, c.Version, "an edit changes the version")
	assert.Len(t, a.Version, 12)
}

func TestBuild_PutsTheVersionInTheFrontmatter(t *testing.T) {
	s := build("x", "---\nname: x\ndescription: d\n---\n\nbody\n")
	assert.Equal(t, "---\nname: x\ndescription: d\nmetadata:\n  version: \""+s.Version+"\"\n---\n\nbody\n", s.Content)
}

func TestWithVersion_RejectsAFileWithoutFrontmatter(t *testing.T) {
	assert.Panics(t, func() { withVersion("# no frontmatter\n", "v") })
	assert.Panics(t, func() { withVersion("---\nname: x\nnever closed\n", "v") })
}

func TestNexulMemory(t *testing.T) {
	got, ok := Get("nexul-memory")
	require.True(t, ok)
	assert.Equal(t, NexulMemory, got)
	assert.True(t, strings.HasPrefix(got.Content, "---\nname: nexul-memory\n"))
	assert.Contains(t, got.Content, "  version: \""+got.Version+"\"\n---\n")
	assert.Equal(t, []string{"~/.claude/skills/nexul-memory/SKILL.md", "~/.agents/skills/nexul-memory/SKILL.md"}, got.Paths())
	assert.Equal(t, []string{"nexul-memory", "nexul-triage"}, Names())

	_, ok = Get("missing")
	assert.False(t, ok)
}

func TestSetVersion_ChangesWithAnySkill(t *testing.T) {
	assert.Len(t, SetVersion, 12)
	assert.NotEqual(t, NexulMemory.Version, SetVersion)
	assert.NotEqual(t, NexulTriage.Version, SetVersion)
}

// The skill's "Fields" list and the Triage report form name the same ids, so renaming one breaks the build rather than filing a half-filled issue.
func TestNexulTriage_FillsExactlyTheFieldsOfTheTriageReportForm(t *testing.T) {
	raw, err := os.ReadFile("../../../.github/ISSUE_TEMPLATE/triage_report.yml")
	require.NoError(t, err)
	var form struct {
		Labels []string `yaml:"labels"`
		Body   []struct {
			ID string `yaml:"id"`
		} `yaml:"body"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &form))
	assert.Equal(t, []string{"via-triage"}, form.Labels)
	var formIDs []string
	for _, f := range form.Body {
		formIDs = append(formIDs, f.ID)
	}

	_, fields, ok := strings.Cut(NexulTriage.Content, "\n## Fields\n")
	require.True(t, ok, "the skill has a Fields section")
	fields, _, _ = strings.Cut(fields, "\n## ")
	var skillIDs []string
	for _, m := range regexp.MustCompile("(?m)^- `([a-z_]+)` ").FindAllStringSubmatch(fields, -1) {
		skillIDs = append(skillIDs, m[1])
	}
	assert.Equal(t, formIDs, skillIDs)
	assert.Contains(t, NexulTriage.Content, "`via-triage`")
}
