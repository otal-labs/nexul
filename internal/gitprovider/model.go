package gitprovider

// Repo is a git repository as seen through a provider.
type Repo struct {
	ID            int64  `json:"id"`
	Owner         string `json:"owner"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
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
