package automations

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
	assert.ElementsMatch(t, []string{
		"automation_create", "automation_list", "automation_get",
		"automation_update_config", "automation_set_enabled",
		"automation_delete", "automation_mint_token", "automation_revoke_token",
	}, names)
}

func TestMCPTools_CreateAndLifecycle(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	tools := MCPTools(svc)
	ctx := identity.WithActor(context.Background(), identity.Actor{ID: "owner"})

	created, err := mcpToolByName(t, tools, "automation_create").Call(ctx, map[string]any{
		"name":   "My automation",
		"scopes": []any{"tickets:read"},
	})
	require.NoError(t, err)
	resp, ok := created.(tokenResponse)
	require.True(t, ok)
	id := resp.Automation.ID

	t.Run("get resolves it", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "automation_get").Call(ctx, map[string]any{"id": id})
		require.NoError(t, err)
		assert.Equal(t, id, got.(*Automation).ID)
	})
	t.Run("set_enabled toggles it", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "automation_set_enabled").Call(ctx, map[string]any{"id": id, "enabled": true})
		require.NoError(t, err)
		assert.True(t, got.(*Automation).Enabled)
	})
	t.Run("update_config replaces values", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "automation_update_config").Call(ctx, map[string]any{
			"id":            id,
			"config_values": map[string]any{"status": "done"},
		})
		require.NoError(t, err)
		assert.JSONEq(t, `{"status":"done"}`, string(got.(*Automation).ConfigValues))
	})
	t.Run("mint_token rotates it", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "automation_mint_token").Call(ctx, map[string]any{"id": id})
		require.NoError(t, err)
		minted := got.(tokenResponse)
		assert.NotEqual(t, resp.Token, minted.Token)
	})
	t.Run("revoke_token marks it revoked", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "automation_revoke_token").Call(ctx, map[string]any{"id": id})
		require.NoError(t, err)
		assert.NotNil(t, got.(*Automation).TokenRevokedAt)
	})
	t.Run("list returns it", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "automation_list").Call(ctx, map[string]any{})
		require.NoError(t, err)
		assert.Len(t, got.([]Automation), 1)
	})
	t.Run("delete removes it", func(t *testing.T) {
		got, err := mcpToolByName(t, tools, "automation_delete").Call(ctx, map[string]any{"id": id})
		require.NoError(t, err)
		assert.Equal(t, id, got.(map[string]string)["id"])
	})
}
