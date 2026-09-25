package repository

import (
	"context"
	"errors"
	"fmt"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the repository tools: the project wizard's first two steps, list and scan.
func MCPTools(s Scanner) []mcptool.Tool {
	return []mcptool.Tool{repositoryListTool(s), repositoryScanTool(s)}
}

type repositoryListIn struct {
	mcptool.PageArgs
}

func repositoryListTool(s Scanner) mcptool.Tool {
	return mcptool.New("repository_list", "List repositories",
		"Lists the repositories the connected git provider installation can read, with owner, name, and default "+
			"branch. Use it to pick a repository for repository_scan or pull_request_list. Returns at most 100 per page.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in repositoryListIn) (any, error) {
			repos, err := ListRepos(ctx, s)
			if err != nil {
				return nil, err
			}
			return mcptool.Paginate(repos, in.PageArgs), nil
		})
}

type repositoryScanIn struct {
	Owner string `json:"owner" jsonschema:"The repository's owner, for example acme."`
	Repo  string `json:"repo" jsonschema:"The repository's name, for example api."`
	Ref   string `json:"ref,omitempty" jsonschema:"The branch, tag, or commit to scan, for example main. Defaults to the repository's default branch."`
}

func repositoryScanTool(s Scanner) mcptool.Tool {
	return mcptool.New("repository_scan", "Scan repository",
		"Reads a repository's file tree and proposes deployable candidates: one per compose file and one per "+
			"standalone Dockerfile, each with its services, ports, and env keys, plus the env keys of any "+
			".env.example. Pass a candidate unchanged to stack_create to create a stack from it. Reads the git "+
			"provider only; nothing is created.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in repositoryScanIn) (any, error) {
			out, err := Scan(ctx, s, in.Owner, in.Repo, in.Ref)
			if errors.Is(err, apperrors.ErrNotFound) {
				return nil, fmt.Errorf("%w; repository_list lists the repositories Nexul can read", err)
			}
			if err != nil {
				return nil, err
			}
			return out, nil
		})
}
