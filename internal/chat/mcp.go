package chat

import (
	"context"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the chat tool definitions; list/post act as whoever the identity context authenticates.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "chat_list_conversations",
			Description: "List the authenticated user's chat surface in a workspace: every public channel plus every other conversation they're a participant in.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace_id": map[string]any{"type": "string"},
				},
				"required": []string{"workspace_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, err := mcptool.RequiredString(args, "workspace_id")
				if err != nil {
					return nil, err
				}
				return s.ListConversations(ctx, workspaceID, actorIDFromCtx(ctx))
			},
		},
		{
			Name:        "chat_list_messages",
			Description: "List a conversation's messages, oldest first.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"conversation_id": map[string]any{"type": "string"},
					"limit":           map[string]any{"type": "integer"},
				},
				"required": []string{"conversation_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				conversationID, err := mcptool.RequiredString(args, "conversation_id")
				if err != nil {
					return nil, err
				}
				return s.ListMessages(ctx, conversationID, actorIDFromCtx(ctx), intArg(args["limit"]))
			},
		},
		{
			Name:        "doc_thread_get",
			Description: "Get or lazily create the authenticated user's doc thread, gated by docs:thread on the doc.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace_id": map[string]any{"type": "string"},
					"doc_id":       map[string]any{"type": "string"},
				},
				"required": []string{"workspace_id", "doc_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "workspace_id", "doc_id")
				if err != nil {
					return nil, err
				}
				return s.GetOrCreateDocThread(ctx, vals[0], vals[1], actorIDFromCtx(ctx))
			},
		},
		{
			Name:        "interview_thread_get",
			Description: "Get or lazily create a project's interview thread, the conversation the Interview play asks its questions in.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace_id": map[string]any{"type": "string"},
					"project_id":   map[string]any{"type": "string"},
				},
				"required": []string{"workspace_id", "project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "workspace_id", "project_id")
				if err != nil {
					return nil, err
				}
				return s.GetOrCreateInterviewThread(ctx, vals[0], vals[1], actorIDFromCtx(ctx))
			},
		},
		{
			Name:        "chat_post_message",
			Description: "Post a markdown message to a conversation as the authenticated user, parsing @user/@Agent mentions.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"conversation_id": map[string]any{"type": "string"},
					"body":            map[string]any{"type": "string"},
				},
				"required": []string{"conversation_id", "body"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				conversationID, err := mcptool.RequiredString(args, "conversation_id")
				if err != nil {
					return nil, err
				}
				body, err := mcptool.RequiredString(args, "body")
				if err != nil {
					return nil, err
				}
				return s.PostMessage(ctx, conversationID, actorIDFromCtx(ctx), body)
			},
		},
	}
}

func actorIDFromCtx(ctx context.Context) string {
	if a, ok := identity.ActorFromCtx(ctx); ok {
		return a.ID
	}
	return ""
}

func intArg(v any) int {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return int(n)
		}
	case int:
		if n > 0 {
			return n
		}
	}
	return 0
}
