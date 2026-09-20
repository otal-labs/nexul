package mentions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func mcpService() *Service {
	return New(Config{
		Tickets: &fakeTicketSource{
			tickets: map[string]Ticket{
				"t-1": {ID: "t-1", Title: "Fix the bug", Status: "open"},
			},
			search: []SearchHit{{ID: "t-1", Title: "Fix the bug"}},
		},
		Docs: &fakeDocSource{
			docs: map[string]Doc{
				"d-1": {ID: "d-1", Title: "Architecture"},
			},
			search: []SearchHit{{ID: "d-1", Title: "Architecture"}},
		},
		Statuses:    &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open"}}},
		Access:      &fakeAccessChecker{canOpen: map[string]bool{"d-1": true}},
		Projects:    &fakeProjectSource{projects: map[string]Project{}},
		TicketTypes: &fakeTicketTypeSource{types: map[string]TicketType{}},
	})
}

func findTool(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

func TestMCPTools_Registered(t *testing.T) {
	tools := MCPTools(mcpService())
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	assert.True(t, names["mention_search"])
	assert.True(t, names["mention_resolve"])
}

func TestMentionSearchTool(t *testing.T) {
	tool := findTool(t, MCPTools(mcpService()), "mention_search")

	t.Run("requires query", func(t *testing.T) {
		_, err := tool.Call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("returns combined results", func(t *testing.T) {
		res, err := tool.Call(identity.WithActor(context.Background(), identity.Actor{ID: "u-1"}), map[string]any{"query": "fix"})
		require.NoError(t, err)
		results, ok := res.([]SearchResult)
		require.True(t, ok)
		require.NotEmpty(t, results)
		assert.Equal(t, "ticket", results[0].Type)
		assert.Equal(t, "t-1", results[0].ID)
	})
}

func TestMentionResolveTool(t *testing.T) {
	tool := findTool(t, MCPTools(mcpService()), "mention_resolve")

	t.Run("requires refs", func(t *testing.T) {
		_, err := tool.Call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("rejects empty refs", func(t *testing.T) {
		_, err := tool.Call(context.Background(), map[string]any{"refs": []any{}})
		require.Error(t, err)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("resolves batch", func(t *testing.T) {
		res, err := tool.Call(identity.WithActor(context.Background(), identity.Actor{ID: "u-1"}), map[string]any{
			"refs": []any{
				map[string]any{"type": "ticket", "id": "t-1"},
				map[string]any{"type": "doc", "id": "d-1"},
				map[string]any{"type": "ticket", "id": "missing"},
			},
		})
		require.NoError(t, err)
		chips, ok := res.([]Chip)
		require.True(t, ok)
		require.Len(t, chips, 2)
		assert.Equal(t, "Fix the bug", chips[0].Title)
		assert.Equal(t, "Open", chips[0].StatusLabel)
		assert.Equal(t, "Architecture", chips[1].Title)
		assert.True(t, chips[1].CanOpen)
	})
}
