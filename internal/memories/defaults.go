package memories

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// DefaultsKind names the default memories among the instance templates: each new project copies them in once (ADR 0194).
const DefaultsKind = "memory"

//go:embed defaults/*.md
var defaultFiles embed.FS

// DefaultKeys are the shipped default memories' stable keys, in the order a new project lists them.
var DefaultKeys = []string{"writing-a-doc", "writing-tickets", "building-a-ticket", "reviewing-a-change", "finishing-a-ticket"}

// Default is one shipped default memory: its key and its SKILL.md text.
type Default struct {
	Key  string
	Text string
}

// Defaults returns the shipped default memories in DefaultKeys order.
func Defaults() []Default {
	out := make([]Default, len(DefaultKeys))
	for i, key := range DefaultKeys {
		text, err := defaultFiles.ReadFile("defaults/" + key + ".md")
		if err != nil {
			panic(fmt.Sprintf("default memory %s is not embedded: %v", key, err)) // the embed pattern above includes every key
		}
		out[i] = Default{Key: key, Text: string(text)}
	}
	return out
}

// SkillFile is a memory written as a SKILL.md: frontmatter naming it, then its markdown body.
type SkillFile struct {
	Name           string `yaml:"name"`
	Description    string `yaml:"description"`
	AlwaysIncluded bool   `yaml:"always_included"`
	Footer         bool   `yaml:"footer"`
	Body           string `yaml:"-"`
}

// ParseSkillFile reads a default memory's text, naming what a person got wrong.
func ParseSkillFile(text string) (SkillFile, error) {
	front, body, ok := splitFrontmatter(text)
	if !ok {
		return SkillFile{}, fmt.Errorf("%w: a default memory opens with frontmatter between two --- lines: name, description, and optionally always_included and footer", apperrs.ErrInvalid)
	}
	var f SkillFile
	dec := yaml.NewDecoder(strings.NewReader(front))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil && !errors.Is(err, io.EOF) {
		return SkillFile{}, fmt.Errorf("%w: the frontmatter takes name, description, always_included and footer, the last two true or false, and a value holding \": \" in quotes: %v", apperrs.ErrInvalid, err)
	}
	f.Name = strings.TrimSpace(f.Name)
	f.Description = strings.TrimSpace(f.Description)
	f.Body = strings.TrimSpace(body)
	if f.Name == "" {
		return SkillFile{}, fmt.Errorf("%w: the frontmatter needs a name, the memory's title", apperrs.ErrInvalid)
	}
	if f.Description == "" {
		return SkillFile{}, fmt.Errorf("%w: the frontmatter needs a description, the line saying when to read the memory", apperrs.ErrInvalid)
	}
	if len(f.Description) > maxWhenToUseChars {
		return SkillFile{}, fmt.Errorf("%w: the description is a when-to-use line of at most %d characters", apperrs.ErrInvalid, maxWhenToUseChars)
	}
	if f.Body == "" {
		return SkillFile{}, fmt.Errorf("%w: the memory needs a body below the frontmatter", apperrs.ErrInvalid)
	}
	if _, err := richtext.Normalize(f.Body); err != nil {
		return SkillFile{}, fmt.Errorf("%w: the body is not markdown the editor can hold: %v", apperrs.ErrInvalid, err)
	}
	return f, nil
}

// splitFrontmatter cuts text into the YAML between its opening and closing --- lines and the rest.
func splitFrontmatter(text string) (front, body string, ok bool) {
	text = strings.ReplaceAll(strings.TrimPrefix(text, "\ufeff"), "\r\n", "\n")
	rest, found := strings.CutPrefix(strings.TrimLeft(text, " \t\n"), "---\n")
	if !found {
		return "", "", false
	}
	front, body, found = strings.Cut("\n"+rest, "\n---\n")
	if found {
		return front, body, true
	}
	front, found = strings.CutSuffix(strings.TrimRight("\n"+rest, " \t\n"), "\n---")
	return front, "", found
}

// CheckSkillFile validates a default memory's text before the instance stores it.
func CheckSkillFile(text string) error {
	_, err := ParseSkillFile(text)
	return err
}

// SkillFileName is the title a default memory's text carries, "" for one that doesn't parse.
func SkillFileName(text string) string {
	f, _ := ParseSkillFile(text) // an unparsable text has no title to show
	return f.Name
}

// Seed is a memory a new project starts with, its body already in the stored rich-text form.
type Seed struct {
	Title          string
	WhenToUse      string
	Body           string
	AlwaysIncluded bool
	Footer         bool
}

// Seeds turns default memory texts into what a new project stores, skipping and logging one that doesn't parse, so a
// bad template never stops a project from being created.
func Seeds(ctx context.Context, texts []string) []Seed {
	var out []Seed
	for _, text := range texts {
		seed, err := SeedOf(text)
		if err != nil {
			logging.FromCtx(ctx).Warn("default memory skipped", slog.String("name", SkillFileName(text)), slog.Any("error", err))
			continue
		}
		out = append(out, seed)
	}
	return out
}

// SeedOf is one default memory's text as a project stores it.
func SeedOf(text string) (Seed, error) {
	f, err := ParseSkillFile(text)
	if err != nil {
		return Seed{}, err
	}
	body, err := richtext.Normalize(f.Body)
	if err != nil {
		return Seed{}, fmt.Errorf("%w: %v", apperrs.ErrInvalid, err)
	}
	return Seed{Title: f.Name, WhenToUse: f.Description, Body: body, AlwaysIncluded: f.AlwaysIncluded, Footer: f.Footer}, nil
}
