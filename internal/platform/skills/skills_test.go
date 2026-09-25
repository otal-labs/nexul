package skills

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, []string{"nexul-memory"}, Names())

	_, ok = Get("missing")
	assert.False(t, ok)
}
