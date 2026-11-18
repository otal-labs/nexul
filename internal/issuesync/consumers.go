package issuesync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// issueEvent is git.issue as the sync reads it, declared here so issuesync never imports gitprovider (ADR 0017).
type issueEvent struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Action string `json:"action"`
	Issue  Issue  `json:"issue"`
	Sender sender `json:"sender"`
}

type sender struct {
	Login string `json:"login"`
}

// commentEvent is git.issue_comment as the sync reads it.
type commentEvent struct {
	Owner       string  `json:"owner"`
	Repo        string  `json:"repo"`
	IssueNumber int     `json:"issue_number"`
	Action      string  `json:"action"`
	Comment     Comment `json:"comment"`
	Sender      sender  `json:"sender"`
}

// HandleIssue is registered on git.issue.
func (s *Service) HandleIssue(ctx context.Context, ev eventbus.Event) error {
	var e issueEvent
	if err := decode(ev, &e); err != nil {
		return err
	}
	if e.Owner == "" || e.Repo == "" || e.Issue.Number < 1 {
		return apperrs.Fatal(errors.New("git.issue missing owner, repo, or issue number"))
	}
	if err := s.receive(ctx, e.Owner, e.Repo, e.Action, e.Sender.Login, e.Issue); err != nil {
		return apperrs.Retryable(err)
	}
	return nil
}

// HandleIssueComment is registered on git.issue_comment.
func (s *Service) HandleIssueComment(ctx context.Context, ev eventbus.Event) error {
	var e commentEvent
	if err := decode(ev, &e); err != nil {
		return err
	}
	if e.Owner == "" || e.Repo == "" || e.IssueNumber < 1 || e.Comment.ID == 0 {
		return apperrs.Fatal(errors.New("git.issue_comment missing owner, repo, issue number, or comment id"))
	}
	if err := s.receiveComment(ctx, e.Owner, e.Repo, e.IssueNumber, e.Action, e.Sender.Login, e.Comment); err != nil {
		return apperrs.Retryable(err)
	}
	return nil
}

func decode(ev eventbus.Event, into any) error {
	if err := json.Unmarshal(ev.Payload, into); err != nil {
		return apperrs.Fatal(fmt.Errorf("decode %s: %w", ev.Topic, err))
	}
	return nil
}
