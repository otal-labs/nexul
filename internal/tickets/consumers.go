package tickets

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// prEvent is declared consumer-side so tickets stays decoupled from gitprovider (ADR 0017).
type prEvent struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	PR    prRef  `json:"pr"`
}

type prRef struct {
	Number          int      `json:"number"`
	Title           string   `json:"title"`
	HeadSHA         string   `json:"head_sha"`
	LinkedTicketIDs []string `json:"linked_ticket_ids"`
}

// HandlePRMerged is registered on git.pr_merged; re-evaluates each linked ticket for completion.
func (s *Service) HandlePRMerged(ctx context.Context, ev eventbus.Event) error {
	return s.handlePRState(ctx, ev, PRStateMerged)
}

// HandlePRClosed is registered on git.pr_closed; all-closed-unmerged never finishes a ticket.
func (s *Service) HandlePRClosed(ctx context.Context, ev eventbus.Event) error {
	return s.handlePRState(ctx, ev, PRStateClosed)
}

func (s *Service) handlePRState(ctx context.Context, ev eventbus.Event, state PRState) error {
	e, err := unmarshalPREvent(ev)
	if err != nil {
		return err
	}
	ref := PRRef{Owner: e.Owner, Repo: e.Repo, Number: e.PR.Number, Title: e.PR.Title, SHA: e.PR.HeadSHA}
	for _, id := range e.PR.LinkedTicketIDs {
		if err := s.repo.LinkPR(ctx, id, ref, state); err != nil {
			return apperrs.Retryable(err)
		}
	}
	affected, err := s.repo.MarkPRState(ctx, e.Owner, e.Repo, e.PR.Number, state)
	if err != nil {
		return apperrs.Retryable(err)
	}
	for _, id := range affected {
		if err := s.evaluateCompletion(ctx, id); err != nil {
			return apperrs.Retryable(err)
		}
	}
	return nil
}

func unmarshalPREvent(ev eventbus.Event) (prEvent, error) {
	var e prEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return prEvent{}, apperrs.Fatal(err)
	}
	if e.Owner == "" || e.Repo == "" || e.PR.Number < 1 {
		return prEvent{}, apperrs.Fatal(fmt.Errorf("git event missing owner, repo, or number"))
	}
	return e, nil
}
