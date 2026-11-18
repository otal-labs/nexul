package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/issuesync"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

var (
	_ issuesync.Desk     = issueDesk{}
	_ issuesync.Board    = issueDesk{}
	_ issuesync.Notes    = issueThread{}
	_ issuesync.Comments = issueThread{}
	_ issuesync.People   = issuePeople{}
	_ issuesync.Issues   = issueReader{}
	_ issuesync.Bot      = issueBot{}
	_ issuesync.Repos    = issueRepos{}
)

func newIssueSync(store *storage.Store, t *tickets.Service, c *chat.Service, w *workspace.Service, a *access.Service, git gitProviderRouter, appConfigs connectors.AppConfigStore) *issuesync.Service {
	desk := issueDesk{tickets: t, statuses: store.Statuses, types: store.TicketTypes}
	thread := issueThread{tickets: t, chat: c, projects: store.Projects, users: store.Users}
	return issuesync.New(issuesync.Deps{
		Repo: store.IssueSync, Desk: desk, Board: desk, Notes: thread, Comments: thread,
		People: issuePeople{users: store.Users, access: a}, Issues: issueReader{git: git},
		Bot: issueBot{configs: appConfigs}, Repos: issueRepos{workspace: w}, Gate: a,
	})
}

// issueDesk adapts tickets to the sync's Desk and Board seams (ADR 0017); it runs from event consumers and the sweep,
// so no person is behind its writes.
type issueDesk struct {
	tickets  *tickets.Service
	statuses *storage.StatusesRepo
	types    *storage.TicketTypesRepo
}

func (d issueDesk) File(ctx context.Context, projectID string, f issuesync.TicketFields, developer, reporter string) (string, error) {
	column, err := d.firstOfStage(ctx, projectID, workspace.StatusKindBacklog)
	if err != nil {
		return "", err
	}
	opt := tickets.CreateOptions{Status: tickets.Status(column), Reporter: &tickets.Reporter{Kind: tickets.ReporterKindGitHub, Login: reporter}}
	if opt.TypeID, err = d.types.FirstType(ctx, projectID); err != nil {
		return "", err
	}
	if opt.TypeID != "" {
		name, err := d.types.TypeName(ctx, opt.TypeID)
		if err != nil {
			return "", err
		}
		opt.OriginUnknown = tickets.IsBugType(name)
	}
	t, err := d.tickets.Create(ctx, projectID, f.Title, f.Body, "", developer, opt)
	if err != nil {
		return "", err
	}
	return t.ID, d.labels(ctx, t.ID, f.Labels)
}

func (d issueDesk) Apply(ctx context.Context, ticketID string, f issuesync.TicketFields, developer string) error {
	t, err := d.tickets.Get(ctx, ticketID)
	if err != nil {
		return err
	}
	if t.Title != f.Title || t.Body != f.Body {
		if _, err := d.tickets.UpdateTicket(ctx, ticketID, f.Title, f.Body); err != nil {
			return err
		}
	}
	if err := d.labels(ctx, ticketID, f.Labels); err != nil {
		return err
	}
	if developer == "" || developer == t.Developer {
		return nil
	}
	_, err = d.tickets.SetPerson(ctx, ticketID, tickets.RoleDeveloper, developer)
	return err
}

// labels makes the ticket carry exactly want.
func (d issueDesk) labels(ctx context.Context, ticketID string, want []string) error {
	have, err := d.tickets.ListLabels(ctx, ticketID)
	if err != nil {
		return err
	}
	for _, l := range want {
		if slices.Contains(have, l) {
			continue
		}
		if _, err := d.tickets.AddLabel(ctx, ticketID, l); err != nil {
			return err
		}
	}
	for _, l := range have {
		if slices.Contains(want, l) {
			continue
		}
		if _, err := d.tickets.RemoveLabel(ctx, ticketID, l); err != nil {
			return err
		}
	}
	return nil
}

func (d issueDesk) Done(ctx context.Context, ticketID string) (bool, error) {
	t, err := d.tickets.Get(ctx, ticketID)
	if err != nil {
		return false, err
	}
	return ticketDone(ctx, d.statuses, t)
}

func (d issueDesk) MoveTo(ctx context.Context, ticketID string, stage issuesync.Stage) error {
	t, err := d.tickets.Get(ctx, ticketID)
	if err != nil {
		return err
	}
	column, err := d.firstOfStage(ctx, t.ProjectID, workspace.StatusKind(stage))
	if err != nil || column == "" {
		return err
	}
	_, err = d.tickets.UpdateStatus(ctx, ticketID, tickets.Status(column))
	return err
}

// firstOfStage is the project's first column of kind, empty when it has none.
func (d issueDesk) firstOfStage(ctx context.Context, projectID string, kind workspace.StatusKind) (string, error) {
	columns, err := d.statuses.ListByProject(ctx, projectID)
	if err != nil {
		return "", err
	}
	for _, c := range columns {
		if c.Kind == kind {
			return c.ID, nil
		}
	}
	return "", nil
}

// issueThread adapts chat to the sync's Notes and Comments seams; it writes as the server, for the workspace's Owner,
// as crash tickets' comments are.
type issueThread struct {
	tickets  *tickets.Service
	chat     *chat.Service
	projects *storage.ProjectsRepo
	users    *storage.UsersRepo
}

