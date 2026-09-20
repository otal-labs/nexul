package memories

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func toolByName(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo()))
	require.Len(t, tools, 8)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{
		"memory_list", "memory_get", "memory_create", "memory_update", "memory_delete",
		"memory_list_versions", "memory_revert", "memory_clone",
	}, names)
}

func TestMCPTools_MemoryCreate(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo()))
	call := toolByName(t, tools, "memory_create").Call

	t.Run("happy path returns created memory as markdown", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"project_id": "project-1", "title": "Deploy quirks", "when_to_use": "when deploying", "body": "**bold**", "always_included": true})
		require.NoError(t, err)
		m, ok := got.(*Memory)
		require.True(t, ok)
		assert.Equal(t, "Deploy quirks", m.Title)
		assert.Equal(t, "project-1", m.ProjectID)
		assert.Equal(t, "**bold**", m.Body)
		assert.True(t, m.AlwaysIncluded)
	})
	t.Run("missing project_id and workspace_id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"title": "Title"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing title is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"project_id": "project-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("no project_id with workspace_id creates a workspace-scoped memory", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"workspace_id": "workspace-1", "title": "Team tone"})
		require.NoError(t, err)
		m, ok := got.(*Memory)
		require.True(t, ok)
		assert.Empty(t, m.ProjectID)
		assert.Equal(t, "workspace-1", m.WorkspaceID)
	})
}

func TestMCPTools_MemoryGet(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "memory_get").Call

	t.Run("returns stored memory", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.(*Memory).ID)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown memory is not found", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_MemoryList(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)
	_, err = s.Create(testCtx(), "", "workspace-1", "Team tone", "when", "body", false, "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "memory_list").Call

	t.Run("lists a project's memories, workspace ones included", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"project_id": "project-1"})
		require.NoError(t, err)
		items, ok := got.([]*Memory)
		require.True(t, ok)
		require.Len(t, items, 2)
	})
	t.Run("no project_id lists workspace-scoped memories only", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"workspace_id": "workspace-1"})
		require.NoError(t, err)
		items, ok := got.([]*Memory)
		require.True(t, ok)
		require.Len(t, items, 1)
		assert.Equal(t, "Team tone", items[0].Title)
	})
	t.Run("missing project_id and workspace_id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_MemoryUpdate(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "", "Title", "when", "v1", false, "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "memory_update").Call

	t.Run("updates fields", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID, "title": "Renamed", "body": "v2", "always_included": true})
		require.NoError(t, err)
		m := got.(*Memory)
		assert.Equal(t, "Renamed", m.Title)
		assert.Equal(t, "v2", m.Body)
		assert.True(t, m.AlwaysIncluded)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"title": "x"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing title is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": created.ID})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_MemoryListVersions(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "memory_list_versions").Call

	t.Run("lists version history", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		versions, ok := got.([]*MemoryVersion)
		require.True(t, ok)
		require.Len(t, versions, 1)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_MemoryRevert(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "", "Title", "when", "v1", false, "")
	require.NoError(t, err)
	_, err = s.Update(testCtx(), created.ID, "Title", "when", "v2", false, "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "memory_revert").Call

	t.Run("reverts to a numeric version and tags the version via mcp", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID, "version": float64(1)})
		require.NoError(t, err)
		m := got.(*Memory)
		assert.Equal(t, 3, m.Version)

		v, err := s.GetVersion(testCtx(), created.ID, 3)
		require.NoError(t, err)
		assert.Equal(t, "mcp", v.AuthorVia)
	})
	t.Run("accepts a numeric string version", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": created.ID, "version": "1"})
		require.NoError(t, err)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"version": float64(1)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-numeric version is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": created.ID, "version": true})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unparseable string version is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": created.ID, "version": "nope"})
		require.Error(t, err)
	})
}

func TestMCPTools_MemoryClone(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "memory_clone").Call

	t.Run("clones to another project", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID, "project_id": "project-2"})
		require.NoError(t, err)
		clone := got.(*Memory)
		assert.Equal(t, "project-2", clone.ProjectID)
	})
	t.Run("missing project_id and workspace_id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": created.ID})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("no project_id with workspace_id clones to the workspace", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID, "workspace_id": "workspace-1"})
		require.NoError(t, err)
		clone := got.(*Memory)
		assert.Empty(t, clone.ProjectID)
		assert.Equal(t, "workspace-1", clone.WorkspaceID)
	})
}

func TestMCPTools_MemoryDelete(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "", "Title", "when", "body", false, "")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "memory_delete").Call

	t.Run("deletes a memory", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		_, err = s.Get(testCtx(), created.ID)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}
