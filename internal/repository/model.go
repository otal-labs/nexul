// Package repository reads a git repository's tree and proposes deployable candidates: compose
// stacks and standalone Dockerfiles, with their services, ports and env keys. It never imports gitprovider or
// deploy directly (ADR 0017) — server/cmd adapts a connector-backed provider to the Scanner seam below.
package repository

import (
	"strings"
	"time"
)

// Repo is a git repository visible through a connector's installation.
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
	// Workspaces see this installation's repositories; none means it is unassigned.
	Workspaces []InstallationWorkspace `json:"workspaces"`
	// Problem says why GitHub refused to read this installation, a suspension for one; the others read on.
	Problem string `json:"problem,omitempty"`
	// Gone is an assignment whose installation GitHub no longer lists: the App was uninstalled from the account.
	Gone bool `json:"gone,omitempty"`
}

// Assignment is one workspace an installation's account is assigned to; AccountID is 0 until its id is recorded.
type Assignment struct {
	AccountID     int64
	AccountLogin  string
	WorkspaceID   string
	WorkspaceName string
	Gone          bool
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

// InstallState is what an install link's state stands for: the workspace and the person who asked for the link.
type InstallState struct {
	WorkspaceID string
	UserID      string
	ExpiresAt   time.Time
}

// Installer is who GitHub returned from an install link, as their own token sees the installation.
type Installer struct {
	AccountID    int64
	AccountLogin string
	// Admin is true when the installer is the account itself or an admin of the organisation; false when unverifiable.
	Admin bool
}

// InstallationWorkspace is one workspace an installation is assigned to.
type InstallationWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