func (n issueThread) thread(ctx context.Context, ticketID string) (conversationID, ownerID string, err error) {
	t, err := n.tickets.Get(ctx, ticketID)
	if err != nil {
		return "", "", err
	}
	project, err := n.projects.Get(ctx, t.ProjectID)
	if err != nil {
		return "", "", err
	}
	owner, err := n.users.FirstWorkspaceOwner(ctx, project.WorkspaceID)
	if err != nil {
		return "", "", fmt.Errorf("find an owner of workspace %s to post for: %w", project.WorkspaceID, err)
	}
	c, err := n.chat.GetOrCreateTicketThread(ctx, project.WorkspaceID, ticketID, owner)
	if err != nil {
		return "", "", err
	}
	return c.ID, owner, nil
}

func (n issueThread) Note(ctx context.Context, ticketID, key, body string) error {
	conversationID, owner, err := n.thread(ctx, ticketID)
	if err != nil {
		return err
	}
	return n.chat.PostSystemNotice(ctx, conversationID, owner, key, body)
}

func (n issueThread) Post(ctx context.Context, ticketID string, c issuesync.Comment) (string, error) {
	conversationID, _, err := n.thread(ctx, ticketID)
	if err != nil {
		return "", err
	}
	id := ids.From(fmt.Sprintf("github-comment\x00%d", c.ID))
	_, err = n.chat.PostExternalMessage(ctx, chat.ExternalPost{
		ID: id, ConversationID: conversationID, AuthorID: githubAuthor(c), Name: c.Author, AvatarURL: c.AuthorAvatarURL, Body: c.Body, Via: "github",
	})
	if errors.Is(err, apperrs.ErrConflict) {
		return id, nil
	}
	return id, err
}

func (n issueThread) Edit(ctx context.Context, messageID string, c issuesync.Comment) error {
	_, err := n.chat.EditMessage(ctx, messageID, githubAuthor(c), c.Body)
	return err
}

func (n issueThread) Delete(ctx context.Context, messageID string, c issuesync.Comment) error {
	return n.chat.DeleteMessage(ctx, messageID, githubAuthor(c))
}

func githubAuthor(c issuesync.Comment) string { return "github:" + c.Author }

// issuePeople maps a GitHub login to the member who signs in with it, while they may open the project.
type issuePeople struct {
	users  *storage.UsersRepo
	access *access.Service
}

func (p issuePeople) Member(ctx context.Context, projectID, githubLogin string) (string, error) {
	userID, found, err := p.users.UserIDByIdentityLogin(ctx, auth.ProviderGitHub, githubLogin)
	if err != nil || !found || !p.access.CanInProject(ctx, userID, projectID, permissions.Member) {
		return "", err
	}
	u, err := p.users.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Login, nil
}

// issueReader reads a synced repository's issues as the router resolves it, the App's installation once a key is set.
type issueReader struct {
	git gitProviderRouter
}

func (r issueReader) ListIssues(ctx context.Context, owner, repo string, q issuesync.IssueQuery) ([]issuesync.Issue, int, error) {
	issues, next, err := r.git.ListIssues(ctx, owner, repo, gitprovider.IssueOpts{State: q.State, Labels: q.Labels, Since: q.Since, Page: q.Page})
	if err != nil {
		return nil, 0, err
	}
	out := make([]issuesync.Issue, 0, len(issues))
	for _, i := range issues {
		out = append(out, issuesync.Issue{
			ID: i.ID, Number: i.Number, Title: i.Title, Body: i.Body, State: i.State, StateReason: i.StateReason,
			Labels: i.Labels, Assignees: i.Assignees, Author: i.Author, HTMLURL: i.HTMLURL, UpdatedAt: i.UpdatedAt,
		})
	}
	return out, next, nil
}

func (r issueReader) ListComments(ctx context.Context, owner, repo string, number int, since time.Time) ([]issuesync.Comment, error) {
	comments, err := r.git.ListIssueComments(ctx, owner, repo, number, since)
	if err != nil {
		return nil, err
	}
	out := make([]issuesync.Comment, 0, len(comments))
	for _, c := range comments {
		out = append(out, issuesync.Comment{ID: c.ID, Body: c.Body, Author: c.Author, AuthorAvatarURL: c.AuthorAvatarURL, UpdatedAt: c.UpdatedAt})
	}
	return out, nil
}

// RateLimit reads the limit of the token the repository is read with; a provider that can't say never pauses the sweep.
func (r issueReader) RateLimit(ctx context.Context, owner, repo string) (issuesync.Rate, error) {
	p, err := r.git.resolve(ctx, owner, repo)
	if err != nil {
		return issuesync.Rate{}, err
	}
	reader, ok := p.(gitprovider.RateReader)
	if !ok {
		return issuesync.Rate{Remaining: int(^uint(0) >> 1)}, nil
	}
	rate, err := reader.RateLimit(ctx)
	return issuesync.Rate{Remaining: rate.Remaining, Reset: rate.Reset}, err
}

// issueBot names the App's account from its slug, so the sync skips what Nexul itself wrote to GitHub.
type issueBot struct {
	configs connectors.AppConfigStore
}

func (b issueBot) BotLogin(ctx context.Context) (string, error) {
	cfg, err := b.configs.GetAppConfig(ctx, githubConnectorID)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && cfg.AppSlug == "") {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return cfg.AppSlug + "[bot]", nil
}

type issueRepos struct {
	workspace gitRepoResolver
}

func (r issueRepos) ProjectOf(ctx context.Context, owner, repo string) (string, error) {
	ref, err := r.workspace.GetRepoByFullName(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	return ref.ProjectID, nil
}
