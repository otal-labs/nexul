// Package issuesync keeps a project's GitHub issues and its tickets in step (ADR 0200).
package issuesync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Mode says which of a repository's issues become tickets.
type Mode string

const (
	// ModeLabel syncs the issues carrying the sync's label; removing it pauses the issue's sync.
	ModeLabel Mode = "label"
	// ModeAll syncs every issue.
	ModeAll Mode = "all"
)

// DefaultLabel is the label a new sync triggers on.
const DefaultLabel = "nexul"

// Sync is one repository's issue sync, owned by the project the repository is attached to.
type Sync struct {
	ProjectID string `json:"project_id"`
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	Label     string `json:"label"`
	Mode      Mode   `json:"mode"`
	Enabled   bool   `json:"enabled"`
	// LastSweptAt is when issues were last listed from GitHub; nil until the first backfill.
	LastSweptAt *time.Time `json:"last_swept_at,omitempty"`
}

// Change turns a sync on or off and sets its label and mode; a nil field keeps its value.
type Change struct {
	Owner   string
	Repo    string
	Label   *string
	Mode    *Mode
	Enabled *bool
}

// Link joins one GitHub issue to the ticket it became.
type Link struct {
	Owner       string
	Repo        string
	IssueNumber int
	IssueID     int64
	TicketID    string
	// Paused is a label-mode issue whose label was removed; the link stays so adding it back resumes.
	Paused bool
	// InboundHash is Hash of the issue last applied to the ticket; OutboundHash of what Nexul last wrote to GitHub.
	InboundHash  string
	OutboundHash string
	// TicketHash is Hash of the ticket as its issue takes it when the two were last in step, so what GitHub changed is
	// never written back.
	TicketHash     string
	IssueUpdatedAt time.Time
	LinkedAt       time.Time
}

// TicketPush is a ticket's fields waiting to be written to its issue.
type TicketPush struct {
	TicketID string
	Owner    string
	Repo     string
	// Create makes the issue in Owner/Repo when the ticket has none, once its trigger label was added.
	Create bool
	Due    time.Time
	// Tried is when a create was first sent, zero before; a retry looks for the issue it may have made.
	Tried    time.Time
	Attempts int
}

// CommentPush is a person's message on a linked ticket's thread waiting to be mirrored to its issue.
type CommentPush struct {
	MessageID string
	TicketID  string
	Due       time.Time
	// Tried is when the comment was first sent, zero before; a retry looks for the comment it may have made.
	Tried    time.Time
	Attempts int
}

// CommentLink joins a GitHub comment to the ticket-thread message it became.
type CommentLink struct {
	GitHubCommentID int64
	MessageID       string
	TicketID        string
}

// Issue is a GitHub issue as git.issue and a listing carry it, declared here so the domain stays apart from gitprovider (ADR 0017).
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
	HTMLURL     string    `json:"html_url"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Comment is a comment on a GitHub issue as git.issue_comment and a listing carry it.
type Comment struct {
	ID              int64     `json:"id"`
	Body            string    `json:"body"`
	Author          string    `json:"author"`
	AuthorAvatarURL string    `json:"author_avatar_url"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TicketFields is what a ticket takes from a GitHub issue.
type TicketFields struct {
	Title  string
	Body   string
	Closed bool
	// StateReason is why GitHub closed or reopened it, such as completed or not_planned.
	StateReason string
	// Labels are the issue's labels, sorted, without the trigger label.
	Labels []string
	// Assignees are the issue's assignee logins in GitHub's order; the first one a member holds becomes the developer.
	Assignees []string
}

// FieldsFromIssue maps an issue onto ticket fields, leaving out trigger, the label that turns a label-mode sync on, and
// the hidden marker of the ticket Nexul made the issue for.
func FieldsFromIssue(i Issue, trigger string) TicketFields {
	labels := make([]string, 0, len(i.Labels))
	for _, l := range i.Labels {
		if trigger != "" && strings.EqualFold(l, trigger) {
			continue
		}
		labels = append(labels, l)
	}
	slices.Sort(labels)
	return TicketFields{
		Title: strings.TrimSpace(i.Title), Body: withoutMarker(i.Body), Closed: i.State == "closed", StateReason: i.StateReason,
		Labels: labels, Assignees: slices.Clone(i.Assignees),
	}
}

// Hash is SHA-256 over the title, body, state, sorted labels and first assignee, in that order: an issue whose hash
// matches what was last applied or last written carries nothing new.
func Hash(f TicketFields) string {
	state, assignee := "open", ""
	if f.Closed {
		state = "closed"
	}
	if len(f.Assignees) > 0 {
		assignee = f.Assignees[0]
	}
	labels := slices.Sorted(slices.Values(f.Labels))
	raw, _ := json.Marshal([]any{f.Title, f.Body, state, labels, assignee}) // strings and a string slice always marshal
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// syncs reports whether an issue belongs to s: every issue in all mode, else one carrying the label.
func (s Sync) syncs(i Issue) bool {
	if s.Mode == ModeAll {
		return true
	}
	return slices.ContainsFunc(i.Labels, func(l string) bool { return strings.EqualFold(l, s.Label) })
}

// trigger is the label FieldsFromIssue leaves out: the sync's label in label mode, none in all mode.
func (s Sync) trigger() string {
	if s.Mode == ModeAll {
		return ""
	}
	return s.Label
}

// apply returns s with c's fields set, validated.
func (s Sync) apply(c Change) (Sync, error) {
	if c.Label != nil {
		s.Label = strings.TrimSpace(*c.Label)
	}
	if c.Mode != nil {
		s.Mode = *c.Mode
	}
	if c.Enabled != nil {
		s.Enabled = *c.Enabled
	}
	if s.Mode != ModeLabel && s.Mode != ModeAll {
		return Sync{}, fmt.Errorf("%w: mode is label or all", apperrs.ErrInvalid)
	}
	if s.Mode == ModeLabel && s.Label == "" {
		return Sync{}, fmt.Errorf("%w: label mode needs a label, such as %s", apperrs.ErrInvalid, DefaultLabel)
	}
	if len(s.Label) > 50 {
		return Sync{}, fmt.Errorf("%w: a GitHub label is at most 50 characters", apperrs.ErrInvalid)
	}
	return s, nil
}
