package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/imports"
	"github.com/otal-labs/nexul/internal/issuesync"
	"github.com/otal-labs/nexul/internal/tickets"
)

// fakeRepoIssues is one repository's issues, served a hundred to a page as GitHub does.
type fakeRepoIssues struct {
	issues   []imports.GitHubIssue
	comments map[int][]imports.GitHubComment
}

func newFakeRepoIssues(n int) *fakeRepoIssues {
	f := &fakeRepoIssues{comments: map[int][]imports.GitHubComment{}}
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for i := 1; i <= n; i++ {
		issue := imports.GitHubIssue{
			ID: int64(9000 + i), Number: i, Title: fmt.Sprintf("Issue %d", i), Body: "Steps to reproduce", State: "open",
			Labels: []string{"bug"}, Author: "alice", HTMLURL: fmt.Sprintf("https://github.com/acme/app/issues/%d", i),
		}
		if i%2 == 0 {
			issue.State = "closed"
		}
		comments := []imports.GitHubComment{{ID: int64(70000 + i), Body: fmt.Sprintf("Seen on %d", i), Author: "bob", CreatedAt: at}}
		if i%10 == 0 {
			comments = append(comments, imports.GitHubComment{ID: int64(80000 + i), Body: "Again", Author: "carol", CreatedAt: at})
		}
		f.comments[i] = comments
		issue.Comments = len(comments)
		f.issues = append(f.issues, issue)
	}
	return f
}

func (f *fakeRepoIssues) ListIssues(_ context.Context, _, _ string, page int) ([]imports.GitHubIssue, int, error) {
	from := (page - 1) * 100
	if from >= len(f.issues) {
		return nil, 0, nil
	}
	to := min(from+100, len(f.issues))
	next := page + 1
	if to == len(f.issues) {
		next = 0
	}
	return f.issues[from:to], next, nil
}

func (f *fakeRepoIssues) ListComments(_ context.Context, _, _ string, number int) ([]imports.GitHubComment, error) {
	return f.comments[number], nil
}

type oneRepo string

func (r oneRepo) Repos(context.Context, string) ([]string, error) { return []string{string(r)}, nil }

// crashingDesk files tickets until it has filed crashAt of them, then stops the server right after the last commit,
// before the import could record it.
type crashingDesk struct {
	imports.Desk
	crashAt int
	filed   int
	crash   context.CancelFunc
}

func (d *crashingDesk) FileTicket(ctx context.Context, t imports.NewTicket) error {
	if err := d.Desk.FileTicket(ctx, t); err != nil {
		return err
	}
	if d.filed++; d.filed == d.crashAt {
		d.crash()
		return context.Canceled
	}
	return nil
}

func newImportService(f permFixture, issues *fakeRepoIssues, desk imports.Desk) *imports.Service {
	return imports.NewService(imports.Deps{
		Repo: f.store.Imports, Owners: f.svc.accessSvc, Projects: importProjects{projects: f.store.Projects},
		Board:   importBoard{svc: f.svc.workspaceSvc},
		People:  importPeople{store: f.store.Imports, projects: f.store.Projects, access: f.svc.accessSvc},
		Desk:    desk,
		Sources: map[string]imports.Source{imports.SourceGitHub: imports.NewGitHubSource(issues, oneRepo("acme/app"), importIssueSyncs{repo: f.store.IssueSync})},
	})
}

func newImportDesk(f permFixture) importDesk {
	return importDesk{tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, attachments: f.svc.attachmentsSvc, types: f.store.TicketTypes, projects: f.store.Projects}
}

