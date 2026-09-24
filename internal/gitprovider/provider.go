package gitprovider

import "context"

// GitProvider is the seam to a git host; GitLab/Gitea add a package satisfying this interface.
type GitProvider interface {
	GetRepo(ctx context.Context, owner, name string) (*Repo, error)
	ListPRs(ctx context.Context, owner, name string, opts PROpts) ([]*PR, error)
	GetPR(ctx context.Context, owner, name string, number int) (*PR, error)
	// PRsForCommit lists the pull requests with a commit: the merged one that introduced it, else the open ones carrying it.
	PRsForCommit(ctx context.Context, owner, name, sha string) ([]*PR, error)
	CreateWebhook(ctx context.Context, owner, name string, cfg WebhookConfig) (string, error)
	DeleteWebhook(ctx context.Context, owner, name, hookID string) error
	// ListInstallationRepos lists every repository the connected user's App installations grant.
	ListInstallationRepos(ctx context.Context) ([]*Repo, error)
	// GetTree returns the recursive git tree at ref; an empty ref resolves to the repo's default branch.
	GetTree(ctx context.Context, owner, name, ref string) ([]TreeEntry, error)
	// GetFile returns a file's decoded content at ref; an empty ref resolves to the repo's default branch.
	GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error)
}
