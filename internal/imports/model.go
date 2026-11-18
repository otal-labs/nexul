// Package imports brings another tool's records into a project: a scan, a confirmed job, live progress, and re-runs
// that skip what came in before (ADR 0201). Each tool is a Source; the framework owns every mapping.
package imports

import (
	"time"
)

// State is where an import is: scanning, ready to confirm, importing, or one of the three final states.
type State string

const (
	StateScanning  State = "scanning"
	StateReady     State = "ready"
	StateImporting State = "importing"
	StateCompleted State = "completed"
	StateFailed    State = "failed"
	StateCancelled State = "cancelled"
)

// Final reports whether the import has ended, which also means its credential is gone.
func (s State) Final() bool {
	return s == StateCompleted || s == StateFailed || s == StateCancelled
}

// Limits on one import.
const (
	MaxTickets = 20_000
	MaxDocs    = 5_000
	// MaxListedFailures caps the failures a report names; the count goes on past it.
	MaxListedFailures = 200
	// ReadyFor is how long a scanned import waits to be confirmed before it is cancelled.
	ReadyFor = 24 * time.Hour
)

// Item kinds an import records.
const (
	KindTicket  = "ticket"
	KindComment = "comment"
	KindDoc     = "doc"
)

// Import is one run of one source into one project.
type Import struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Source      string `json:"source"`
	// From is what in the source is imported, such as owner/name for a GitHub repository.
	From      string `json:"from"`
	State     State  `json:"state" enum:"scanning,ready,importing,completed,failed,cancelled"`
	CreatedBy string `json:"created_by"`
	// Credential is the source's secret while the import runs; never sent anywhere.
	Credential string    `json:"-"`
	Total      int       `json:"total"`
	Done       int       `json:"done"`
	Failed     int       `json:"failed"`
	Report     Report    `json:"report"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Report is what a scan found and what the import did, kept on the import.
type Report struct {
	// Tickets, Comments, Docs and Files are what the scan counted.
	Tickets  int `json:"tickets"`
	Comments int `json:"comments"`
	Docs     int `json:"docs"`
	// People are everyone the scan found, each with the member they map to, if any.
	People []PersonMatch `json:"people"`
	// Statuses are the source's statuses and the column each lands in.
	Statuses []StatusMatch `json:"statuses"`
	// Skipped are records an earlier import already brought in.
	Skipped int `json:"skipped"`
	// FilesCopied went into attachments; FilesLinked stayed links, too large or not downloadable.
	FilesCopied int `json:"files_copied"`
	FilesLinked int `json:"files_linked"`
	// Unsupported are source blocks with no counterpart here, brought in as text or links instead.
	Unsupported int `json:"unsupported"`
	// Failures names up to MaxListedFailures records that did not come in; Import.Failed counts them all.
	Failures []Failure `json:"failures"`
	// Error is why a failed import stopped.
	Error string `json:"error,omitempty"`
	// Cursor is where a resumed import starts reading the source again.
	Cursor string `json:"cursor,omitempty"`
}

// PersonMatch is one person from the source and the login of the member they map to; empty Member is unmatched.
type PersonMatch struct {
	Person
	Member string `json:"member"`
}

// StatusMatch is one source status and the column it lands in; New is a column the import adds.
type StatusMatch struct {
	StatusRef
	Column string `json:"column"`
	New    bool   `json:"new"`
}

// Failure is one record that did not come in, and why.
type Failure struct {
	Kind     string `json:"kind"`
	SourceID string `json:"source_id"`
	Title    string `json:"title,omitempty"`
	Error    string `json:"error"`
}

// Start is what begins an import: a scan of From in source, into projectID.
type Start struct {
	Source     string
	ProjectID  string
	From       string
	Credential string
}