// runImport scans and confirms an import of the fake repository and runs it to its end.
func runImport(t *testing.T, svc *imports.Service) *imports.Import {
	t.Helper()
	ctx := as(uOwner)
	imp, err := svc.Start(ctx, imports.Start{Source: imports.SourceGitHub, ProjectID: pGeneral})
	require.NoError(t, err)
	svc.Pass(t.Context())
	imp, err = svc.Get(ctx, imp.ID)
	require.NoError(t, err)
	require.Equal(t, imports.StateReady, imp.State, imp.Report.Error)
	_, err = svc.Confirm(ctx, imp.ID)
	require.NoError(t, err)
	svc.Pass(t.Context())
	imp, err = svc.Get(ctx, imp.ID)
	require.NoError(t, err)
	return imp
}

// importedTickets maps each imported title to how many tickets carry it.
func importedTickets(t *testing.T, f permFixture) (map[string]int, []*tickets.Ticket) {
	t.Helper()
	all, err := f.svc.ticketsSvc.ListByProject(as(uOwner), pGeneral)
	require.NoError(t, err)
	titles := map[string]int{}
	var imported []*tickets.Ticket
	for _, tk := range all {
		if tk.Reporter.Imported() {
			titles[tk.Title]++
			imported = append(imported, tk)
		}
	}
	return titles, imported
}

