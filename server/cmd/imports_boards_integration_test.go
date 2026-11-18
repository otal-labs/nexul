package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/imports"
)

// stubJSON serves each path's JSON body; a key written path?key=value answers only a query holding that pair, ahead
// of the bare path.
func stubJSON(t *testing.T, bodies map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, body := range bodies {
			p, q, ok := strings.Cut(key, "?")
			if ok && p == r.URL.Path && r.URL.Query().Get(strings.Split(q, "=")[0]) == strings.Split(q, "=")[1] {
				_, _ = w.Write([]byte(body)) // test server: write errors are irrelevant
				return
			}
		}
		if body, ok := bodies[r.URL.Path]; ok {
			_, _ = w.Write([]byte(body)) // test server: write errors are irrelevant
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// anyFile serves every file a source links to.
type anyFile struct{}

func (anyFile) Fetch(_ context.Context, rawURL string, _ []string, _ http.Header) (string, []byte, error) {
	return path.Base(strings.SplitN(rawURL, "?", 2)[0]), []byte("file bytes"), nil
}

func newSourceImports(f permFixture, name string, src imports.Source) *imports.Service {
	return imports.NewService(imports.Deps{
		Repo: f.store.Imports, Owners: f.svc.accessSvc, Projects: importProjects{projects: f.store.Projects},
		Board:   importBoard{svc: f.svc.workspaceSvc},
		People:  importPeople{store: f.store.Imports, projects: f.store.Projects, access: f.svc.accessSvc},
		Desk:    newImportDesk(f),
		Files:   anyFile{},
		Sources: map[string]imports.Source{name: src},
	})
}

func runSourceImport(t *testing.T, svc *imports.Service, source, from, credential string) *imports.Import {
	t.Helper()
	ctx := as(uOwner)
	imp, err := svc.Start(ctx, imports.Start{Source: source, ProjectID: pGeneral, From: from, Credential: credential})
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

func columnNames(t *testing.T, f permFixture) map[string]string {
	t.Helper()
	statuses, err := f.svc.workspaceSvc.ListStatusesByProject(as(uOwner), pGeneral)
	require.NoError(t, err)
	out := map[string]string{}
	for _, st := range statuses {
		out[st.ID] = st.Name
	}
	return out
}

func ticketFiles(t *testing.T, f permFixture, ticketID string) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM attachments WHERE ticket_id = ?`, ticketID).Scan(&n))
	return n
}

func TestImport_ATrelloBoardComesInWithColumnsLabelsCommentsAndFiles_ArchivedOnlyCounted_AndARerunCreatesNone(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	srv := stubJSON(t, map[string]string{
		"/boards/abc":        `{"id":"b1","name":"Roadmap","shortLink":"abc"}`,
		"/boards/b1/lists":   `[{"id":"L1","name":"To do"},{"id":"L2","name":"Review it"},{"id":"L3","name":"Shipped"},{"id":"L4","name":"Old","closed":true}]`,
		"/boards/b1/members": `[{"id":"m1","username":"` + uWriter + `","fullName":"Writer"}]`,
		"/boards/b1/cards/all": `[
			{"id":"c3","name":"Login page","desc":"Two fields","idList":"L2","idMembers":["m1"],"due":"2026-12-24T12:00:00.000Z",
			 "shortUrl":"https://trello.com/c/c3","labels":[{"name":"ui","color":"blue"}],"badges":{"comments":1},
			 "checklists":[{"name":"Steps","checkItems":[{"name":"Draw","state":"complete"}]}],
			 "attachments":[{"name":"mock.png","url":"https://trello.com/1/cards/c3/attachments/a1/download/mock.png","isUpload":true}]},
			{"id":"c2","name":"Shipped thing","idList":"L3"},
			{"id":"c1","name":"Archived card","idList":"L1","closed":true},
			{"id":"c0","name":"On an archived list","idList":"L4"}]`,
		"/cards/c3/actions": `[{"id":"act1","date":"2026-11-01T10:00:00Z","data":{"text":"Ship it"},"memberCreator":{"username":"bob","fullName":"Bob Roe"}}]`,
	})
	svc := newSourceImports(f, imports.SourceTrello, imports.NewTrelloSource(srv.URL, srv.Client()))

	imp := runSourceImport(t, svc, imports.SourceTrello, "https://trello.com/b/abc/roadmap", "key:token")
	assert.Equal(t, imports.StateCompleted, imp.State, imp.Report.Error)
	assert.Equal(t, 2, imp.Done)
	assert.Equal(t, 3, imp.Report.Archived, "the archived list, its card and an archived card are counted")
	assert.Equal(t, 1, imp.Report.FilesCopied)
	titles, imported := importedTickets(t, f)
	require.Len(t, imported, 2)
	assert.NotContains(t, titles, "Archived card")
	login := titled(t, imported, "Login page")
	cols := columnNames(t, f)
	assert.Equal(t, "Review it", cols[string(login.Status)], "a list becomes a column of its name")
	assert.Equal(t, "Shipped", cols[string(titled(t, imported, "Shipped thing").Status)])
	assert.Equal(t, []string{"ui"}, login.Labels)
	assert.Equal(t, uWriter, login.Developer, "a Trello username matches a member's login")
	assert.Equal(t, "2026-12-24", login.DueOn)
	assert.Contains(t, login.Body, "- [x] Draw")
	assert.Contains(t, login.Body, "[mock.png](/api/attachments/")
	assert.Equal(t, 1, threadMessages(t, f, login.ID))
	assert.Equal(t, 1, ticketFiles(t, f, login.ID))

	again := runSourceImport(t, svc, imports.SourceTrello, "abc", "key:token")
	assert.Equal(t, imports.StateCompleted, again.State, again.Report.Error)
	assert.Equal(t, 2, again.Report.Skipped)
	_, imported = importedTickets(t, f)
	assert.Len(t, imported, 2, "a re-run creates none")
	assert.Equal(t, 1, threadMessages(t, f, login.ID))
	assert.Equal(t, 1, ticketFiles(t, f, login.ID))
}

func TestImport_AnAsanaProjectComesInWithSectionsTagsCommentsFilesAndSubtasks_AndARerunCreatesNone(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	task := func(gid, name, extra string) string {
		return `{"gid":"` + gid + `","name":"` + name + `","permalink_url":"https://app.asana.com/0/200/` + gid + `"` + extra + `}`
	}
	empty := `{"data":[],"next_page":null}`
	srv := stubJSON(t, map[string]string{
		"/projects/200":          `{"data":{"gid":"200","name":"Roadmap"}}`,
		"/projects/200/sections": `{"data":[{"gid":"401","name":"Ideas"},{"gid":"402","name":"Doing"},{"gid":"403","name":"Done"}],"next_page":null}`,
		"/tasks?offset=o2":       `{"data":[` + task("303", "Launch", `,"completed":true`) + `],"next_page":null}`,
		"/tasks": `{"data":[` + task("301", "Design", `,"num_subtasks":1,"due_on":"2026-12-05","tags":[{"name":"ui"}],`+
			`"memberships":[{"project":{"gid":"200"},"section":{"gid":"402"}}]`) + `],"next_page":{"offset":"o2"}}`,
		"/tasks/301/subtasks":     `{"data":[` + task("302", "Colours", "") + `],"next_page":null}`,
		"/tasks/301/stories":      `{"data":[{"gid":"501","type":"comment","text":"Ready","created_at":"2026-11-03T09:00:00Z","created_by":{"name":"Bob Roe"}}],"next_page":null}`,
		"/tasks/302/stories":      empty,
		"/tasks/303/stories":      empty,
		"/attachments?parent=301": `{"data":[{"gid":"601","name":"mock.png","host":"asana","download_url":"https://asana-user-private-us-east-1.s3.amazonaws.com/a/mock.png?sig=1","permanent_url":"https://app.asana.com/x/601"}],"next_page":null}`,
		"/attachments":            empty,
	})
	svc := newSourceImports(f, imports.SourceAsana, imports.NewAsanaSource(srv.URL, srv.Client()))

	imp := runSourceImport(t, svc, imports.SourceAsana, "200", "pat")
	assert.Equal(t, imports.StateCompleted, imp.State, imp.Report.Error)
	assert.Equal(t, 3, imp.Done)
	assert.Equal(t, 1, imp.Report.FilesCopied)
	_, imported := importedTickets(t, f)
	require.Len(t, imported, 3)
	cols := columnNames(t, f)
	design, colours, launch := titled(t, imported, "Design"), titled(t, imported, "Colours"), titled(t, imported, "Launch")
	assert.Equal(t, "Doing", cols[string(design.Status)], "a section becomes a column of its name")
	assert.Equal(t, "Done", cols[string(launch.Status)], "a completed task goes to the done column")
	assert.Equal(t, []string{"ui"}, design.Labels)
	assert.Equal(t, "2026-12-05", design.DueOn)
	assert.Equal(t, 1, threadMessages(t, f, design.ID))
	assert.Equal(t, 1, ticketFiles(t, f, design.ID))
	project, err := f.store.Projects.Get(t.Context(), pGeneral)
	require.NoError(t, err)
	assert.Contains(t, colours.Body, fmt.Sprintf("Subtask of %s-%d", project.Prefix, design.Number), "a subtask names its parent's key")

	again := runSourceImport(t, svc, imports.SourceAsana, "200", "pat")
	assert.Equal(t, imports.StateCompleted, again.State, again.Report.Error)
	assert.Equal(t, 3, again.Report.Skipped)
	_, imported = importedTickets(t, f)
	assert.Len(t, imported, 3, "a re-run creates none")
	assert.Equal(t, 1, ticketFiles(t, f, design.ID))
}
