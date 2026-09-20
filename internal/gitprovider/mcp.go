package gitprovider

import (
	"context"
	"fmt"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the gitprovider tool definitions (named domain_action).
func MCPTools(p GitProvider) []mcptool.Tool {
	return []mcptool.Tool{
		{
			Name:        "git_list_prs",
			Description: "List pull requests for a repository.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"owner": map[string]any{"type": "string"},
					"repo":  map[string]any{"type": "string"},
					"state": map[string]any{"type": "string", "enum": []string{"open", "closed", "all"}},
					"limit": map[string]any{"type": "integer"},
				},
				"required": []string{"owner", "repo"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				owner, repo, err := ownerRepoArgs(args)
				if err != nil {
					return nil, err
				}
				opts := PROpts{State: strArg(args["state"], "open"), Limit: intArg(args["limit"])}
				return ListPRs(ctx, p, owner, repo, opts)
			},
		},
		{
			Name:        "git_get_pr",
			Description: "Fetch a single pull request by number.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"owner":  map[string]any{"type": "string"},
					"repo":   map[string]any{"type": "string"},
					"number": map[string]any{"type": "integer"},
				},
				"required": []string{"owner", "repo", "number"},
			},
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				owner, repo, err := ownerRepoArgs(args)
				if err != nil {
					return nil, err
				}
				number, ok := args["number"].(float64)
				if !ok || number < 1 {
					return nil, fmt.Errorf("%w: number must be a positive integer", apperrors.ErrInvalid)
				}
				return GetPR(ctx, p, owner, repo, int(number))
			},
		},
	}
}

func ownerRepoArgs(args map[string]any) (owner, repo string, err error) {
	owner, ok := args["owner"].(string)
	if !ok || owner == "" {
		return "", "", fmt.Errorf("%w: owner is required", apperrors.ErrInvalid)
	}
	repo, ok = args["repo"].(string)
	if !ok || repo == "" {
		return "", "", fmt.Errorf("%w: repo is required", apperrors.ErrInvalid)
	}
	return owner, repo, nil
}

func strArg(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func intArg(v any) int {
	if f, ok := v.(float64); ok && f > 0 {
		return int(f)
	}
	return 0
}
