// Package issuesync keeps a project's GitHub issues and their tickets in step, GitHub to Nexul (ADR 0200).
package issuesync

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"
)

// Mode says which issues become tickets.
type Mode string

const (
	// ModeLabel links the issues carrying the sync's label; removing the label pauses the link.
	ModeLabel Mode = "label"
	// ModeAll links every issue.
	ModeAll Mode = "all"
)

// DefaultLabel is the label a new sync links issues by.
const DefaultLabel = "nexul"

// Sync is one repository's issue sync in its project.
type Sync struct {
	ProjectID string `json:"project_id"`
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	Label     string `json:"label"`
	Mode      Mode   `json:"mode" enum:"label,all"`
	Enabled   bool   `json:"enabled"`
	// LastSweptAt is when the last sweep began; nil until the first one links the open issues.
	LastSweptAt *time.Time `json:"last_swept_at,omitempty"`
}

// Matches reports whether an issue with labels is one this sync links.
func (s *Sync) Matches(labels []string) bool {
	if s.Mode == ModeAll {
		return true
	}
	return slices.ContainsFunc(labels, func(l string) bool { return strings.EqualFold(l, s.Label) })
}

// Link ties one issue to its ticket.
type Link struct {
	Owner          string
	Repo           string
	Number         int
	IssueID        int64
	TicketID       string
	Paused         bool
	InboundHash    string
	OutboundHash   string
	IssueUpdatedAt time.Time
	LinkedAt       time.Time
}

// CommentLink ties a GitHub comment to the ticket-thread message it became.
type CommentLink struct {
	CommentID int64
	MessageID string
	TicketID  string
}

// Issue is an issue as the git host sends it.
type Issue struct {
	ID          int64     `json:"id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	StateReason string    `json:"state_reason"`
	Labels      []string  `json:"labels"`
	Assignees   []string  `json:"assignees"`
	Author      string    `json:"author"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Closed reports whether the issue is closed.
func (i Issue) Closed() bool { return i.State == "closed" }

// Hash is SHA-256 over the fields a ticket follows, in a fixed order, so an unchanged issue writes nothing.
func (i Issue) Hash() string {
	labels := slices.Clone(i.Labels)
	slices.Sort(labels)
	assignee := ""
	if len(i.Assignees) > 0 {
		assignee = i.Assignees[0]
	}
	h := sha256.New()
	for _, part := range []string{i.Title, i.Body, i.State, strings.Join(labels, "\n"), assignee} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Comment is a comment on an issue as the git host sends it.
type Comment struct {
	ID              int64  `json:"id"`
	Body            string `json:"body"`
	Author          string `json:"author"`
	AuthorAvatarURL string `json:"author_avatar_url"`
}

// Stage names the column a ticket moves to: the project's first column of that stage.
type Stage string

const (
	StageBacklog Stage = "backlog"
	StageDone    Stage = "done"
)

// Ticket is what the sync reads of a linked ticket.
type Ticket struct {
	ID        string
	ProjectID string
	Title     string
	Body      string
	Labels    []string
	Developer string
	// Done is a ticket in a done-stage column.
	Done bool
}

// NewTicket is a ticket the sync files for an issue.
type NewTicket struct {
	Title string
	Body  string
	// Reporter is the GitHub login that opened the issue.
	Reporter string
}

// Input changes a sync; a nil field keeps its value, or its default on a new sync.
type Input struct {
	Owner   string
	Repo    string
	Label   *string
	Mode    *string
	Enabled *bool
}
