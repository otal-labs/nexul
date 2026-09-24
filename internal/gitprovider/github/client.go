// Package github implements gitprovider.GitProvider against GitHub via go-github.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	githubapi "github.com/google/go-github/v71/github"

	"github.com/otal-labs/nexul/internal/gitprovider"
	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
)

// Client is the go-github-backed GitProvider implementation.
type Client struct {
	gh *githubapi.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL points the client at a different API root (tests, GHE, Gitea).
func WithBaseURL(u *url.URL) Option {
	return func(c *Client) {
		if u == nil {
			return
		}
		base := *u
		if !strings.HasSuffix(base.Path, "/") {
			base.Path += "/"
		}
		c.gh.BaseURL = &base
	}
}

// New builds a GitHub client; an empty token sends no auth header, giving unconfigured workspaces public access.
func New(token string, opts ...Option) *Client {
	gh := githubapi.NewClient(nil)
	if token != "" {
		gh = gh.WithAuthToken(token)
	}
	c := &Client{gh: gh}
	for _, o := range opts {
		o(c)
	}
	return c
}

// GetRepo implements gitprovider.GitProvider.
func (c *Client) GetRepo(ctx context.Context, owner, name string) (*gitprovider.Repo, error) {
	r, _, err := c.gh.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("get repo %s/%s: %w", owner, name, mapErr(err))
	}
	return toRepo(r), nil
}

// ListPRs implements gitprovider.GitProvider.
func (c *Client) ListPRs(ctx context.Context, owner, name string, opts gitprovider.PROpts) ([]*gitprovider.PR, error) {
	if opts.State == "" {
		opts.State = "open"
	}
	listOpts := &githubapi.PullRequestListOptions{State: opts.State}
	if opts.Limit > 0 {
		listOpts.PerPage = opts.Limit
	}
	prs, _, err := c.gh.PullRequests.List(ctx, owner, name, listOpts)
	if err != nil {
		return nil, fmt.Errorf("list PRs %s/%s: %w", owner, name, mapErr(err))
	}
	out := make([]*gitprovider.PR, 0, len(prs))
	for _, p := range prs {
		out = append(out, toPR(p))
	}
	return out, nil
}

// GetPR implements gitprovider.GitProvider.
func (c *Client) GetPR(ctx context.Context, owner, name string, number int) (*gitprovider.PR, error) {
	p, _, err := c.gh.PullRequests.Get(ctx, owner, name, number)
	if err != nil {
		return nil, fmt.Errorf("get PR %s/%s#%d: %w", owner, name, number, mapErr(err))
	}
	return toPR(p), nil
}

// PRsForCommit implements gitprovider.GitProvider.
func (c *Client) PRsForCommit(ctx context.Context, owner, name, sha string) ([]*gitprovider.PR, error) {
	prs, _, err := c.gh.PullRequests.ListPullRequestsWithCommit(ctx, owner, name, sha, nil)
	if err != nil {
		return nil, fmt.Errorf("list PRs for commit %s in %s/%s: %w", sha, owner, name, mapErr(err))
	}
	out := make([]*gitprovider.PR, 0, len(prs))
	for _, p := range prs {
		out = append(out, toPR(p))
	}
	return out, nil
}

// CreateWebhook implements gitprovider.GitProvider, returning the hook id.
func (c *Client) CreateWebhook(ctx context.Context, owner, name string, cfg gitprovider.WebhookConfig) (string, error) {
	hook, _, err := c.gh.Repositories.CreateHook(ctx, owner, name, &githubapi.Hook{
		Events: []string{"pull_request"},
		Config: &githubapi.HookConfig{
			URL:         &cfg.URL,
			ContentType: githubapi.Ptr("json"),
			Secret:      &cfg.Secret,
		},
	})
	if err != nil {
		return "", fmt.Errorf("create webhook %s/%s: %w", owner, name, mapErr(err))
	}
	return strconv.FormatInt(hook.GetID(), 10), nil
}

// DeleteWebhook implements gitprovider.GitProvider.
func (c *Client) DeleteWebhook(ctx context.Context, owner, name, hookID string) error {
	id, err := strconv.ParseInt(hookID, 10, 64)
	if err != nil {
		return fmt.Errorf("delete webhook %s/%s: %w: hook id %q is not an integer", owner, name, apperrors.ErrInvalid, hookID)
	}
	if _, err := c.gh.Repositories.DeleteHook(ctx, owner, name, id); err != nil {
		return fmt.Errorf("delete webhook %s/%s: %w", owner, name, mapErr(err))
	}
	return nil
}

