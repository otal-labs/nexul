package imports

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// handbook is a 3-level tree with a database of 5 rows, an image, and links between pages.
func handbook(t *testing.T) (*stubNotion, map[string]string) {
	s := newStubNotion(t)
	ids := map[string]string{}
	ids["root"] = s.page("", "Handbook")
	ids["child"] = s.page(ids["root"], "Onboarding")
	ids["grandchild"] = s.page(ids["child"], "First week")
	s.add(ids["root"], "paragraph", map[string]any{"rich_text": []map[string]any{
		{"type": "text", "plain_text": "Start with ", "annotations": map[string]any{}},
		{"type": "mention", "plain_text": "First week", "mention": map[string]any{"type": "page", "page": map[string]any{"id": ids["grandchild"]}}, "annotations": map[string]any{}},
	}})
	s.image(ids["child"], "welcome.png")
	s.add(ids["grandchild"], "paragraph", map[string]any{"rich_text": []map[string]any{
		{"type": "text", "plain_text": "Back to the handbook", "href": "/" + strings.ReplaceAll(ids["root"], "-", ""), "annotations": map[string]any{}},
	}})
	db := s.database(ids["root"], "Policies")
	for i := 1; i <= 5; i++ {
		ids[fmt.Sprintf("row%d", i)] = s.row(db, fmt.Sprintf("Policy %d", i), map[string]any{
			"Status": map[string]any{"type": "status", "status": map[string]any{"name": "Live"}},
		})
	}
	return s, ids
}

func runNotion(t *testing.T, svc *Service, repo *fakeRepo, from string) *Import {
	t.Helper()
	imp, err := svc.Start(as(testOwner), Start{Source: SourceNotion, ProjectID: testProject, From: from, Credential: stubToken})
	require.NoError(t, err)
	svc.Pass(t.Context())
	ready := repo.stored(imp.ID)
	require.Equal(t, StateReady, ready.State, ready.Report.Error)
	_, err = svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)
	svc.Pass(t.Context())
	done := repo.stored(imp.ID)
	require.Equal(t, StateCompleted, done.State, done.Report.Error)
	return &done
}

func TestNotionImport_BringsATreeIntoOneFolderWithWorkingLinksAndCopiedImages(t *testing.T) {
	stub, ids := handbook(t)
	shelf := newFakeShelf()
	svc, repo := newNotionService(stub, shelf)

	imp := runNotion(t, svc, repo, "")

	assert.Equal(t, "Handbook ("+ids["root"]+")", imp.From)
	assert.Equal(t, 8, imp.Total, "three pages and five rows")
	assert.Equal(t, 8, imp.Done)
	assert.Equal(t, 1, imp.Report.FilesCopied)
	assert.Equal(t, map[string]string{"handbook": "folder-handbook"}, shelf.folders)
	for _, d := range shelf.docs {
		assert.Equal(t, "folder-handbook", d.FolderID, d.Title)
	}
	link := func(key string) string { return "/docs/" + derivedID(testWorkspace, SourceNotion, KindDoc, ids[key]) }

	root, ok := shelf.byTitle("Handbook")
	require.True(t, ok)
	assert.Contains(t, root.Body, "Start with [First week]("+link("grandchild")+")")
	assert.Contains(t, root.Body, "[Onboarding]("+link("child")+")", "a parent lists its child pages")
	assert.Contains(t, root.Body, "**Policies**")
	assert.Contains(t, root.Body, "- [Policy 3]("+link("row3")+")")

	grandchild, _ := shelf.byTitle("First week")
	assert.Contains(t, grandchild.Body, "[Back to the handbook]("+link("root")+")")
	assert.Equal(t, root.ID, strings.TrimPrefix(link("root"), "/docs/"), "a link points at the doc it names")

	child, _ := shelf.byTitle("Onboarding")
	assert.Contains(t, child.Body, "![](/api/attachments/att-welcome.png)")
	assert.Equal(t, []byte("PNG bytes of welcome.png"), shelf.files["att-welcome.png"])

	row, _ := shelf.byTitle("Policy 2")
	assert.True(t, strings.HasPrefix(row.Body, "| Property | Value |\n| --- | --- |\n| Status | Live |"), row.Body)
}

