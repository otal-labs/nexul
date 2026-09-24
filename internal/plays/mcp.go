package plays

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the play definition tools (named domain_action); the run tools live in RunMCPTools.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name: "play_list",
			Description: "List a workspace's plays, sorted by label. Pass type (and project_id, stage for a " +
				"ticket play) to switch to the applicable list for the caller: enabled, not excluded for the " +
				"project, matching stage, and not denied plays:run.",
			InputSchema: mcptool.ObjectSchema(map[string]any{
				"workspace_id": map[string]any{"type": "string"},
				"project_id":   map[string]any{"type": "string"},
				"type":         map[string]any{"type": "string", "enum": targetTypeNames},
				"stage":        map[string]any{"type": "string", "enum": stageNames()},
			}, "workspace_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, err := mcptool.RequiredString(args, "workspace_id")
				if err != nil {
					return nil, err
				}
				typeStr := mcptool.OptionalString(args["type"])
				if typeStr == "" {
					return s.List(ctx, workspaceID)
				}
				return s.ListApplicable(ctx, workspaceID, actorID(ctx), mcptool.OptionalString(args["project_id"]), Type(typeStr), optionalStage(args["stage"]))
			},
		},
		{
			Name: "play_create",
			Description: "Create a play in a workspace: label (button text), type (ticket, doc, or interview), description, " +
				"instructions, enabled, show-when stage (ticket plays only), excluded projects.",
			InputSchema: mcptool.ObjectSchema(map[string]any{
				"workspace_id":         map[string]any{"type": "string"},
				"label":                map[string]any{"type": "string"},
				"type":                 map[string]any{"type": "string", "enum": targetTypeNames},
				"description":          map[string]any{"type": "string"},
				"instructions":         map[string]any{"type": "string"},
				"enabled":              map[string]any{"type": "boolean"},
				"show_when_stage":      map[string]any{"type": "string", "enum": stageNames()},
				"excluded_project_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}, "workspace_id", "label", "type"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, label, err := requiredWorkspaceAndLabel(args)
				if err != nil {
					return nil, err
				}
				return s.Create(ctx, workspaceID, CreateInput{
					Label: label, Type: Type(mcptool.OptionalString(args["type"])),
					Description:  mcptool.OptionalString(args["description"]),
					Instructions: mcptool.OptionalString(args["instructions"]),
					Enabled:      optionalBool(args["enabled"]), ShowWhenStage: optionalStage(args["show_when_stage"]),
					ExcludedProjectIDs: stringsArg(args["excluded_project_ids"]),
				})
			},
		},
		{
			Name: "play_update",
			Description: "Replace a play's owner-edited fields: label, description, instructions, enabled, " +
				"show-when stage (ticket plays only), excluded projects. Type is immutable after create.",
			InputSchema: mcptool.ObjectSchema(map[string]any{
				"workspace_id":         map[string]any{"type": "string"},
				"id":                   map[string]any{"type": "string"},
				"label":                map[string]any{"type": "string"},
				"description":          map[string]any{"type": "string"},
				"instructions":         map[string]any{"type": "string"},
				"enabled":              map[string]any{"type": "boolean"},
				"show_when_stage":      map[string]any{"type": "string", "enum": stageNames()},
				"excluded_project_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}, "workspace_id", "id", "label"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, label, err := requiredWorkspaceAndLabel(args)
				if err != nil {
					return nil, err
				}
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Update(ctx, workspaceID, id, UpdateInput{
					Label: label, Description: mcptool.OptionalString(args["description"]),
					Instructions: mcptool.OptionalString(args["instructions"]),
					Enabled:      optionalBool(args["enabled"]), ShowWhenStage: optionalStage(args["show_when_stage"]),
					ExcludedProjectIDs: stringsArg(args["excluded_project_ids"]),
				})
			},
		},
		{
			Name:        "play_delete",
			Description: "Delete a play.",
			InputSchema: mcptool.ObjectSchema(map[string]any{
				"workspace_id": map[string]any{"type": "string"},
				"id":           map[string]any{"type": "string"},
			}, "workspace_id", "id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, err := mcptool.RequiredString(args, "workspace_id")
				if err != nil {
					return nil, err
				}
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				if err := s.Delete(ctx, workspaceID, id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "deleted"}, nil
			},
		},
	}
}

func requiredWorkspaceAndLabel(args map[string]any) (string, string, error) {
	vals, err := mcptool.RequiredStrings(args, "workspace_id", "label")
	if err != nil {
		return "", "", err
	}
	return vals[0], vals[1], nil
}

// targetTypeNames lists the play types, which are also the run target types.
var targetTypeNames = []string{string(TypeTicket), string(TypeDoc), string(TypeInterview)}

func stageNames() []string {
	out := make([]string, len(stages))
	for i, s := range stages {
		out[i] = string(s)
	}
	return out
}

func optionalStage(v any) *Stage {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	stage := Stage(s)
	return &stage
}

func optionalBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func stringsArg(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
