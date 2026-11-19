package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

func TestMigration0166_EveryFolderStaysTopLevelAndNamesAreUniquePerParent(t *testing.T) {
	db := migrateBefore(t, "0166")
	_, err := db.Exec(`
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0);
INSERT INTO doc_folders (id, project_id, name, is_default, created_at, updated_at) VALUES
    ('f-clients', 'p-web', 'Clients', 0, 0, 0),
    ('f-drafts', 'p-web', 'Drafts', 0, 0, 0);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0166 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	folders, err := s.Docs.ListFolders(ctx, "p-web")
	require.NoError(t, err)
	for _, f := range folders {
		assert.Empty(t, f.ParentID, "%s stays at the top level", f.Name)
	}
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-drafts-2", ProjectID: "p-web", ParentID: "f-clients", Name: "drafts"}),
		"a name taken at the top is free inside another folder")
	err = s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-drafts-3", ProjectID: "p-web", ParentID: "f-clients", Name: "DRAFTS"})
	require.ErrorIs(t, err, apperrs.ErrConflict, "siblings still can't share a name, ignoring case")
	err = s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-clients-2", ProjectID: "p-web", Name: "clients"})
	require.ErrorIs(t, err, apperrs.ErrConflict, "nor can two top-level folders")
}

func TestDocFolders_UpdateRefusesAParentInsideTheFolder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-a", ProjectID: "project-general", Name: "A"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-b", ProjectID: "project-general", ParentID: "f-a", Name: "B"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-c", ProjectID: "project-general", ParentID: "f-b", Name: "C"}))

	for _, parent := range []string{"f-a", "f-b", "f-c"} {
		err := s.Docs.UpdateFolder(ctx, &docs.Folder{ID: "f-a", Name: "A", ParentID: parent})
		require.ErrorIs(t, err, apperrs.ErrInvalid, "inside %s", parent)
	}
	require.NoError(t, s.Docs.UpdateFolder(ctx, &docs.Folder{ID: "f-c", Name: "C", ParentID: "f-a"}))
	got, err := s.Docs.GetFolder(ctx, "f-c")
	require.NoError(t, err)
	assert.Equal(t, "f-a", got.ParentID)
}

func TestDocFolders_DeleteTakesTheSubtreeAndItsRules(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-a", ProjectID: "project-general", Name: "A"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-b", ProjectID: "project-general", ParentID: "f-a", Name: "B"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-c", ProjectID: "project-general", ParentID: "f-b", Name: "C"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-keep", ProjectID: "project-general", ParentID: "f-a", Name: "Keep"}))
	d := newTestDoc("doc-deep")
	d.FolderID = "f-c"
	require.NoError(t, s.Docs.Create(ctx, d))
	mustCreateUser(t, s, "u-r", "ur")
	require.NoError(t, s.Access.Set(ctx, "folder", "f-c", "u-r", nil, permissions.SetOf(permissions.DocsRead)))

	subtree, err := s.Docs.FolderSubtree(ctx, "f-b")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"f-b", "f-c"}, subtree)

	require.NoError(t, s.Docs.DeleteFolder(ctx, "f-b", "f-a"))

	got, err := s.Docs.GetByID(ctx, "doc-deep")
	require.NoError(t, err)
	assert.Equal(t, "f-a", got.FolderID)
	for _, id := range []string{"f-b", "f-c"} {
		_, err := s.Docs.GetFolder(ctx, id)
		require.ErrorIs(t, err, apperrs.ErrNotFound, id)
	}
	_, err = s.Docs.GetFolder(ctx, "f-keep")
	require.NoError(t, err, "a sibling stays")
	rules, err := s.Access.ListByResource(ctx, "folder", "f-c")
	require.NoError(t, err)
	assert.Empty(t, rules, "an inner folder's rules go with it")
}

func TestDocsRepo_FolderScope_NamesTheFoldersAroundIt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-a", ProjectID: "project-general", Name: "A"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-b", ProjectID: "project-general", ParentID: "f-a", Name: "B"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-c", ProjectID: "project-general", ParentID: "f-b", Name: "C"}))

	scope, err := s.Docs.FolderScope(ctx, "f-c")
	require.NoError(t, err)
	assert.Equal(t, "project-general", scope.ProjectID)
	assert.Equal(t, "workspace-default", scope.WorkspaceID)
	assert.Equal(t, []string{"f-a", "f-b"}, scope.Folders, "outermost first")
	_, err = s.Docs.FolderScope(ctx, "nope")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

// TestAccessRepo_NestedFolders: a doc's rules come with every folder above it, outermost first, and a folder's Deny
// turns the folders inside it until a nearer rule turns it back.
func TestAccessRepo_NestedFolders(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: "role-r", WorkspaceID: "workspace-default", Name: "role-r", CreatedAt: now, UpdatedAt: now}))
	mustCreateUser(t, s, "u-r", "ur")
	require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-r", WorkspaceID: "workspace-default", RoleID: "role-r", CreatedAt: now}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-a", ProjectID: "project-general", Name: "A"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-b", ProjectID: "project-general", ParentID: "f-a", Name: "B"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-c", ProjectID: "project-general", ParentID: "f-b", Name: "C"}))
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-d", ProjectID: "project-general", ParentID: "f-c", Name: "D"}))
	d := newTestDoc("doc-deep")
	d.FolderID = "f-d"
	require.NoError(t, s.Docs.Create(ctx, d))
	read := permissions.SetOf(permissions.DocsRead)
	require.NoError(t, s.Access.SetRole(ctx, "folder", "f-a", "role-r", nil, read))
	require.NoError(t, s.Access.Set(ctx, "folder", "f-c", "u-r", read, nil))

	got, err := s.Access.DocRules(ctx, []string{"doc-deep"}, "u-r")
	require.NoError(t, err)
	require.Len(t, got["doc-deep"].Folders, 2)
	assert.Equal(t, "f-a", got["doc-deep"].Folders[0].Roles[0].ResourceID, "the outermost folder first")
	assert.Equal(t, "f-c", got["doc-deep"].Folders[1].Person.ResourceID)

	reach, err := s.Access.DocRuleTurns(ctx, "u-r", permissions.DocsRead, []string{"project-general"}, []string{"project-general"})
	require.NoError(t, err)
	assert.Equal(t, []string{"f-a", "f-b"}, reach.DeniedFolders, "A's Deny reaches B")
	assert.Empty(t, reach.AllowedFolders, "C's Allow only gives back what the project already gives")
	assert.Empty(t, reach.Denied)

	reach, err = s.Access.DocRuleTurns(ctx, "u-r", permissions.DocsRead, []string{"project-general"}, []string{})
	require.NoError(t, err)
	assert.Equal(t, []string{"f-c", "f-d"}, reach.AllowedFolders, "C's Allow reaches D where the project reads nothing")
	assert.Empty(t, reach.DeniedFolders)
}