func TestNotionImport_ARerunUpdatesNothing(t *testing.T) {
	stub, _ := handbook(t)
	shelf := newFakeShelf()
	svc, repo := newNotionService(stub, shelf)
	runNotion(t, svc, repo, "")
	before := shelf.created
	stub.requests = 0

	again := runNotion(t, svc, repo, "")

	assert.Equal(t, before, shelf.created)
	assert.Equal(t, 8, again.Report.Skipped)
	assert.Zero(t, again.Report.FilesCopied)
	assert.Equal(t, 4, stub.requests, "a search and a database read for the scan and again for the import; no page is read again")
}

func TestNotionImport_CountsAPageUnsharedMidwayAsFailed(t *testing.T) {
	stub, ids := handbook(t)
	delete(stub.children, ids["grandchild"])
	stub.children[ids["grandchild"]] = nil
	svc, repo := newNotionService(stub, newFakeShelf())
	imp, err := svc.Start(as(testOwner), Start{Source: SourceNotion, ProjectID: testProject, Credential: stubToken})
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, err = svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)
	stub.mu.Lock()
	stub.unshared = ids["grandchild"]
	stub.mu.Unlock()
	svc.Pass(t.Context())

	done := repo.stored(imp.ID)
	assert.Equal(t, StateCompleted, done.State, done.Report.Error)
	assert.Equal(t, 7, done.Done)
	assert.Equal(t, 1, done.Failed)
	require.Len(t, done.Report.Failures, 1)
	assert.Equal(t, Failure{Kind: KindDoc, SourceID: ids["grandchild"], Title: "First week", Error: "the page is no longer shared with the integration"}, done.Report.Failures[0])
}

func TestNotionImport_ACancelWhileADocFillsLeavesThatDocIn(t *testing.T) {
	stub := newStubNotion(t)
	root := stub.page("", "Handbook")
	stub.image(root, "logo.png")
	stub.page(root, "Later")
	shelf := newFakeShelf()
	svc, repo := newNotionService(stub, shelf)
	imp, err := svc.Start(as(testOwner), Start{Source: SourceNotion, ProjectID: testProject, Credential: stubToken})
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, err = svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)
	shelf.onAttach = func() {
		shelf.onAttach = nil
		_, err := svc.Cancel(as(testOwner), imp.ID)
		require.NoError(t, err)
	}

	svc.Pass(t.Context())

	assert.Equal(t, StateCancelled, repo.stored(imp.ID).State)
	docID := derivedID(testWorkspace, SourceNotion, KindDoc, root)
	assert.False(t, shelf.archived[docID], "the doc finished filling, so it stays")
	_, before, err := repo.Imported(t.Context(), testWorkspace, SourceNotion, KindDoc, root)
	require.NoError(t, err)
	assert.True(t, before)
	_, ok := shelf.byTitle("Later")
	assert.False(t, ok, "nothing after the cancel is written")
}

func TestNotionImport_APickedRootUnderAnotherJoinsItsFolder(t *testing.T) {
	stub, ids := handbook(t)
	shelf := newFakeShelf()
	svc, repo := newNotionService(stub, shelf)

	imp := runNotion(t, svc, repo, ids["child"]+"\n"+ids["root"])

	assert.Equal(t, 8, imp.Done)
	assert.Equal(t, map[string]string{"handbook": "folder-handbook"}, shelf.folders)
}

func TestNotionImport_ImportsOnlyTheRootsItNames(t *testing.T) {
	stub, ids := handbook(t)
	other := stub.page("", "Archive")
	stub.page(other, "Old notes")
	shelf := newFakeShelf()
	svc, repo := newNotionService(stub, shelf)

	imp := runNotion(t, svc, repo, "https://www.notion.so/acme/Onboarding-"+strings.ReplaceAll(ids["child"], "-", "")+"?pvs=4")

	assert.Equal(t, 2, imp.Done, "Onboarding and First week")
	assert.Equal(t, map[string]string{"onboarding": "folder-onboarding"}, shelf.folders)
	grandchild, _ := shelf.byTitle("First week")
	assert.Contains(t, grandchild.Body, "(https://www.notion.so/"+strings.ReplaceAll(ids["root"], "-", "")+")", "a page outside the import links back to Notion")
}

