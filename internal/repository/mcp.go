package repository

import (
	"context"
	"errors"
	"fmt"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the repository tools: the project wizard's first two steps, list and scan.
func MCPTools(s Scanner, l InstallationLister, g Gate) []mcptool.Tool {
	return []mcptool.Tool{repositoryListTool(s, l, g), repositoryScanTool(s, g)}
}

type repositoryListIn struct {
	mcptool.PageArgs
	Query         string `json:"q,omitempty" jsonschema:"Keeps only repositories whose owner/name contains this text, ignoring case. At least 3 characters. Omit it to list every repository."`
	Refresh       bool   `json:"refresh,omitempty" jsonschema:"When true, asks the git provider again instead of using the answer from the last minute, for an App just installed on another account. Defaults to false."`
	Installations bool   `json:"installations,omitempty" jsonschema:"When true, also return the accounts and organisations Nexul's GitHub App is installed on. Defaults to false."`
}

// repositoryListOut is the page plus the installations behind it, returned only when they were asked for.
type repositoryListOut struct {
	mcptool.Page[Repo]
	Installations []Installation `json:"installations"`
}

func repositoryListTool(s Scanner, l InstallationLister, g Gate) mcptool.Tool {
	return mcptool.New("repository_list", "List repositories",
		"Lists the repositories the connected git provider installation can read, with owner, name, and default "+
			"branch. Use it to pick a repository for repository_scan or pull_request_list. Returns at most 100 per page; "+
			"pass q to search by owner/name text. "+
			"Only accounts with Nexul's GitHub App installed are listed; a missing repository means the App is not "+
			"installed on its owner, which installs it at https://github.com/apps/<app slug>/installations/new. "+
			"Pass installations to also see those accounts and organisations, whether each grants all or selected "+
			"repositories, and the GitHub page where its access is managed.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in repositoryListIn) (any, error) {
			repos, err := ListRepos(ctx, g, s, in.Query, in.Refresh)
			if err != nil {
				return nil, err
			}
			page := mcptool.Paginate(repos, in.PageArgs)
			if !in.Installations {
				return page, nil
			}
			installs, err := ListInstallations(ctx, g, l)
			if err != nil {
				return nil, err
			}
			return repositoryListOut{Page: page, Installations: installs}, nil
		})
}

type repositoryScanIn struct {
	Owner string `json:"owner" jsonschema:"The repository's owner, for example acme."`
	Repo  string `json:"repo" jsonschema:"The repository's name, for example api."`
	Ref   string `json:"ref,omitempty" jsonschema:"The branch, tag, or commit to scan, for example main. Defaults to the repository's default branch."`
}

func repositoryScanTool(s Scanner, g Gate) mcptool.Tool {
	return mcptool.New("repository_scan", "Scan repository",
		"Reads a repository's file tree and proposes deployable candidates: one per compose file and one per "+
			"standalone Dockerfile, each with its services, ports, and env keys, plus the env keys of any "+
			".env.example. Pass a candidate unchanged to stack_create to create a stack from it. Reads the git "+
			"provider only; nothing is created.",
		mcptool.Hints{ReadOnly: true},
		func(ctx context.Context, in repositoryScanIn) (any, error) {
			out, err := Scan(ctx, g, s, in.Owner, in.Repo, in.Ref)
			if errors.Is(err, apperrors.ErrNotFound) {
				return nil, fmt.Errorf("%w; repository_list lists the repositories Nexul can read", err)
			}
			if err != nil {
				return nil, err
			}
			return out, nil
		})
}
