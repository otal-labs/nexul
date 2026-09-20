package docs

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
	require.Len(t, tools, 6)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{"doc_create", "doc_get", "doc_search", "doc_update", "doc_archive", "doc_restore"}, names)
}

func TestMCPTools_DocCreate(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo()))
	call := toolByName(t, tools, "doc_create").Call

	t.Run("happy path returns created doc", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"project_id": "project-1", "title": "Spec", "body": "body"})
		require.NoError(t, err)
		d, ok := got.(*Doc)
		require.True(t, ok)
		assert.Equal(t, "Spec", d.Title)
		assert.Equal(t, "project-1", d.ProjectID)
		assert.Equal(t, 1, d.Version)
	})
	t.Run("missing project_id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"title": "Spec", "body": "body"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing title is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"project_id": "project-1", "body": "body"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty title is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"project_id": "project-1", "title": "  "})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_DocGet(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "Spec", "body")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "doc_get").Call

	t.Run("returns stored doc", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.(*Doc).ID)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown doc is not found", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_DocSearch(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.Create(testCtx(), "project-1", "Storage Spine", "SQLite migrations")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "doc_search").Call

	t.Run("finds matching docs", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"query": "sqlite"})
		require.NoError(t, err)
		results, ok := got.([]SearchResult)
		require.True(t, ok)
		require.Len(t, results, 1)
	})
	t.Run("missing query is invalid", func(t *testing.T) {
		_, err := call(testCtx(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_DocUpdate(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "Spec", "v1 body")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "doc_update").Call

	t.Run("updates and bumps version", func(t *testing.T) {
		got, err := call(testCtx(), map[string]any{"id": created.ID, "title": "Renamed", "body": "v2 body"})
		require.NoError(t, err)
		d := got.(*Doc)
		assert.Equal(t, "Renamed", d.Title)
		assert.Equal(t, 2, d.Version)
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

func TestMCPTools_DocArchiveRestore(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "Spec", "v1")
	require.NoError(t, err)

	tools := MCPTools(s)
	t.Run("archives", func(t *testing.T) {
		got, err := toolByName(t, tools, "doc_archive").Call(testCtx(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		assert.True(t, got.(*Doc).Archived)
	})
	t.Run("restores", func(t *testing.T) {
		got, err := toolByName(t, tools, "doc_restore").Call(testCtx(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		assert.False(t, got.(*Doc).Archived)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := toolByName(t, tools, "doc_archive").Call(testCtx(), map[string]any{})
		require.Error(t, err)
	})
}

func TestMCPTools_DocGet_ReturnsMarkdown(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "Spec", "# Title\n\nBody **bold**")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "doc_get").Call

	got, err := call(testCtx(), map[string]any{"id": created.ID})
	require.NoError(t, err)
	d := got.(*Doc)
	assert.Equal(t, "# Title\n\nBody **bold**", d.Body, "LLM gets markdown, not JSON")
}

func TestMCPTools_DocCreate_AcceptsMarkdown(t *testing.T) {
	s := newTestService(newFakeRepo())
	call := toolByName(t, MCPTools(s), "doc_create").Call

	got, err := call(testCtx(), map[string]any{"project_id": "project-1", "title": "Spec", "body": "# Heading"})
	require.NoError(t, err)
	d := got.(*Doc)
	assert.Equal(t, "# Heading", d.Body, "create returns markdown rendering")
}

func TestMCPTools_DocUpdate_AcceptsMarkdown(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(testCtx(), "project-1", "Spec", "v1")
	require.NoError(t, err)
	call := toolByName(t, MCPTools(s), "doc_update").Call

	got, err := call(testCtx(), map[string]any{"id": created.ID, "title": "Renamed", "body": "**v2**"})
	require.NoError(t, err)
	d := got.(*Doc)
	assert.Equal(t, "**v2**", d.Body)
	assert.Equal(t, 2, d.Version)
}
