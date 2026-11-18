package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/otal-labs/nexul/internal/access"
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
	_ issuesync.Tickets      = issueSyncTickets{}
	_ issuesync.TicketFields = issueSyncTickets{}
	_ issuesync.Thread       = issueSyncThread{}
	_ issuesync.Comments     = issueSyncThread{}
	_ issuesync.Members      = issueSyncMembers{}
	_ issuesync.Host         = issueSyncHost{}
	_ issuesync.ProjectRepos = issueSyncRepos{}
)

// newIssueSync wires the GitHub issue sync over the tickets, chat and git provider it reads and writes (ADR 0017).
func newIssueSync(store *storage.Store, t *tickets.Service, c *chat.Service, w *workspace.Service, a *access.Service, git gitprovider.GitProvider, appConfigs connectors.AppConfigStore, log *slog.Logger) *issuesync.Service {
	desk := issueSyncTickets{tickets: t, statuses: store.Statuses}
	thread := issueSyncThread{tickets: t, chat: c, projects: store.Projects, users: store.Users}
	return issuesync.NewService(issuesync.Deps{
		Repo: store.IssueSync, Tickets: desk, Fields: desk, Thread: thread, Comments: thread,
		Members: issueSyncMembers{identities: store.IssueSync, users: store.Users, access: a},
		Host:    issueSyncHost{git: git, appConfigs: appConfigs},
		Gate:    a, Repos: issueSyncRepos{workspace: w}, Log: log,
	})
}

// issueSyncTickets adapts tickets to the sync's Tickets and TicketFields seams; it runs as the server, so the writes
// pass the gate as the sync's own.
type issueSyncTickets struct {
	tickets  *tickets.Service
	statuses *storage.StatusesRepo
}

func (d issueSyncTickets) Get(ctx context.Context, id string) (issuesync.Ticket, error) {
	t, err := d.tickets.Get(ctx, id)
	if err != nil {
		return issuesync.Ticket{}, err
	}
	done, err := ticketDone(ctx, d.statuses, t)
	if err != nil {
		return issuesync.Ticket{}, err
	}
	return issuesync.Ticket{ID: t.ID, ProjectID: t.ProjectID, Title: t.Title, Body: t.Body, Labels: t.Labels, Developer: t.Developer, Done: done}, nil
}

func (d issueSyncTickets) Create(ctx context.Context, projectID string, n issuesync.NewTicket) (string, error) {
	t, err := d.tickets.Create(ctx, projectID, n.Title, n.Body, "", "", tickets.CreateOptions{GitHubReporter: n.Reporter})
	if err != nil {
		return "", err
	}
	return t.ID, d.moveTo(ctx, t, issuesync.StageBacklog)
}

func (d issueSyncTickets) Move(ctx context.Context, id string, stage issuesync.Stage) error {
	t, err := d.tickets.Get(ctx, id)
	if err != nil {
		return err
	}
	return d.moveTo(ctx, t, stage)
}

// moveTo puts t in its project's first column of stage; a board without one leaves it where it is.
func (d issueSyncTickets) moveTo(ctx context.Context, t *tickets.Ticket, stage issuesync.Stage) error {
	columns, err := d.statuses.ListByProject(ctx, t.ProjectID)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(columns, func(c *workspace.Status) bool { return string(c.Kind) == string(stage) })
	if i < 0 || tickets.Status(columns[i].ID) == t.Status {
		return nil
	}
	_, err = d.tickets.UpdateStatus(ctx, t.ID, tickets.Status(columns[i].ID))
	return err
}

func (d issueSyncTickets) SetText(ctx context.Context, id, title, body string) error {
	_, err := d.tickets.ReplaceText(ctx, id, title, body)
	return err
}

