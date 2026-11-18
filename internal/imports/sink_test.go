package imports

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runToEnd scans, confirms and runs one import of src and returns it as stored.
func runToEnd(t *testing.T, svc *Service, repo *fakeRepo) Import {
	t.Helper()
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, err = svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)
	svc.Pass(t.Context())
	return repo.stored(imp.ID)
}

func TestSink_FilesEachTicketWithItsCommentsFilesAndPeople(t *testing.T) {
	at := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)
	src := fakeSource{tickets: []Ticket{{
		SourceID: "1", Title: "Crash", Body: "![a](https://files.example.com/a.png) ![b](https://files.example.com/huge.mov) ![c](https://elsewhere.example.com/c.png)",
		Status: StatusRef{Name: "In QA", Kind: StatusTesting}, Labels: []string{"bug"}, Assignee: Person{Login: "Alice"},
		Comments: []Comment{{SourceID: "c1", Body: "same as ![a](https://files.example.com/a.png)", Author: Person{Login: "bob"}, At: at}, {SourceID: "c2", Body: "  "}},
		Files:    []string{"https://files.example.com/a.png", "https://files.example.com/huge.mov", "https://elsewhere.example.com/c.png", "https://files.example.com/a.png"},
	}, {
		SourceID: "2", Title: "", Assignee: Person{Login: "zed", Name: "Zed Z"},
	}}}
	svc, repo, desk := newDeskService(src, newFakeDesk())
	got := runToEnd(t, svc, repo)

	assert.Equal(t, StateCompleted, got.State)
	assert.Equal(t, 2, got.Done)
	assert.Equal(t, 1, got.Report.FilesCopied)
	assert.Equal(t, 2, got.Report.FilesLinked)
	first := desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "1")]
	assert.Equal(t, "alice", first.Developer)
	assert.Equal(t, "col-In QA", first.Status, "a status the project lacks becomes a new column of its kind")
	assert.Equal(t, []string{"bug"}, first.Labels)
	assert.Equal(t, "fake", first.Source)
	assert.Equal(t, "![a](/api/attachments/att-a.png) ![b](https://files.example.com/huge.mov) ![c](https://elsewhere.example.com/c.png)", first.Body)
	comments := desk.comments[first.ID]
	require.Len(t, comments, 1, "an empty comment is left out")
	assert.Equal(t, NewComment{
		ID: derivedID(testWorkspace, "fake", KindComment, "c1"), AuthorKey: "fake:bob", Name: "bob", Body: "same as ![a](/api/attachments/att-a.png)", Via: "fake", At: at,
	}, comments[0])

	second := desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "2")]
	assert.Equal(t, "Untitled 2", second.Title)
	assert.Empty(t, second.Developer)
	assert.Equal(t, "Assigned in Fake to Zed Z", second.Body)
}

func TestSink_NamesTheParentsKeyAndKeepsTheDueDate(t *testing.T) {
	src := fakeSource{tickets: []Ticket{
		{SourceID: "p", Title: "Parent", DueOn: "2026-12-01"},
		{SourceID: "s", Title: "Child", Body: "Notes", Parent: "p"},
		{SourceID: "o", Title: "Orphan", Parent: "never-came-in"},
	}}
	svc, repo, desk := newDeskService(src, newFakeDesk())
	runToEnd(t, svc, repo)
	assert.Equal(t, "2026-12-01", desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "p")].DueOn)
	assert.Equal(t, "Notes\n\nSubtask of REF-Parent", desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "s")].Body)
	assert.Empty(t, desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "o")].Body, "a parent that never came in is not named")
}

// headerFiles records the header each download was signed with.
type headerFiles struct{ got map[string]http.Header }

func (f headerFiles) Fetch(_ context.Context, rawURL string, _ []string, header http.Header) (string, []byte, error) {
	f.got[rawURL] = header
	return "f", []byte("bytes"), nil
}

type signedSource struct{ fakeSource }

func (signedSource) Info() Info {
	return Info{Title: "Signed", FileHosts: []string{"api.example.com", "elsewhere.example.com"}, FileHeader: func(credential string, u *url.URL) http.Header {
		if u.Host != "api.example.com" {
			return nil
		}
		return http.Header{"Authorization": {"Token " + credential}}
	}}
}

func TestSink_SignsADownloadFromTheSourcesOwnAPIAlone(t *testing.T) {
	files := headerFiles{got: map[string]http.Header{}}
	src := signedSource{fakeSource{tickets: []Ticket{{SourceID: "1", Title: "One", Files: []string{"https://api.example.com/f", "https://elsewhere.example.com/f"}}}}}
	repo := newFakeRepo()
	svc := NewService(Deps{
		Repo: repo, Owners: fakeOwners{owner: testOwner}, Projects: fakeProjects{testProject: testWorkspace},
		Board: &fakeBoard{}, People: fakePeople{}, Desk: newFakeDesk(), Files: files, Sources: map[string]Source{"fake": src},
	})
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject, Credential: "s3cret"})
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, err = svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)
	svc.Pass(t.Context())
	assert.Equal(t, http.Header{"Authorization": {"Token s3cret"}}, files.got["https://api.example.com/f"])
	assert.Nil(t, files.got["https://elsewhere.example.com/f"])
}

