package codereview

import (
	"context"
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
	tools := MCPTools(newTestService(newFakeRepo(), newFakeBus()))
	require.Len(t, tools, 2)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{"review_list_by_ticket", "review_get"}, names)
}

func TestMCPTools_ReviewListByTicket(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, newFakeBus())
	mustCreate(t, s, "acme/app", 42)
	repo.link("t-1", "acme/app", 42)
	call := toolByName(t, MCPTools(s), "review_list_by_ticket").Call

	t.Run("returns the ticket reviews", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"ticket_id": "t-1"})
		require.NoError(t, err)
		rs, ok := got.([]*CodeReview)
		require.True(t, ok)
		require.Len(t, rs, 1)
		assert.Equal(t, 42, rs[0].PRNumber)
	})
	t.Run("missing ticket_id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ReviewGet(t *testing.T) {
	s := newTestService(newFakeRepo(), newFakeBus())
	created := mustCreate(t, s, "acme/app", 42)
	call := toolByName(t, MCPTools(s), "review_get").Call

	t.Run("returns the stored review", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": created.ID})
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.(*CodeReview).ID)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown review is not found", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}
