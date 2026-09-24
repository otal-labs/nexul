package tickets

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func idSchema(extra map[string]any, required ...string) map[string]any {
	props := map[string]any{"id": map[string]any{"type": "string"}}
	for k, v := range extra {
		props[k] = v
	}
	return map[string]any{"type": "object", "properties": props, "required": append([]string{"id"}, required...)}
}

func ticketLinkMCPTools(s *Service) []mcptool.Tool {
	str := map[string]any{"type": "string"}
	return []mcptool.Tool{
		{
			Name:        "ticket_get_ticket_links",
			Description: "List a ticket's found-in and blocked-by links in both directions: what it was found in, bugs found in it, what blocks it, and what it blocks. Each linked ticket carries done, read from its column's stage; blocked is true while any blocker is not done.",
			InputSchema: idSchema(nil),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Links(ctx, id)
			},
		},
		{
			Name:        "ticket_set_found_in",
			Description: "Record the ticket a bug was found in (origin_id), or set origin_unknown true when nobody knows. Replaces any earlier found-in.",
			InputSchema: idSchema(map[string]any{"origin_id": str, "origin_unknown": map[string]any{"type": "boolean"}}),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				unknown, _ := args["origin_unknown"].(bool)
				return s.SetFoundIn(ctx, id, mcptool.OptionalString(args["origin_id"]), unknown)
			},
		},
		{
			Name:        "ticket_remove_found_in",
			Description: "Remove a ticket's found-in link or origin-unknown marker.",
			InputSchema: idSchema(nil),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.RemoveFoundIn(ctx, id)
			},
		},
		{
			Name:        "ticket_add_blocker",
			Description: "Mark a ticket as blocked by another (blocker_id) that must reach a done-stage column first. Refused if it would form a cycle. Never stops the ticket moving.",
			InputSchema: idSchema(map[string]any{"blocker_id": str}, "blocker_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "blocker_id")
				if err != nil {
					return nil, err
				}
				return s.AddBlocker(ctx, vals[0], vals[1])
			},
		},
		{
			Name:        "ticket_remove_blocker",
			Description: "Remove a blocked-by link from a ticket.",
			InputSchema: idSchema(map[string]any{"blocker_id": str}, "blocker_id"),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "blocker_id")
				if err != nil {
					return nil, err
				}
				return s.RemoveBlocker(ctx, vals[0], vals[1])
			},
		},
		{
			Name:        "ticket_list_blocked",
			Description: "List every blocked ticket id with the blockers it still waits on (those not yet in a done-stage column).",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				return s.UnclearedBlockers(ctx)
			},
		},
	}
}
