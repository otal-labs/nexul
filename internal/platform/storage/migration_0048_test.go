package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/workspace"
)

func TestMigration0048_EveryProjectGetsMainHoldingAllItsDocs(t *testing.T) {
	db := migrateBefore(t, "0048")
	_, err := db.Exec(`
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES
    ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0),
    ('p-api', 'API', 'API', 1, 'workspace-default', 0, 0);
INSERT INTO docs (id, title, body, project_id, version, archived, locked, created_at, updated_at) VALUES
    ('d-ep01', 'GetSource EP01', '', 'p-web', 4, 0, 0, 100, 900),
    ('d-ep02', 'GetSource EP02', '', 'p-web', 1, 0, 0, 200, 200),
    ('d-old', 'Old plan', '', 'p-web', 2, 1, 0, 300, 300),
    ('d-spec', 'Signed-off spec', '', 'p-api', 7, 0, 1, 400, 400);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0048 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	for project, want := range map[string][]string{"p-web": {"d-ep01", "d-ep02", "d-old"}, "p-api": {"d-spec"}} {
		folders, err := s.Docs.ListFolders(ctx, project)
		require.NoError(t, err)
		require.Len(t, folders, 1, project)
		assert.Equal(t, "Main", folders[0].Name)
		assert.True(t, folders[0].IsDefault)
		for _, id := range want {
			d, err := s.Docs.GetByID(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, folders[0].ID, d.FolderID, "%s lands in its own project's Main", id)
		}
	}

	ep01, err := s.Docs.GetByID(ctx, "d-ep01")
	require.NoError(t, err)
	assert.Equal(t, 4, ep01.Version)
	assert.Equal(t, time.Unix(900, 0).UTC(), ep01.UpdatedAt, "the backfill is not an edit")
	archived, err := s.Docs.GetByID(ctx, "d-old")
	require.NoError(t, err)
	assert.True(t, archived.Archived)
	locked, err := s.Docs.GetByID(ctx, "d-spec")
	require.NoError(t, err)
	assert.True(t, locked.Locked)

	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-gs", ProjectID: "p-web", Name: "GetSource"}))
	require.NoError(t, s.Docs.SetDocFolder(ctx, "d-ep01", "f-gs"), "an upgraded doc moves like a new one")
}

func TestDocFolders_ANewProjectGetsMain(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	require.NoError(t, s.Projects.Create(ctx, &workspace.Project{ID: "p-new", Name: "New", Prefix: "NEW", WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now}))

	folders, err := s.Docs.ListFolders(ctx, "p-new")
	require.NoError(t, err)
	require.Len(t, folders, 1)
	assert.Equal(t, "Main", folders[0].Name)
	main, err := s.Docs.DefaultFolder(ctx, "p-new")
	require.NoError(t, err)
	assert.Equal(t, folders[0].ID, main.ID)
}

func TestDocFolders_NamesAreUniquePerProjectIgnoringCase(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now().UTC()
	require.NoError(t, s.Projects.Create(ctx, &workspace.Project{ID: "p-two", Name: "Two", Prefix: "TWO", WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-gs", ProjectID: "project-general", Name: "GetSource"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-other", ProjectID: "project-general", Name: "Other"}))

	err := s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-dup", ProjectID: "project-general", Name: "getsource"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
	err = s.Docs.RenameFolder(ctx, &docs.Folder{ID: "f-other", Name: "GETSOURCE"})
	require.ErrorIs(t, err, apperrs.ErrConflict, "a rename into a taken name conflicts too")
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-gs-2", ProjectID: "p-two", Name: "GetSource"}), "another project may reuse it")
	err = s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-main-2", ProjectID: "p-two", Name: "Spare", IsDefault: true})
	require.ErrorIs(t, err, apperrs.ErrConflict, "a project has one default folder")
}

func TestDocFolders_DeleteMovesItsDocsInTheSameTransaction(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-gs", ProjectID: "project-general", Name: "GetSource"}))
	d := newTestDoc("doc-1")
	d.FolderID = "f-gs"
	require.NoError(t, s.Docs.Create(ctx, d))

	require.NoError(t, s.Docs.DeleteFolder(ctx, "f-gs", "folder-general-main"))

	got, err := s.Docs.GetByID(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "folder-general-main", got.FolderID)
	_, err = s.Docs.GetFolder(ctx, "f-gs")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, s.Docs.DeleteFolder(ctx, "f-gs", "folder-general-main"), apperrs.ErrNotFound)
}
