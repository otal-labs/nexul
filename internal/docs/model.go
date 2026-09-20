package docs

import "time"

type Doc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
	// ProjectID is the project this doc belongs to; mandatory, mirrors Ticket.ProjectID (ADR 0025).
	ProjectID string    `json:"project_id"`
	Version   int       `json:"version"`
	Archived  bool      `json:"archived"`
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
	Title     string    `json:"title"`
	Version   int       `json:"version"`
	Archived  bool      `json:"archived"`
	CanOpen   bool      `json:"can_open"`
	UpdatedAt time.Time `json:"updated_at"`
}
