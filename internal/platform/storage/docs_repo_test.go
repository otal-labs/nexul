package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
	err := s.Docs.Update(context.Background(), newTestDoc("missing"))
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
	require.NoError(t, s.Docs.Update(context.Background(), d))

	got, err := s.Docs.GetByID(context.Background(), "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Title)
	assert.Equal(t, "changed body", got.Body)
	assert.Equal(t, 2, got.Version)
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
	require.NoError(t, s.Docs.Update(context.Background(), updated))

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
	require.NoError(t, s.Docs.CommitBody(ctx, updated))

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
	err := s.Docs.CommitBody(context.Background(), newTestDoc("nope"))
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
