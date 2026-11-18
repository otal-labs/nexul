// Package github implements gitprovider.GitProvider against GitHub via go-github.
package github

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	githubapi "github.com/google/go-github/v71/github"

	"github.com/otal-labs/nexul/internal/gitprovider"
	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
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
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	r, _, err := c.gh.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("get repo %s/%s: %w", owner, name, mapErr(err))
	}
	return toRepo(r), nil
}

// ListPRs implements gitprovider.GitProvider.
func (c *Client) ListPRs(ctx context.Context, owner, name string, opts gitprovider.PROpts) ([]*gitprovider.PR, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
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
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	p, _, err := c.gh.PullRequests.Get(ctx, owner, name, number)
	if err != nil {
		return nil, fmt.Errorf("get PR %s/%s#%d: %w", owner, name, number, mapErr(err))
	}
	return toPR(p), nil
}

// PRsForCommit implements gitprovider.GitProvider.
func (c *Client) PRsForCommit(ctx context.Context, owner, name, sha string) ([]*gitprovider.PR, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
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

// MergedInto implements gitprovider.GitProvider through GitHub's compare of the pull request's merge commit with sha.
func (c *Client) MergedInto(ctx context.Context, owner, name string, number int, sha string) (bool, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return false, err
	}
	p, _, err := c.gh.PullRequests.Get(ctx, owner, name, number)
	if err != nil {
		return false, fmt.Errorf("get PR %s/%s#%d: %w", owner, name, number, mapErr(err))
	}
	merge := p.GetMergeCommitSHA()
	if p.GetMergedAt().IsZero() || merge == "" {
		return false, fmt.Errorf("%w: PR %s/%s#%d has not merged", apperrors.ErrConflict, owner, name, number)
	}
	cmp, _, err := c.gh.Repositories.CompareCommits(ctx, owner, name, merge, sha, &githubapi.ListOptions{PerPage: 1})
	if err != nil {
		return false, fmt.Errorf("compare %s...%s in %s/%s: %w", merge, sha, owner, name, mapErr(err))
	}
	status := cmp.GetStatus()
	return status == "identical" || status == "ahead", nil
}

// webhookEvents are the delivery types the /hooks/github receiver dispatches; push also carries branch deletes.
var webhookEvents = []string{"pull_request", "pull_request_review", "pull_request_review_comment", "issues", "issue_comment", "push"}

// staleEvents reports whether a hook's events differ from webhookEvents, in any order.
func staleEvents(events []string) bool {
	have, want := slices.Sorted(slices.Values(events)), slices.Sorted(slices.Values(webhookEvents))
	return !slices.Equal(have, want)
}