func TestSink_ARerunSkipsWhatAnEarlierImportBroughtIn(t *testing.T) {
	src := fakeSource{tickets: []Ticket{{SourceID: "1", Title: "One"}, {SourceID: "2", Title: "Two"}}}
	svc, repo, desk := newDeskService(src, newFakeDesk())
	runToEnd(t, svc, repo)
	again := runToEnd(t, svc, repo)
	assert.Equal(t, StateCompleted, again.State)
	assert.Equal(t, 2, again.Report.Skipped)
	assert.Equal(t, 2, again.Done)
	assert.Len(t, desk.tickets, 2)
}

func TestSink_ARecordThatFailsIsListedAndTheRestComeIn(t *testing.T) {
	var tickets []Ticket
	for i := range MaxListedFailures + 2 {
		tickets = append(tickets, Ticket{SourceID: fmt.Sprint(i), Title: "refused"})
	}
	tickets = append(tickets, Ticket{SourceID: "ok", Title: "fine"})
	desk := newFakeDesk()
	desk.fail = "refused"
	svc, repo, _ := newDeskService(fakeSource{tickets: tickets}, desk)
	got := runToEnd(t, svc, repo)
	assert.Equal(t, StateCompleted, got.State)
	assert.Equal(t, MaxListedFailures+2, got.Failed)
	assert.Len(t, got.Report.Failures, MaxListedFailures)
	assert.Equal(t, Failure{Kind: KindTicket, SourceID: "0", Title: "refused", Error: "title refused"}, got.Report.Failures[0])
	assert.Equal(t, 1, got.Done)
}

func TestSink_ACancelMeanwhileStopsTheRun(t *testing.T) {
	desk := newFakeDesk()
	src := fakeSource{tickets: []Ticket{{SourceID: "1", Title: "One"}, {SourceID: "2", Title: "Two"}}}
	svc, repo, _ := newDeskService(src, desk)
	imp, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, err = svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)
	desk.onFile = func() {
		desk.onFile = nil
		_, err := svc.Cancel(as(testOwner), imp.ID)
		assert.NoError(t, err)
	}
	svc.Pass(t.Context())
	got := repo.stored(imp.ID)
	assert.Equal(t, StateCancelled, got.State)
	assert.Len(t, desk.tickets, 1, "the run stopped at the next record")
}

// TestSink_ARerunNeverFilesATicketTwice holds the next import to one ticket per record wherever the last one stopped:
// cancelled just after a ticket was filed, failed filling a ticket that exists, or with a filed ticket's record lost.
func TestSink_ARerunNeverFilesATicketTwice(t *testing.T) {
	tickets := []Ticket{
		{SourceID: "1", Title: "One", Comments: []Comment{{SourceID: "c1", Body: "first"}}},
		{SourceID: "2", Title: "Two"},
	}
	tests := []struct {
		name  string
		setup func(svc *Service, desk *fakeDesk, id func() string)
	}{
		{"a cancel right after the first ticket was filed", func(svc *Service, desk *fakeDesk, id func() string) {
			desk.onFile = func() {
				desk.onFile = nil
				_, err := svc.Cancel(as(testOwner), id())
				assert.NoError(t, err)
			}
		}},
		{"a comment refused after the ticket exists", func(_ *Service, desk *fakeDesk, _ func() string) { desk.refuseComments = true }},
		{"the first ticket's record lost after its commit", func(_ *Service, desk *fakeDesk, _ func() string) { desk.lostAfterFile = "One" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desk := newFakeDesk()
			svc, repo, _ := newDeskService(fakeSource{tickets: tickets}, desk)
			first, err := svc.Start(as(testOwner), Start{Source: "fake", ProjectID: testProject})
			require.NoError(t, err)
			svc.Pass(t.Context())
			_, err = svc.Confirm(as(testOwner), first.ID)
			require.NoError(t, err)
			tt.setup(svc, desk, func() string { return first.ID })
			svc.Pass(t.Context())
			require.True(t, repo.stored(first.ID).State.Final())

			desk.refuseComments = false
			again := runToEnd(t, svc, repo)
			assert.Equal(t, StateCompleted, again.State)
			assert.Equal(t, map[string]int{"One": 1, "Two": 1}, desk.filed, "each record filed once across both imports")
		})
	}
}

func TestSink_GivesEachTicketItsCategoryAndDueDate_AndLinksABlockerWhicheverComesInSecond(t *testing.T) {
	src := fakeSource{tickets: []Ticket{
		{SourceID: "waits", Title: "Waits", Category: "Mobile", DueOn: "2026-12-01", BlockedBy: []string{"blocker", "elsewhere"}},
		{SourceID: "blocker", Title: "Blocker", Category: "mobile", Blocks: []string{"waits", "later"}},
		{SourceID: "later", Title: "Later", Category: "Web", BlockedBy: []string{"blocker"}},
	}}
	svc, repo, desk := newDeskService(src, newFakeDesk())
	runToEnd(t, svc, repo)
	again := runToEnd(t, svc, repo)
	assert.Equal(t, 3, again.Report.Skipped)

	waits := desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "waits")]
	blocker := desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "blocker")]
	later := desk.tickets[derivedID(testWorkspace, "fake", KindTicket, "later")]
	assert.Equal(t, "2026-12-01", waits.DueOn)
	assert.Equal(t, "cat-Mobile", waits.CategoryID)
	assert.Equal(t, "cat-Mobile", blocker.CategoryID, "a category of the same name, ignoring case, is reused")
	assert.Equal(t, "cat-Web", later.CategoryID)
	assert.Equal(t, map[string][]string{waits.ID: {blocker.ID}, later.ID: {blocker.ID}}, desk.blocks,
		"each link once, made by whichever end came in second; one to a record never imported is left out")
}
