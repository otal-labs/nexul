package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/imports"
	"github.com/otal-labs/nexul/internal/issuesync"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// importSources is every tool an import reads, by the name import_run takes; a new source is one line here.
func importSources(git gitprovider.GitProvider, workspaceSvc *workspace.Service, syncs *storage.IssueSyncRepo) map[string]imports.Source {
	return map[string]imports.Source{
		imports.SourceGitHub: imports.NewGitHubSource(importGitHubIssues{git: git}, importProjectRepos{svc: workspaceSvc}, importIssueSyncs{repo: syncs}),
		imports.SourceNotion: imports.NewNotionSource(imports.NewNotionClient("", nil, 3)),
		imports.SourceLinear: imports.NewLinearSource(imports.LinearAPI, nil),
	}
}

// importIssueSyncs answers from issue sync's own table whether a repository's issues already sync to tickets.
type importIssueSyncs struct {
	repo *storage.IssueSyncRepo
}

func (s importIssueSyncs) SyncOn(ctx context.Context, owner, name string) (bool, error) {
	sync, err := s.repo.GetSync(ctx, owner, name)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return sync.Enabled, nil
}

var (
	_ imports.Shelf    = importShelf{}
	_ imports.Desk     = importDesk{}
	_ imports.Board    = importBoard{}
	_ imports.People   = importPeople{}
	_ imports.Projects = importProjects{}
)

// importDesk writes an import's records through tickets, chat and attachments, as the person who started it (ADR 0017).
type importDesk struct {
	tickets     *tickets.Service
	chat        *chat.Service
	attachments *attachments.Service
	types       *storage.TicketTypesRepo
	projects    *storage.ProjectsRepo
}

// FileTicket files the ticket with the project's first type that is not a bug, since a bug needs the ticket it was
// found in; a project whose only type is bug marks the origin unknown.
func (d importDesk) FileTicket(ctx context.Context, t imports.NewTicket) error {
	types, err := d.types.ListByProject(ctx, t.ProjectID)
	if err != nil {
		return err
	}
	opt := tickets.CreateOptions{
		Status: tickets.Status(t.Status), CategoryID: t.CategoryID,
		Import: &tickets.ImportedTicket{ID: t.ID, Source: t.Source, Labels: t.Labels, DueOn: t.DueOn},
	}
	for _, ty := range types {
		if !tickets.IsBugType(ty.Name) {
			opt.TypeID = ty.ID
			break
		}
	}
	if opt.TypeID == "" && len(types) > 0 {
		opt.TypeID, opt.OriginUnknown = types[0].ID, true
	}
	_, err = d.tickets.Create(ctx, t.ProjectID, t.Title, t.Body, "", t.Developer, opt)
	return err
}

func (d importDesk) SetBody(ctx context.Context, ticketID, title, body string) error {
	_, err := d.tickets.UpdateTicket(ctx, ticketID, title, body)
	return err
}

func (d importDesk) Comment(ctx context.Context, ticketID string, c imports.NewComment) error {
	actor, _ := identity.ActorFromCtx(ctx) // the import loop always acts as the person who started it
	t, err := d.tickets.Get(ctx, ticketID)
	if err != nil {
		return err
	}
	project, err := d.projects.Get(ctx, t.ProjectID)
	if err != nil {
		return err
	}
	// Opening the thread as the importer is the read check; the copied comment then posts as the server, as sync's do.
	thread, err := d.chat.GetOrCreateTicketThread(ctx, project.WorkspaceID, ticketID, actor.ID)
	if err != nil {
		return err
	}
	_, err = d.chat.PostExternalMessage(ctx, chat.ExternalPost{
		ID: c.ID, ConversationID: thread.ID, AuthorID: c.AuthorKey, Name: c.Name,
		AvatarURL: c.AvatarURL, Body: c.Body, Via: c.Via, At: c.At,
	})
	return err
}

func (d importDesk) Attach(ctx context.Context, ticketID, name string, data []byte) (string, error) {
	a, err := d.attachments.Upload(ctx, attachments.Owner{TicketID: ticketID}, name, data)
	if err != nil {
		return "", err
	}
	return a.ID, nil
}

// importShelf writes an import's docs through docs and attachments, as the person who started it (ADR 0017).
type importShelf struct {
	docs        *docs.Service
	attachments *attachments.Service
}

// Folder finds the folder by name first, so a re-run and a person's own folder of that name are both reused.
func (f importShelf) Folder(ctx context.Context, projectID, name string) (string, error) {
	find := func() (string, error) {
		folders, err := f.docs.ListFolders(ctx, projectID)
		if err != nil {
			return "", err
		}
		for _, folder := range folders {
			if strings.EqualFold(strings.TrimSpace(folder.Name), strings.TrimSpace(name)) {
				return folder.ID, nil
			}
		}
		return "", nil
	}
	if id, err := find(); err != nil || id != "" {
		return id, err
	}
	folder, err := f.docs.CreateFolder(ctx, projectID, name)
	if errors.Is(err, apperrs.ErrConflict) {
		return find()
	}
	if err != nil {
		return "", err
	}
	return folder.ID, nil
}

// CreateDoc restores a doc of the id an earlier, stopped import archived, so a re-run fills it again.
func (f importShelf) CreateDoc(ctx context.Context, d imports.NewDoc) error {
	_, err := f.docs.CreateWithID(ctx, d.ID, d.ProjectID, d.FolderID, d.Title, d.Body)
	if !errors.Is(err, apperrs.ErrConflict) {
		return err
	}
	existing, getErr := f.docs.Get(ctx, d.ID)
	if getErr == nil && existing.Archived {
		if _, err := f.docs.Restore(ctx, d.ID); err != nil {
			return err
		}
		_, err := f.docs.Update(ctx, d.ID, d.Title, d.Body)
		if err != nil {
			return err
		}
	}
	return apperrs.ErrConflict
}

