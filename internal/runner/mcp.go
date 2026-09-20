package runner

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the runner tool definitions over the visibility use-cases plus the
// machine tools: a machine is a runner-domain concept, discovered here too.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "runner_list",
			Description: "List runners with connection state, last seen time, and the job each is currently running.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListRunners(ctx)
			},
		},
		{
			Name:        "runner_queue",
			Description: "List deploys waiting for a runner to pick them up.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListQueue(ctx)
			},
		},
		{
			Name:        "machine_list",
			Description: "List machines runners have reported, with their stack root and last-seen time.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.ListMachines(ctx)
			},
		},
		{
			Name:        "machine_discover",
			Description: "Dispatch a discover job to a machine and return what it found, grouped by compose project, standalone containers, and recognised gateways.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"machine_id": map[string]any{"type": "string"},
				},
				"required": []string{"machine_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				machineID, err := mcptool.RequiredString(args, "machine_id")
				if err != nil {
					return nil, err
				}
				return s.DiscoverForImport(ctx, machineID)
			},
		},
		{
			Name:        "instance_upgrade_status",
			Description: "Report the instance's running version, the channel's newest release, and whether an upgrade can start now.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.UpgradeStatus(ctx)
			},
		},
		{
			Name:        "instance_upgrade",
			Description: "Upgrade the instance to the channel's newest release, exactly as the settings page's Upgrade button does.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.RequestUpgrade(ctx, mcpUpgradeActor(ctx))
			},
		},
	}
}

// mcpUpgradeActor resolves the acting user for upgrade provenance (ADR 0049): "<user id>:mcp", empty when no
// actor is attached to the call.
func mcpUpgradeActor(ctx context.Context) string {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return ""
	}
	return a.ID + ":mcp"
}
