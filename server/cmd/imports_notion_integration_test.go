package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/imports"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const notionStubToken = "ntn_stub"

func nid(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

func notionText(s string) []map[string]any {
	return []map[string]any{{"type": "text", "plain_text": s, "annotations": map[string]any{}}}
}

// notionStub is a Notion workspace of a 3-level tree, a database of 5 rows, an image and links between pages.
func notionStub(t *testing.T) *httptest.Server {
	root, child, grandchild, db := nid(1), nid(2), nid(3), nid(4)
	page := func(id, title string, parent map[string]any) map[string]any {
		return map[string]any{"object": "page", "id": id, "created_time": "2026-05-01T09:00:00Z", "parent": parent,
			"properties": map[string]any{"Name": map[string]any{"type": "title", "title": notionText(title)}}}
	}
	pages := []map[string]any{
		page(root, "Handbook", map[string]any{"type": "workspace", "workspace": true}),
		page(child, "Onboarding", map[string]any{"type": "page_id", "page_id": root}),
		page(grandchild, "First week", map[string]any{"type": "page_id", "page_id": child}),
	}
	for i := 1; i <= 5; i++ {
		row := page(nid(10+i), fmt.Sprintf("Policy %d", i), map[string]any{"type": "data_source_id", "data_source_id": nid(5), "database_id": db})
		row["properties"].(map[string]any)["Status"] = map[string]any{"type": "status", "status": map[string]any{"name": "Live"}}
		pages = append(pages, row)
	}
	var srv *httptest.Server
	children := func(id string) []map[string]any {
		switch id {
		case root:
			return []map[string]any{
				{"type": "paragraph", "paragraph": map[string]any{"rich_text": []map[string]any{
					{"type": "text", "plain_text": "Start with ", "annotations": map[string]any{}},
					{"type": "mention", "plain_text": "First week", "mention": map[string]any{"type": "page", "page": map[string]any{"id": grandchild}}, "annotations": map[string]any{}},
				}}},
				{"id": child, "type": "child_page", "has_children": true, "child_page": map[string]any{"title": "Onboarding"}},
				{"id": db, "type": "child_database", "child_database": map[string]any{"title": "Policies"}},
			}
		case child:
			return []map[string]any{
				{"type": "image", "image": map[string]any{"type": "file", "file": map[string]any{"url": srv.URL + "/files/welcome.png"}, "caption": []any{}}},
				{"id": grandchild, "type": "child_page", "has_children": true, "child_page": map[string]any{"title": "First week"}},
			}
		case grandchild:
			return []map[string]any{{"type": "paragraph", "paragraph": map[string]any{"rich_text": []map[string]any{
				{"type": "text", "plain_text": "Back to the handbook", "href": "/" + strings.ReplaceAll(root, "-", ""), "annotations": map[string]any{}},
			}}}}
		}
		return nil
	}
	list := func(w http.ResponseWriter, items []map[string]any) {
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "results": items, "has_more": false, "next_cursor": nil})
	}
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/files/") {
			_, _ = w.Write([]byte("\x89PNG welcome"))
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+notionStubToken || r.Header.Get("Notion-Version") != imports.NotionVersion {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/v1/search":
			list(w, pages)
		case r.URL.Path == "/v1/databases/"+db:
			_ = json.NewEncoder(w).Encode(map[string]any{"object": "database", "id": db, "title": notionText("Policies"), "parent": map[string]any{"type": "page_id", "page_id": root}})
		case strings.HasSuffix(r.URL.Path, "/children"):
			list(w, children(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/blocks/"), "/children")))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// stubDownloads fetches from the stub over plain HTTP, where the real fetcher keeps to Notion's HTTPS hosts.
type stubDownloads struct{}

func (stubDownloads) Fetch(ctx context.Context, rawURL string, _ []string, _ http.Header) (string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	return rawURL[strings.LastIndex(rawURL, "/")+1:], data, err
}

func runNotionImport(t *testing.T, svc *imports.Service) *imports.Import {
	t.Helper()
	ctx := as(uOwner)
	imp, err := svc.Start(ctx, imports.Start{Source: imports.SourceNotion, ProjectID: pGeneral, Credential: notionStubToken})
	require.NoError(t, err)
	svc.Pass(t.Context())
	_, err = svc.Confirm(ctx, imp.ID)
	require.NoError(t, err)
	svc.Pass(t.Context())
	imp, err = svc.Get(ctx, imp.ID)
	require.NoError(t, err)
	require.Equal(t, imports.StateCompleted, imp.State, imp.Report.Error)
	return imp
}

func newNotionImportService(f permFixture, stub *httptest.Server, shelf imports.Shelf) *imports.Service {
	return imports.NewService(imports.Deps{
		Repo: f.store.Imports, Owners: f.svc.accessSvc, Projects: importProjects{projects: f.store.Projects},
		Board:  importBoard{svc: f.svc.workspaceSvc},
		People: importPeople{store: f.store.Imports, projects: f.store.Projects, access: f.svc.accessSvc},
		Desk:   newImportDesk(f), Shelf: shelf, Files: stubDownloads{},
		Sources: map[string]imports.Source{imports.SourceNotion: imports.NewNotionSource(imports.NewNotionClient(stub.URL, stub.Client(), 0))},
	})
}

// stallingShelf fails filling the first doc's body for a passing reason, as a busy database would.
type stallingShelf struct {
	importShelf
	stalled bool
}

func (s *stallingShelf) SetDocBody(ctx context.Context, docID, title, body string) error {
	if !s.stalled {
		s.stalled = true
		return fmt.Errorf("%w: database is busy", apperrs.ErrRetryable)
	}
	return s.importShelf.SetDocBody(ctx, docID, title, body)
}

func TestImport_ACancelledNotionImportArchivesTheDocItNeverFilled_AndARerunBringsItBack(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	stub := notionStub(t)
	shelf := importShelf{docs: f.svc.docsSvc, attachments: f.svc.attachmentsSvc}
	ctx := as(uOwner)
	stalling := newNotionImportService(f, stub, &stallingShelf{importShelf: shelf})
	imp, err := stalling.Start(ctx, imports.Start{Source: imports.SourceNotion, ProjectID: pGeneral, Credential: notionStubToken})
	require.NoError(t, err)
	stalling.Pass(t.Context())
	_, err = stalling.Confirm(ctx, imp.ID)
	require.NoError(t, err)
	stalling.Pass(t.Context())

	_, err = stalling.Cancel(ctx, imp.ID)
	require.NoError(t, err)
	onboarding, err := f.svc.docsSvc.Get(ctx, derivedDocID(f, nid(2)))
	require.NoError(t, err)
	assert.True(t, onboarding.Archived, "the doc whose image never landed is archived")

	again := runNotionImport(t, newNotionImportService(f, stub, shelf))
	assert.Equal(t, 1, again.Report.FilesCopied)
	onboarding, err = f.svc.docsSvc.Get(ctx, onboarding.ID)
	require.NoError(t, err)
	assert.False(t, onboarding.Archived, "a re-run brings it back")
	md, err := richtext.ToMarkdown(onboarding.Body)
	require.NoError(t, err)
	assert.Contains(t, md, "/api/attachments/")
}

// derivedDocID is the id an import gives a Notion page in the fixture's workspace.
func derivedDocID(f permFixture, pageID string) string {
	p, err := f.store.Projects.Get(context.Background(), pGeneral)
	if err != nil {
		panic(err)
	}
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("nexul-import:"+p.WorkspaceID+"/"+imports.SourceNotion+"/"+imports.KindDoc+"/"+pageID)).String()
}

func TestImport_ANotionTreeBecomesOneFolderOfLinkedDocsWithItsImages_AndARerunUpdatesNothing(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	stub := notionStub(t)
	svc := newNotionImportService(f, stub, importShelf{docs: f.svc.docsSvc, attachments: f.svc.attachmentsSvc})
	ctx := as(uOwner)

	imp := runNotionImport(t, svc)
	assert.Equal(t, 8, imp.Done)
	assert.Equal(t, 1, imp.Report.FilesCopied)

	folders, err := f.svc.docsSvc.ListFolders(ctx, pGeneral)
	require.NoError(t, err)
	var folderID string
	for _, fo := range folders {
		if fo.Name == "Handbook" {
			folderID = fo.ID
		}
	}
	require.NotEmpty(t, folderID, "the root page is a folder")
	all, err := f.svc.docsSvc.ListByProject(ctx, pGeneral)
	require.NoError(t, err)
	byTitle := map[string]string{}
	for _, d := range all {
		if d.FolderID == folderID {
			byTitle[d.Title] = d.ID
		}
	}
	require.Len(t, byTitle, 8, "every page and row is a doc in that folder")

	markdown := func(title string) string {
		d, err := f.svc.docsSvc.Get(ctx, byTitle[title])
		require.NoError(t, err)
		md, err := richtext.ToMarkdown(d.Body)
		require.NoError(t, err)
		return md
	}
	root := markdown("Handbook")
	assert.Contains(t, root, "/docs/"+byTitle["First week"], "a page mention links its doc")
	assert.Contains(t, root, "/docs/"+byTitle["Onboarding"], "a parent lists its child pages")
	assert.Contains(t, root, "/docs/"+byTitle["Policy 4"], "a database lists its rows")
	assert.Contains(t, markdown("First week"), "/docs/"+byTitle["Handbook"])
	assert.Contains(t, markdown("Policy 1"), "| Status | Live |")

	files, err := f.svc.attachmentsSvc.List(ctx, attachments.Owner{DocID: byTitle["Onboarding"]})
	require.NoError(t, err)
	require.Len(t, files, 1, "the image was copied while its link still worked")
	assert.Contains(t, markdown("Onboarding"), "/api/attachments/"+files[0].ID)

	before, err := f.svc.docsSvc.Get(ctx, byTitle["Handbook"])
	require.NoError(t, err)
	again := runNotionImport(t, svc)
	assert.Equal(t, 8, again.Report.Skipped)
	after, err := f.svc.docsSvc.Get(ctx, byTitle["Handbook"])
	require.NoError(t, err)
	assert.Equal(t, before.Version, after.Version, "a re-run updates nothing")
	all, err = f.svc.docsSvc.ListByProject(ctx, pGeneral)
	require.NoError(t, err)
	var count int
	for _, d := range all {
		if d.FolderID == folderID {
			count++
		}
	}
	assert.Equal(t, 8, count)
}