func TestNotionImport_CopiesAFileFromTheReadThatLinksIt(t *testing.T) {
	stub, _ := handbook(t)
	shelf := newFakeShelf()
	svc, repo := newNotionService(stub, shelf)
	imp, err := svc.Start(as(testOwner), Start{Source: SourceNotion, ProjectID: testProject, Credential: stubToken})
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, err = svc.Confirm(as(testOwner), imp.ID)
	require.NoError(t, err)

	stub.advance(3 * time.Hour)
	svc.Pass(t.Context())

	done := repo.stored(imp.ID)
	assert.Equal(t, StateCompleted, done.State, done.Report.Error)
	assert.Equal(t, 1, done.Report.FilesCopied, "the link the scan would have seen is long expired; the import reads a fresh one")
}

func TestNotionScan_RefusesWhatItCannotImport(t *testing.T) {
	tests := []struct {
		name  string
		build func(s *stubNotion)
		from  string
		token string
		want  string
	}{
		{"no token", func(s *stubNotion) { s.page("", "A") }, "", "", "needs the internal integration's secret"},
		{"a refused token", func(s *stubNotion) { s.page("", "A") }, "", "ntn_wrong", "Notion refused the token"},
		{"nothing shared", func(*stubNotion) {}, "", stubToken, "nothing is shared with the integration yet"},
		{"a page not shared", func(s *stubNotion) { s.page("", "A") }, stubID(999), stubToken, "is not shared with the integration"},
		{"a tree 25 levels deep", func(s *stubNotion) {
			parent := s.page("", "Level 0")
			for i := 1; i <= MaxNotionDepth+1; i++ {
				parent = s.page(parent, fmt.Sprintf("Level %d", i))
			}
		}, "", stubToken, "deeper than the 24 one import reads"},
		{"more than 5,000 pages", func(s *stubNotion) {
			root := s.page("", "Wiki")
			for i := 0; i < MaxDocs; i++ {
				s.page(root, "Page")
			}
		}, "", stubToken, "more than the 5000 one import takes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newStubNotion(t)
			tt.build(stub)
			svc, repo := newNotionService(stub, newFakeShelf())
			imp, err := svc.Start(as(testOwner), Start{Source: SourceNotion, ProjectID: testProject, From: tt.from, Credential: tt.token})
			require.NoError(t, err)
			svc.Pass(t.Context())
			got := repo.stored(imp.ID)
			assert.Equal(t, StateFailed, got.State)
			assert.Contains(t, got.Report.Error, tt.want)
			assert.Empty(t, got.Credential, "a failed import keeps no token")
		})
	}
}

func TestNotionScan_AtTheLimitsItCounts(t *testing.T) {
	stub := newStubNotion(t)
	parent := stub.page("", "Level 0")
	for i := 1; i <= MaxNotionDepth; i++ {
		parent = stub.page(parent, fmt.Sprintf("Level %d", i))
	}
	got, err := NewNotionSource(NewNotionClient(stub.srv.URL, stub.srv.Client(), 0)).Scan(t.Context(), Job{Credential: stubToken})
	require.NoError(t, err)
	assert.Equal(t, MaxNotionDepth+1, got.Docs)
}

func TestNotionClient_WaitsOutRetryAfter(t *testing.T) {
	stub := newStubNotion(t)
	stub.page("", "A")
	stub.throttle = 7
	client := NewNotionClient(stub.srv.URL, stub.srv.Client(), 0)
	var waited []time.Duration
	client.sleep = func(_ context.Context, d time.Duration) error {
		waited = append(waited, d)
		return nil
	}
	pages, err := client.searchPages(t.Context(), stubToken)
	require.NoError(t, err)
	assert.Len(t, pages, 1)
	assert.Equal(t, []time.Duration{7 * time.Second}, waited)
}

