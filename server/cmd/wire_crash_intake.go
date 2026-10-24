package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/crashintake"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

var (
	_ crashintake.Desk  = crashIntakeDesk{}
	_ crashintake.Notes = crashIntakeNotes{}
)

// crashIntakeDesk adapts tickets to the crash intake's Desk seam (ADR 0017).
type crashIntakeDesk struct {
	tickets  *tickets.Service
	statuses *storage.StatusesRepo
	types    *storage.TicketTypesRepo
}

// Find searches the project for the fingerprint and keeps the tickets whose body has the exact line, since a search
// also matches notes and prefixes.
func (d crashIntakeDesk) Find(ctx context.Context, projectID, fingerprint string) ([]crashintake.Match, error) {
	found, _, err := d.tickets.PageTickets(ctx, tickets.TicketFilter{ProjectID: projectID, Query: fingerprint}, paging.Window{Limit: paging.MaxLimit})
	if err != nil {
		return nil, err
	}
	line := crashintake.FingerprintLine + fingerprint
	var out []crashintake.Match
	for _, t := range found {
		if !hasLine(t.Body, line) {
			continue
		}
		m, err := d.match(ctx, t)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func hasLine(body, line string) bool {
	for l := range strings.SplitSeq(body, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

func (d crashIntakeDesk) match(ctx context.Context, t *tickets.Ticket) (crashintake.Match, error) {
	done, err := ticketDone(ctx, d.statuses, t)
	if err != nil {
		return crashintake.Match{}, err
	}
	m := crashintake.Match{ID: t.ID, CreatedAt: t.CreatedAt, Done: done}
	if !done {
		return m, nil
	}
	prs, _, err := d.tickets.ListLinks(ctx, t.ID)
	if err != nil {
		return crashintake.Match{}, err
	}
	for _, pr := range prs {
		if pr.State == tickets.PRStateMerged {
			m.Merged = append(m.Merged, crashintake.PR{Owner: pr.Owner, Repo: pr.Repo, Number: pr.Number})
		}
	}
	return m, nil
}

// File creates the bug in the project's first column; a labelled one, a returned crash, moves to the top of it.
func (d crashIntakeDesk) File(ctx context.Context, bug crashintake.Bug) (string, error) {
	typeID, err := d.typeNamed(ctx, bug.ProjectID, bug.TypeName)
	if err != nil {
		return "", err
	}
	opt := tickets.CreateOptions{TypeID: typeID, OriginID: bug.FoundIn, OriginUnknown: bug.FoundIn == ""}
	t, err := d.tickets.Create(ctx, bug.ProjectID, bug.Title, bug.Body, "", "", opt)
	if err != nil {
		return "", err
	}
	if bug.Label == "" {
		return t.ID, nil
	}
	if _, err := d.tickets.AddLabel(ctx, t.ID, bug.Label); err != nil {
		return "", err
	}
	if _, err := d.tickets.SetPosition(ctx, t.ID, 0); err != nil {
		return "", err
	}
	return t.ID, nil
}

func (d crashIntakeDesk) typeNamed(ctx context.Context, projectID, name string) (string, error) {
	types, err := d.types.ListByProject(ctx, projectID)
	if err != nil {
		return "", err
	}
	for _, t := range types {
		if strings.EqualFold(strings.TrimSpace(t.Name), name) {
			return t.ID, nil
		}
	}
	return "", fmt.Errorf("%w: project %s has no ticket type named %q", apperrs.ErrInvalid, projectID, name)
}

// crashIntakeNotes adapts chat to the crash intake's Notes seam. A note is posted on behalf of the caller, or, for a
// shipped default automation, which has no creator, on behalf of the workspace's Owner, as crash tickets' comments are.
type crashIntakeNotes struct {
	tickets  *tickets.Service
	chat     *chat.Service
	projects *storage.ProjectsRepo
	users    *storage.UsersRepo
}

func (n crashIntakeNotes) Find(ctx context.Context, ticketID, name string) (crashintake.Note, bool, error) {
	_, poster, err := n.poster(ctx, ticketID)
	if err != nil {
		return crashintake.Note{}, false, err
	}
	notes, err := n.chat.TicketNotes(ctx, ticketID, poster)
	if err != nil {
		return crashintake.Note{}, false, err
	}
	files, err := n.chat.NoteFiles(ctx, notes.Notes)
	if err != nil {
		return crashintake.Note{}, false, err
	}
	for _, m := range notes.Notes {
		if f := files[m.AttachmentID]; f != nil && f.Name == name {
			return crashintake.Note{ID: m.ID, Markdown: f.Markdown}, true, nil
		}
	}
	return crashintake.Note{}, false, nil
}

func (n crashIntakeNotes) Post(ctx context.Context, ticketID, name, markdown string) error {
	workspaceID, poster, err := n.poster(ctx, ticketID)
	if err != nil {
		return err
	}
	thread, err := n.chat.GetOrCreateTicketThread(ctx, workspaceID, ticketID, poster)
	if err != nil {
		return err
	}
	_, _, err = n.chat.PostNote(ctx, thread.ID, poster, "Crash intake keeps this note up to date.", chat.NoteFileInput{Name: name, Markdown: markdown})
	return err
}

func (n crashIntakeNotes) Replace(ctx context.Context, ticketID, noteID, markdown string) error {
	_, poster, err := n.poster(ctx, ticketID)
	if err != nil {
		return err
	}
	_, _, err = n.chat.ReplaceNote(ctx, noteID, poster, markdown)
	return err
}

// poster is the ticket's workspace and who the intake writes there as.
func (n crashIntakeNotes) poster(ctx context.Context, ticketID string) (workspaceID, userID string, err error) {
	t, err := n.tickets.Get(ctx, ticketID)
	if err != nil {
		return "", "", err
	}
	project, err := n.projects.Get(ctx, t.ProjectID)
	if err != nil {
		return "", "", err
	}
	if actor, ok := identity.ActorFromCtx(ctx); ok && actor.ID != "" {
		return project.WorkspaceID, actor.ID, nil
	}
	owner, err := n.users.FirstWorkspaceOwner(ctx, project.WorkspaceID)
	if err != nil {
		return "", "", fmt.Errorf("find an owner of workspace %s to post for: %w", project.WorkspaceID, err)
	}
	return project.WorkspaceID, owner, nil
}
