package docs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func (f *fakeRepo) ListFolders(_ context.Context, projectID string) ([]*Folder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureDefault(projectID)
	var out []*Folder
	for _, folder := range f.folders {
		if folder.ProjectID == projectID {
			out = append(out, folder)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetFolder(_ context.Context, id string) (*Folder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, folder := range f.folders {
		if folder.ID == id {
			return folder, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) DefaultFolder(_ context.Context, projectID string) (*Folder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ensureDefault(projectID), nil
}

// ensureDefault stands in for the Main folder storage seeds with every project.
func (f *fakeRepo) ensureDefault(projectID string) *Folder {
	for _, folder := range f.folders {
		if folder.ProjectID == projectID && folder.IsDefault {
			return folder
		}
	}
	main := &Folder{ID: "main-" + projectID, ProjectID: projectID, Name: "Main", IsDefault: true}
	f.folders = append([]*Folder{main}, f.folders...)
	return main
}

func (f *fakeRepo) CreateFolder(_ context.Context, folder *Folder, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.folders = append(f.folders, folder)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) RenameFolder(_ context.Context, folder *Folder, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, have := range f.folders {
		if have.ID == folder.ID {
			f.folders[i] = folder
			f.events = append(f.events, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) DeleteFolder(_ context.Context, id, toFolderID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.docs {
		if d.FolderID == id {
			d.FolderID = toFolderID
		}
	}
	for i, have := range f.folders {
		if have.ID == id {
			f.folders = append(f.folders[:i], f.folders[i+1:]...)
			f.events = append(f.events, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeRepo) SetDocFolder(_ context.Context, docID, folderID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.docs[docID]
	if !ok {
		return apperrs.ErrNotFound
	}
	d.FolderID = folderID
	f.events = append(f.events, evts...)
	return nil
}

func folderNames(folders []*Folder) []string {
	var out []string
	for _, f := range folders {
		out = append(out, f.Name)
	}
	return out
}

func TestCreate_LandsInTheProjectsDefaultFolder(t *testing.T) {
	s := newTestService(newFakeRepo())
	d, err := s.Create(testCtx(), "project-1", "EP01", "")
	require.NoError(t, err)
	assert.Equal(t, "main-project-1", d.FolderID)
}

func TestCreateInFolder(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	getSource, err := s.CreateFolder(testCtx(), "project-1", "GetSource")
	require.NoError(t, err)
	other, err := s.CreateFolder(testCtx(), "project-2", "Elsewhere")
	require.NoError(t, err)

	t.Run("lands in the folder it was created from", func(t *testing.T) {
		d, err := s.CreateInFolder(testCtx(), "project-1", getSource.ID, "EP02", "")
		require.NoError(t, err)
		assert.Equal(t, getSource.ID, d.FolderID)
		created := repo.eventsFor(TopicCreated)
		assert.Equal(t, getSource.ID, created[len(created)-1].Payload.(CreatedEvent).Doc.FolderID, "doc.created carries the folder")
	})
	t.Run("a folder of another project is invalid", func(t *testing.T) {
		_, err := s.CreateInFolder(testCtx(), "project-1", other.ID, "EP03", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("a missing folder is invalid", func(t *testing.T) {
		_, err := s.CreateInFolder(testCtx(), "project-1", "nope", "EP03", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
}

func TestCreateFolder(t *testing.T) {
	t.Run("a blank name is invalid", func(t *testing.T) {
		_, err := newTestService(newFakeRepo()).CreateFolder(testCtx(), "project-1", "  ")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("needs docs:write in the project", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, fakeAccess{can: true, projectAllow: []permissions.Action{permissions.DocsRead}}, nil)
		_, err := s.CreateFolder(testCtx(), "project-1", "GetSource")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Empty(t, repo.eventsFor(TopicFolderCreated))
	})
	t.Run("creates a trimmed, non-default folder and enqueues doc.folder.created", func(t *testing.T) {
		repo := newFakeRepo()
		f, err := newTestService(repo).CreateFolder(testCtx(), "project-1", " GetSource ")
		require.NoError(t, err)
		assert.Equal(t, "GetSource", f.Name)
		assert.False(t, f.IsDefault)
		require.Len(t, repo.eventsFor(TopicFolderCreated), 1)
	})
}

func TestRenameFolder(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	main, err := repo.DefaultFolder(t.Context(), "project-1")
	require.NoError(t, err)

	renamed, err := s.RenameFolder(testCtx(), main.ID, "Episodes")
	require.NoError(t, err, "the default folder can be renamed")
	assert.Equal(t, "Episodes", renamed.Name)
	assert.True(t, renamed.IsDefault, "a rename keeps it the default")
	updated := repo.eventsFor(TopicFolderUpdated)
	require.Len(t, updated, 1)
	assert.Equal(t, "Main", updated[0].Payload.(FolderUpdatedEvent).PreviousName)

	_, err = s.RenameFolder(testCtx(), main.ID, "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = s.RenameFolder(testCtx(), "nope", "x")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestDeleteFolder_MovesItsDocsToTheDefaultFolder(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	getSource, err := s.CreateFolder(testCtx(), "project-1", "GetSource")
	require.NoError(t, err)
	d, err := s.CreateInFolder(testCtx(), "project-1", getSource.ID, "EP01", "")
	require.NoError(t, err)

	require.NoError(t, s.DeleteFolder(testCtx(), getSource.ID))

	got, err := s.Get(testCtx(), d.ID)
	require.NoError(t, err, "the doc survives its folder")
	assert.Equal(t, "main-project-1", got.FolderID)
	folders, err := s.ListFolders(testCtx(), "project-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"Main"}, folderNames(folders))
	deleted := repo.eventsFor(TopicFolderDeleted)
	require.Len(t, deleted, 1)
	assert.Equal(t, "main-project-1", deleted[0].Payload.(FolderDeletedEvent).MovedToFolderID)
}

func TestDeleteFolder_RefusesTheDefaultFolder(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	main, err := repo.DefaultFolder(t.Context(), "project-1")
	require.NoError(t, err)

	err = s.DeleteFolder(testCtx(), main.ID)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = repo.GetFolder(t.Context(), main.ID)
	require.NoError(t, err, "the default folder is still there")
	assert.Empty(t, repo.eventsFor(TopicFolderDeleted))
}

func TestDeleteFolder_NeedsDocsWrite(t *testing.T) {
	repo := newFakeRepo()
	getSource, err := newTestService(repo).CreateFolder(testCtx(), "project-1", "GetSource")
	require.NoError(t, err)
	reader := NewService(repo, fakeAccess{can: true, projectAllow: []permissions.Action{permissions.DocsRead, permissions.DocsDelete}}, nil)

	require.ErrorIs(t, reader.DeleteFolder(testCtx(), getSource.ID), apperrs.ErrForbidden)
	_, err = repo.GetFolder(t.Context(), getSource.ID)
	require.NoError(t, err)
}

func TestMoveToFolder(t *testing.T) {
	setup := func(t *testing.T) (*fakeRepo, *Doc, *Folder) {
		repo := newFakeRepo()
		s := newTestService(repo)
		d, err := s.Create(testCtx(), "project-1", "EP01", "")
		require.NoError(t, err)
		getSource, err := s.CreateFolder(testCtx(), "project-1", "GetSource")
		require.NoError(t, err)
		return repo, d, getSource
	}

	t.Run("needs docs:write on the doc", func(t *testing.T) {
		repo, d, getSource := setup(t)
		reader := NewService(repo, fakeAccess{allow: []permissions.Action{permissions.DocsRead}}, nil)
		_, err := reader.MoveToFolder(testCtx(), d.ID, getSource.ID)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Equal(t, "main-project-1", repo.docs[d.ID].FolderID)
		assert.Empty(t, repo.eventsFor(TopicMoved))
	})
	t.Run("a folder of another project is invalid", func(t *testing.T) {
		repo, d, _ := setup(t)
		elsewhere, err := newTestService(repo).CreateFolder(testCtx(), "project-2", "Elsewhere")
		require.NoError(t, err)
		_, err = newTestService(repo).MoveToFolder(testCtx(), d.ID, elsewhere.ID)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("moves the doc and enqueues doc.moved with where it came from", func(t *testing.T) {
		repo, d, getSource := setup(t)
		moved, err := newTestService(repo).MoveToFolder(testCtx(), d.ID, getSource.ID)
		require.NoError(t, err)
		assert.Equal(t, getSource.ID, moved.FolderID)
		assert.Equal(t, getSource.ID, repo.docs[d.ID].FolderID)
		evts := repo.eventsFor(TopicMoved)
		require.Len(t, evts, 1)
		assert.Equal(t, "main-project-1", evts[0].Payload.(MovedEvent).FromFolderID)
		assert.Empty(t, repo.eventsFor(TopicUpdated), "a move is not an edit, so nobody is notified of one")
	})
	t.Run("moving into its own folder changes nothing", func(t *testing.T) {
		repo, d, _ := setup(t)
		_, err := newTestService(repo).MoveToFolder(testCtx(), d.ID, "main-project-1")
		require.NoError(t, err)
		assert.Empty(t, repo.eventsFor(TopicMoved))
	})
}

func TestListFolders_WritersSeeEveryFolderReadersOnlyThoseHoldingADocTheyCanOpen(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	getSource, err := s.CreateFolder(testCtx(), "project-1", "GetSource")
	require.NoError(t, err)
	_, err = s.CreateFolder(testCtx(), "project-1", "Empty")
	require.NoError(t, err)
	_, err = s.CreateInFolder(testCtx(), "project-1", getSource.ID, "EP01", "")
	require.NoError(t, err)

	writer, err := s.ListFolders(testCtx(), "project-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"Main", "GetSource", "Empty"}, folderNames(writer), "default first, then creation order")

	reader := NewService(repo, fakeAccess{can: true, projectAllow: []permissions.Action{permissions.DocsRead}}, nil)
	got, err := reader.ListFolders(testCtx(), "project-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"GetSource"}, folderNames(got))

	blind := NewService(repo, fakeAccess{can: false, projectAllow: []permissions.Action{}}, nil)
	got, err = blind.ListFolders(testCtx(), "project-1")
	require.NoError(t, err)
	assert.Empty(t, got, "a folder whose docs the viewer cannot open stays hidden")
}

func TestClone_IntoAnotherProject_LandsInItsDefaultFolder(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	getSource, err := s.CreateFolder(testCtx(), "project-1", "GetSource")
	require.NoError(t, err)
	source, err := s.CreateInFolder(testCtx(), "project-1", getSource.ID, "EP01", "")
	require.NoError(t, err)

	elsewhere, err := s.Clone(testCtx(), source.ID, "project-2")
	require.NoError(t, err)
	assert.Equal(t, "main-project-2", elsewhere.FolderID)

	beside, err := s.Clone(testCtx(), source.ID, "")
	require.NoError(t, err)
	assert.Equal(t, getSource.ID, beside.FolderID, "a duplicate in its own project sits beside the original")
}
