package workspace

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func wsTool(t *testing.T, s *Service, name string) mcptool.Tool {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcptool.Tool{}
}

func asOwner(ctx context.Context) context.Context {
	return identity.WithActor(ctx, identity.Actor{ID: "u-1"})
}

func TestMCPTools_Names(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	var names []string
	for _, tool := range MCPTools(s) {
		names = append(names, tool.Name)
	}
	assert.Equal(t, []string{"project_list", "project_create", "project_delete"}, names)
}

func TestProjectCreateTool_ListsEverySuggestedIcon(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	desc := wsTool(t, s, "project_create").InputSchema.Properties["icon"].Description
	for icon := range validProjectIcons {
		assert.Contains(t, desc, string(icon))
	}
}

func TestProjectListTool(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", WorkspaceID: "ws-1"}
	repo.projects["p-2"] = &Project{ID: "p-2", Name: "Other", WorkspaceID: "ws-2"}
	call := wsTool(t, s, "project_list").Call

	_, err := call(t.Context(), json.RawMessage(`{}`))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "workspace_id is required")
	_, err = call(t.Context(), json.RawMessage(`{"workspace_id":"ws-1","user_id":"u-2"}`))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "an unknown argument is refused")

	got, err := call(t.Context(), json.RawMessage(`{"workspace_id":"ws-1"}`))
	require.NoError(t, err)
	page := got.(mcptool.Page[*Project])
	require.Len(t, page.Items, 1)
	assert.Equal(t, "p-1", page.Items[0].ID)
	assert.False(t, page.HasMore)
}

func TestProjectCreateTool(t *testing.T) {
	tests := []struct {
		name    string
		ctx     func(context.Context) context.Context
		owner   bool
		args    string
		wantErr error
	}{
		{"missing prefix is invalid", asOwner, true, `{"workspace_id":"ws-1","name":"Backend"}`, apperrs.ErrInvalid},
		{"a bogus icon is invalid", asOwner, true, `{"workspace_id":"ws-1","name":"Backend","prefix":"BE","icon":"bogus"}`, apperrs.ErrInvalid},
		{"no caller is unauthorized", func(ctx context.Context) context.Context { return ctx }, true, `{"workspace_id":"ws-1","name":"Backend","prefix":"BE"}`, apperrs.ErrUnauthorized},
		{"a caller who is not an owner is forbidden", asOwner, false, `{"workspace_id":"ws-1","name":"Backend","prefix":"BE"}`, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, _ := newOwnerRepo(t, tt.owner)
			_, err := wsTool(t, s, "project_create").Call(tt.ctx(t.Context()), json.RawMessage(tt.args))
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, repo.projects)
		})
	}
	t.Run("creates a project with an icon", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		got, err := wsTool(t, s, "project_create").Call(asOwner(t.Context()),
			json.RawMessage(`{"workspace_id":"ws-1","name":"Backend","prefix":"srv","icon":"Server"}`))
		require.NoError(t, err)
		p := got.(*Project)
		assert.Equal(t, "SRV", p.Prefix)
		assert.Equal(t, ProjectIconServer, p.Icon)
		assert.Equal(t, "ws-1", p.WorkspaceID)
	})
}

func TestProjectDeleteTool(t *testing.T) {
	tests := []struct {
		name    string
		ctx     func(context.Context) context.Context
		owner   bool
		id      string
		wantErr error
	}{
		{"no caller is unauthorized", func(ctx context.Context) context.Context { return ctx }, true, "p-1", apperrs.ErrUnauthorized},
		{"a caller who is not an owner is forbidden", asOwner, false, "p-1", apperrs.ErrForbidden},
		{"a missing project is not found", asOwner, true, "nope", apperrs.ErrNotFound},
		{"a project with tickets is a conflict", asOwner, true, "p-busy", apperrs.ErrConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, _ := newOwnerRepo(t, tt.owner)
			repo.projects["p-1"] = &Project{ID: "p-1"}
			repo.projects["p-busy"] = &Project{ID: "p-busy"}
			repo.ticketPro["t-1"] = "p-busy"
			_, err := wsTool(t, s, "project_delete").Call(tt.ctx(t.Context()), json.RawMessage(`{"id":"`+tt.id+`"}`))
			require.ErrorIs(t, err, tt.wantErr)
			assert.Len(t, repo.projects, 2)
		})
	}
	t.Run("deletes an empty project and says so", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		got, err := wsTool(t, s, "project_delete").Call(asOwner(t.Context()), json.RawMessage(`{"id":"p-1"}`))
		require.NoError(t, err)
		assert.Equal(t, mcptool.Gone("p-1"), got)
		assert.Empty(t, repo.projects)
	})
}
