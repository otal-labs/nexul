package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

var _ deploy.TicketDesk = crashTicketDesk{}

// crashTicketDesk adapts tickets and chat to deploy's TicketDesk seam (ADR 0017). It runs from event consumers, so a
// ticket it files has no person behind it, and a comment is a system message in the ticket's thread posted for the
// project's workspace Owner, who is also the thread's first participant.
type crashTicketDesk struct {
	tickets  *tickets.Service
	chat     *chat.Service
	statuses *storage.StatusesRepo
	types    *storage.TicketTypesRepo
	projects *storage.ProjectsRepo
	users    *storage.UsersRepo
}

// File gives the ticket the project's default type; a bug type needs a found-in, which no one can name here, so it is marked unknown.
func (d crashTicketDesk) File(ctx context.Context, projectID, title, body string) (string, error) {
	typeID, err := d.types.FirstType(ctx, projectID)
	if err != nil {
		return "", err
	}
	opt := tickets.CreateOptions{TypeID: typeID}
	if typeID != "" {
		name, err := d.types.TypeName(ctx, typeID)
		if err != nil {
			return "", err
		}
		opt.OriginUnknown = tickets.IsBugType(name)
	}
	t, err := d.tickets.Create(ctx, projectID, title, body, "", "", opt)
	if err != nil {
		return "", err
	}
	return t.ID, nil
}

// Open treats a ticket in a done column, or in the legacy done or closed status, as finished.
func (d crashTicketDesk) Open(ctx context.Context, ticketID string) (bool, error) {
	t, err := d.tickets.Get(ctx, ticketID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	done, err := ticketDone(ctx, d.statuses, t)
	return !done, err
}

// ticketDone is a ticket in a done column, or in the legacy done or closed status.
func ticketDone(ctx context.Context, statuses *storage.StatusesRepo, t *tickets.Ticket) (bool, error) {
	if t.Status == tickets.StatusDone || t.Status == tickets.StatusClosed {
		return true, nil
	}
	column, err := statuses.Get(ctx, string(t.Status))
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return column.Kind == workspace.StatusKindDone, nil
}

func (d crashTicketDesk) Comment(ctx context.Context, ticketID, body string) error {
	t, err := d.tickets.Get(ctx, ticketID)
	if err != nil {
		return err
	}
	project, err := d.projects.Get(ctx, t.ProjectID)
	if err != nil {
		return err
	}
	owner, err := d.users.FirstWorkspaceOwner(ctx, project.WorkspaceID)
	if err != nil {
		return fmt.Errorf("find an owner of workspace %s to post for: %w", project.WorkspaceID, err)
	}
	thread, err := d.chat.GetOrCreateTicketThread(ctx, project.WorkspaceID, ticketID, owner)
	if err != nil {
		return err
	}
	_, err = d.chat.PostSystemMessage(ctx, thread.ID, owner, body)
	return err
}
