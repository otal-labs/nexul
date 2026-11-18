package issuesync

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// The topics the sync consumes, published by gitprovider; named here so the domain stays apart from it (ADR 0017).
const (
	TopicIssue        = "git.issue"
	TopicIssueComment = "git.issue_comment"
)

type sender struct {
	Login string `json:"login"`
}

// issueEvent mirrors the git.issue payload.
type issueEvent struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Action string `json:"action"`
	Issue  Issue  `json:"issue"`
	Sender sender `json:"sender"`
}

// commentEvent mirrors the git.issue_comment payload.
type commentEvent struct {
	Owner       string  `json:"owner"`
	Repo        string  `json:"repo"`
	IssueNumber int     `json:"issue_number"`
	Action      string  `json:"action"`
	Comment     Comment `json:"comment"`
	Sender      sender  `json:"sender"`
}

// HandleIssueEvent is registered on git.issue.
func (s *Service) HandleIssueEvent(ctx context.Context, ev eventbus.Event) error {
	var e issueEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicIssue, err))
	}
	return s.HandleIssue(ctx, e.Owner, e.Repo, e.Action, e.Issue, e.Sender.Login)
}

// HandleCommentEvent is registered on git.issue_comment.
func (s *Service) HandleCommentEvent(ctx context.Context, ev eventbus.Event) error {
	var e commentEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicIssueComment, err))
	}
	return s.HandleComment(ctx, e.Owner, e.Repo, e.IssueNumber, e.Action, e.Comment, e.Sender.Login)
}
