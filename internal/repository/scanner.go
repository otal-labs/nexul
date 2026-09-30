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

// InstallationLister is the seam over the git provider's installation read, kept apart from Scanner so each
// interface stays small (go.md §6).
type InstallationLister interface {
	// ListInstallations lists the App installations the connector's user can see.
	ListInstallations(ctx context.Context) ([]Installation, error)
}
