package repository

import (
	"context"
	"errors"
	"fmt"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the repository tools: the project wizard's first two steps, list and scan.
func MCPTools(svc *Service) []mcptool.Tool {
	return []mcptool.Tool{repositoryListTool(svc), repositoryScanTool(svc)}
}

type repositoryListIn struct {
	mcptool.PageArgs
	WorkspaceID   string `json:"workspace_id,omitempty" jsonschema:"A workspace's id from workspace_list where you can create projects. Omit it to need that in any workspace; the list is your own GitHub view either way."`
	Query         string `json:"q,omitempty" jsonschema:"Keeps only repositories whose owner/name contains this text, ignoring case. At least 3 characters. Omit it to list every repository."`
	Refresh       bool   `json:"refresh,omitempty" jsonschema:"Ask the git provider again instead of using the last minute's answer, for an App just installed on another account."`
	Installations bool   `json:"installations,omitempty" jsonschema:"Also return the App's installations your GitHub account can see."`
}

// repositoryListOut is the page plus the installations behind it, returned only when they were asked for.
type repositoryListOut struct {
	mcptool.Page[Repo]
	Installations []Installation `json:"installations"`
}

func repositoryListTool(svc *Service) mcptool.Tool {
	return mcptool.New("repository_list", "List repositories",
		"Lists the repositories you can make a project from: those your own GitHub account can open where Nexul's "+
			"GitHub App is installed. Use it to pick a repository for repository_scan or project_update's "+
			"add_repositories; q searches by owner/name text. A missing one means your GitHub account cannot open it or "+
			"the App is not installed on its owner (install it at https://github.com/apps/<app slug>/installations/new). "+
			"installations adds the accounts and organisations the App is installed on, whether each grants all or "+
			"selected repositories, the workspaces that use it, and where its access is managed.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in repositoryListIn) (any, error) {
			repos, err := svc.ListRepos(ctx, in.WorkspaceID, in.Query, in.Refresh)
			if err != nil {
				return nil, err
			}
			page := mcptool.Paginate(repos, in.PageArgs)
			if !in.Installations {
				return page, nil
			}
			installs, err := svc.ListInstallations(ctx)
			if err != nil {
				return nil, err
			}
			return repositoryListOut{Page: page, Installations: installs}, nil
		})
}

type repositoryScanIn struct {
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"A workspace where you can create projects, from workspace_list. Omit it to need that in any workspace."`
	Owner       string `json:"owner" jsonschema:"The repository's owner."`
	Repo        string `json:"repo" jsonschema:"The repository's name."`
	Ref         string `json:"ref,omitempty" jsonschema:"The branch, tag, or commit to scan. Defaults to the repository's default branch."`
}

func repositoryScanTool(svc *Service) mcptool.Tool {
	return mcptool.New("repository_scan", "Scan repository",
		"Reads a repository's file tree and proposes deployable candidates: one per compose file and standalone "+
			"Dockerfile, each with its services, ports, and env keys. Pass a candidate unchanged to stack_create to "+
			"create a stack from it. Nothing is created; only a repository you can open scans.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in repositoryScanIn) (any, error) {
			out, err := svc.Scan(ctx, in.WorkspaceID, in.Owner, in.Repo, in.Ref)
			if errors.Is(err, apperrors.ErrNotFound) {
				return nil, fmt.Errorf("%w; repository_list lists the repositories Nexul can read", err)
			}
			if err != nil {
				return nil, err
			}
			return out, nil
		})
}
