package storage

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newTestDoc(id string) *docs.Doc {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	return &docs.Doc{ID: id, ProjectID: "project-general", Title: "Storage Spine", Body: "SQLite, migrations, FTS5", Version: 1, CreatedAt: now, UpdatedAt: now}
}

func TestDocsRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Docs.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDocsRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	err := s.Docs.Create(context.Background(), newTestDoc("doc-1"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestDocsRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestDoc("doc-1")
	require.NoError(t, s.Docs.Create(context.Background(), want))

	got, err := s.Docs.GetByID(context.Background(), "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "doc-1", got.ID)
	assert.Equal(t, want.ProjectID, got.ProjectID)
	assert.Equal(t, want.Title, got.Title)
	assert.Equal(t, want.Body, got.Body)
	assert.Equal(t, want.Version, got.Version)
	assert.Equal(t, want.CreatedAt, got.CreatedAt)
	assert.Equal(t, want.UpdatedAt, got.UpdatedAt)
}

// Acceptance criterion (ticket 10): docs.project_id is a real FK — creating a
// doc against an unknown project id is rejected, not silently accepted.
func TestDocsRepo_Create_UnknownProjectIsConflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	d := newTestDoc("doc-1")
	d.ProjectID = "does-not-exist"
	err := s.Docs.Create(context.Background(), d)
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestDocsRepo_ListByProject_FiltersToOneProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	require.NoError(t, s.Projects.Create(context.Background(), &workspace.Project{
		ID: "project-other", Name: "Other", Position: 1, WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now,
	}))

	inGeneral := newTestDoc("doc-1")
	require.NoError(t, s.Docs.Create(context.Background(), inGeneral))
	inOther := newTestDoc("doc-2")
	inOther.ProjectID = "project-other"
	require.NoError(t, s.Docs.Create(context.Background(), inOther))

	got, err := s.Docs.ListByProject(context.Background(), "project-general")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "doc-1", got[0].ID)
}

func TestDocsRepo_List_ReturnsAll(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-2")))

	got, err := s.Docs.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestDocsRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Docs.Update(context.Background(), newTestDoc("missing"), "")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDocsRepo_Update_PersistsChanges(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	d := newTestDoc("doc-1")
	d.Title = "Renamed"
	d.Body = "changed body"
	d.Version = 2
	require.NoError(t, s.Docs.Update(context.Background(), d, ""))

	got, err := s.Docs.GetByID(context.Background(), "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Title)
	assert.Equal(t, "changed body", got.Body)
	assert.Equal(t, 2, got.Version)
}

func TestDocsRepo_Update_TwoWritersFromTheSameRead_BothLand(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(t.Context(), newTestDoc("doc-1")))

	first, second := newTestDoc("doc-1"), newTestDoc("doc-1")
	first.Version, second.Version = 2, 2
	second.Title = "Renamed last"
	require.NoError(t, s.Docs.Update(t.Context(), first, ""))
	require.NoError(t, s.Docs.Update(t.Context(), second, ""))

	got, err := s.Docs.GetByID(t.Context(), "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed last", got.Title)
	assert.Equal(t, 3, got.Version)
}

func TestDocsRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Docs.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDocsRepo_Delete_RemovesDoc(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Docs.Delete(context.Background(), "doc-1"))
	_, err := s.Docs.GetByID(context.Background(), "doc-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDocsRepo_RichTextBody_Searchable(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	// Canonical structured body (ws-23): the repo stores the markdown
	// rendering in body_md, which docs_fts indexes.
	d := newTestDoc("doc-1")
	d.Body = `{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Storage"}]},{"type":"paragraph","content":[{"type":"text","text":"SQLite is the spine"}]}]}`
	require.NoError(t, s.Docs.Create(context.Background(), d))

	got, err := s.Docs.Search(context.Background(), "sqlite", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "doc-1", got[0].ID)

	got, err = s.Docs.Search(context.Background(), "spine", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)

	// The canonical JSON is stored verbatim for the editor; search indexes the
	// rendering, not the JSON keys.
	_, err = s.Docs.Search(context.Background(), "content", 10)
	require.NoError(t, err)
}

func TestDocsRepo_LegacyMarkdownBody_Searchable(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	// Pre-ws-23 rows held plain markdown; they stay searchable after the
	// migration backfill and on write.
	d := newTestDoc("doc-1")
	d.Body = "legacy markdown body about migrations"
	require.NoError(t, s.Docs.Create(context.Background(), d))

	got, err := s.Docs.Search(context.Background(), "migrations", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "doc-1", got[0].ID)
}

func TestDocsRepo_UpdateRewritesSearchText(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	d := newTestDoc("doc-1")
	d.Body = "old body"
	require.NoError(t, s.Docs.Create(context.Background(), d))

	updated := newTestDoc("doc-1")
	updated.Body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"new searchable content"}]}]}`
	updated.Version = 2
	require.NoError(t, s.Docs.Update(context.Background(), updated, ""))

	got, err := s.Docs.Search(context.Background(), "new", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "doc-1", got[0].ID)

	// The old body is no longer searchable.
	got, err = s.Docs.Search(context.Background(), "old", 10)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestDocsRepo_CommitBody_NoVersionRowAndReindexes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))

	updated := newTestDoc("doc-1")
	updated.Title = "Committed"
	updated.Body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"merged-state"}]}]}`
	updated.Version = 2
	require.NoError(t, s.Docs.CommitBody(ctx, updated, ""))

	got, err := s.Docs.GetByID(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Committed", got.Title)
	assert.Equal(t, updated.Body, got.Body)
	assert.Equal(t, 2, got.Version)

	// A commit writes no heavyweight version row.
	vs, err := s.Docs.ListVersions(ctx, "doc-1")
	require.NoError(t, err)
	assert.Len(t, vs, 1, "only the create-time row exists")

	// The FTS trigger re-indexes the committed body immediately (search
	// consumes meaningful commits, not keystrokes).
	results, err := s.Docs.Search(ctx, "merged-state", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc-1", results[0].ID)
}

func TestDocsRepo_CommitBody_MissingDoc(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Docs.CommitBody(context.Background(), newTestDoc("nope"), "")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDocsRepo_CreateNamedVersion_AdvancesCounter(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))

	v := &docs.DocVersion{
		DocID: "doc-1", Version: 2, Title: "Storage Spine", Body: "SQLite, migrations, FTS5",
		Name: "MVP cut", AuthorID: "user-1", CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.Docs.CreateNamedVersion(ctx, v))

	got, err := s.Docs.GetByID(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, 2, got.Version, "doc counter advances to the named row's key")

	vs, err := s.Docs.ListVersions(ctx, "doc-1")
	require.NoError(t, err)
	require.Len(t, vs, 2)
	assert.Equal(t, "MVP cut", vs[0].Name)
	assert.Equal(t, "user-1", vs[0].AuthorID)
	assert.Empty(t, vs[1].Name, "create-time auto row stays unnamed")

	back, err := s.Docs.GetVersion(ctx, "doc-1", 2)
	require.NoError(t, err)
	assert.Equal(t, "MVP cut", back.Name)
	assert.Equal(t, "user-1", back.AuthorID)
}

