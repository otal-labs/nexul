package gitprovider

import (
	"context"
	"time"
)

// GitProvider is the seam to a git host; GitLab/Gitea add a package satisfying this interface.
type GitProvider interface {
	GetRepo(ctx context.Context, owner, name string) (*Repo, error)
	ListPRs(ctx context.Context, owner, name string, opts PROpts) ([]*PR, error)
	GetPR(ctx context.Context, owner, name string, number int) (*PR, error)
	// PRsForCommit lists the pull requests with a commit: the merged one that introduced it, else the open ones carrying it.
	PRsForCommit(ctx context.Context, owner, name, sha string) ([]*PR, error)
	// MergedInto reports whether merged pull request number's merge commit is sha or one of sha's ancestors.
	MergedInto(ctx context.Context, owner, name string, number int, sha string) (bool, error)
	CreateWebhook(ctx context.Context, owner, name string, cfg WebhookConfig) (string, error)
	ListWebhooks(ctx context.Context, owner, name string) ([]Webhook, error)
	// UpdateWebhook sets the hook's events to the ones the receiver dispatches, what a Stale hook lacks.
	UpdateWebhook(ctx context.Context, owner, name, hookID string) error
	DeleteWebhook(ctx context.Context, owner, name, hookID string) error
	// GetTree returns the recursive git tree at ref; an empty ref resolves to the repo's default branch.
	GetTree(ctx context.Context, owner, name, ref string) ([]TreeEntry, error)
	// GetFile returns a file's decoded content at ref; an empty ref resolves to the repo's default branch.
	GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error)
	// ListIssues returns one page of issues, pull requests left out, and the next page number, 0 after the last.
	ListIssues(ctx context.Context, owner, name string, opts IssueOpts) ([]*Issue, int, error)
	GetIssue(ctx context.Context, owner, name string, number int) (*Issue, error)
	CreateIssue(ctx context.Context, owner, name string, in IssueInput) (*Issue, error)
	UpdateIssue(ctx context.Context, owner, name string, number int, patch IssuePatch) (*Issue, error)
	// ListIssueComments returns every comment on issue number updated at or after since; a zero since returns them all.
	ListIssueComments(ctx context.Context, owner, name string, number int, since time.Time) ([]*IssueComment, error)
	CreateIssueComment(ctx context.Context, owner, name string, number int, body string) (*IssueComment, error)
	UpdateIssueComment(ctx context.Context, owner, name string, commentID int64, body string) (*IssueComment, error)
	DeleteIssueComment(ctx context.Context, owner, name string, commentID int64) error
}

// RateReader is a GitProvider that can say how much of its rate limit is left, which reading costs none of it.
type RateReader interface {
	RateLimit(ctx context.Context) (Rate, error)
}
