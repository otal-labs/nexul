package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func wsToolByName(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
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
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
	tools := MCPTools(s)
	require.Len(t, tools, 30)
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
}

func TestMCPTools_ProjectCreate(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	call := wsToolByName(t, MCPTools(s), "project_create").Call

	t.Run("creates a project", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"workspace_id": "ws-1", "name": "Backend", "prefix": "BE"})
		require.NoError(t, err)
		p, ok := got.(*Project)
		require.True(t, ok)
		assert.Equal(t, "Backend", p.Name)
		assert.Equal(t, "BE", p.Prefix)
		assert.Equal(t, "ws-1", p.WorkspaceID)
	})
	t.Run("missing workspace_id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"name": "Backend", "prefix": "BE"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing name is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"workspace_id": "ws-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("creates a project with an icon", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"workspace_id": "ws-1", "name": "Backend", "prefix": "SRV", "icon": "Server"})
		require.NoError(t, err)
		assert.Equal(t, ProjectIconServer, got.(*Project).Icon)
	})
	t.Run("bogus icon is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"workspace_id": "ws-1", "name": "Backend", "prefix": "BOG", "icon": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ProjectList(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", WorkspaceID: "ws-1"}
	got, err := wsToolByName(t, MCPTools(s), "project_list").Call(context.Background(), map[string]any{"workspace_id": "ws-1"})
	require.NoError(t, err)
	projects, ok := got.([]*Project)
	require.True(t, ok)
	require.Len(t, projects, 1)
}

