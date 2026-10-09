package docs

import "time"

type Doc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
	// ProjectID is the project this doc belongs to; mandatory, mirrors Ticket.ProjectID (ADR 0025).
	ProjectID string `json:"project_id"`
	// FolderID is the project folder the doc lives in; every doc is in exactly one (ADR 0096).
	FolderID  string    `json:"folder_id" jsonschema:"The project folder the doc lives in."`
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
	IsDefault bool      `json:"is_default" jsonschema:"The project's default folder, where new docs land; it is never deleted."`
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

// Clarification is a doc's rounds of questions and answers as one person sees them; NoGapsAt shows only when CanClose.
type Clarification struct {
	Rounds []*ClarificationRound `json:"rounds"`
	// Running is a round being written now; Closed is the newest round closed, until another round reopens it.
	Running bool `json:"running"`
	Closed  bool `json:"closed"`
	// CanClose is docs:write plus plays:run on the Clarify play, which closing and the no-gaps signal take.
	CanClose bool `json:"can_close"`
}

// ClarificationRound is one Clarify run's batch of questions, numbered from 1 within its doc.
type ClarificationRound struct {
	DocID     string    `json:"doc_id"`
	Round     int       `json:"round"`
	StartedBy string    `json:"started_by"`
	TrailID   string    `json:"trail_id"`
	StartedAt time.Time `json:"started_at"`
	Running   bool      `json:"running"`
	// TookLock is whether this round's run locked the doc, so its end unlocks it (ADR 0121).
	TookLock       bool       `json:"-"`
	AnythingElse   string     `json:"anything_else"`
	AnythingElseBy string     `json:"anything_else_by,omitempty"`
	AnythingElseAt *time.Time `json:"anything_else_at,omitempty"`
	// AnythingElseReply is the next round's one-line answer to this round's AnythingElse.
	AnythingElseReply string `json:"anything_else_reply"`
	// NoGapsAt is when this round found no gaps left and wrote the doc.
	NoGapsAt  *time.Time               `json:"no_gaps_at,omitempty"`
	ClosedBy  string                   `json:"closed_by,omitempty"`
	ClosedAt  *time.Time               `json:"closed_at,omitempty"`
	Questions []*ClarificationQuestion `json:"questions"`
}

// ClarificationQuestion is one question of a round with its answer; no picks, no text, and not skipped is pending.
type ClarificationQuestion struct {
	ID          string           `json:"id"`
	DocID       string           `json:"doc_id"`
	Round       int              `json:"round"`
	Position    int              `json:"position"`
	Question    string           `json:"question"`
	Why         string           `json:"why"`
	Options     []QuestionOption `json:"options"`
	MultiSelect bool             `json:"multi_select"`
	Selected    []string         `json:"selected"`
	Text        string           `json:"text"`
	Skipped     bool             `json:"skipped"`
	AnsweredBy  string           `json:"answered_by,omitempty"`
	AnsweredAt  *time.Time       `json:"answered_at,omitempty"`
}

// Pending reports a question nobody has answered or skipped yet.
func (q *ClarificationQuestion) Pending() bool {
	return len(q.Selected) == 0 && q.Text == "" && !q.Skipped
}

// QuestionOption is one choice a question offers.
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Answer is what someone gives a question: picked labels, free text, or a skip, which carries neither.
type Answer struct {
	Selected []string `json:"selected"`
	Text     string   `json:"text"`
	Skipped  bool     `json:"skipped"`
}
