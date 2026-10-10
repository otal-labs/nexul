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
	WorkspaceID   string `json:"workspace_id,omitempty" jsonschema:"A workspace's id (a UUID) from workspace_list where you can create projects. Omit it to need that in any workspace; the list is your own GitHub view either way."`
	Query         string `json:"q,omitempty" jsonschema:"Keeps only repositories whose owner/name contains this text, ignoring case. At least 3 characters. Omit it to list every repository."`
	Refresh       bool   `json:"refresh,omitempty" jsonschema:"When true, asks the git provider again instead of using the answer from the last minute, for an App just installed on another account. Defaults to false."`
	Installations bool   `json:"installations,omitempty" jsonschema:"When true, also return the accounts and organisations the GitHub App is installed on that your own GitHub account can see, with the workspaces of yours that use each. Defaults to false."`
}

// repositoryListOut is the page plus the installations behind it, returned only when they were asked for.
type repositoryListOut struct {
	mcptool.Page[Repo]
	Installations []Installation `json:"installations"`
}

func repositoryListTool(svc *Service) mcptool.Tool {
	return mcptool.New("repository_list", "List repositories",
		"Lists the repositories you can make a project from: the ones your own GitHub account can open where "+
			"Nexul's GitHub App is installed, read with your GitHub sign-in, never with another person's or the "+
			"instance's credential. Use it to pick a repository for repository_scan or project_update's "+
			"add_repositories. Returns at most 100 per page; pass q to search by owner/name text. If you have not "+
			"connected GitHub it fails and says so; the person connects it in Settings → Profile. A missing "+
			"repository means your GitHub account cannot open it or the App is not installed on its owner, which "+
			"installs it at https://github.com/apps/<app slug>/installations/new. "+
			"Pass installations to also see those accounts and organisations, whether each grants all or selected "+
			"repositories, the workspaces of yours that use it, and the GitHub page where its access is managed.",
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
	Owner       string `json:"owner" jsonschema:"The repository's owner, for example acme."`
	Repo        string `json:"repo" jsonschema:"The repository's name, for example api."`
	Ref         string `json:"ref,omitempty" jsonschema:"The branch, tag, or commit to scan, for example main. Defaults to the repository's default branch."`
}

func repositoryScanTool(svc *Service) mcptool.Tool {
	return mcptool.New("repository_scan", "Scan repository",
		"Reads a repository's file tree and proposes deployable candidates: one per compose file and one per "+
			"standalone Dockerfile, each with its services, ports, and env keys, plus the env keys of any "+
			".env.example. Pass a candidate unchanged to stack_create to create a stack from it. Reads GitHub "+
			"with your own GitHub sign-in, so only a repository you can open scans; nothing is created.",
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