func TestMCPTools_ProjectGet(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
	call := wsToolByName(t, MCPTools(s), "project_get").Call

	t.Run("returns a stored project", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": "p-1"})
		require.NoError(t, err)
		assert.Equal(t, "p-1", got.(*Project).ID)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("unknown project is not found", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestMCPTools_ProjectRename(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old"}
	call := wsToolByName(t, MCPTools(s), "project_rename").Call

	t.Run("renames a project", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": "p-1", "name": "New"})
		require.NoError(t, err)
		assert.Equal(t, "New", got.(*Project).Name)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"name": "New"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing name is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": "p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("renames with an icon", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": "p-1", "name": "New", "icon": "Rocket"})
		require.NoError(t, err)
		assert.Equal(t, ProjectIconRocket, got.(*Project).Icon)
	})
	t.Run("bogus icon is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": "p-1", "name": "New", "icon": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ProjectDelete(t *testing.T) {
	t.Run("deletes an empty project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		call := wsToolByName(t, MCPTools(s), "project_delete").Call
		got, err := call(context.Background(), map[string]any{"id": "p-1"})
		require.NoError(t, err)
		assert.Equal(t, "deleted", got.(map[string]string)["status"])
	})
	t.Run("project with tickets conflicts", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "p-1"
		call := wsToolByName(t, MCPTools(s), "project_delete").Call
		_, err := call(context.Background(), map[string]any{"id": "p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(newTestService(newFakeRepo(), nil)), "project_delete").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ProjectReorder(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
	repo.projects["p-2"] = &Project{ID: "p-2", Name: "B", WorkspaceID: "ws-1"}
	call := wsToolByName(t, MCPTools(s), "project_reorder").Call

	t.Run("reorders projects", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"workspace_id": "ws-1", "ids": []any{"p-2", "p-1"}})
		require.NoError(t, err)
		assert.Equal(t, "reordered", got.(map[string]string)["status"])
	})
	t.Run("missing workspace_id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"ids": []any{"p-1"}})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ids is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"workspace_id": "ws-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-string ids are invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"workspace_id": "ws-1", "ids": []any{1}})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ProjectAddRepo(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
	call := wsToolByName(t, MCPTools(s), "project_add_repo").Call

	t.Run("associates a repo", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "owner": "acme", "name": "app"})
		require.NoError(t, err)
		assert.Equal(t, "acme", got.(map[string]string)["owner"])
	})
	t.Run("defaults connector_id to github when omitted", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1", "owner": "acme2", "name": "app2"})
		require.NoError(t, err)
		repos, err := s.ListRepos(context.Background(), "p-1")
		require.NoError(t, err)
		found := false
		for _, r := range repos {
			if r.Owner == "acme2" && r.Name == "app2" {
				found = true
				assert.Equal(t, "github", r.ConnectorID)
			}
		}
		assert.True(t, found)
	})
	t.Run("missing project_id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"owner": "acme", "name": "app"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ProjectRemoveRepo(t *testing.T) {
	t.Run("dissociates a repo", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
		call := wsToolByName(t, MCPTools(s), "project_remove_repo").Call
		got, err := call(context.Background(), map[string]any{"owner": "acme", "name": "app"})
		require.NoError(t, err)
		assert.Equal(t, "removed", got.(map[string]string)["status"])
	})
	t.Run("missing repo is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		call := wsToolByName(t, MCPTools(s), "project_remove_repo").Call
		_, err := call(context.Background(), map[string]any{"owner": "acme", "name": "app"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("missing name is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(newTestService(newFakeRepo(), nil)), "project_remove_repo").Call
		_, err := call(context.Background(), map[string]any{"owner": "acme"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_ProjectListRepos(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
	call := wsToolByName(t, MCPTools(s), "project_list_repos").Call

	got, err := call(context.Background(), map[string]any{"project_id": "p-1"})
	require.NoError(t, err)
	repos, ok := got.([]RepoRef)
	require.True(t, ok)
	require.Len(t, repos, 1)

	_, err = call(context.Background(), map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestMCPTools_ProjectMoveTicket(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
	repo.ticketPro["t-1"] = "old"
	call := wsToolByName(t, MCPTools(s), "project_move_ticket").Call

	got, err := call(context.Background(), map[string]any{"ticket_id": "t-1", "project_id": "p-1"})
	require.NoError(t, err)
	assert.Equal(t, "moved", got.(map[string]string)["status"])

	_, err = call(context.Background(), map[string]any{"ticket_id": "t-1"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestMCPTools_ProjectDeleteImpact(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
	repo.repos["p-1"] = []RepoRef{{Owner: "acme", Name: "app"}}
	call := wsToolByName(t, MCPTools(s), "project_delete_impact").Call
	got, err := call(context.Background(), map[string]any{"id": "p-1"})
	require.NoError(t, err)
	assert.Equal(t, DeleteImpact{Repos: 1}, got.(DeleteImpact))

	_, err = call(context.Background(), map[string]any{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrInvalid))
}

func TestMCPTools_CategoryCreate(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
	call := wsToolByName(t, MCPTools(s), "category_create").Call

	t.Run("creates a category", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Sprint 1"})
		require.NoError(t, err)
		assert.Equal(t, "Sprint 1", got.(*Category).Name)
	})
	t.Run("creates a category with a color", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Sprint 1", "color": "cyan"})
		require.NoError(t, err)
		assert.Equal(t, colors.Cyan, got.(*Category).Color)
	})
	t.Run("bogus color is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Sprint 1", "color": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing project id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"name": "Sprint 1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing name is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_CategoryList(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1"}
	s.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}

	t.Run("lists all categories", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_list").Call
		got, err := call(context.Background(), map[string]any{})
		require.NoError(t, err)
		require.Len(t, got.([]*Category), 1)
	})
	t.Run("lists by project", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_list").Call
		got, err := call(context.Background(), map[string]any{"project_id": "p-1"})
		require.NoError(t, err)
		require.Len(t, got.([]*Category), 1)
	})
}

func TestMCPTools_CategoryMoveTicket(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1"}
	s.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}

	t.Run("moves a ticket", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_move_ticket").Call
		got, err := call(context.Background(), map[string]any{"ticket_id": "t-1", "category_id": "c-1"})
		require.NoError(t, err)
		assert.Equal(t, "moved", got.(map[string]string)["status"])
	})
	t.Run("missing category is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_move_ticket").Call
		_, err := call(context.Background(), map[string]any{"ticket_id": "t-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_CategoryClearTicket(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1"}
	call := wsToolByName(t, MCPTools(s), "category_clear_ticket").Call
	got, err := call(context.Background(), map[string]any{"ticket_id": "t-1"})
	require.NoError(t, err)
	assert.Equal(t, "uncategorized", got.(map[string]string)["status"])
}

func TestMCPTools_TicketTypeCreate(t *testing.T) {
	call := wsToolByName(t, MCPTools(newTestService(newFakeRepo(), nil)), "ticket_type_create").Call

	t.Run("creates a ticket type", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "bug"})
		require.NoError(t, err)
		assert.Equal(t, "bug", got.(*TicketType).Name)
	})
	t.Run("missing name is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("creates a ticket type with a color", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "bug", "color": "cyan"})
		require.NoError(t, err)
		assert.Equal(t, colors.Cyan, got.(*TicketType).Color)
	})
	t.Run("bogus color is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "bug", "color": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_StatusCreate(t *testing.T) {
	call := wsToolByName(t, MCPTools(newTestService(newFakeRepo(), nil)), "status_create").Call

	t.Run("creates a status", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Blocked", "kind": "progress"})
		require.NoError(t, err)
		assert.Equal(t, "Blocked", got.(*Status).Name)
		assert.Equal(t, StatusKindProgress, got.(*Status).Kind)
	})
	t.Run("missing kind is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Blocked"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("bogus kind is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Blocked", "kind": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("creates a status with an icon", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Blocked", "kind": "progress", "icon": "Circle"})
		require.NoError(t, err)
		assert.Equal(t, StatusIconTodo, got.(*Status).Icon)
	})
	t.Run("bogus icon is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"project_id": "p-1", "name": "Blocked", "kind": "progress", "icon": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_StatusRename(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	s.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
	call := wsToolByName(t, MCPTools(s), "status_rename").Call

	t.Run("renames a status", func(t *testing.T) {
		got, err := call(context.Background(), map[string]any{"id": "s-1", "name": "In review", "kind": "done"})
		require.NoError(t, err)
		assert.Equal(t, "In review", got.(*Status).Name)
		assert.Equal(t, StatusKindDone, got.(*Status).Kind)
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"name": "x", "kind": "progress"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("bogus icon is invalid", func(t *testing.T) {
		_, err := call(context.Background(), map[string]any{"id": "s-1", "name": "Blocked", "kind": "progress", "icon": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_StatusDelete(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	s.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1"}

	t.Run("deletes unused status", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "status_delete").Call
		got, err := call(context.Background(), map[string]any{"id": "s-1"})
		require.NoError(t, err)
		assert.Equal(t, "deleted", got.(map[string]string)["status"])
	})
	t.Run("missing id is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "status_delete").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestMCPTools_CategoryDetailErrors(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1"}
	s.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}

	t.Run("category_get happy + missing id", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_get").Call
		got, err := call(context.Background(), map[string]any{"id": "c-1"})
		require.NoError(t, err)
		assert.Equal(t, "Sprint 1", got.(*Category).Name)
		_, err = call(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("category_get missing is not found", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_get").Call
		_, err := call(context.Background(), map[string]any{"id": "nope"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("category_rename happy + missing args", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_rename").Call
		got, err := call(context.Background(), map[string]any{"id": "c-1", "name": "Sprint 2"})
		require.NoError(t, err)
		assert.Equal(t, "Sprint 2", got.(*Category).Name)
		_, err = call(context.Background(), map[string]any{"id": "c-1"})
		require.Error(t, err)
	})
	t.Run("category_rename with a color", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_rename").Call
		got, err := call(context.Background(), map[string]any{"id": "c-1", "name": "Sprint 2", "color": "orange"})
		require.NoError(t, err)
		assert.Equal(t, colors.Orange, got.(*Category).Color)
	})
	t.Run("category_rename bogus color is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_rename").Call
		_, err := call(context.Background(), map[string]any{"id": "c-1", "name": "Sprint 2", "color": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("category_delete happy + missing id", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_delete").Call
		got, err := call(context.Background(), map[string]any{"id": "c-1"})
		require.NoError(t, err)
		assert.Equal(t, "deleted", got.(map[string]string)["status"])
		_, err = call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
	t.Run("category_reorder happy + missing args", func(t *testing.T) {
		s.cats.(*fakeCategoryRepo).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		call := wsToolByName(t, MCPTools(s), "category_reorder").Call
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "ids": []any{"c-1"}})
		require.NoError(t, err)
		assert.Equal(t, "reordered", got.(map[string]string)["status"])
		_, err = call(context.Background(), map[string]any{"project_id": "p-1"})
		require.Error(t, err)
	})
	t.Run("category_move_ticket missing category is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_move_ticket").Call
		_, err := call(context.Background(), map[string]any{"ticket_id": "t-1", "category_id": ""})
		require.Error(t, err)
	})
	t.Run("category_clear_ticket missing id is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "category_clear_ticket").Call
		_, err := call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

func TestMCPTools_TicketTypeDetailErrors(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	s.types.(*fakeTicketTypeRepo).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}

	t.Run("ticket_type_rename happy + missing args", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "ticket_type_rename").Call
		got, err := call(context.Background(), map[string]any{"id": "tt-1", "name": "defect"})
		require.NoError(t, err)
		assert.Equal(t, "defect", got.(*TicketType).Name)
		_, err = call(context.Background(), map[string]any{"id": "tt-1"})
		require.Error(t, err)
	})
	t.Run("ticket_type_rename with a color", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "ticket_type_rename").Call
		got, err := call(context.Background(), map[string]any{"id": "tt-1", "name": "defect", "color": "orange"})
		require.NoError(t, err)
		assert.Equal(t, colors.Orange, got.(*TicketType).Color)
	})
	t.Run("ticket_type_rename bogus color is invalid", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "ticket_type_rename").Call
		_, err := call(context.Background(), map[string]any{"id": "tt-1", "name": "defect", "color": "bogus"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("ticket_type_set_template happy + missing id", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "ticket_type_set_template").Call
		got, err := call(context.Background(), map[string]any{"id": "tt-1", "body_template": "## Why\n\n"})
		require.NoError(t, err)
		assert.Equal(t, "## Why\n\n", got.(*TicketType).BodyTemplate)
		_, err = call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
	t.Run("ticket_type_delete happy + missing id", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "ticket_type_delete").Call
		got, err := call(context.Background(), map[string]any{"id": "tt-1"})
		require.NoError(t, err)
		assert.Equal(t, "deleted", got.(map[string]string)["status"])
		_, err = call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
}

func TestMCPTools_StatusDetailErrors(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	s.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1", Name: "Blocked", Kind: StatusKindProgress}

	t.Run("status_reorder happy + missing ids", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "status_reorder").Call
		got, err := call(context.Background(), map[string]any{"project_id": "p-1", "ids": []any{"s-1"}})
		require.NoError(t, err)
		assert.Equal(t, "reordered", got.(map[string]string)["status"])
		_, err = call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
	t.Run("status_delete happy + missing id", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "status_delete").Call
		got, err := call(context.Background(), map[string]any{"id": "s-1"})
		require.NoError(t, err)
		assert.Equal(t, "deleted", got.(map[string]string)["status"])
		_, err = call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
	t.Run("status_rename happy + missing args", func(t *testing.T) {
		s.statuses.(*fakeStatusRepo).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1", Name: "Blocked", Kind: StatusKindProgress}
		call := wsToolByName(t, MCPTools(s), "status_rename").Call
		got, err := call(context.Background(), map[string]any{"id": "s-1", "name": "In review", "kind": "done"})
		require.NoError(t, err)
		assert.Equal(t, "In review", got.(*Status).Name)
		_, err = call(context.Background(), map[string]any{"id": "s-1", "name": "x"})
		require.Error(t, err)
	})
	t.Run("status_list lists", func(t *testing.T) {
		call := wsToolByName(t, MCPTools(s), "status_list").Call
		got, err := call(context.Background(), map[string]any{"project_id": "p-1"})
		require.NoError(t, err)
		require.Len(t, got.([]*Status), 1)
	})
}