func TestDocsRepo_CreateNamedVersion_RejectsDuplicateKey(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))

	// A named version at an already-occupied key must be refused, never
	// silently overwritten (the use-case always bumps to a fresh key).
	v := &docs.DocVersion{DocID: "doc-1", Version: 1, Title: "x", Body: "y", Name: "dup", CreatedAt: time.Now().UTC()}
	require.Error(t, s.Docs.CreateNamedVersion(ctx, v))
}

// TestDocsRepo_PageAndSearchIDs_FilterInSQL: browsing pages oldest first and a search ranks as Search does, each
// keeping only the docs its project, folder, archived state, and scope allow.
func TestDocsRepo_PageAndSearchIDs_FilterInSQL(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s := newTestStore(t)
	seedProject(t, s, "p-hidden", "workspace-default", "HID")
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-ops", ProjectID: "project-general", Name: "Ops"}))
	var all []*docs.Doc
	for i := range 40 {
		d := newTestDoc(fmt.Sprintf("doc-%02d", i))
		d.Title = strings.Repeat("rollback ", 1+i%3) + "plan"
		d.CreatedAt = d.CreatedAt.Add(time.Duration(i/4) * time.Second)
		d.FolderID = "folder-general-main"
		if i%5 == 0 {
			d.FolderID = "f-ops"
		}
		if i%4 == 0 {
			d.ProjectID, d.FolderID = "p-hidden", ""
		}
		require.NoError(t, s.Docs.Create(ctx, d))
		if i%6 == 0 {
			require.NoError(t, s.Docs.SetArchived(ctx, d.ID, true))
			d.Archived = true
		}
		all = append(all, d)
	}
	keeps := func(d *docs.Doc, f docs.DocFilter, scope docs.DocScope) bool {
		return (scope.All || slices.Contains(scope.ProjectIDs, d.ProjectID)) && (f.ProjectID == "" || d.ProjectID == f.ProjectID) &&
			(f.FolderID == "" || d.FolderID == f.FolderID) && (f.IncludeArchived || !d.Archived)
	}
	hits, err := s.Docs.Search(ctx, "rollback", 1000)
	require.NoError(t, err)
	byID := map[string]*docs.Doc{}
	for _, d := range all {
		byID[d.ID] = d
	}
	filters := []docs.DocFilter{{}, {IncludeArchived: true}, {ProjectID: "project-general"}, {FolderID: "f-ops"}, {ProjectID: "p-hidden", IncludeArchived: true}}
	for _, scope := range []docs.DocScope{{All: true}, {ProjectIDs: []string{"project-general"}}, {ProjectIDs: []string{}}} {
		for _, f := range filters {
			var want []string
			for _, d := range all {
				if keeps(d, f, scope) {
					want = append(want, d.ID)
				}
			}
			got := pageAll(t, 6, func(offset, limit int) ([]string, int) {
				ds, total, err := s.Docs.Page(ctx, f, scope, paging.Window{Offset: offset, Limit: limit})
				require.NoError(t, err)
				ids := make([]string, len(ds))
				for i, d := range ds {
					ids[i] = d.ID
				}
				return ids, total
			})
			assert.Equal(t, want, got, "browse %+v %+v", f, scope)

			f.Query = "rollback"
			ranked := []string{}
			for _, h := range hits {
				if keeps(byID[h.ID], f, scope) {
					ranked = append(ranked, h.ID)
				}
			}
			searched, err := s.Docs.SearchIDs(ctx, f, scope)
			require.NoError(t, err)
			assert.Equal(t, ranked, searched, "search %+v %+v", f, scope)
		}
	}
}
