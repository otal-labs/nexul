package repository

import "context"

// Scanner reads one repository's tree and files, defined at the consumer side (go.md §6) so this domain never imports
// gitprovider.
type Scanner interface {
	// GetTree returns the recursive tree at ref, plus the branch actually used — ref resolves to the repo's
	// default branch when empty, and Scan needs that resolved branch to fill in ScanResult.DefaultBranch.
	GetTree(ctx context.Context, owner, name, ref string) (resolvedRef string, entries []TreeEntry, err error)
	// GetFile returns a file's content at ref.
	GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error)
}

// GitHubView is GitHub as one person's own token shows it (ADR 0147): the repositories they can open where the App
// is installed, and nothing else.
type GitHubView interface {
	Scanner
	// Repos lists every such repository. An implementation may serve a recent answer; refresh asks for a fresh one.
	Repos(ctx context.Context, refresh bool) ([]Repo, error)
	// Installations lists the App's installations the person can see.
	Installations(ctx context.Context) ([]Installation, error)
}

// People opens one person's GitHub view; a person who has not connected GitHub gets an error saying how to.
type People interface {
	GitHubView(ctx context.Context, userID string) (GitHubView, error)
}

// InstallationLister is how Nexul reads GitHub and where the App is installed from.
type InstallationLister interface {
	// ReadsAsApp reports whether a private key is set, so background work reads GitHub as its App (ADR 0144).
	ReadsAsApp(ctx context.Context) (bool, error)
	// InstallURL is GitHub's page for installing the App on another account, or "" while no App slug is registered.
	InstallURL(ctx context.Context) (string, error)
}

// AccountResolver names the GitHub account behind an installation or a repository by its numeric id, as the App.
type AccountResolver interface {
	// InstallationAccounts lists every installation of the App with its account.
	InstallationAccounts(ctx context.Context) ([]Installation, error)
	// AccountOf is the id of the account whose installation covers owner/name, ErrNotFound when none does.
	AccountOf(ctx context.Context, owner, name string) (int64, error)
}
