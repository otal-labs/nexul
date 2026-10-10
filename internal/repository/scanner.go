package repository

import "context"

// Scanner is the seam over a git provider's three read operations Scan needs; defined at the consumer
// side (go.md §6) so this domain never imports gitprovider — server/cmd adapts a connector-backed provider to it.
type Scanner interface {
	// ListInstallationRepos lists every repository the connected App installation grants. An implementation may
	// serve a recent answer; refresh asks for a fresh one.
	ListInstallationRepos(ctx context.Context, refresh bool) ([]Repo, error)
	// GetTree returns the recursive tree at ref, plus the branch actually used — ref resolves to the repo's
	// default branch when empty, and Scan needs that resolved branch to fill in ScanResult.DefaultBranch.
	GetTree(ctx context.Context, owner, name, ref string) (resolvedRef string, entries []TreeEntry, err error)
	// GetFile returns a file's content at ref.
	GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error)
}

// InstallationLister is the seam over the git provider's installations, apart from Scanner to keep both small.
type InstallationLister interface {
	// ListInstallations lists every installation read as the App, else those the connector's user can see.
	ListInstallations(ctx context.Context) ([]Installation, error)
	// ReadsAsApp reports whether a private key is set, so Nexul reads GitHub as its App (ADR 0144).
	ReadsAsApp(ctx context.Context) (bool, error)
	// InstallURL is GitHub's page for installing the App on another account, or "" while no App slug is registered.
	InstallURL(ctx context.Context) (string, error)
}

// AccountResolver names the GitHub account behind an installation or a repository by its numeric id.
type AccountResolver interface {
	// InstallationAccounts lists every installation with its account, without the reads ListInstallations adds.
	InstallationAccounts(ctx context.Context) ([]Installation, error)
	// AccountOf is the id of the account whose installation covers owner/name, ErrNotFound when none does.
	AccountOf(ctx context.Context, owner, name string) (int64, error)
}

// InstallerChecker confirms who GitHub returned from an install link, with the code GitHub sent along.
type InstallerChecker interface {
	// Installer is installationID's account as the installer's own token sees it, ErrForbidden if unseen.
	Installer(ctx context.Context, code string, installationID int64) (Installer, error)
}
