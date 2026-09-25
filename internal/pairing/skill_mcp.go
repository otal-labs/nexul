package pairing

import (
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	shipped "github.com/otal-labs/nexul/internal/platform/skills"
)

type skillGetIn struct {
	Name string `json:"name" jsonschema:"The skill's name, for example nexul-memory."`
}

type skillResult struct {
	shipped.Skill
	Paths []string `json:"paths"`
}

func skillGetTool() mcptool.Tool {
	return mcptool.New("skill_get", "Get skill",
		"Returns the current version of a skill Nexul installs on paired computers, with the full SKILL.md content and "+
			"the paths it belongs at. Compare its version with the metadata.version of the installed copy; when they "+
			"differ, write the returned content over every listed path. Setup does the same on a re-run, and "+
			"computer_list flags a provider whose installed skills are out of date.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(_ context.Context, in skillGetIn) (any, error) {
			skill, ok := shipped.Get(in.Name)
			if !ok {
				return nil, fmt.Errorf("%w: no skill named %q; Nexul ships %s", apperrs.ErrNotFound, in.Name, strings.Join(shipped.Names(), ", "))
			}
			return skillResult{Skill: skill, Paths: skill.Paths()}, nil
		})
}