// ListInstallationRepos implements gitprovider.GitProvider: every repository across every App installation the
// connected user's token grants, fetched a page of installations at a time, a page of repos at a time.
func (c *Client) ListInstallationRepos(ctx context.Context) ([]*gitprovider.Repo, error) {
	var out []*gitprovider.Repo
	opts := &githubapi.ListOptions{PerPage: 100}
	for {
		installs, resp, err := c.gh.Apps.ListUserInstallations(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("list installations: %w", mapErr(err))
		}
		for _, inst := range installs {
			repos, err := c.listInstallationRepos(ctx, inst.GetID())
			if err != nil {
				return nil, fmt.Errorf("list repos for installation %d: %w", inst.GetID(), err)
			}
			out = append(out, repos...)
		}
		if resp == nil || resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

// installationReposResponse is GitHub's paginated body for GET /user/installations/{id}/repositories.
type installationReposResponse struct {
	Repositories []*githubapi.Repository `json:"repositories"`
}

// listInstallationRepos paginates one installation's repository list; go-github has no typed helper for this
// user-token endpoint (only the app-token app/installations/{id}/repositories variant), so the request is raw.
func (c *Client) listInstallationRepos(ctx context.Context, installID int64) ([]*gitprovider.Repo, error) {
	var out []*gitprovider.Repo
	page := 1
	for {
		u := fmt.Sprintf("user/installations/%d/repositories?per_page=100&page=%d", installID, page)
		req, err := c.gh.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		var body installationReposResponse
		resp, err := c.gh.Do(ctx, req, &body)
		if err != nil {
			return nil, mapErr(err)
		}
		for _, r := range body.Repositories {
			out = append(out, toRepo(r))
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		page = resp.NextPage
	}
}

// GetTree implements gitprovider.GitProvider: the recursive tree at ref, resolving ref to the repo's default
// branch via GetRepo when empty.
func (c *Client) GetTree(ctx context.Context, owner, name, ref string) ([]gitprovider.TreeEntry, error) {
	ref, err := c.resolveRef(ctx, owner, name, ref)
	if err != nil {
		return nil, err
	}
	tree, _, err := c.gh.Git.GetTree(ctx, owner, name, ref, true)
	if err != nil {
		return nil, fmt.Errorf("get tree %s/%s@%s: %w", owner, name, ref, mapErr(err))
	}
	out := make([]gitprovider.TreeEntry, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		out = append(out, gitprovider.TreeEntry{Path: e.GetPath(), Type: e.GetType()})
	}
	return out, nil
}

// GetFile implements gitprovider.GitProvider: a file's decoded content at ref, resolving ref to the repo's
// default branch via GetRepo when empty.
func (c *Client) GetFile(ctx context.Context, owner, name, ref, filePath string) ([]byte, error) {
	ref, err := c.resolveRef(ctx, owner, name, ref)
	if err != nil {
		return nil, err
	}
	file, dir, _, err := c.gh.Repositories.GetContents(ctx, owner, name, filePath, &githubapi.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return nil, fmt.Errorf("get file %s/%s@%s:%s: %w", owner, name, ref, filePath, mapErr(err))
	}
	if file == nil {
		return nil, fmt.Errorf("get file %s/%s@%s:%s: %w: path is a directory (%d entries)", owner, name, ref, filePath, apperrors.ErrInvalid, len(dir))
	}
	content, err := file.GetContent()
	if err != nil {
		return nil, fmt.Errorf("decode file %s/%s@%s:%s: %w", owner, name, ref, filePath, err)
	}
	return []byte(content), nil
}

// resolveRef returns ref as-is, or the repo's default branch when ref is empty.
func (c *Client) resolveRef(ctx context.Context, owner, name, ref string) (string, error) {
	if ref != "" {
		return ref, nil
	}
	repo, err := c.GetRepo(ctx, owner, name)
	if err != nil {
		return "", fmt.Errorf("resolve default branch for %s/%s: %w", owner, name, err)
	}
	return repo.DefaultBranch, nil
}

func toRepo(r *githubapi.Repository) *gitprovider.Repo {
	return &gitprovider.Repo{
		ID:            r.GetID(),
		Owner:         r.GetOwner().GetLogin(),
		Name:          r.GetName(),
		FullName:      r.GetFullName(),
		DefaultBranch: r.GetDefaultBranch(),
		HTMLURL:       r.GetHTMLURL(),
	}
}

func toPR(p *githubapi.PullRequest) *gitprovider.PR {
	return &gitprovider.PR{
		Number:          p.GetNumber(),
		Title:           p.GetTitle(),
		Body:            p.GetBody(),
		State:           gitprovider.PRState(p.GetState()),
		HeadSHA:         p.GetHead().GetSHA(),
		BaseBranch:      p.GetBase().GetRef(),
		Author:          p.GetUser().GetLogin(),
		LinkedTicketIDs: gitprovider.LinkedTicketIDs(p.GetBody(), p.GetHead().GetRef()),
	}
}

// mapErr translates go-github errors into the platform sentinels so adapters can
// map them to HTTP/MCP codes. Network-level errors are treated as retryable.
func mapErr(err error) error {
	var ghe *githubapi.ErrorResponse
	if !errors.As(err, &ghe) {
		return apperrors.Retryable(err)
	}
	switch ghe.Response.StatusCode {
	case http.StatusNotFound:
		return apperrors.ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return apperrors.ErrUnauthorized
	case http.StatusBadRequest:
		return apperrors.ErrInvalid
	case http.StatusConflict, http.StatusUnprocessableEntity:
		return apperrors.ErrConflict
	}
	if ghe.Response.StatusCode >= 500 {
		return apperrors.Retryable(err)
	}
	return err
}
