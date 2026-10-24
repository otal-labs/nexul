package automations

import _ "embed"

// The board pair and crash intake; regenerate via automations/defaults/build.ts after editing the source.
var (
	//go:embed defaults/ticket-finished.bundle.js
	ticketFinishedBundle string
	//go:embed defaults/pr-opened.bundle.js
	prOpenedBundle string
	//go:embed defaults/crash-intake.bundle.js
	crashIntakeBundle string
)

// DefaultDefinitions is the composition root's supply of shipped defaults
// (the board pair and crash intake), wired into NewSeeder in server/cmd/main.go.
func DefaultDefinitions() []DefaultDefinition {
	return []DefaultDefinition{
		{
			ID:          "default-ticket-finished",
			Name:        "Ticket finished",
			Description: "Moves a ticket to its workspace's completed status once every linked PR has merged.",
			Code:        ticketFinishedBundle,
			Scopes:      []string{"tickets:write"},
			Enabled:     true,
		},
		{
			ID:          "default-pr-opened",
			Name:        "PR opened",
			Description: "Moves a ticket to its workspace's in-review status when a linked pull request opens.",
			Code:        prOpenedBundle,
			Scopes:      []string{"tickets:write"},
			Enabled:     true,
		},
		{
			ID:          "default-crash-intake",
			Name:        "Crash intake",
			Description: "Files each crash report a bot posts in a channel as one bug ticket, notes repeats on it, and files a high-priority bug when a fixed crash comes back.",
			Code:        crashIntakeBundle,
			Scopes:      []string{"tickets:write"},
			// Off until someone picks its channel and project.
			Enabled: false,
		},
	}
}
