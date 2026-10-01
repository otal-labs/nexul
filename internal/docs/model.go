package docs

import "time"

type Doc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
	// ProjectID is the project this doc belongs to; mandatory, mirrors Ticket.ProjectID (ADR 0025).
	ProjectID string `json:"project_id"`
	// FolderID is the project folder the doc lives in; every doc is in exactly one (ADR 0096).
	FolderID  string    `json:"folder_id"`
	Version   int       `json:"version"`
	Archived  bool      `json:"archived"`
	Locked    bool      `json:"locked"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DocVersion is one historical snapshot; each update appends a row while the docs table holds the current pointer.
type DocVersion struct {
	DocID     string    `json:"doc_id"`
	Version   int       `json:"version"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Name      string    `json:"name,omitempty"`
	AuthorID  string    `json:"author_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// DocListItem is the disclosure-aware list view: unopenable docs show title/status with can_open=false, no body.
type DocListItem struct {
	ID string `json:"id"`
	// ProjectID lets the frontend group/filter the list by project (ticket 10), matching Ticket.ProjectID.
	ProjectID string    `json:"project_id"`
	FolderID  string    `json:"folder_id"`
	Title     string    `json:"title"`
	Version   int       `json:"version"`
	Archived  bool      `json:"archived"`
	Locked    bool      `json:"locked"`
	CanOpen   bool      `json:"can_open"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Filled only when CanOpen, so a doc the caller can't open discloses its title alone.
	CreatedBy string `json:"created_by,omitempty"`
	Snippet   string `json:"snippet,omitempty"`
}

// Folder groups a project's docs one level deep; each project has one default folder, which is never deleted.
type Folder struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WatcherSource says how someone came to watch a doc: added for creating or editing it, or by choosing to.
type WatcherSource string

const (
	WatcherAuto   WatcherSource = "auto"
	WatcherManual WatcherSource = "manual"
)

// Watcher is a person who gets a doc's change notifications (ADR 0101).
type Watcher struct {
	UserID    string        `json:"user_id"`
	Source    WatcherSource `json:"source"`
	CreatedAt time.Time     `json:"created_at"`
}

// Watchers is a doc's watchers as one person sees them, with whether that person is one.
type Watchers struct {
	Watchers []*Watcher `json:"watchers"`
	Watching bool       `json:"watching"`
}
