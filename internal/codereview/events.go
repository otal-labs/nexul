package codereview

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicStatusChanged is published on every review status change.
const TopicStatusChanged = "review.status_changed"

// Topics returns every topic the codereview domain publishes.
func Topics() []string {
	return []string{TopicStatusChanged}
}

// StatusChangedEvent is the review.status_changed payload.
type StatusChangedEvent struct {
	ID       string `json:"id"`
	Repo     string `json:"repo"`
	PRNumber int    `json:"pr_number"`
	Status   string `json:"status"`
	Reviewer string `json:"reviewer,omitempty"`
}

// prEvent is declared consumer-side so codereview stays decoupled from gitprovider (ADR 0017).
type prEvent struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	PR    prRef  `json:"pr"`
}

type prRef struct {
	Number int `json:"number"`
}

// reviewSubmittedEvent mirrors the git.pr_review_submitted payload published by the gitprovider domain.
type reviewSubmittedEvent struct {
	Owner string             `json:"owner"`
	Repo  string             `json:"repo"`
	PR    reviewSubmittedRef `json:"pr"`
}

type reviewSubmittedRef struct {
	Number   int    `json:"number"`
	State    string `json:"state"`
	Reviewer string `json:"reviewer"`
}

// HandlePROpened is registered on git.pr_opened.
func HandlePROpened(ctx context.Context, svc *Service, ev eventbus.Event) error {
	e, err := unmarshalPREvent(ev)
	if err != nil {
		return err
	}
	if _, err := svc.Create(ctx, e.Owner+"/"+e.Repo, e.PR.Number); err != nil {
		return apperrs.Retryable(err)
	}
	return nil
}

// HandlePRReviewSubmitted is registered on git.pr_review_submitted.
func HandlePRReviewSubmitted(ctx context.Context, svc *Service, ev eventbus.Event) error {
	var e reviewSubmittedEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(err)
	}
	if e.Owner == "" || e.Repo == "" || e.PR.Number < 1 {
		return apperrs.Fatal(fmt.Errorf("git.pr_review_submitted missing owner, repo, or number"))
	}
	if _, err := svc.ApplyReviewSubmitted(ctx, e.Owner+"/"+e.Repo, e.PR.Number, e.PR.State, e.PR.Reviewer); err != nil {
		return apperrs.Retryable(err)
	}
	return nil
}

// HandlePRMerged is registered on git.pr_merged.
func HandlePRMerged(ctx context.Context, svc *Service, ev eventbus.Event) error {
	e, err := unmarshalPREvent(ev)
	if err != nil {
		return err
	}
	if _, err := svc.ApplyPRMerged(ctx, e.Owner+"/"+e.Repo, e.PR.Number); err != nil {
		return apperrs.Retryable(err)
	}
	return nil
}

// HandlePRClosed is registered on git.pr_closed.
func HandlePRClosed(ctx context.Context, svc *Service, ev eventbus.Event) error {
	e, err := unmarshalPREvent(ev)
	if err != nil {
		return err
	}
	if _, err := svc.ApplyPRClosed(ctx, e.Owner+"/"+e.Repo, e.PR.Number); err != nil {
		return apperrs.Retryable(err)
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
