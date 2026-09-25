package plays

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// callTool runs a tool the way the adapter does, with raw JSON arguments; "$WS" stands for the test workspace.
func callTool(t *testing.T, tools []mcptool.Tool, ctx context.Context, name, args string) (any, error) {
	t.Helper()
	args = strings.ReplaceAll(args, "$WS", workspaceID)
	for _, tool := range tools {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func toolNames(tools []mcptool.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name)
	}
	return out
}

// playToolsFixture seeds one enabled ticket play; "owner" manages plays, "reader" only reads, "alice" only runs.
func playToolsFixture(t *testing.T) (*fakePerm, []mcptool.Tool, *Play) {
	t.Helper()
	perm := newFakePerm(map[string][]permissions.Action{
		"owner":  {permissions.PlaysRead, permissions.PlaysWrite, permissions.PlaysDelete},
		"reader": {permissions.PlaysRead},
		"alice":  {permissions.PlaysRun},
	})
	svc := newTestService(newFakeRepo(), perm)
	p, err := svc.Create(ctxAs("owner"), workspaceID, CreateInput{
		Label: "Fix with AI", Type: TypeTicket, Description: "Fixes it.", Instructions: "Fix the ticket.",
		Enabled: true, ShowWhenStage: ticketStage(), ExcludedProjectIDs: []string{"proj-9"},
	})
	require.NoError(t, err)
	return perm, MCPTools(svc), p
}

func TestMCPTools_Surface(t *testing.T) {
	_, tools, _ := playToolsFixture(t)
	assert.ElementsMatch(t, []string{"play_list", "play_create", "play_update", "play_delete"}, toolNames(tools))
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Title, tool.Name)
	}
}