func TestNotionClient_PausesTheImportWhenStillThrottled(t *testing.T) {
	stub := newStubNotion(t)
	client := NewNotionClient(stub.srv.URL, stub.srv.Client(), 0)
	client.sleep = func(context.Context, time.Duration) error {
		stub.mu.Lock()
		stub.throttle = 1
		stub.mu.Unlock()
		return nil
	}
	stub.throttle = 1
	_, err := client.searchPages(t.Context(), stubToken)
	assert.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestNotionClient_KeepsToItsRate(t *testing.T) {
	stub := newStubNotion(t)
	stub.page("", "A")
	client := NewNotionClient(stub.srv.URL, stub.srv.Client(), 20)
	start := time.Now()
	for range 25 {
		_, err := client.searchPages(t.Context(), stubToken)
		require.NoError(t, err)
	}
	assert.GreaterOrEqual(t, time.Since(start), 200*time.Millisecond, "25 requests at 20 a second, a burst of 20 first")
}

func TestNotionImport_ArchivesTheDocsAStoppedImportNeverFilled(t *testing.T) {
	tests := []struct {
		name string
		stop func(t *testing.T, svc *Service, stub *stubNotion, id string)
	}{
		{"cancelled", func(t *testing.T, svc *Service, _ *stubNotion, id string) {
			_, err := svc.Cancel(as(testOwner), id)
			require.NoError(t, err)
		}},
		{"failed", func(t *testing.T, svc *Service, stub *stubNotion, _ string) {
			stub.mu.Lock()
			stub.pages = nil
			stub.mu.Unlock()
			svc.now = func() time.Time { return time.Now().Add(retryAfter) }
			svc.Pass(t.Context())
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newStubNotion(t)
			root := stub.page("", "Handbook")
			stub.image(root, "logo.png")
			shelf := newFakeShelf()
			shelf.failBody = fmt.Errorf("%w: database is busy", apperrs.ErrRetryable)
			svc, repo := newNotionService(stub, shelf)
			imp, err := svc.Start(as(testOwner), Start{Source: SourceNotion, ProjectID: testProject, Credential: stubToken})
			require.NoError(t, err)
			svc.Pass(t.Context())
			_, err = svc.Confirm(as(testOwner), imp.ID)
			require.NoError(t, err)
			svc.Pass(t.Context())
			require.Equal(t, StateImporting, repo.stored(imp.ID).State, "a passing failure leaves the doc written, not filled")

			tt.stop(t, svc, stub, imp.ID)

			docID := derivedID(testWorkspace, SourceNotion, KindDoc, root)
			assert.True(t, shelf.archived[docID])
			_, before, err := repo.Imported(t.Context(), testWorkspace, SourceNotion, KindDoc, root)
			require.NoError(t, err)
			assert.False(t, before, "a re-run brings it in again")
		})
	}
}

func TestNotionError_SaysWhatToFix(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{401, apperrs.ErrUnauthorized},
		{403, apperrs.ErrForbidden},
		{404, apperrs.ErrNotFound},
		{429, apperrs.ErrRetryable},
		{502, apperrs.ErrRetryable},
		{400, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			assert.ErrorIs(t, notionError(tt.status, []byte(`{"code":"x","message":"y"}`)), tt.want)
		})
	}
	assert.Equal(t, 7*time.Second, retryAfterOf("7"))
	assert.Equal(t, time.Second, retryAfterOf("Wed, 21 Oct 2026 07:28:00 GMT"), "a date waits the least")
}

func TestNotionClient_NeverPutsTheTokenInAnError(t *testing.T) {
	client := NewNotionClient("http://127.0.0.1:1", nil, 0)
	_, err := client.searchPages(t.Context(), "ntn_secret_value")
	require.ErrorIs(t, err, apperrs.ErrRetryable)
	assert.NotContains(t, err.Error(), "ntn_secret_value")
}
