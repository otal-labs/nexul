// Package repository reads a git repository's tree and proposes deployable candidates: compose
// stacks and standalone Dockerfiles, with their services, ports and env keys. It never imports gitprovider or
// deploy directly (ADR 0017) — server/cmd adapts each person's own GitHub view to the GitHubView seam (ADR 0147).
package repository

import "strings"

// Repo is a git repository a person's own GitHub account can open where the App is installed.
type Repo struct {
	ID            int64  `json:"id"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	// Provider is the connector that listed the repository ("github"); the web picks its mark from it.
	Provider string `json:"provider"`
	// AccountID is GitHub's id of the account whose installation listed the repository; what assignment checks.
	AccountID int64 `json:"-"`
}

// Installation is one account or organisation the GitHub App is installed on, with the workspaces that list it.
type Installation struct {
	ID int64 `json:"id"`
	// AccountID is GitHub's numeric id of the account, which survives a rename; assignments are keyed by it.
	AccountID    int64  `json:"account_id"`
	AccountLogin string `json:"account_login"`
	// AccountType is "user" or "organization".
	AccountType      string `json:"account_type"`
	AccountAvatarURL string `json:"account_avatar_url"`
	// RepositorySelection is "all" or "selected".
	RepositorySelection string `json:"repository_selection"`
	// RepositoryCount is present only when RepositorySelection is "selected".
	RepositoryCount *int `json:"repository_count,omitempty"`
	// HTMLURL is where the installation's repository access is managed on GitHub.
	HTMLURL string `json:"html_url"`
	// Workspaces use this installation: a project there attaches one of its repositories (ADR 0147).
	Workspaces []InstallationWorkspace `json:"workspaces"`
	// Problem says why GitHub refused to read this installation, a suspension for one; the others read on.
	Problem string `json:"problem,omitempty"`
}

// Assignment links an installation's account to a workspace whose project attached one of its repositories, so
// background work there may read it as the App; AccountID is 0 until its id is recorded.
type Assignment struct {
	AccountID     int64
	AccountLogin  string
	WorkspaceID   string
	WorkspaceName string
	Gone          bool
	// Attached is set by AssignmentsIn when a project in the workspace still attaches one of the account's repositories.
	Attached bool
}

// sameAccount reports whether a is an assignment of the account with id and login, by login only while a has no id.
func (a Assignment) sameAccount(id int64, login string) bool {
	if a.AccountID != 0 {
		return a.AccountID == id
	}
	return a.AccountLogin == strings.ToLower(login)
}

// AccountSync is what reading the App's installations changes in the assignments.
type AccountSync struct {
	// Resolved records the id of each login assigned before ids were kept.
	Resolved map[string]int64
	// Renamed is each account's login as GitHub now names it.
	Renamed map[int64]string
	// Gone are the assignments whose installation GitHub no longer lists.
	Gone []Assignment
	// Reinstalled are accounts back on GitHub whose old, gone assignments are dropped, so a reinstall lands unassigned.
	Reinstalled []int64
}

func (s AccountSync) empty() bool {
	return len(s.Resolved) == 0 && len(s.Renamed) == 0 && len(s.Gone) == 0 && len(s.Reinstalled) == 0
}

// InstallationWorkspace is one workspace that uses an installation.
type InstallationWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// CanDetach is true where the viewer holds projects:write, what detaching the account there takes.
	CanDetach bool `json:"can_detach"`
}

// TreeEntry is one entry of a recursive git tree: a file (blob), directory (tree), or submodule (commit).
type TreeEntry struct {
	Path string
	Type string
}

// Kind distinguishes a compose-stack candidate from a standalone-Dockerfile candidate.
type Kind string

const (
	KindCompose    Kind = "compose"
	KindDockerfile Kind = "dockerfile"
)

// Build captures a service's build directive; nil when the service pulls Image instead of building.
type Build struct {
	Context    string `json:"context"`
	Dockerfile string `json:"dockerfile"`
}

// DeclaredService is one service a candidate declares — a compose service, or a standalone Dockerfile's own build.
type DeclaredService struct {
	Name    string   `json:"name"`
	Image   string   `json:"image,omitempty"`
	Build   *Build   `json:"build,omitempty"`
	Ports   []int    `json:"ports"`
	Expose  []int    `json:"expose"`
	EnvKeys []string `json:"env_keys"`
}

// Reachable names the first declared service with a published or exposed port, and which port.
type Reachable struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
}

// Candidate is one deployable stack proposed from the repository tree.
type Candidate struct {
	Kind      Kind              `json:"kind"`
	Path      string            `json:"path"`
	Name      string            `json:"name"`
	Services  []DeclaredService `json:"services"`
	Reachable *Reachable        `json:"reachable,omitempty"`
}

// ScanResult is what POST /api/repositories/scan returns: everything the stack-creation wizard needs.
type ScanResult struct {
	DefaultBranch string      `json:"default_branch"`
	Candidates    []Candidate `json:"candidates"`
	EnvKeys       []string    `json:"env_keys"`
}
