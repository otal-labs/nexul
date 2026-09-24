package memories

import (
	"context"
	"fmt"
	"strconv"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// viaMCP marks every save made through an MCP tool call (ADR 0049), so the version's author_via records
// where it came from — an Agent turn's memory_update is attributed to the Agent via the mentioning user.
const viaMCP = "mcp"

// MCPTools returns the memories tool definitions (named domain_action).
func MCPTools(s *Service) []mcptool.Tool {
	return append([]mcptool.Tool{
		{
			Name:        "memory_list",
			Description: "List memories (title and when-to-use only, no body). With project_id, returns the project's workspace memories first, then its own. With no project_id, workspace_id is required and returns that workspace's workspace-scoped memories only.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id":   map[string]any{"type": "string"},
					"workspace_id": map[string]any{"type": "string", "description": "Required when project_id is omitted"},
				},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				return memoryListCall(ctx, s, args)
			},
		},
		{
			Name:        "memory_get",
			Description: "Fetch a single memory by id, rendered as markdown.",
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
				m, err := s.Get(ctx, id)
				if err != nil {
					return nil, err
				}
				return memoryToMarkdown(m)
			},
		},
		{
			Name:        "memory_create",
			Description: "Create a memory from a markdown body and return it as markdown. Omit project_id to save at workspace scope, in which case workspace_id is required.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id":      map[string]any{"type": "string"},
					"workspace_id":    map[string]any{"type": "string", "description": "Required when project_id is omitted"},
					"title":           map[string]any{"type": "string"},
					"when_to_use":     map[string]any{"type": "string", "description": "One-line hint for when this memory applies"},
					"body":            map[string]any{"type": "string", "description": "Markdown body"},
					"always_included": map[string]any{"type": "boolean"},
				},
				"required": []string{"title"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				title, err := mcptool.RequiredString(args, "title")
				if err != nil {
					return nil, err
				}
				projectID := mcptool.OptionalString(args["project_id"])
				workspaceID := mcptool.OptionalString(args["workspace_id"])
				m, err := s.Create(ctx, projectID, workspaceID, title, mcptool.OptionalString(args["when_to_use"]), mcptool.OptionalString(args["body"]), boolArg(args["always_included"]), viaMCP)
				if err != nil {
					return nil, err
				}
				return memoryToMarkdown(m)
			},
		},
		{
			Name:        "memory_update",
			Description: "Update a memory's title, when-to-use, body (markdown), and always-included flag. The interview memory (kind \"interview\") stays always included whatever the flag says, and its body is capped at 8,000 characters of markdown.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":              map[string]any{"type": "string"},
					"title":           map[string]any{"type": "string"},
					"when_to_use":     map[string]any{"type": "string"},
					"body":            map[string]any{"type": "string", "description": "Markdown body"},
					"always_included": map[string]any{"type": "boolean"},
				},
				"required": []string{"id", "title"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				vals, err := mcptool.RequiredStrings(args, "id", "title")
				if err != nil {
					return nil, err
				}
				id, title := vals[0], vals[1]
				m, err := s.Update(ctx, id, title, mcptool.OptionalString(args["when_to_use"]), mcptool.OptionalString(args["body"]), boolArg(args["always_included"]), viaMCP)
				if err != nil {
					return nil, err
				}
				return memoryToMarkdown(m)
			},
		},
		{
			Name:        "memory_delete",
			Description: "Delete a memory.",
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
				return nil, s.Delete(ctx, id)
			},
		},
		{
			Name:        "memory_list_versions",
			Description: "List a memory's version history, newest first.",
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
				return s.ListVersions(ctx, id)
			},
		},
		{
			Name:        "memory_revert",
			Description: "Revert a memory to a prior version, appending a new version with that version's content.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "string"},
					"version": map[string]any{"type": "number"},
				},
				"required": []string{"id", "version"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				version, err := intArg(args["version"])
				if err != nil {
					return nil, err
				}
				m, err := s.Revert(ctx, id, version, viaMCP)
				if err != nil {
					return nil, err
				}
				return memoryToMarkdown(m)
			},
		},
		{
			Name:        "memory_clone",
			Description: "Clone a memory into another project, or into a workspace directly (omit project_id, workspace_id then required), in this workspace or another the caller belongs to.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":           map[string]any{"type": "string"},
					"project_id":   map[string]any{"type": "string", "description": "Destination project id"},
					"workspace_id": map[string]any{"type": "string", "description": "Destination workspace id, required when project_id is omitted"},
				},
				"required": []string{"id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := mcptool.RequiredString(args, "id")
				if err != nil {
					return nil, err
				}
				m, err := s.Clone(ctx, id, mcptool.OptionalString(args["project_id"]), mcptool.OptionalString(args["workspace_id"]))
				if err != nil {
					return nil, err
				}
				return memoryToMarkdown(m)
			},
		},
	}, interviewTools(s)...)
}

// interviewTools are the interview memory and Interview template tools (ADR 0065).
func interviewTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "memory_create_interview",
			Description: "Return the project's interview memory, the rules sent in full with every agent turn in the project, creating it from the workspace's Interview template if it does not exist yet. Edit it with memory_update; its body is capped at 8,000 characters of markdown.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
				},
				"required": []string{"project_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				projectID, err := mcptool.RequiredString(args, "project_id")
				if err != nil {
					return nil, err
				}
				m, err := s.CreateInterview(ctx, projectID, viaMCP)
				if err != nil {
					return nil, err
				}
				return memoryToMarkdown(m)
			},
		},
		{
			Name:        "interview_template_get",
			Description: "Get the workspace's Interview template (markdown), the starting point each new project's interview memory is copied from.",
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
				return s.InterviewTemplate(ctx, workspaceID)
			},
		},
		{
			Name:        "interview_template_update",
			Description: "Replace the workspace's Interview template (markdown, at most 8,000 characters). Existing interview memories are not changed.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace_id": map[string]any{"type": "string"},
					"body":         map[string]any{"type": "string", "description": "Markdown body"},
				},
				"required": []string{"workspace_id"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				workspaceID, err := mcptool.RequiredString(args, "workspace_id")
				if err != nil {
					return nil, err
				}
				return s.SaveInterviewTemplate(ctx, workspaceID, mcptool.OptionalString(args["body"]))
			},
		},
	}
}

// memoryListCall backs memory_list: a project id returns its workspace memories then its own, no project id
// falls back to the given workspace's workspace-scoped memories only.
func memoryListCall(ctx context.Context, s *Service, args map[string]any) (any, error) {
	if projectID := mcptool.OptionalString(args["project_id"]); projectID != "" {
		return s.ListForProject(ctx, projectID)
	}
	workspaceID, err := mcptool.RequiredString(args, "workspace_id")
	if err != nil {
		return nil, err
	}
	return s.ListWorkspaceScoped(ctx, workspaceID)
}

// intArg accepts a JSON number (float64, as the MCP transport decodes it) or a numeric string.
func intArg(v any) (int, error) {
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case string:
		return strconv.Atoi(n)
	default:
		return 0, fmt.Errorf("%w: version must be a number", apperrs.ErrInvalid)
	}
}

// memoryToMarkdown returns a copy of the memory with its body rendered as markdown for LLM consumption.
func memoryToMarkdown(m *Memory) (any, error) {
	md, err := richtext.ToMarkdown(m.Body)
	if err != nil {
		return nil, err
	}
	out := *m
	out.Body = md
	return &out, nil
}

func boolArg(v any) bool {
	b, ok := v.(bool)
	return ok && b
}
