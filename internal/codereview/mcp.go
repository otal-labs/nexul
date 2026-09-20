package codereview

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools is read-only, since provider events are the only writer (ADR 0034).
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "review_list_by_ticket",
			Description: "List code review records for a ticket.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ticket_id": map[string]any{"type": "string"},
				},
				"required": []string{"ticket_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				ticketID, err := mcptool.RequiredString(args, "ticket_id")
				if err != nil {
					return nil, err
				}
				return s.ListByTicket(ctx, ticketID)
			},
		},
		{
			Name:        "review_get",
			Description: "Fetch a single code review record by id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				return s.Get(ctx, id)
			},
		},
	}
}
