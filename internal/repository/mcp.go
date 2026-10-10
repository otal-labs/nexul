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
	WorkspaceID   string `json:"workspace_id,omitempty" jsonschema:"A workspace's id (a UUID) from workspace_list: only the repositories it can make a project from. Omit it for every workspace in which you can create projects."`
	Query         string `json:"q,omitempty" jsonschema:"Keeps only repositories whose owner/name contains this text, ignoring case. At least 3 characters. Omit it to list every repository."`
	Refresh       bool   `json:"refresh,omitempty" jsonschema:"When true, asks the git provider again instead of using the answer from the last minute, for an App just installed on another account. Defaults to false."`
	Installations bool   `json:"installations,omitempty" jsonschema:"When true, also return every account and organisation Nexul's GitHub App is installed on, with the workspaces that see each one's repositories. Needs connectors:read. Defaults to false."`
}

// repositoryListOut is the page plus the installations behind it, returned only when they were asked for.
type repositoryListOut struct {
	mcptool.Page[Repo]
	Installations []Installation `json:"installations"`
}

func repositoryListTool(svc *Service) mcptool.Tool {
	return mcptool.New("repository_list", "List repositories",
		"Lists the repositories a workspace can make a project from, with owner, name, and default branch. Use it to "+
			"pick a repository for repository_scan or pull_request_list. Returns at most 100 per page; pass q to search "+
			"by owner/name text. "+
			"Once the GitHub App's private key is set, Nexul reads GitHub as the App: a workspace lists the repositories "+
			"of the installations assigned to it, whoever connected GitHub. Without a key it reads as the account "+
			"connected in Settings → Connectors, and every workspace lists what that account can open where the App "+
			"is installed. A missing repository means the App is not installed on its owner, which installs it at "+
			"https://github.com/apps/<app slug>/installations/new, or that installation is not assigned to the "+
			"workspace, which workspace_update assigns. "+
			"Pass installations to also see every account and organisation, whether each grants all or selected "+
			"repositories, its workspaces, and the GitHub page where its access is managed.",
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
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"The workspace from repository_list. Omit for the union of workspaces where you can create projects."`
	Owner       string `json:"owner" jsonschema:"The repository's owner, for example acme."`
	Repo        string `json:"repo" jsonschema:"The repository's name, for example api."`
	Ref         string `json:"ref,omitempty" jsonschema:"The branch, tag, or commit to scan, for example main. Defaults to the repository's default branch."`
}

func repositoryScanTool(svc *Service) mcptool.Tool {
	return mcptool.New("repository_scan", "Scan repository",
		"Reads a repository's file tree and proposes deployable candidates: one per compose file and one per "+
			"standalone Dockerfile, each with its services, ports, and env keys, plus the env keys of any "+
			".env.example. Pass a candidate unchanged to stack_create to create a stack from it. Reads the git "+
			"provider only; nothing is created.",
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
