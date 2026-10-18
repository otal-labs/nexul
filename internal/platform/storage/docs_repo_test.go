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
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newTestDoc(id string) *docs.Doc {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	return &docs.Doc{ID: id, ProjectID: "project-general", Title: "Storage Spine", Body: "SQLite, migrations, FTS5", Version: 1, CreatedAt: now, UpdatedAt: now}
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

func TestDocsRepo_Delete_RemovesDoc(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Docs.Delete(context.Background(), "doc-1"))
	_, err := s.Docs.GetByID(context.Background(), "doc-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
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

// seedPagedDocs files 40 docs, three to a second, across two projects and two folders, a sixth of them archived.
func seedPagedDocs(t *testing.T, s *Store) []*docs.Doc {
	t.Helper()
	ctx := t.Context()
	seedProject(t, s, "p-hidden", "workspace-default", "HID")
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-ops", ProjectID: "project-general", Name: "Ops"}))
	var all []*docs.Doc
	for i := range 40 {
		d := newTestDoc(fmt.Sprintf("doc-%02d", i))
		d.Title = strings.Repeat("rollback ", 1+i%3) + "plan"
		d.CreatedAt = d.CreatedAt.Add(time.Duration(i/4) * time.Second)
		d.FolderID = map[bool]string{true: "f-ops", false: "folder-general-main"}[i%5 == 0]
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
	return all
}

// reaches is whether read keeps d: its own rule first, then its folder's, then its project's.
func reaches(read *permissions.DocReach, d *docs.Doc) bool {
	if read == nil || slices.Contains(read.Allowed, d.ID) {
		return true
	}
	if slices.Contains(read.Denied, d.ID) {
		return false
	}
	return (slices.Contains(read.ProjectIDs, d.ProjectID) && !slices.Contains(read.DeniedFolders, d.FolderID)) || slices.Contains(read.AllowedFolders, d.FolderID)
}

// docKept is the per-row rule a doc page applies in SQL.
func docKept(d *docs.Doc, f docs.DocFilter, scope docs.DocScope) bool {
	return reaches(scope.Read, d) && (scope.All || slices.Contains(scope.ProjectIDs, d.ProjectID)) && (f.ProjectID == "" || d.ProjectID == f.ProjectID) &&
		(f.FolderID == "" || d.FolderID == f.FolderID) && (f.Query != "" || f.IncludeArchived || !d.Archived) && (f.Query == "" || !d.Archived)
}

func pageDocIDs(t *testing.T, s *Store, f docs.DocFilter, scope docs.DocScope) []string {
	t.Helper()
	return pageAll(t, 6, func(offset, limit int) ([]string, int) {
		ds, total, err := s.Docs.Page(t.Context(), f, scope, paging.Window{Offset: offset, Limit: limit})
		require.NoError(t, err)
		ids := make([]string, len(ds))
		for i, d := range ds {
			ids[i] = d.ID
		}
		return ids, total
	})
}

// TestDocsRepo_Page_FiltersAndSearchesInSQL: browsing pages oldest first and a search ranks as Search does, each keeping
// only the docs its project, folder, archived state, and scope allow, and a search only those its read scope reads.
func TestDocsRepo_Page_FiltersAndSearchesInSQL(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	all := seedPagedDocs(t, s)
	hits, err := s.Docs.Search(t.Context(), "rollback", 1000)
	require.NoError(t, err)
	byID := map[string]*docs.Doc{}
	for _, d := range all {
		byID[d.ID] = d
	}
	read := &permissions.DocReach{ProjectIDs: []string{"project-general"}, Allowed: []string{"doc-04"}, Denied: []string{"doc-01", "doc-02"}}
	deniedFolder := &permissions.DocReach{ProjectIDs: []string{"project-general"}, DeniedFolders: []string{"f-ops"}, Allowed: []string{"doc-05"}, Denied: []string{"doc-01"}}
	allowedFolder := &permissions.DocReach{ProjectIDs: []string{}, AllowedFolders: []string{"f-ops"}, Denied: []string{"doc-10"}}
	filters := []docs.DocFilter{{}, {IncludeArchived: true}, {ProjectID: "project-general"}, {FolderID: "f-ops"}, {ProjectID: "p-hidden", IncludeArchived: true}}
	for _, scope := range []docs.DocScope{{All: true}, {ProjectIDs: []string{"project-general"}}, {ProjectIDs: []string{}}} {
		for _, f := range filters {
			var want []string
			for _, d := range all {
				if docKept(d, f, scope) {
					want = append(want, d.ID)
				}
			}
			assert.Equal(t, want, pageDocIDs(t, s, f, scope), "browse %+v %+v", f, scope)

			f.Query = "rollback"
			for _, scope.Read = range []*permissions.DocReach{nil, read, deniedFolder, allowedFolder} {
				var ranked []string
				for _, h := range hits {
					if docKept(byID[h.ID], f, scope) {
						ranked = append(ranked, h.ID)
					}
				}
				assert.Equal(t, ranked, pageDocIDs(t, s, f, scope), "search %+v %+v", f, scope)
			}
			scope.Read = nil
		}
	}
}
