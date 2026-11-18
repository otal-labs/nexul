package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/imports"
	"github.com/otal-labs/nexul/internal/imports/testutil"
	"github.com/otal-labs/nexul/internal/tickets"
)

// linearFiles serves every file on a host the source allows, as a signed Linear upload link would.
type linearFiles struct{}

func (linearFiles) Fetch(_ context.Context, rawURL string, hosts []string) (string, []byte, error) {
	for _, h := range hosts {
		if strings.HasPrefix(rawURL, "https://"+h+"/") {
			return path.Base(strings.SplitN(rawURL, "?", 2)[0]), []byte("png bytes"), nil
		}
	}
	return "", nil, fmt.Errorf("host not allowed: %s", rawURL)
}

// linearTeamStub is one team with three states, two projects, 120 issues, comments, one blocking relation and an
// uploaded file.
func linearTeamStub() *testutil.LinearStub {
	stub := &testutil.LinearStub{Key: "lin_api_test"}
	stub.AddTeam("team-1", "ENG", "Engineering")
	states := []struct{ name, kind string }{{"Todo", "unstarted"}, {"In Progress", "started"}, {"Done", "completed"}}
	at := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	for n := 1; n <= 120; n++ {
		st := states[n%3]
		issue := testutil.LinearIssue{
			ID: fmt.Sprintf("issue-%d", n), Identifier: fmt.Sprintf("ENG-%d", n), Title: fmt.Sprintf("Linear %d", n),
			Description: "Steps", StateName: st.name, StateType: st.kind, Priority: 2, Labels: []string{"bug"},
			Project: []string{"Mobile", "Web"}[n%2],
		}
		stub.Issues = append(stub.Issues, issue)
		stub.Comments = append(stub.Comments, testutil.LinearComment{
			ID: fmt.Sprintf("comment-%d", n), IssueID: issue.ID, Body: fmt.Sprintf("Seen on %d", n), UserName: "Bob", UserEmail: "bob@example.com", CreatedAt: at,
		})
	}
	first := &stub.Issues[0]
	first.Description = "Broken ![shot](https://uploads.linear.app/ws/abc/shot.png?signature=s1)"
	first.AssigneeName, first.AssigneeEmail, first.DueDate = "Writer", "writer@example.com", "2026-12-24"
	// ENG-110, on the second page, blocks ENG-5 on the first, so the link is made when its blocker comes in.
	stub.Issues[109].Blocks = []string{"issue-5"}
	stub.Issues[109].Related = []string{"issue-6"}
	return stub
}

func newLinearImportService(f permFixture, endpoint string) *imports.Service {
	return imports.NewService(imports.Deps{
		Repo: f.store.Imports, Owners: f.svc.accessSvc, Projects: importProjects{projects: f.store.Projects},
		Board:   importBoard{svc: f.svc.workspaceSvc},
		People:  importPeople{store: f.store.Imports, projects: f.store.Projects, access: f.svc.accessSvc},
		Desk:    newImportDesk(f),
		Files:   linearFiles{},
		Sources: map[string]imports.Source{imports.SourceLinear: imports.NewLinearSource(endpoint, nil)},
	})
}

