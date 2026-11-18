package gitprovider

import "time"

// Repo is a git repository as seen through a provider.
type Repo struct {
	ID            int64  `json:"id"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	// AccountID is the id of the account whose installation listed the repository, set by installation reads.
	AccountID int64 `json:"-"`
}

// Installation is one account or organisation the git host's App is installed on, as the connected user sees it.
type Installation struct {
	ID int64
	// AccountID is the git host's numeric id of the account, which survives a rename.
	AccountID    int64
	AccountLogin string
	// AccountType is "user" or "organization".
	AccountType      string
	AccountAvatarURL string
	// RepositorySelection is "all" or "selected".
	RepositorySelection string
	// RepositoryCount is set only for a "selected" installation; "all" follows the account's repositories.
	RepositoryCount *int
	// HTMLURL is the installation's settings page on the git host.
	HTMLURL string
	// Problem says why the git host refused to read this installation; the others read on.
	Problem string
	// MissingPermissions names what the App asks for that this installation has not accepted, such as "issues".
	MissingPermissions []string
}

// PRState is the lifecycle state of a pull request.
type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateClosed PRState = "closed"
)

// PR is a provider-neutral pull request.
type PR struct {
	Number          int      `json:"number"`
	Title           string   `json:"title"`
	Body            string   `json:"body"`
	State           PRState  `json:"state"`
	Merged          bool     `json:"merged"`
	HeadSHA         string   `json:"head_sha"`
	BaseBranch      string   `json:"base_branch"`
	Author          string   `json:"author"`
	LinkedTicketIDs []string `json:"linked_ticket_ids"`
}

// PROpts filters ListPRs results.
type PROpts struct {
	State string
	Limit int
}

// WebhookConfig describes a repository webhook to create.
type WebhookConfig struct {
	URL    string
	Secret string
}

// Webhook is a webhook already registered on a repository.
type Webhook struct {
	ID  string
	URL string
	// Stale is true when the hook's events differ from the ones the receiver dispatches; UpdateWebhook fixes them.
	Stale bool
}

// PRRef identifies a pull request for ticket linking.
type PRRef struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	SHA    string `json:"sha"`
}

// TreeEntry is one entry of a recursive git tree: a file (blob), directory (tree), or submodule (commit).
type TreeEntry struct {
	Path string
	Type string
}

// Issue is a provider-neutral issue, never a pull request; it is also the git.issue payload's issue.
type Issue struct {
	ID          int64     `json:"id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	State       string    `json:"state" enum:"open,closed"`
	StateReason string    `json:"state_reason" jsonschema:"Why it was closed or reopened (completed, not_planned, duplicate, reopened); empty when the host gives none."`
	Labels      []string  `json:"labels" jsonschema:"Label names."`
	Assignees   []string  `json:"assignees" jsonschema:"Assignee logins."`
	Author      string    `json:"author"`
	HTMLURL     string    `json:"html_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IssueComment is a comment on an issue; it is also the git.issue_comment payload's comment.
type IssueComment struct {
	ID              int64     `json:"id"`
	Body            string    `json:"body"`
	Author          string    `json:"author"`
	AuthorAvatarURL string    `json:"author_avatar_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// IssueOpts filters ListIssues; Page starts at 1, and 0 reads the first page.
type IssueOpts struct {
	// State is open, closed or all; empty reads open issues.
	State  string
	Labels []string
	// Since keeps only issues updated at or after it; zero keeps all.
	Since time.Time
	Page  int
}

// IssueInput is a new issue.
type IssueInput struct {
	Title     string
	Body      string
	Labels    []string
	Assignees []string
}

// IssuePatch changes an issue; a nil field keeps its value, and an empty list clears labels or assignees.
type IssuePatch struct {
	Title       *string
	Body        *string
	State       *string
	StateReason *string
	Labels      *[]string
	Assignees   *[]string
}

// Rate is what is left of a token's core rate limit, and when it resets.
type Rate struct {
	Remaining int
	Reset     time.Time
}