// CreateWebhook implements gitprovider.GitProvider, returning the hook id.
func (c *Client) CreateWebhook(ctx context.Context, owner, name string, cfg gitprovider.WebhookConfig) (string, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return "", err
	}
	hook, _, err := c.gh.Repositories.CreateHook(ctx, owner, name, &githubapi.Hook{
		Events: webhookEvents,
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

// ListWebhooks implements gitprovider.GitProvider.
func (c *Client) ListWebhooks(ctx context.Context, owner, name string) ([]gitprovider.Webhook, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	var out []gitprovider.Webhook
	opts := &githubapi.ListOptions{PerPage: 100}
	for {
		hooks, resp, err := c.gh.Repositories.ListHooks(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("list webhooks %s/%s: %w", owner, name, mapErr(err))
		}
		for _, h := range hooks {
			out = append(out, gitprovider.Webhook{ID: strconv.FormatInt(h.GetID(), 10), URL: h.GetConfig().GetURL(), Stale: staleEvents(h.Events)})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

// UpdateWebhook implements gitprovider.GitProvider, setting the hook's events to webhookEvents.
func (c *Client) UpdateWebhook(ctx context.Context, owner, name, hookID string) error {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return err
	}
	id, err := strconv.ParseInt(hookID, 10, 64)
	if err != nil {
		return fmt.Errorf("update webhook %s/%s: %w: hook id %q is not an integer", owner, name, apperrors.ErrInvalid, hookID)
	}
	if _, _, err := c.gh.Repositories.EditHook(ctx, owner, name, id, &githubapi.Hook{Events: webhookEvents}); err != nil {
		return fmt.Errorf("update webhook %s/%s: %w", owner, name, mapErr(err))
	}
	return nil
}

// DeleteWebhook implements gitprovider.GitProvider.
func (c *Client) DeleteWebhook(ctx context.Context, owner, name, hookID string) error {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return err
	}
	id, err := strconv.ParseInt(hookID, 10, 64)
	if err != nil {
		return fmt.Errorf("delete webhook %s/%s: %w: hook id %q is not an integer", owner, name, apperrors.ErrInvalid, hookID)
	}
	if _, err := c.gh.Repositories.DeleteHook(ctx, owner, name, id); err != nil {
		return fmt.Errorf("delete webhook %s/%s: %w", owner, name, mapErr(err))
	}
	return nil
}

// issuesPerPage is GitHub's largest page, so an import reads the fewest pages.
const issuesPerPage = 100

// ListIssues implements gitprovider.GitProvider: one page of issues, pull requests left out, and the next page, 0 after the last.
func (c *Client) ListIssues(ctx context.Context, owner, name string, opts gitprovider.IssueOpts) ([]*gitprovider.Issue, int, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, 0, err
	}
	listOpts := &githubapi.IssueListByRepoOptions{
		State:       opts.State,
		Labels:      opts.Labels,
		Since:       opts.Since,
		ListOptions: githubapi.ListOptions{Page: opts.Page, PerPage: issuesPerPage},
	}
	issues, resp, err := c.gh.Issues.ListByRepo(ctx, owner, name, listOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("list issues %s/%s: %w", owner, name, mapErr(err))
	}
	out := make([]*gitprovider.Issue, 0, len(issues))
	for _, i := range issues {
		if i.IsPullRequest() {
			continue
		}
		issue := gitprovider.IssueFromGitHub(i)
		out = append(out, &issue)
	}
	return out, resp.NextPage, nil
}

// RateLimit implements gitprovider.RateReader.
func (c *Client) RateLimit(ctx context.Context) (gitprovider.Rate, error) {
	limits, _, err := c.gh.RateLimit.Get(ctx)
	if err != nil {
		return gitprovider.Rate{}, fmt.Errorf("read rate limit: %w", mapErr(err))
	}
	core := limits.GetCore()
	return gitprovider.Rate{Remaining: core.Remaining, Reset: core.Reset.Time}, nil
}

// GetIssue implements gitprovider.GitProvider.
func (c *Client) GetIssue(ctx context.Context, owner, name string, number int) (*gitprovider.Issue, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	i, _, err := c.gh.Issues.Get(ctx, owner, name, number)
	if err != nil {
		return nil, fmt.Errorf("get issue %s/%s#%d: %w", owner, name, number, mapErr(err))
	}
	issue := gitprovider.IssueFromGitHub(i)
	return &issue, nil
}

// CreateIssue implements gitprovider.GitProvider.
func (c *Client) CreateIssue(ctx context.Context, owner, name string, in gitprovider.IssueInput) (*gitprovider.Issue, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	req := &githubapi.IssueRequest{Title: &in.Title, Body: &in.Body}
	if in.Labels != nil {
		req.Labels = &in.Labels
	}
	if in.Assignees != nil {
		req.Assignees = &in.Assignees
	}
	i, _, err := c.gh.Issues.Create(ctx, owner, name, req)
	if err != nil {
		return nil, fmt.Errorf("create issue in %s/%s: %w", owner, name, mapErr(err))
	}
	issue := gitprovider.IssueFromGitHub(i)
	return &issue, nil
}

// UpdateIssue implements gitprovider.GitProvider, sending only the fields the patch sets.
func (c *Client) UpdateIssue(ctx context.Context, owner, name string, number int, p gitprovider.IssuePatch) (*gitprovider.Issue, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	req := &githubapi.IssueRequest{Title: p.Title, Body: p.Body, State: p.State, StateReason: p.StateReason, Labels: p.Labels, Assignees: p.Assignees}
	i, _, err := c.gh.Issues.Edit(ctx, owner, name, number, req)
	if err != nil {
		return nil, fmt.Errorf("update issue %s/%s#%d: %w", owner, name, number, mapErr(err))
	}
	issue := gitprovider.IssueFromGitHub(i)
	return &issue, nil
}

// ListIssueComments implements gitprovider.GitProvider: every comment on issue number updated at or after since, all pages.
func (c *Client) ListIssueComments(ctx context.Context, owner, name string, number int, since time.Time) ([]*gitprovider.IssueComment, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	opts := &githubapi.IssueListCommentsOptions{ListOptions: githubapi.ListOptions{PerPage: issuesPerPage}}
	if !since.IsZero() {
		opts.Since = &since
	}
	var out []*gitprovider.IssueComment
	for {
		comments, resp, err := c.gh.Issues.ListComments(ctx, owner, name, number, opts)
		if err != nil {
			return nil, fmt.Errorf("list comments of %s/%s#%d: %w", owner, name, number, mapErr(err))
		}
		for _, cm := range comments {
			comment := gitprovider.IssueCommentFromGitHub(cm)
			out = append(out, &comment)
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

// CreateIssueComment implements gitprovider.GitProvider.
func (c *Client) CreateIssueComment(ctx context.Context, owner, name string, number int, body string) (*gitprovider.IssueComment, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	cm, _, err := c.gh.Issues.CreateComment(ctx, owner, name, number, &githubapi.IssueComment{Body: &body})
	if err != nil {
		return nil, fmt.Errorf("comment on %s/%s#%d: %w", owner, name, number, mapErr(err))
	}
	comment := gitprovider.IssueCommentFromGitHub(cm)
	return &comment, nil
}

// UpdateIssueComment implements gitprovider.GitProvider.
func (c *Client) UpdateIssueComment(ctx context.Context, owner, name string, commentID int64, body string) (*gitprovider.IssueComment, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	cm, _, err := c.gh.Issues.EditComment(ctx, owner, name, commentID, &githubapi.IssueComment{Body: &body})
	if err != nil {
		return nil, fmt.Errorf("update comment %d in %s/%s: %w", commentID, owner, name, mapErr(err))
	}
	comment := gitprovider.IssueCommentFromGitHub(cm)
	return &comment, nil
}

// DeleteIssueComment implements gitprovider.GitProvider.
func (c *Client) DeleteIssueComment(ctx context.Context, owner, name string, commentID int64) error {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return err
	}
	if _, err := c.gh.Issues.DeleteComment(ctx, owner, name, commentID); err != nil {
		return fmt.Errorf("delete comment %d in %s/%s: %w", commentID, owner, name, mapErr(err))
	}
	return nil
}

// ListInstallationRepos lists every repository across every App installation the user's token grants, fetched a page of installations at a time, a page of repos at a time.
func (c *Client) ListInstallationRepos(ctx context.Context) ([]*gitprovider.Repo, error) {
	var out []*gitprovider.Repo
	err := c.eachInstallation(ctx, func(inst *githubapi.Installation) error {
		repos, err := c.listInstallationRepos(ctx, inst.GetID())
		if err != nil {
			return fmt.Errorf("list repos for installation %d: %w", inst.GetID(), err)
		}
		for _, r := range repos {
			r.AccountID = inst.GetAccount().GetID()
		}
		out = append(out, repos...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListInstallations lists the App installations the user's token sees; only a "selected" installation costs a second request,
// a one-item page read for its total_count.
func (c *Client) ListInstallations(ctx context.Context) ([]*gitprovider.Installation, error) {
	var out []*gitprovider.Installation
	err := c.eachInstallation(ctx, func(inst *githubapi.Installation) error {
		i := toInstallation(inst)
		if i.RepositorySelection == "selected" {
			n, err := c.installationRepoCount(ctx, inst.GetID())
			if err != nil {
				return fmt.Errorf("count repos for installation %d: %w", inst.GetID(), err)
			}
			i.RepositoryCount = &n
		}
		out = append(out, i)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// eachInstallation walks every App installation the connected user's token can see, a page of 100 at a time.
func (c *Client) eachInstallation(ctx context.Context, fn func(*githubapi.Installation) error) error {
	opts := &githubapi.ListOptions{PerPage: 100}
	for {
		installs, resp, err := c.gh.Apps.ListUserInstallations(ctx, opts)
		if err != nil {
			return fmt.Errorf("list installations: %w", mapErr(err))
		}
		for _, inst := range installs {
			if err := fn(inst); err != nil {
				return err
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return nil
		}
		opts.Page = resp.NextPage
	}
}

// installationReposResponse is GitHub's paginated body for GET /user/installations/{id}/repositories.
type installationReposResponse struct {
	TotalCount   int                     `json:"total_count"`
	Repositories []*githubapi.Repository `json:"repositories"`
}

// installationRepoCount reads one installation's repository total from a one-item page instead of paging them all.
func (c *Client) installationRepoCount(ctx context.Context, installID int64) (int, error) {
	req, err := c.gh.NewRequest("GET", fmt.Sprintf("user/installations/%d/repositories?per_page=1", installID), nil)
	if err != nil {
		return 0, err
	}
	var body installationReposResponse
	if _, err := c.gh.Do(ctx, req, &body); err != nil {
		return 0, mapErr(err)
	}
	return body.TotalCount, nil
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
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
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
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
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

func toInstallation(inst *githubapi.Installation) *gitprovider.Installation {
	account := inst.GetAccount()
	return &gitprovider.Installation{
		ID:                  inst.GetID(),
		AccountID:           account.GetID(),
		AccountLogin:        account.GetLogin(),
		AccountType:         strings.ToLower(cmp.Or(account.GetType(), inst.GetTargetType())),
		AccountAvatarURL:    account.GetAvatarURL(),
		RepositorySelection: inst.GetRepositorySelection(),
		HTMLURL:             inst.GetHTMLURL(),
		MissingPermissions:  missingPermissions(inst.GetPermissions()),
	}
}

// missingPermissions names the permissions the App asks for that an installation has not accepted yet.
func missingPermissions(p *githubapi.InstallationPermissions) []string {
	missing := []string{}
	if p.GetIssues() != "write" {
		missing = append(missing, "issues")
	}
	return missing
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
		Number:     p.GetNumber(),
		Title:      p.GetTitle(),
		Body:       p.GetBody(),
		State:      gitprovider.PRState(p.GetState()),
		Merged:     !p.GetMergedAt().IsZero(),
		HeadSHA:    p.GetHead().GetSHA(),
		BaseBranch: p.GetBase().GetRef(),
		Author:     p.GetUser().GetLogin(),
	}
}

// mapErr translates go-github errors into the platform sentinels so adapters can
// map them to HTTP/MCP codes. Network-level errors and rate limits are treated as retryable.
func mapErr(err error) error {
	if wait, ok := rateLimitWait(err, time.Now()); ok {
		return &apperrors.RetryAfterError{Wait: wait, Err: err}
	}
	var ghe *githubapi.ErrorResponse
	if !errors.As(err, &ghe) {
		return apperrors.Retryable(err)
	}
	switch ghe.Response.StatusCode {
	case http.StatusNotFound, http.StatusGone:
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

// rateLimitWait reads a 403 or 429 carrying X-RateLimit-Remaining: 0 or Retry-After as a rate limit, and how long it lasts:
// Retry-After's seconds, else until X-RateLimit-Reset.
func rateLimitWait(err error, now time.Time) (time.Duration, bool) {
	var rle *githubapi.RateLimitError
	if errors.As(err, &rle) {
		return max(rle.Rate.Reset.Sub(now), 0), true
	}
	var abuse *githubapi.AbuseRateLimitError
	if errors.As(err, &abuse) {
		return max(abuse.GetRetryAfter(), 0), true
	}
	var ghe *githubapi.ErrorResponse
	if !errors.As(err, &ghe) || ghe.Response == nil {
		return 0, false
	}
	resp := ghe.Response
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return 0, false
	}
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" && resp.Header.Get("X-RateLimit-Remaining") != "0" {
		return 0, false
	}
	if secs, perr := strconv.Atoi(retryAfter); perr == nil {
		return max(time.Duration(secs)*time.Second, 0), true
	}
	reset, perr := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
	if perr != nil {
		return 0, true
	}
	return max(time.Unix(reset, 0).Sub(now), 0), true
}
