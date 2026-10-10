package gitprovider

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