func runLinearImport(t *testing.T, svc *imports.Service) *imports.Import {
	t.Helper()
	ctx := as(uOwner)
	imp, err := svc.Start(ctx, imports.Start{Source: imports.SourceLinear, ProjectID: pGeneral, Credential: "lin_api_test"})
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

func TestImport_ALinearTeamOf120IssuesMapsEveryField_AndARerunAddsNothing(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	_, err := f.db.ExecContext(t.Context(),
		`INSERT INTO user_identities (user_id, provider, provider_user_id, login, created_at) VALUES (?, 'google', 'g-writer', 'writer@example.com', 0)`, uWriter)
	require.NoError(t, err)
	stub := linearTeamStub()
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)
	svc := newLinearImportService(f, srv.URL)

	imp := runLinearImport(t, svc)
	assert.Equal(t, imports.StateCompleted, imp.State, imp.Report.Error)
	assert.Equal(t, "ENG", imp.From)
	assert.Equal(t, 120, imp.Done)
	assert.Zero(t, imp.Failed)
	assert.Equal(t, 120, imp.Report.Comments)
	assert.Equal(t, 1, imp.Report.RelationsSkipped)
	assert.Equal(t, 1, imp.Report.FilesCopied)
	assert.Empty(t, imp.Credential, "the key is gone once the import ends")
	stored, err := f.store.Imports.Get(t.Context(), imp.ID)
	require.NoError(t, err)
	assert.Empty(t, stored.Credential)

	titles, imported := importedTickets(t, f)
	require.Len(t, imported, 120)
	for title, n := range titles {
		assert.Equal(t, 1, n, title)
	}

	cats, err := f.svc.workspaceSvc.ListCategoriesByProject(as(uOwner), pGeneral)
	require.NoError(t, err)
	catName := map[string]string{}
	for _, c := range cats {
		catName[c.ID] = c.Name
	}
	kinds := map[string]string{}
	for _, tk := range imported {
		status, err := f.svc.workspaceSvc.GetStatus(as(uOwner), string(tk.Status))
		require.NoError(t, err)
		kinds[status.Name] = string(status.Kind)
	}
	assert.Equal(t, map[string]string{"Todo": "backlog", "In progress": "progress", "Done": "done"}, kinds,
		"each state lands in the column of its name, ignoring case, else a new one of the kind its type maps to")

	first := titled(t, imported, "Linear 1")
	assert.Equal(t, uWriter, first.Developer, "the assignee is matched by email")
	assert.Equal(t, "2026-12-24", first.DueOn)
	assert.Equal(t, "Web", catName[first.CategoryID])
	assert.Equal(t, "Mobile", catName[titled(t, imported, "Linear 2").CategoryID])
	assert.ElementsMatch(t, []string{"bug", "priority:high"}, first.Labels)
	assert.Contains(t, first.Body, "Imported from [ENG-1](https://linear.app/acme/issue/ENG-1).")
	assert.Contains(t, first.Body, "![shot](/api/attachments/", "the uploaded file is copied and the body points at the copy")
	assert.Equal(t, tickets.Reporter{Kind: tickets.ActorKindImport, Login: uOwner, Source: imports.SourceLinear}, first.Reporter)

	var via, author string
	require.NoError(t, f.db.QueryRowContext(t.Context(),
		`SELECT m.via, m.author_name FROM messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.ticket_id = ? AND m.author_kind = 'external'`,
		first.ID).Scan(&via, &author))
	assert.Equal(t, "linear", via)
	assert.Equal(t, "Bob", author)

	blocked := titled(t, imported, "Linear 5")
	links, err := f.svc.ticketsSvc.Links(as(uOwner), blocked.ID)
	require.NoError(t, err)
	require.Len(t, links.BlockedBy, 1)
	assert.Equal(t, titled(t, imported, "Linear 110").ID, links.BlockedBy[0].ID)
	related, err := f.svc.ticketsSvc.Links(as(uOwner), titled(t, imported, "Linear 6").ID)
	require.NoError(t, err)
	assert.Empty(t, related.BlockedBy, "a related relation is counted and skipped")

	again := runLinearImport(t, svc)
	assert.Equal(t, imports.StateCompleted, again.State, again.Report.Error)
	assert.Equal(t, 120, again.Report.Skipped)
	_, imported = importedTickets(t, f)
	assert.Len(t, imported, 120, "a re-run adds no ticket")
	assert.Equal(t, 1, threadMessages(t, f, first.ID), "nor a comment")
	links, err = f.svc.ticketsSvc.Links(as(uOwner), blocked.ID)
	require.NoError(t, err)
	assert.Len(t, links.BlockedBy, 1, "nor a link")
	after, err := f.svc.workspaceSvc.ListCategoriesByProject(as(uOwner), pGeneral)
	require.NoError(t, err)
	assert.Len(t, after, len(cats), "nor a category")
	var files int
	require.NoError(t, f.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM attachments WHERE ticket_id = ?`, first.ID).Scan(&files))
	assert.Equal(t, 1, files, "nor a file")
	assert.True(t, slices.ContainsFunc(stub.Requests(), func(r testutil.LinearRequest) bool { return r.After == "cursor-99" }), "the second page was read by its cursor")
}
