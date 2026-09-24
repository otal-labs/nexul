package main

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

// ticketStages adapts workspace's status columns to tickets' Stages seam (ADR 0017).
type ticketStages struct {
	statuses *storage.StatusesRepo
}

func (a ticketStages) StageOf(ctx context.Context, statusID string) (string, error) {
	st, err := a.statuses.Get(ctx, statusID)
	if err != nil {
		return "", err
	}
	return string(st.Kind), nil
}

// FirstOfStage relies on ListByProject's board order: stage first, then position.
func (a ticketStages) FirstOfStage(ctx context.Context, projectID, stage string) (tickets.Status, error) {
	statuses, err := a.statuses.ListByProject(ctx, projectID)
	if err != nil {
		return "", err
	}
	for _, st := range statuses {
		if string(st.Kind) == stage {
			return tickets.Status(st.ID), nil
		}
	}
	return "", fmt.Errorf("%s column in project %s: %w", stage, projectID, apperrs.ErrNotFound)
}

// ticketThreads adapts chat to tickets' TicketThreads seam; the thread lives in the ticket's project's workspace.
type ticketThreads struct {
	chat     *chat.Service
	projects *storage.ProjectsRepo
}

func (a ticketThreads) PostToTicketThread(ctx context.Context, t *tickets.Ticket, authorID, body string) error {
	project, err := a.projects.Get(ctx, t.ProjectID)
	if err != nil {
		return err
	}
	thread, err := a.chat.GetOrCreateTicketThread(ctx, project.WorkspaceID, t.ID, authorID)
	if err != nil {
		return err
	}
	_, err = a.chat.PostMessage(ctx, thread.ID, authorID, body)
	return err
}

// ticketTestTargets adapts deploy's test target resolution to tickets' TestTargets seam.
type ticketTestTargets struct {
	deploy *deploy.Service
}

func (a ticketTestTargets) ResolveTestTarget(ctx context.Context, projectID string, branches []tickets.BranchLink) (tickets.TestTarget, error) {
	refs := make([]deploy.BranchRef, 0, len(branches))
	for _, b := range branches {
		refs = append(refs, deploy.BranchRef{Owner: b.Owner, Repo: b.Repo, Branch: b.Branch})
	}
	target, err := a.deploy.ResolveTestTarget(ctx, projectID, refs)
	if err != nil {
		return tickets.TestTarget{}, err
	}
	return tickets.TestTarget{URL: target.URL, Kind: string(target.Kind), Branch: target.Branch}, nil
}
