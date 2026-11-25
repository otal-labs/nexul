package imports

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Source is one tool people move from. It only reads and describes; the framework maps people, statuses and files,
// writes the records, skips what came in before, and resumes after a restart (ADR 0201).
type Source interface {
	// Scan counts what Import would bring and names the people and statuses it would map, without writing anything.
	// A scan that pauses on an error may return what it found so far with Cursor set; it resumes from job.Cursor,
	// adding to job.Scanned.
	Scan(ctx context.Context, job Job) (Scan, error)
	// Import reads from job.Cursor onward, handing each record to sink in a stable order, and returns once the source
	// is exhausted. A record handed again after a restart is skipped by the sink, so Import never tracks what it sent.
	Import(ctx context.Context, job Job, sink Sink) error
	Info() Info
}

// Info describes a source to the framework.
type Info struct {
	// Title is how people name the tool, such as GitHub.
	Title string
	// FileHosts are the hosts files may be downloaded from, redirects included; a file anywhere else stays a link.
	FileHosts []string
	// FileHeader, when set, signs a download from the source's own API with its credential; nil for any other URL,
	// and a redirect to another domain drops it, so the credential only ever goes back to the service that issued it.
	FileHeader func(credential string, u *url.URL) http.Header
}

// Job is what a source reads: the import's project, what in the source to read, the credential, and where to resume.
type Job struct {
	ImportID    string
	WorkspaceID string
	ProjectID   string
	// From is what in the source to read, such as owner/name; a source may fill it in during Scan.
	From       string
	Credential string
	// Cursor is the last Checkpoint, or where a paused scan stopped; empty on a first run.
	Cursor string
	// Scanned is what a paused scan found before Cursor.
	Scanned Scan
}

// Scan is what a source found.
type Scan struct {
	// From is what the scan read, resolved when the import named none.
	From     string
	Tickets  int
	Comments int
	Docs     int
	// RelationsSkipped counts links between records an import has no place for, such as Linear's related or duplicate.
	RelationsSkipped int
	// Archived are records the source keeps archived, counted but not imported.
	Archived int
	People   []Person
	Statuses []StatusRef
	// Cursor is where a scan that paused resumes reading; everything before it is counted above.
	Cursor string
}

// Sink takes the records a source reads.
type Sink interface {
	// Ticket files one ticket with its comments and files, or skips it when an import already brought it in.
	Ticket(ctx context.Context, t Ticket) error
	// Doc writes one doc with its files, or skips it when an import already brought it in.
	Doc(ctx context.Context, d Doc) error
	// DocLink is the in-app link of the doc a source record becomes, so a body can link a doc not written yet.
	DocLink(sourceID string) string
	// Skip reports whether a record of kind is already in, counting it as skipped, so a source need not read it again.
	Skip(ctx context.Context, kind, sourceID string) (bool, error)
	// Fail records a record the source could not read, with why, and carries on.
	Fail(ctx context.Context, kind, sourceID, title string, cause error) error
	// Checkpoint records where a resumed import starts reading again; everything before it has gone to the sink.
	Checkpoint(ctx context.Context, cursor string) error
}

// Person is someone in the source, matched to a member by Login (their GitHub login) or Email, or, for a source that
// gives neither, by Username (a member's login) and then Name.
type Person struct {
	Login    string `json:"login,omitempty"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Name     string `json:"name"`
}

// Key identifies the person within one import.
func (p Person) Key() string {
	if p.Login != "" {
		return "login:" + strings.ToLower(p.Login)
	}
	if p.Email != "" {
		return "email:" + strings.ToLower(p.Email)
	}
	return "username:" + strings.ToLower(p.Username)
}

// Zero reports whether the person names nobody.
func (p Person) Zero() bool { return p.Login == "" && p.Email == "" && p.Username == "" }

// StatusKind is the stage a source status maps to, one of the board's column kinds.
type StatusKind string

const (
	StatusBacklog  StatusKind = "backlog"
	StatusProgress StatusKind = "progress"
	StatusReview   StatusKind = "review"
	StatusTesting  StatusKind = "testing"
	StatusDone     StatusKind = "done"
)

// StatusRef is a source status: a column of the same name, ignoring case, or else a new column of Kind. An empty
// Name is the project's first column of Kind.
type StatusRef struct {
	Name string     `json:"name"`
	Kind StatusKind `json:"kind" enum:"backlog,progress,review,testing,done"`
}

// Ticket is one record that becomes a ticket.
type Ticket struct {
	// SourceID is stable in the source, so a later import of it is skipped.
	SourceID string
	Title    string
	// Body is markdown; links to files the source hosts are copied and pointed at the copies.
	Body     string
	Status   StatusRef
	Labels   []string
	Assignee Person
	// Category is the name of the project category it goes in, added the first time the project lacks it.
	Category string
	// DueOn is its due date as YYYY-MM-DD.
	DueOn string
	// BlockedBy and Blocks are the source ids of records it waits on and that wait on it; a link is made once both
	// ends are in, whichever comes second.
	BlockedBy []string
	Blocks    []string
	// Parent is the source id of the ticket this one is a subtask of, handed before it; the body names its key.
	Parent   string
	Comments []Comment
	// Files are the file URLs in Body and the comments to copy into attachments.
	Files []string
}

// Comment is one comment on a ticket, copied into its thread under the author's name there.
type Comment struct {
	SourceID  string
	Body      string
	Author    Person
	AvatarURL string
	At        time.Time
}

// Doc is one record that becomes a doc.
type Doc struct {
	// SourceID is stable in the source, so a later import of it is skipped.
	SourceID string
	// Folder names the doc folder it lands in, made the first time the project lacks one of that name.
	Folder string
	Title  string
	// Body is markdown; links to files the source hosts are copied and pointed at the copies.
	Body  string
	Files []string
	// Unsupported counts the source blocks the body carries as text or links.
	Unsupported int
}
