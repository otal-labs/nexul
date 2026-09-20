package workspace

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// NotificationMCPTools takes user_id explicitly so an agent acting as a user sees that user's inbox.
func NotificationMCPTools(s *NotificationService) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "notification_list",
			Description: "List a user's notifications, newest first (unread state included).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user_id": map[string]any{"type": "string"},
					"limit":   map[string]any{"type": "integer"},
				},
				"required": []string{"user_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				userID, err := mcptool.RequiredString(args, "user_id")
				if err != nil {
					return nil, err
				}
				return s.List(ctx, userID, notifIntArg(args["limit"]))
			},
		},
		{
			Name:        "notification_mark_read",
			Description: "Mark a single notification as read.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user_id": map[string]any{"type": "string"},
					"id":      map[string]any{"type": "string"},
				},
				"required": []string{"user_id", "id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				userID, err := mcptool.RequiredString(args, "user_id")
				if err != nil {
					return nil, err
				}
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				if err := s.MarkRead(ctx, userID, id); err != nil {
					return nil, err
				}
				return map[string]string{"id": id, "status": "read"}, nil
			},
		},
		{
			Name:        "notification_mark_all_read",
			Description: "Mark every notification of a user as read.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user_id": map[string]any{"type": "string"},
				},
				"required": []string{"user_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				userID, err := mcptool.RequiredString(args, "user_id")
				if err != nil {
					return nil, err
				}
				if err := s.MarkAllRead(ctx, userID); err != nil {
					return nil, err
				}
				return map[string]string{"status": "all read"}, nil
			},
		},
	}
}

// notifIntArg is named distinctly from intArg in the projects MCP file since both live in this package.
func notifIntArg(v any) int {
	if f, ok := v.(float64); ok && f > 0 {
		return int(f)
	}
	return 0
}
