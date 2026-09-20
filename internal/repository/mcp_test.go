package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestMCPTools_Shape(t *testing.T) {
	tools := MCPTools(&fakeScanner{})
	require.Len(t, tools, 2)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{"repository_list", "repository_scan"}, names)
}

func TestMCPTools_RepositoryList(t *testing.T) {
	s := &fakeScanner{repos: []Repo{{ID: 1, Owner: "acme", Name: "api", FullName: "acme/api"}}}
	call := toolByName(t, MCPTools(s), "repository_list").Call
	got, err := call(context.Background(), map[string]any{})
	require.NoError(t, err)
	repos, ok := got.([]Repo)
	require.True(t, ok)
	require.Len(t, repos, 1)
	assert.Equal(t, "acme/api", repos[0].FullName)
}

func TestMCPTools_RepositoryList_ProviderError(t *testing.T) {
	s := &fakeScanner{listErr: apperrors.ErrUnauthorized}
	call := toolByName(t, MCPTools(s), "repository_list").Call
	_, err := call(context.Background(), map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrUnauthorized))
}

func TestMCPTools_RepositoryScan(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		s := &fakeScanner{
			resolvedRef: "main",
			tree:        []TreeEntry{{Path: "docker-compose.yml", Type: "blob"}},
			files: map[string][]byte{
				"docker-compose.yml": []byte("services:\n  web:\n    image: nginx\n    ports:\n      - \"80:80\"\n"),
			},
		}
		call := toolByName(t, MCPTools(s), "repository_scan").Call
		got, err := call(context.Background(), map[string]any{"owner": "acme", "name": "api"})
		require.NoError(t, err)
		result, ok := got.(*ScanResult)
		require.True(t, ok)
		assert.Equal(t, "main", result.DefaultBranch)
		require.Len(t, result.Candidates, 1)
		assert.Equal(t, KindCompose, result.Candidates[0].Kind)
	})
	t.Run("missing owner is invalid", func(t *testing.T) {
		call := toolByName(t, MCPTools(&fakeScanner{}), "repository_scan").Call
		_, err := call(context.Background(), map[string]any{"name": "api"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrors.ErrInvalid))
	})
	t.Run("ref is passed through", func(t *testing.T) {
		s := &fakeScanner{}
		call := toolByName(t, MCPTools(s), "repository_scan").Call
		got, err := call(context.Background(), map[string]any{"owner": "acme", "name": "api", "ref": "dev"})
		require.NoError(t, err)
		assert.Equal(t, "dev", got.(*ScanResult).DefaultBranch)
	})
}

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