func TestPlayTools_ErrorPaths(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		tool  string
		args  string
		want  error
	}{
		{"create without a label is invalid", "owner", "play_create", `{"workspace_id":"$WS","type":"doc"}`, apperrs.ErrInvalid},
		{"a ticket play without a stage is invalid", "owner", "play_create", `{"workspace_id":"$WS","label":"Go","type":"ticket"}`, apperrs.ErrInvalid},
		{"an unknown play type is invalid", "owner", "play_list", `{"workspace_id":"$WS","type":"bogus"}`, apperrs.ErrInvalid},
		{"project_id without type is invalid, not ignored", "owner", "play_list", `{"workspace_id":"$WS","project_id":"proj-1"}`, apperrs.ErrInvalid},
		{"stage without type is invalid, not ignored", "owner", "play_list", `{"workspace_id":"$WS","stage":"progress"}`, apperrs.ErrInvalid},
		{"type cannot be updated", "owner", "play_update", `{"workspace_id":"$WS","id":"$ID","type":"doc"}`, apperrs.ErrInvalid},
		{"an unknown stage is invalid", "owner", "play_update", `{"workspace_id":"$WS","id":"$ID","show_when_stage":"shipped"}`, apperrs.ErrInvalid},
		{"update of a missing play", "owner", "play_update", `{"workspace_id":"$WS","id":"missing","label":"x"}`, apperrs.ErrNotFound},
		{"update of a play in another workspace", "owner", "play_update", `{"workspace_id":"$WS-other","id":"$ID","label":"x"}`, apperrs.ErrNotFound},
		{"delete of a missing play", "owner", "play_delete", `{"workspace_id":"$WS","id":"missing"}`, apperrs.ErrNotFound},
		{"a reader cannot create", "reader", "play_create", `{"workspace_id":"$WS","label":"Go","type":"doc"}`, apperrs.ErrForbidden},
		{"a reader cannot update", "reader", "play_update", `{"workspace_id":"$WS","id":"$ID","label":"x"}`, apperrs.ErrForbidden},
		{"a reader cannot delete", "reader", "play_delete", `{"workspace_id":"$WS","id":"$ID"}`, apperrs.ErrForbidden},
		{"a runner without plays:read cannot list every definition", "alice", "play_list", `{"workspace_id":"$WS"}`, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, tools, p := playToolsFixture(t)
			_, err := callTool(t, tools, ctxAs(tt.actor), tt.tool, strings.ReplaceAll(tt.args, "$ID", p.ID))
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPlayUpdate_MissingPlayNamesTheListTool(t *testing.T) {
	_, tools, _ := playToolsFixture(t)
	_, err := callTool(t, tools, ctxAs("owner"), "play_update", `{"workspace_id":"$WS","id":"missing","label":"x"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, err.Error(), "play_list")
}

func TestPlayUpdate_OmittedFieldsKeepTheirValues(t *testing.T) {
	_, tools, p := playToolsFixture(t)
	update := func(args string) playResult {
		t.Helper()
		out, err := callTool(t, tools, ctxAs("owner"), "play_update", strings.ReplaceAll(args, "$ID", p.ID))
		require.NoError(t, err)
		return out.(playResult)
	}

	got := update(`{"workspace_id":"$WS","id":"$ID","label":"Fix it"}`)
	assert.Equal(t, "Fix it", got.Label)
	assert.Equal(t, "Fixes it.", got.Description)
	assert.Equal(t, "Fix the ticket.", got.Instructions, "a rename keeps the instructions")
	assert.True(t, got.Enabled, "a rename keeps the play enabled")
	assert.Equal(t, StageProgress, *got.ShowWhenStage)
	assert.Equal(t, []string{"proj-9"}, got.ExcludedProjectIDs)

	got = update(`{"workspace_id":"$WS","id":"$ID","enabled":false,"show_when_stage":"review","instructions":"Fix it well.","label":null}`)
	assert.False(t, got.Enabled)
	assert.Equal(t, "Fix it well.", got.Instructions)
	assert.Equal(t, StageReview, *got.ShowWhenStage)
	assert.Equal(t, "Fix it", got.Label, "a null field is the same as an omitted one")

	got = update(`{"workspace_id":"$WS","id":"$ID","excluded_project_ids":[],"description":""}`)
	assert.Empty(t, got.ExcludedProjectIDs, "an empty list clears the exclusions")
	assert.Empty(t, got.Description, "an empty string clears the description")
	assert.Equal(t, "Fix it well.", got.Instructions)
	assert.False(t, got.Enabled)
}

func TestPlayTools_CreateListDelete(t *testing.T) {
	perm, tools, fix := playToolsFixture(t)
	owner := ctxAs("owner")

	out, err := callTool(t, tools, owner, "play_create",
		`{"workspace_id":"$WS","label":"To tickets via AI","type":"doc","enabled":true,"excluded_project_ids":["proj-2","proj-1"]}`)
	require.NoError(t, err)
	doc := out.(playResult)
	assert.Equal(t, TypeDoc, doc.Type)
	assert.Nil(t, doc.ShowWhenStage)
	assert.Equal(t, []string{"proj-1", "proj-2"}, doc.ExcludedProjectIDs)

	t.Run("without type, every definition is listed and paged", func(t *testing.T) {
		out, err := callTool(t, tools, owner, "play_list", `{"workspace_id":"$WS","limit":1}`)
		require.NoError(t, err)
		page := out.(mcptool.Page[playResult])
		assert.Equal(t, 2, page.Total)
		assert.True(t, page.HasMore)
		require.Len(t, page.Items, 1)
	})
	t.Run("with type, only the plays the caller may run on that target", func(t *testing.T) {
		args := `{"workspace_id":"$WS","type":"ticket","stage":"progress","project_id":"proj-1"}`
		out, err := callTool(t, tools, ctxAs("alice"), "play_list", args)
		require.NoError(t, err)
		page := out.(mcptool.Page[playResult])
		require.Len(t, page.Items, 1)
		assert.Equal(t, fix.ID, page.Items[0].ID)

		out, err = callTool(t, tools, ctxAs("alice"), "play_list", strings.ReplaceAll(args, "proj-1", "proj-9"))
		require.NoError(t, err)
		assert.Empty(t, out.(mcptool.Page[playResult]).Items, "excluded from proj-9")

		perm.deny("alice", fix.ID)
		out, err = callTool(t, tools, ctxAs("alice"), "play_list", args)
		require.NoError(t, err)
		assert.Empty(t, out.(mcptool.Page[playResult]).Items, "denied plays:run on this play")
	})
	t.Run("delete reports what it deleted", func(t *testing.T) {
		out, err := callTool(t, tools, owner, "play_delete", `{"workspace_id":"$WS","id":"`+doc.ID+`"}`)
		require.NoError(t, err)
		assert.Equal(t, mcptool.Gone(doc.ID), out)
		_, err = callTool(t, tools, owner, "play_delete", `{"workspace_id":"$WS","id":"`+doc.ID+`"}`)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}
