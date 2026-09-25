// Package skills holds the agent skills Nexul writes to paired computers, each versioned by its own content.
package skills

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"strings"
)

//go:embed nexul-memory.md
var nexulMemorySource string

// Skill is one skill as Nexul ships it; Content is the SKILL.md to write, its version already in the frontmatter.
type Skill struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Content string `json:"content"`
}

// Paths are where setup installs the skill: Claude Code and opencode read the first, Codex and opencode the second.
func (s Skill) Paths() []string {
	return []string{"~/.claude/skills/" + s.Name + "/SKILL.md", "~/.agents/skills/" + s.Name + "/SKILL.md"}
}

// NexulMemory is the memory protocol every paired provider installs.
var NexulMemory = build("nexul-memory", nexulMemorySource)

var all = []Skill{NexulMemory}

// Get returns a shipped skill by name.
func Get(name string) (Skill, bool) {
	for _, s := range all {
		if s.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}

// Names lists every shipped skill.
func Names() []string {
	out := make([]string, 0, len(all))
	for _, s := range all {
		out = append(out, s.Name)
	}
	return out
}

// build derives the version from the source, so a release that edits the skill changes it and one that doesn't keeps it.
func build(name, source string) Skill {
	sum := sha256.Sum256([]byte(source))
	version := hex.EncodeToString(sum[:])[:12]
	return Skill{Name: name, Version: version, Content: withVersion(source, version)}
}

// withVersion adds metadata.version as the last frontmatter field; it panics on a file without frontmatter, a build-time mistake.
func withVersion(source, version string) string {
	if !strings.HasPrefix(source, "---\n") {
		panic("skills: a skill file must open with frontmatter")
	}
	end := strings.Index(source[4:], "\n---\n")
	if end < 0 {
		panic("skills: a skill file's frontmatter must close with ---")
	}
	end += 4
	return source[:end] + fmt.Sprintf("\nmetadata:\n  version: %q", version) + source[end:]
}
