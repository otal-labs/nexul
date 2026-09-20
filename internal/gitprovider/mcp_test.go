package gitprovider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestMCPTools_Names(t *testing.T) {
	tools := MCPTools(&fakeProvider{})
	require.Len(t, tools, 2)
	names := []string{tools[0].Name, tools[1].Name}
	assert.ElementsMatch(t, []string{"git_list_prs", "git_get_pr"}, names)
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
}

func TestMCPTools_ListPRs(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		want := []*PR{{Number: 1}, {Number: 2}}
		tools := MCPTools(&fakeProvider{prs: want})
		got, err := tools[0].Call(context.Background(), map[string]any{"owner": "acme", "repo": "app", "state": "closed", "limit": float64(10)})
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
	t.Run("missing owner is invalid params", func(t *testing.T) {
		tools := MCPTools(&fakeProvider{})
		_, err := tools[0].Call(context.Background(), map[string]any{"repo": "app"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing repo is invalid params", func(t *testing.T) {
		tools := MCPTools(&fakeProvider{})
		_, err := tools[0].Call(context.Background(), map[string]any{"owner": "acme"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_GetPR(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		want := &PR{Number: 7, Title: "Fix login"}
		tools := MCPTools(&fakeProvider{pr: want})
		got, err := tools[1].Call(context.Background(), map[string]any{"owner": "acme", "repo": "app", "number": float64(7)})
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
	t.Run("non-positive number is invalid params", func(t *testing.T) {
		tools := MCPTools(&fakeProvider{})
		_, err := tools[1].Call(context.Background(), map[string]any{"owner": "acme", "repo": "app", "number": float64(0)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing number is invalid params", func(t *testing.T) {
		tools := MCPTools(&fakeProvider{})
		_, err := tools[1].Call(context.Background(), map[string]any{"owner": "acme", "repo": "app"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}
