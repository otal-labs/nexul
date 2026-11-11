package automations

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
)

// The board pair and crash intake; regenerate via automations/defaults/build.ts after editing the source.
var (
	//go:embed defaults/ticket-finished.bundle.js
	ticketFinishedBundle string
	//go:embed defaults/pr-opened.bundle.js
	prOpenedBundle string
	//go:embed defaults/crash-intake.bundle.js
	crashIntakeBundle string

	//go:embed defaults/ticket-finished.surface.json
	ticketFinishedSurface []byte
	//go:embed defaults/pr-opened.surface.json
	prOpenedSurface []byte
	//go:embed defaults/crash-intake.surface.json
	crashIntakeSurface []byte
)

// withSurface adds the subscriptions and config schema build.ts wrote beside the bundle; the files are embedded, so bad JSON is a build defect.
func withSurface(def DefaultDefinition, surface []byte) DefaultDefinition {
	var parsed struct {
		Subscriptions []string        `json:"subscriptions"`
		Source        string          `json:"source"`
		ConfigSchema  json.RawMessage `json:"config_schema"`
	}
	if err := json.Unmarshal(surface, &parsed); err != nil {
		panic(fmt.Sprintf("default automation %s surface: %v", def.ID, err))
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, parsed.ConfigSchema); err != nil {
		panic(fmt.Sprintf("default automation %s config schema: %v", def.ID, err))
	}
	def.Subscriptions = parsed.Subscriptions
	def.Source = parsed.Source
	def.ConfigSchema = compact.Bytes()
	return def
}

// DefaultDefinitions is the composition root's supply of shipped defaults
// (the board pair and crash intake), wired into NewSeeder in server/cmd/main.go.
func DefaultDefinitions() []DefaultDefinition {
	return []DefaultDefinition{
		withSurface(DefaultDefinition{
			ID:          "default-ticket-finished",
			Name:        "Ticket finished",
			Description: "Moves a ticket to its project's Done status once every linked PR has merged.",
			Code:        ticketFinishedBundle,
			Message:     "Uses each project's own Done status; no setup",
			Scopes:      []string{"projects:read", "tickets:read", "tickets:write"},
			Enabled:     true,
		}, ticketFinishedSurface),
		withSurface(DefaultDefinition{
			ID:          "default-pr-opened",
			Name:        "PR opened",
			Description: "Moves each linked ticket to its project's In review status when a pull request opens.",
			Code:        prOpenedBundle,
			Message:     "Uses each project's own In review status; no setup",
			Scopes:      []string{"projects:read", "tickets:read", "tickets:write"},
			Enabled:     true,
		}, prOpenedSurface),
		withSurface(DefaultDefinition{
			ID:          "default-crash-intake",
			Name:        "Crash intake",
			Description: "Files each crash report a bot posts in a channel as one bug ticket, notes repeats on it, and files a high-priority bug when a fixed crash comes back.",
			Code:        crashIntakeBundle,
			Scopes:      []string{"tickets:read", "tickets:write"},
			// Off until someone picks its channel and project.
			Enabled: false,
		}, crashIntakeSurface),
	}
}