// SetLabels adds and removes labels so the ticket carries exactly labels, ignoring case.
func (d issueSyncTickets) SetLabels(ctx context.Context, id string, labels []string) error {
	current, err := d.tickets.ListLabels(ctx, id)
	if err != nil {
		return err
	}
	for _, l := range current {
		if !slices.ContainsFunc(labels, func(want string) bool { return strings.EqualFold(want, l) }) {
			if _, err := d.tickets.RemoveLabel(ctx, id, l); err != nil {
				return err
			}
		}
	}
	for _, l := range labels {
		if !slices.ContainsFunc(current, func(have string) bool { return strings.EqualFold(have, l) }) {
			if _, err := d.tickets.AddLabel(ctx, id, l); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d issueSyncTickets) SetDeveloper(ctx context.Context, id, login string) error {
	_, err := d.tickets.SetPerson(ctx, id, tickets.RoleDeveloper, login)
	return err
}

// issueSyncThread adapts chat to the sync's Thread and Comments seams, creating the ticket's thread for the workspace's
// Owner when it has none, as crash notes do.
type issueSyncThread struct {
	tickets  *tickets.Service
	chat     *chat.Service
	projects *storage.ProjectsRepo
	users    *storage.UsersRepo
}

func (n issueSyncThread) Note(ctx context.Context, ticketID, key, body string) error {
	thread, owner, err := n.thread(ctx, ticketID)
	if err != nil {
		return err
	}
	return n.chat.PostSystemNotice(ctx, thread, owner, key, body)
}

// Post derives the message id from the comment, so a repeat delivery finds the message it posted.
func (n issueSyncThread) Post(ctx context.Context, ticketID string, c issuesync.Comment) (string, error) {
	thread, _, err := n.thread(ctx, ticketID)
	if err != nil {
		return "", err
	}
	m, err := n.chat.PostExternalMessage(ctx, chat.ExternalPost{
		ID: ids.From("github-comment\x00" + strconv.FormatInt(c.ID, 10)), ConversationID: thread,
		Via: "github", Login: c.Author, AvatarURL: c.AuthorAvatarURL, Body: c.Body,
	})
	if err != nil {
		return "", err
	}
	return m.ID, nil
}

func (n issueSyncThread) Edit(ctx context.Context, messageID string, c issuesync.Comment) error {
	_, err := n.chat.EditExternalMessage(ctx, messageID, chat.ExternalAuthorID("github", c.Author), c.Body)
	return err
}

func (n issueSyncThread) Delete(ctx context.Context, messageID, author string) error {
	return n.chat.DeleteMessage(ctx, messageID, chat.ExternalAuthorID("github", author))
}

func (n issueSyncThread) thread(ctx context.Context, ticketID string) (conversationID, ownerID string, err error) {
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
	thread, err := n.chat.GetOrCreateTicketThread(ctx, project.WorkspaceID, ticketID, owner)
	if err != nil {
		return "", "", err
	}
	return thread.ID, owner, nil
}

// issueSyncMembers maps a GitHub login to the project member who signed in with it.
type issueSyncMembers struct {
	identities *storage.IssueSyncRepo
	users      *storage.UsersRepo
	access     *access.Service
}

func (m issueSyncMembers) MemberLogin(ctx context.Context, projectID, githubLogin string) (string, bool, error) {
	userID, ok, err := m.identities.GitHubUserID(ctx, githubLogin)
	if err != nil || !ok || !m.access.CanInProject(ctx, userID, projectID, permissions.Member) {
		return "", false, err
	}
	u, err := m.users.GetUserByID(ctx, userID)
	if err != nil {
		return "", false, fmt.Errorf("read the login of %s: %w", userID, err)
	}
	return u.Login, true, nil
}

// issueSyncHost reads issues through the git provider router, and the App's bot login from its stored slug.
type issueSyncHost struct {
	git        gitprovider.GitProvider
	appConfigs connectors.AppConfigStore
}

func (h issueSyncHost) BotLogin(ctx context.Context) (string, error) {
	cfg, err := h.appConfigs.GetAppConfig(ctx, githubConnectorID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil || cfg.AppSlug == "" {
		return "", err
	}
	return cfg.AppSlug + "[bot]", nil
}

func (h issueSyncHost) ListIssues(ctx context.Context, owner, repo string, q issuesync.IssueQuery) (issuesync.IssuePage, error) {
	p, err := h.git.ListIssues(ctx, owner, repo, gitprovider.IssueOpts{State: q.State, Labels: q.Labels, Since: q.Since, Page: q.Page})
	if err != nil {
		return issuesync.IssuePage{}, err
	}
	out := issuesync.IssuePage{NextPage: p.NextPage, RateRemaining: p.Rate.Remaining, RateReset: p.Rate.Reset}
	for _, i := range p.Issues {
		out.Issues = append(out.Issues, issuesync.Issue{
			ID: i.ID, Number: i.Number, Title: i.Title, Body: i.Body, State: i.State, StateReason: i.StateReason,
			Labels: i.Labels, Assignees: i.Assignees, Author: i.Author, UpdatedAt: i.UpdatedAt,
		})
	}
	return out, nil
}

// issueSyncRepos answers whether a repository is attached to a project.
type issueSyncRepos struct {
	workspace *workspace.Service
}

func (r issueSyncRepos) HasRepo(ctx context.Context, projectID, owner, repo string) (bool, error) {
	ref, err := r.workspace.GetRepoByFullName(ctx, owner, repo)
	if errors.Is(err, apperrs.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return ref.ProjectID == projectID, nil
}
