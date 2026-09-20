package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func mcpToolByName(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
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
	tools := MCPTools(newService(newFakeRepo(), newFakeUsers()))
	require.Len(t, tools, 2)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{"access_list_grants", "access_set_grants"}, names)
}

func TestMCPTools_ListGrants(t *testing.T) {
	// Ticket 11: no instance-wide bypass — "owner" proves access through a real permissions:write overwrite on doc-1, same as the HTTP handler tests.
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypeDoc, "doc-1", "owner", permissions.SetOf(permissions.PermissionsWrite))
	setOverwrite(repo, resourceTypeDoc, "doc-1", "alice", permissions.SetOf(permissions.DocsRead))
	call := mcpToolByName(t, MCPTools(svc), "access_list_grants").Call
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner", CanCreateWorkspace: true})

	t.Run("returns grants", func(t *testing.T) {
		got, err := call(ctx, map[string]any{"doc_id": "doc-1"})
		require.NoError(t, err)
		grants := got.([]*Overwrite)
		require.Len(t, grants, 2)
	})
	t.Run("missing doc_id is invalid", func(t *testing.T) {
		_, err := call(ctx, map[string]any{})
		require.Error(t, err)
	})
}

func TestMCPTools_SetGrants(t *testing.T) {
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypeDoc, "doc-1", "owner", permissions.SetOf(permissions.PermissionsWrite))
	call := mcpToolByName(t, MCPTools(svc), "access_set_grants").Call
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner", CanCreateWorkspace: true})

	t.Run("applies grants", func(t *testing.T) {
		_, err := call(ctx, map[string]any{
			"doc_ids":  []any{"doc-1"},
			"user_ids": []any{"alice"},
			"actions":  []any{"docs:read", "docs:write"},
			"grant":    true,
		})
		require.NoError(t, err)
		g, err := repo.Get(context.Background(), resourceTypeDoc, "doc-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.DocsWrite), g.Allow)
	})
	t.Run("unknown action is invalid", func(t *testing.T) {
		_, err := call(ctx, map[string]any{
			"doc_ids":  []any{"doc-1"},
			"user_ids": []any{"alice"},
			"actions":  []any{"comment"},
			"grant":    true,
		})
		require.Error(t, err)
	})
	t.Run("missing arrays are invalid", func(t *testing.T) {
		_, err := call(ctx, map[string]any{"doc_ids": []any{}, "user_ids": []any{"alice"}, "actions": []any{"docs:read"}, "grant": true})
		require.Error(t, err)
	})
	t.Run("non-actor ctx is denied", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{
			"doc_ids":  []any{"doc-1"},
			"user_ids": []any{"alice"},
			"actions":  []any{"docs:read"},
			"grant":    true,
		})
		require.Error(t, err)
	})
}

func TestMCPTools_SetGrants_Play(t *testing.T) {
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypePlay, "play-1", "owner", permissions.SetOf(permissions.PlaysWrite))
	call := mcpToolByName(t, MCPTools(svc), "access_set_grants").Call
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	t.Run("denies a user's plays:run", func(t *testing.T) {
		_, err := call(ctx, map[string]any{
			"resource_type": "play",
			"resource_ids":  []any{"play-1"},
			"user_ids":      []any{"alice"},
			"actions":       []any{"plays:run"},
			"grant":         false,
		})
		require.NoError(t, err)
		g, err := repo.Get(context.Background(), resourceTypePlay, "play-1", "alice")
		require.NoError(t, err)
		assert.Equal(t, permissions.SetOf(permissions.PlaysRun), g.Deny)
	})
	t.Run("missing resource_ids is invalid", func(t *testing.T) {
		_, err := call(ctx, map[string]any{
			"resource_type": "play",
			"user_ids":      []any{"alice"},
			"actions":       []any{"plays:run"},
			"grant":         false,
		})
		require.Error(t, err)
	})
}

func TestMCPTools_ListGrants_Play(t *testing.T) {
	svc, repo, _ := newAccessHarness()
	setOverwrite(repo, resourceTypePlay, "play-1", "owner", permissions.SetOf(permissions.PlaysWrite))
	require.NoError(t, repo.Set(context.Background(), resourceTypePlay, "play-1", "alice", nil, permissions.SetOf(permissions.PlaysRun)))
	call := mcpToolByName(t, MCPTools(svc), "access_list_grants").Call
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	t.Run("returns the play's exclusions", func(t *testing.T) {
		got, err := call(ctx, map[string]any{"resource_type": "play", "resource_id": "play-1"})
		require.NoError(t, err)
		grants := got.([]*Overwrite)
		require.Len(t, grants, 2)
		ids := []string{grants[0].UserID, grants[1].UserID}
		assert.ElementsMatch(t, []string{"owner", "alice"}, ids)
	})
	t.Run("missing resource_id is invalid", func(t *testing.T) {
		_, err := call(ctx, map[string]any{"resource_type": "play"})
		require.Error(t, err)
	})
}