func threadMessages(t *testing.T, f permFixture, ticketID string) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.ticket_id = ? AND m.author_kind = 'external'`,
		ticketID).Scan(&n))
	return n
}

func TestImport_AGitHubRepositoryOf250IssuesBecomes250TicketsWithTheirComments_AndARerunCreatesNone(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	issues := newFakeRepoIssues(250)
	svc := newImportService(f, issues, newImportDesk(f))

	imp := runImport(t, svc)
	assert.Equal(t, imports.StateCompleted, imp.State, imp.Report.Error)
	assert.Equal(t, 250, imp.Total)
	assert.Equal(t, 250, imp.Done)
	assert.Zero(t, imp.Failed)
	assert.Equal(t, 275, imp.Report.Comments)
	titles, imported := importedTickets(t, f)
	require.Len(t, imported, 250)
	for title, n := range titles {
		assert.Equal(t, 1, n, title)
	}
	tenth := titled(t, imported, "Issue 10")
	assert.Equal(t, 2, threadMessages(t, f, tenth.ID))
	assert.Equal(t, tickets.Reporter{Kind: tickets.ActorKindImport, Login: uOwner, Source: imports.SourceGitHub}, tenth.Reporter)
	assert.Equal(t, []string{"bug"}, tenth.Labels)
	assert.Contains(t, tenth.Body, "Imported from [acme/app#10](https://github.com/acme/app/issues/10), opened by alice.")
	status, err := f.svc.workspaceSvc.GetStatus(as(uOwner), string(tenth.Status))
	require.NoError(t, err)
	assert.Equal(t, "done", string(status.Kind), "a closed issue lands in a done column")

	again := runImport(t, svc)
	assert.Equal(t, imports.StateCompleted, again.State, again.Report.Error)
	assert.Equal(t, 250, again.Report.Skipped)
	_, imported = importedTickets(t, f)
	assert.Len(t, imported, 250, "a re-run creates none")
	assert.Equal(t, 2, threadMessages(t, f, tenth.ID))
}

func TestImport_ACrashMidwayResumesOnRestartWithNoDuplicates(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	issues := newFakeRepoIssues(250)
	running, crash := context.WithCancel(t.Context())
	desk := &crashingDesk{Desk: newImportDesk(f), crashAt: 130, crash: crash}
	first := newImportService(f, issues, desk)
	ctx := as(uOwner)
	imp, err := first.Start(ctx, imports.Start{Source: imports.SourceGitHub, ProjectID: pGeneral})
	require.NoError(t, err)
	first.Pass(t.Context())
	_, err = first.Confirm(ctx, imp.ID)
	require.NoError(t, err)
	first.Pass(running)
	stopped, err := first.Get(ctx, imp.ID)
	require.NoError(t, err)
	require.Equal(t, imports.StateImporting, stopped.State, "a crash leaves the import to resume")
	require.Equal(t, "2", stopped.Report.Cursor, "the first page was checkpointed")

	restarted := newImportService(f, issues, newImportDesk(f))
	restarted.Pass(t.Context())
	done, err := restarted.Get(ctx, imp.ID)
	require.NoError(t, err)
	assert.Equal(t, imports.StateCompleted, done.State, done.Report.Error)
	assert.Equal(t, 250, done.Done)
	titles, imported := importedTickets(t, f)
	assert.Len(t, imported, 250)
	for title, n := range titles {
		assert.Equal(t, 1, n, title)
	}
	crashed := titled(t, imported, "Issue 130")
	assert.Equal(t, 2, threadMessages(t, f, crashed.ID), "the ticket filed just before the crash got each of its two comments once")
}

func TestImport_AnAssigneeWhoIsAMemberBecomesTheDeveloper_AndAStrangerIsNamedInTheBody(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	issues := newFakeRepoIssues(2)
	issues.issues[0].Assignees = []string{"U-Writer"}
	issues.issues[1].Assignees = []string{"zed"}
	svc := newImportService(f, issues, newImportDesk(f))
	ctx := as(uOwner)
	imp, err := svc.Start(ctx, imports.Start{Source: imports.SourceGitHub, ProjectID: pGeneral})
	require.NoError(t, err)
	svc.Pass(t.Context())
	scanned, err := svc.Get(ctx, imp.ID)
	require.NoError(t, err)
	assert.Equal(t, []imports.PersonMatch{
		{Person: imports.Person{Login: "U-Writer", Name: "U-Writer"}, Member: uWriter},
		{Person: imports.Person{Login: "zed", Name: "zed"}},
	}, scanned.Report.People)

	_, err = svc.Confirm(ctx, imp.ID)
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, imported := importedTickets(t, f)
	require.Len(t, imported, 2)
	first, second := titled(t, imported, "Issue 1"), titled(t, imported, "Issue 2")
	assert.Equal(t, uWriter, first.Developer)
	assert.Empty(t, second.Developer)
	assert.Contains(t, second.Body, "Assigned in GitHub to zed")
}

func TestImport_OnlyTheWorkspaceOwnerRunsOne_AndOneAtATime(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	svc := newImportService(f, newFakeRepoIssues(1), newImportDesk(f))
	_, err := svc.Start(as(uWriter), imports.Start{Source: imports.SourceGitHub, ProjectID: pGeneral})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Owner")

	_, err = svc.Start(as(uOwner), imports.Start{Source: imports.SourceGitHub, ProjectID: pGeneral})
	require.NoError(t, err)
	_, err = svc.Start(as(uOwner), imports.Start{Source: imports.SourceGitHub, ProjectID: pGeneral})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has an import")
}

func titled(t *testing.T, ts []*tickets.Ticket, title string) *tickets.Ticket {
	t.Helper()
	for _, tk := range ts {
		if tk.Title == title {
			return tk
		}
	}
	t.Fatalf("no ticket titled %s", title)
	return nil
}

func TestImport_ARepositoryWhoseIssueSyncIsOnIsRefused(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	require.NoError(t, f.store.IssueSync.PutSync(t.Context(), issuesync.Sync{
		ProjectID: pGeneral, Owner: "acme", Repo: "app", Label: issuesync.DefaultLabel, Mode: issuesync.ModeLabel, Enabled: true,
	}))
	svc := newImportService(f, newFakeRepoIssues(3), newImportDesk(f))
	imp, err := svc.Start(as(uOwner), imports.Start{Source: imports.SourceGitHub, ProjectID: pGeneral})
	require.NoError(t, err)
	svc.Pass(t.Context())
	got, err := svc.Get(as(uOwner), imp.ID)
	require.NoError(t, err)
	assert.Equal(t, imports.StateFailed, got.State)
	assert.Contains(t, got.Report.Error, "issue sync is on for acme/app")
	_, imported := importedTickets(t, f)
	assert.Empty(t, imported)
}
