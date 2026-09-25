package mcp

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/plays"
)

// The rules below are practices/mcp.md sections 4 to 6, checked so they cannot drift.
const toolBudget = 100

var (
	toolName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,51}$`)
	verbs    = []string{
		"list", "get", "create", "update", "delete",
		"deploy", "cancel", "run", "upgrade", "scan", "discover", "import", "replay", "search", "post", "pair", "report",
	}
	// A snake_case word ending in a known verb reads as a tool name wherever it appears in text agents read.
	toolWord = regexp.MustCompile(`\b[a-z]+(?:_[a-z]+)*_(?:` + strings.Join(verbs, "|") + `)\b`)
)

func surface() []mcptool.Tool {
	return registryTools(RegistryOptions{})
}

func TestSurface_StaysInsideTheBudget(t *testing.T) {
	assert.Less(t, len(surface()), toolBudget)
}

func TestSurface_NamesAreObjectVerbAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, tool := range surface() {
		assert.Regexp(t, toolName, tool.Name)
		assert.False(t, seen[tool.Name], "duplicate tool %s", tool.Name)
		seen[tool.Name] = true
		verb := tool.Name[strings.LastIndex(tool.Name, "_")+1:]
		assert.Contains(t, verbs, verb, "%s does not end in a known verb", tool.Name)
	}
}

func TestSurface_EveryToolIsDescribed(t *testing.T) {
	for _, tool := range surface() {
		t.Run(tool.Name, func(t *testing.T) {
			assert.NotEmpty(t, tool.Title)
			assert.GreaterOrEqual(t, sentences(tool.Description), 3, "a description is a contract of at least three sentences")
			require.NotNil(t, tool.InputSchema)
			assert.Equal(t, "object", tool.InputSchema.Type)
			for _, path := range undescribed(tool.InputSchema, "") {
				t.Errorf("parameter %s has no description", path)
			}
		})
	}
}

func TestSurface_ReadShapedToolsAreReadOnly(t *testing.T) {
	for _, tool := range surface() {
		if strings.HasSuffix(tool.Name, "_list") || strings.HasSuffix(tool.Name, "_get") || strings.HasSuffix(tool.Name, "_search") {
			assert.True(t, tool.Hints.ReadOnly, "%s reads, so it is ReadOnly", tool.Name)
		}
	}
}

func TestSurface_TextsAgentsReadNameOnlyRegisteredTools(t *testing.T) {
	registered := map[string]bool{}
	for _, tool := range surface() {
		registered[tool.Name] = true
	}
	texts := map[string]string{"server instructions": instructions}
	for _, p := range workflowPrompts() {
		args := map[string]string{}
		for _, a := range p.args {
			args[a.name] = "x"
		}
		texts["prompt "+p.name] = p.render(args)
	}
	for i, text := range plays.DefaultInstructions() {
		texts["built-in play text "+string(rune('a'+i))] = text
	}
	for _, tool := range surface() {
		texts["description of "+tool.Name] = tool.Description
	}
	for source, text := range texts {
		for _, name := range toolWord.FindAllString(text, -1) {
			assert.True(t, registered[name], "%s names %s, which is not a tool", source, name)
		}
	}
}

func sentences(text string) int {
	n := strings.Count(text, ". ") + strings.Count(text, "? ")
	if strings.HasSuffix(strings.TrimSpace(text), ".") {
		n++
	}
	return n
}

// undescribed walks a schema's properties, array items included, and returns the paths with no description.
func undescribed(s *jsonschema.Schema, prefix string) []string {
	var out []string
	names := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		prop := s.Properties[name]
		path := prefix + name
		if prop.Description == "" {
			out = append(out, path)
		}
		out = append(out, undescribed(prop, path+".")...)
		if prop.Items != nil {
			out = append(out, undescribed(prop.Items, path+"[].")...)
		}
	}
	return out
}
