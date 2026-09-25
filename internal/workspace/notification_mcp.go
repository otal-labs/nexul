package workspace

import (
	"context"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// NotificationMCPTools act on the caller's own inbox, the same one the web app's bell shows.
func NotificationMCPTools(s *NotificationService) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "notification_list",
			Description: "List your notifications, newest first (unread state included).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{"type": "integer"},
				},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				userID, err := callerID(ctx)
				if err != nil {
					return nil, err
				}
				return s.List(ctx, userID, notifIntArg(args["limit"]))
			},
		},
		{
			Name:        "notification_mark_read",
			Description: "Mark one of your notifications as read.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				userID, err := callerID(ctx)
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
			Description: "Mark every one of your notifications as read.",
			InputSchema: mcptool.ObjectSchema(nil),
			Call: func(ctx context.Context, _ map[string]any) (any, error) {
				userID, err := callerID(ctx)
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

// callerID is the authenticated user; an argument never names whose inbox a tool touches.
func callerID(ctx context.Context) (string, error) {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return "", apperrs.ErrUnauthorized
	}
	return a.ID, nil
}

// notifIntArg is named distinctly from intArg in the projects MCP file since both live in this package.
func notifIntArg(v any) int {
	if f, ok := v.(float64); ok && f > 0 {
		return int(f)
	}
	return 0
}