func (f importShelf) SetDocBody(ctx context.Context, docID, title, body string) error {
	_, err := f.docs.Update(ctx, docID, title, body)
	return err
}

func (f importShelf) AttachToDoc(ctx context.Context, docID, name string, data []byte) (string, error) {
	a, err := f.attachments.Upload(ctx, attachments.Owner{DocID: docID}, name, data)
	if err != nil {
		return "", err
	}
	return a.ID, nil
}

func (f importShelf) ArchiveDoc(ctx context.Context, docID string) error {
	_, err := f.docs.Archive(ctx, docID)
	return err
}

func (d importDesk) Block(ctx context.Context, ticketID, blockerID string) error {
	_, err := d.tickets.AddBlocker(ctx, ticketID, blockerID)
	return err
}

// importBoard reads and adds a project's status columns and categories.
type importBoard struct {
	svc *workspace.Service
}

func (b importBoard) Columns(ctx context.Context, projectID string) ([]imports.Column, error) {
	statuses, err := b.svc.ListStatusesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]imports.Column, 0, len(statuses))
	for _, st := range statuses {
		out = append(out, imports.Column{ID: st.ID, Name: st.Name, Kind: imports.StatusKind(st.Kind)})
	}
	return out, nil
}

func (b importBoard) AddColumn(ctx context.Context, projectID, name string, kind imports.StatusKind) (imports.Column, error) {
	st, err := b.svc.CreateStatus(ctx, projectID, name, workspace.StatusKind(kind), "")
	if err != nil {
		return imports.Column{}, err
	}
	return imports.Column{ID: st.ID, Name: st.Name, Kind: imports.StatusKind(st.Kind)}, nil
}

func (b importBoard) Categories(ctx context.Context, projectID string) ([]imports.Category, error) {
	cats, err := b.svc.ListCategoriesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]imports.Category, 0, len(cats))
	for _, c := range cats {
		out = append(out, imports.Category{ID: c.ID, Name: c.Name})
	}
	return out, nil
}

func (b importBoard) AddCategory(ctx context.Context, projectID, name string) (imports.Category, error) {
	c, err := b.svc.CreateCategory(ctx, projectID, name, "")
	if err != nil {
		return imports.Category{}, err
	}
	return imports.Category{ID: c.ID, Name: c.Name}, nil
}

// importPeople is the project's workspace members who may open it, so a matched assignee can be its developer.
type importPeople struct {
	store    *storage.ImportsRepo
	projects *storage.ProjectsRepo
	access   *access.Service
}

func (p importPeople) Members(ctx context.Context, projectID string) ([]imports.Member, error) {
	project, err := p.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	members, err := p.store.People(ctx, project.WorkspaceID)
	if err != nil {
		return nil, err
	}
	var out []imports.Member
	for _, m := range members {
		if p.access.CanInProject(ctx, m.UserID, projectID, permissions.Member) {
			out = append(out, m)
		}
	}
	return out, nil
}

// importProjects resolves a project's workspace.
type importProjects struct {
	projects *storage.ProjectsRepo
}

func (p importProjects) WorkspaceOf(ctx context.Context, projectID string) (string, error) {
	project, err := p.projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return project.WorkspaceID, nil
}

// importProjectRepos lists a project's repositories as owner/name.
type importProjectRepos struct {
	svc *workspace.Service
}

func (r importProjectRepos) Repos(ctx context.Context, projectID string) ([]string, error) {
	repos, err := r.svc.ListRepos(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(repos))
	for _, repo := range repos {
		out = append(out, repo.Owner+"/"+repo.Name)
	}
	return out, nil
}

// importGitHubIssues reads issues through the git router, as the App's installation on the repository.
type importGitHubIssues struct {
	git gitprovider.GitProvider
}

func (g importGitHubIssues) ListIssues(ctx context.Context, owner, name string, page int) ([]imports.GitHubIssue, int, error) {
	issues, next, err := g.git.ListIssues(ctx, owner, name, gitprovider.IssueOpts{State: "all", Page: page})
	if err != nil {
		return nil, 0, fmt.Errorf("list issues of %s/%s: %w", owner, name, err)
	}
	out := make([]imports.GitHubIssue, 0, len(issues))
	for _, i := range issues {
		// Sync's mapping of the shared fields, so an issue reads the same imported as synced.
		f := issuesync.FieldsFromIssue(issuesync.Issue{Title: i.Title, Body: i.Body, State: i.State, Labels: i.Labels, Assignees: i.Assignees}, "")
		out = append(out, imports.GitHubIssue{
			ID: i.ID, Number: i.Number, Title: f.Title, Body: f.Body, State: i.State, Labels: f.Labels,
			Assignees: f.Assignees, Author: i.Author, HTMLURL: i.HTMLURL, Comments: i.Comments,
		})
	}
	return out, next, nil
}

func (g importGitHubIssues) ListComments(ctx context.Context, owner, name string, number int) ([]imports.GitHubComment, error) {
	comments, err := g.git.ListIssueComments(ctx, owner, name, number, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("list comments of %s/%s#%d: %w", owner, name, number, err)
	}
	out := make([]imports.GitHubComment, 0, len(comments))
	for _, c := range comments {
		out = append(out, imports.GitHubComment{
			ID: c.ID, Body: c.Body, Author: c.Author, AuthorAvatarURL: c.AuthorAvatarURL, CreatedAt: c.CreatedAt,
		})
	}
	return out, nil
}
