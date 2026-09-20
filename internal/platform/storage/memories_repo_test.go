package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestMemory(id, projectID string) *memories.Memory {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	return &memories.Memory{
		ID: id, WorkspaceID: "workspace-default", ProjectID: projectID,
		Title: "Deploy quirks", WhenToUse: "use this if touching deploy config", Body: "Body", Version: 1,
		CreatedBy: "user-1", CreatedAt: now, UpdatedBy: "user-1", UpdatedAt: now,
	}
}

func TestMemoriesRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Memories.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestMemoriesRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestMemory("mem-1", "project-general")
	require.NoError(t, s.Memories.Create(context.Background(), want, ""))

	got, err := s.Memories.GetByID(context.Background(), "mem-1")
	require.NoError(t, err)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.WorkspaceID, got.WorkspaceID)
	assert.Equal(t, want.ProjectID, got.ProjectID)
	assert.Equal(t, want.Title, got.Title)
	assert.Equal(t, want.WhenToUse, got.WhenToUse)
	assert.Equal(t, want.Body, got.Body)
	assert.False(t, got.AlwaysIncluded)
	assert.Equal(t, 1, got.Version)
	assert.Equal(t, want.CreatedBy, got.CreatedBy)
	assert.Equal(t, want.CreatedAt, got.CreatedAt)
	assert.Equal(t, want.UpdatedBy, got.UpdatedBy)
	assert.Equal(t, want.UpdatedAt, got.UpdatedAt)
}

func TestMemoriesRepo_Create_WritesVersion1(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := newTestMemory("mem-1", "project-general")
	require.NoError(t, s.Memories.Create(context.Background(), m, "mcp"))

	versions, err := s.Memories.ListVersions(context.Background(), "mem-1")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, 1, versions[0].Version)
	assert.Equal(t, m.Title, versions[0].Title)
	assert.Equal(t, m.Body, versions[0].Body)
	assert.Equal(t, "user-1", versions[0].AuthorID)
	assert.Equal(t, "mcp", versions[0].AuthorVia)
}

func TestMemoriesRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "project-general"), ""))
	err := s.Memories.Create(context.Background(), newTestMemory("mem-1", "project-general"), "")
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestMemoriesRepo_Create_UnknownProjectIsConflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Memories.Create(context.Background(), newTestMemory("mem-1", "does-not-exist"), "")
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestMemoriesRepo_ListByProject_IncludesTheSeededDefault(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "p-1"), ""))

	got, err := s.Memories.ListByProject(context.Background(), "p-1", "workspace-default")
	require.NoError(t, err)
	// One row seeded at project creation (ProjectsRepo.Create) plus the one this test just created.
	require.Len(t, got, 2)
	assert.Equal(t, "Working in this project", got[0].Title)
	assert.True(t, got[0].AlwaysIncluded)
	assert.Equal(t, "mem-1", got[1].ID)
}

func TestMemoriesRepo_ListByProject_WorkspaceScopedComesFirst(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "p-1"), ""))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-ws", ""), ""))

	got, err := s.Memories.ListByProject(context.Background(), "p-1", "workspace-default")
	require.NoError(t, err)
	// Seeded default + mem-1 (both project p-1) plus the workspace-scoped mem-ws.
	require.Len(t, got, 3)
	assert.Equal(t, "mem-ws", got[0].ID)
	assert.Empty(t, got[0].ProjectID)
}

func TestMemoriesRepo_ListWorkspaceScoped_ReturnsOnlyWorkspaceMemories(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "p-1"), ""))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-ws", ""), ""))

	got, err := s.Memories.ListWorkspaceScoped(context.Background(), "workspace-default")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "mem-ws", got[0].ID)
	assert.Empty(t, got[0].ProjectID)
}

func TestMemoriesRepo_ListByWorkspace_OrdersByProjectThenCreatedAt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-2", "Frontend", 1)))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "p-1"), ""))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-2", "p-2"), ""))

	got, err := s.Memories.ListByWorkspace(context.Background(), "workspace-default")
	require.NoError(t, err)
	// The two fresh projects' seeded defaults plus project-general's own seeded row plus the two created here.
	require.Len(t, got, 5)
}

func TestMemoriesRepo_ListByWorkspace_WorkspaceScopedSortsFirst(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "p-1"), ""))
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-ws", ""), ""))

	got, err := s.Memories.ListByWorkspace(context.Background(), "workspace-default")
	require.NoError(t, err)
	require.NotEmpty(t, got)
	assert.Equal(t, "mem-ws", got[0].ID)
	assert.Empty(t, got[0].ProjectID)
}

func TestMemoriesRepo_Update_PersistsFieldsAndAppendsVersion(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := newTestMemory("mem-1", "project-general")
	require.NoError(t, s.Memories.Create(context.Background(), m, ""))

	m.Title = "Deploy quirks v2"
	m.WhenToUse = "updated hint"
	m.Body = "New body"
	m.AlwaysIncluded = true
	m.Version = 2
	m.UpdatedBy = "user-2"
	m.UpdatedAt = m.UpdatedAt.Add(time.Hour)
	require.NoError(t, s.Memories.Update(context.Background(), m, ""))

	got, err := s.Memories.GetByID(context.Background(), "mem-1")
	require.NoError(t, err)
	assert.Equal(t, "Deploy quirks v2", got.Title)
	assert.Equal(t, "updated hint", got.WhenToUse)
	assert.Equal(t, "New body", got.Body)
	assert.True(t, got.AlwaysIncluded)
	assert.Equal(t, 2, got.Version)
	assert.Equal(t, "user-2", got.UpdatedBy)

	versions, err := s.Memories.ListVersions(context.Background(), "mem-1")
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, 2, versions[0].Version) // newest first
	assert.Equal(t, "user-2", versions[0].AuthorID)
	assert.Equal(t, 1, versions[1].Version)
}

func TestMemoriesRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Memories.Update(context.Background(), newTestMemory("missing", "project-general"), "")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestMemoriesRepo_GetVersion(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	m := newTestMemory("mem-1", "project-general")
	require.NoError(t, s.Memories.Create(context.Background(), m, ""))

	v, err := s.Memories.GetVersion(context.Background(), "mem-1", 1)
	require.NoError(t, err)
	assert.Equal(t, m.Title, v.Title)
	assert.Equal(t, m.Body, v.Body)

	_, err = s.Memories.GetVersion(context.Background(), "mem-1", 2)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestMemoriesRepo_Delete_RemovesMemory(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "project-general"), ""))
	require.NoError(t, s.Memories.Delete(context.Background(), "mem-1"))
	_, err := s.Memories.GetByID(context.Background(), "mem-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestMemoriesRepo_Delete_CascadesVersions(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Memories.Create(context.Background(), newTestMemory("mem-1", "project-general"), ""))
	require.NoError(t, s.Memories.Delete(context.Background(), "mem-1"))

	versions, err := s.Memories.ListVersions(context.Background(), "mem-1")
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestMemoriesRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Memories.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestProjectsRepo_Create_SeedsAnAlwaysIncludedMemory(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))

	got, err := s.Memories.ListByProject(context.Background(), "p-1", "workspace-default")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Working in this project", got[0].Title)
	assert.True(t, got[0].AlwaysIncluded)
	assert.Equal(t, "workspace-default", got[0].WorkspaceID)
	assert.Equal(t, 1, got[0].Version)
	assert.NotEmpty(t, got[0].Body)

	versions, err := s.Memories.ListVersions(context.Background(), got[0].ID)
	require.NoError(t, err)
	require.Len(t, versions, 1)
}
