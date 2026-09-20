package plays

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
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
	tools := MCPTools(newTestService(newFakeRepo(), allowAll("owner")))
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{"play_list", "play_create", "play_update", "play_delete"}, names)
}

func TestMCPTools_CreateListUpdateDelete(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	tools := MCPTools(svc)
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	created, err := mcpToolByName(t, tools, "play_create").Call(ctx, map[string]any{
		"workspace_id":    workspaceID,
		"label":           "Fix with AI",
		"type":            "ticket",
		"show_when_stage": "progress",
	})
	require.NoError(t, err)
	p, ok := created.(*Play)
	require.True(t, ok)

	t.Run("list finds it", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "play_list").Call(ctx, map[string]any{"workspace_id": workspaceID})
		require.NoError(t, err)
		list, ok := got.([]*Play)
		require.True(t, ok)
		require.Len(t, list, 1)
		assert.Equal(t, p.ID, list[0].ID)
	})
	t.Run("update replaces fields", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "play_update").Call(ctx, map[string]any{
			"workspace_id":    workspaceID,
			"id":              p.ID,
			"label":           "Fix with AI v2",
			"show_when_stage": "review",
		})
		require.NoError(t, err)
		assert.Equal(t, "Fix with AI v2", got.(*Play).Label)
		assert.Equal(t, StageReview, *got.(*Play).ShowWhenStage)
	})
	t.Run("delete removes it", func(t *testing.T) {
		_, err := mcpToolByName(t, tools, "play_delete").Call(ctx, map[string]any{
			"workspace_id": workspaceID,
			"id":           p.ID,
		})
		require.NoError(t, err)
		_, err = svc.Get(ctx, workspaceID, p.ID)
		require.Error(t, err)
	})
}

func TestMCPTools_Create_MissingRequiredArg(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	tools := MCPTools(svc)
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	_, err := mcpToolByName(t, tools, "play_create").Call(ctx, map[string]any{"workspace_id": workspaceID})
	require.Error(t, err)
}

func TestMCPTools_Create_WithExcludedProjectsAndDocType(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	tools := MCPTools(svc)
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	created, err := mcpToolByName(t, tools, "play_create").Call(ctx, map[string]any{
		"workspace_id":         workspaceID,
		"label":                "To tickets via AI",
		"type":                 "doc",
		"excluded_project_ids": []any{"proj-1", "proj-2"},
	})
	require.NoError(t, err)
	p := created.(*Play)
	assert.Equal(t, []string{"proj-1", "proj-2"}, p.ExcludedProjectIDs)
	assert.Nil(t, p.ShowWhenStage)
}

func TestMCPTools_PlayList_TypeArgSwitchesToApplicable(t *testing.T) {
	perm := runnerPerm("alice")
	svc := newTestService(newFakeRepo(), perm)
	tools := MCPTools(svc)
	ownerCtx := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	created, err := mcpToolByName(t, tools, "play_create").Call(ownerCtx, map[string]any{
		"workspace_id":    workspaceID,
		"label":           "Fix with AI",
		"type":            "ticket",
		"enabled":         true,
		"show_when_stage": "progress",
	})
	require.NoError(t, err)
	p := created.(*Play)

	t.Run("with type set, filters to what alice may run", func(t *testing.T) {
		aliceCtx := identity.WithActor(context.Background(), identity.Actor{ID: "alice"})
		got, err := mcpToolByName(t, tools, "play_list").Call(aliceCtx, map[string]any{
			"workspace_id": workspaceID, "type": "ticket", "stage": "progress", "project_id": "proj-1",
		})
		require.NoError(t, err)
		list, ok := got.([]*Play)
		require.True(t, ok)
		require.Len(t, list, 1)
		assert.Equal(t, p.ID, list[0].ID)
	})
	t.Run("denied user gets none", func(t *testing.T) {
		perm.deny("alice", p.ID)
		aliceCtx := identity.WithActor(context.Background(), identity.Actor{ID: "alice"})
		got, err := mcpToolByName(t, tools, "play_list").Call(aliceCtx, map[string]any{
			"workspace_id": workspaceID, "type": "ticket", "stage": "progress", "project_id": "proj-1",
		})
		require.NoError(t, err)
		assert.Empty(t, got.([]*Play))
	})
	t.Run("without type, plays:read holder still lists the raw definitions", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "play_list").Call(ownerCtx, map[string]any{"workspace_id": workspaceID})
		require.NoError(t, err)
		list, ok := got.([]*Play)
		require.True(t, ok)
		require.Len(t, list, 1)
	})
}

func TestStringsArg_NonArray_ReturnsNil(t *testing.T) {
	assert.Nil(t, stringsArg("not-an-array"))
}

func TestOptionalStage_Empty_ReturnsNil(t *testing.T) {
	assert.Nil(t, optionalStage(""))
	assert.Nil(t, optionalStage(nil))
}
