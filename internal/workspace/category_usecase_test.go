package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func catRepo(s *Service) *fakeCategoryRepo {
	return s.cats.(*fakeCategoryRepo)
}

func typeRepo(s *Service) *fakeTicketTypeRepo {
	return s.types.(*fakeTicketTypeRepo)
}

func statusRepo(s *Service) *fakeStatusRepo {
	return s.statuses.(*fakeStatusRepo)
}

func TestCreateCategory(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateCategory(context.Background(), "u-1", "  ", "Sprint 1", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateCategory(context.Background(), "u-1", "p-1", "  ", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.CreateCategory(context.Background(), "u-1", "p-1", "Sprint 1", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("creates with next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Position: 0}
		c, err := s.CreateCategory(context.Background(), "u-1", "p-1", "Sprint 1", "")
		require.NoError(t, err)
		assert.Equal(t, "Sprint 1", c.Name)
		assert.Equal(t, "p-1", c.ProjectID)
		assert.Equal(t, 1, c.Position)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		catRepo(s).createErr = errors.New("db down")
		_, err := s.CreateCategory(context.Background(), "u-1", "p-1", "Sprint 1", "")
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
	t.Run("invalid color is invalid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		_, err := s.CreateCategory(context.Background(), "u-1", "p-1", "Sprint 1", colors.Color("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty color is valid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		c, err := s.CreateCategory(context.Background(), "u-1", "p-1", "Sprint 1", "")
		require.NoError(t, err)
		assert.Equal(t, colors.Color(""), c.Color)
	})
	t.Run("color from suggested list is valid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		c, err := s.CreateCategory(context.Background(), "u-1", "p-1", "Sprint 1", colors.Cyan)
		require.NoError(t, err)
		assert.Equal(t, colors.Cyan, c.Color)
	})
}

func TestGetCategory(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.GetCategory(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing category is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.GetCategory(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("returns category", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		c, err := s.GetCategory(context.Background(), "c-1")
		require.NoError(t, err)
		assert.Equal(t, "Sprint 1", c.Name)
	})
}

func TestListCategories(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
	catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
	cats, err := s.ListCategories(context.Background())
	require.NoError(t, err)
	require.Len(t, cats, 1)
}

func TestListCategoriesByProject(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.ListCategoriesByProject(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("filters by project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		repo.projects["p-2"] = &Project{ID: "p-2"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		catRepo(s).cats["c-2"] = &Category{ID: "c-2", ProjectID: "p-2"}
		cats, err := s.ListCategoriesByProject(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, cats, 1)
		assert.Equal(t, "c-1", cats[0].ID)
	})
}

func TestRenameCategory(t *testing.T) {
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameCategory(context.Background(), "u-1", "c-1", "  ", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing category is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameCategory(context.Background(), "u-1", "nope", "Sprint 2", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("renames and keeps identity", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		c, err := s.RenameCategory(context.Background(), "u-1", "c-1", "Sprint 2", "")
		require.NoError(t, err)
		assert.Equal(t, "Sprint 2", c.Name)
		assert.Equal(t, "c-1", c.ID)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.RenameCategory(context.Background(), "u-1", "c-1", "Sprint 2", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("invalid color is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameCategory(context.Background(), "u-1", "c-1", "Sprint 2", colors.Color("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("sets and clears color", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", Name: "Sprint 1", Color: colors.Cyan}
		c, err := s.RenameCategory(context.Background(), "u-1", "c-1", "Sprint 2", colors.Emerald)
		require.NoError(t, err)
		assert.Equal(t, colors.Emerald, c.Color)

		c, err = s.RenameCategory(context.Background(), "u-1", "c-1", "Sprint 3", "")
		require.NoError(t, err)
		assert.Equal(t, colors.Color(""), c.Color)
	})
}

func TestReorderCategories(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.ReorderCategories(context.Background(), "u-1", " ", []string{"c-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("duplicate ids are invalid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		err := s.ReorderCategories(context.Background(), "u-1", "p-1", []string{"c-1", "c-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("every category must be listed", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		catRepo(s).cats["c-2"] = &Category{ID: "c-2", ProjectID: "p-1"}
		err := s.ReorderCategories(context.Background(), "u-1", "p-1", []string{"c-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("reorders", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		catRepo(s).cats["c-2"] = &Category{ID: "c-2", ProjectID: "p-1"}
		require.NoError(t, s.ReorderCategories(context.Background(), "u-1", "p-1", []string{"c-2", "c-1"}))
		assert.Equal(t, 0, catRepo(s).cats["c-2"].Position)
		assert.Equal(t, 1, catRepo(s).cats["c-1"].Position)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.ReorderCategories(context.Background(), "u-1", "p-1", []string{"c-1"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestDeleteCategory(t *testing.T) {
	t.Run("missing category is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.DeleteCategory(context.Background(), "u-1", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("deletes category", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		require.NoError(t, s.DeleteCategory(context.Background(), "u-1", "c-1"))
		_, ok := catRepo(s).cats["c-1"]
		assert.False(t, ok)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.DeleteCategory(context.Background(), "u-1", "c-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestMoveTicketToCategory(t *testing.T) {
	t.Run("empty ticket id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.MoveTicketToCategory(context.Background(), " ", "c-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing category is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.MoveTicketToCategory(context.Background(), "t-1", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("moves ticket into category", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		require.NoError(t, s.MoveTicketToCategory(context.Background(), "t-1", "c-1"))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		catRepo(s).moveErr = apperrs.ErrNotFound
		err := s.MoveTicketToCategory(context.Background(), "nope", "c-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("empty category uncategorizes", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		require.NoError(t, s.MoveTicketToCategory(context.Background(), "t-1", "  "))
	})
}

func TestCreateTicketType(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateTicketType(context.Background(), "u-1", "  ", "bug", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateTicketType(context.Background(), "u-1", "p-1", "  ", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.CreateTicketType(context.Background(), "u-1", "p-1", "bug", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("creates with next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", ProjectID: "p-1", Position: 0}
		tt, err := s.CreateTicketType(context.Background(), "u-1", "p-1", "bug", "")
		require.NoError(t, err)
		assert.Equal(t, "bug", tt.Name)
		assert.Equal(t, "p-1", tt.ProjectID)
		assert.Equal(t, 1, tt.Position)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).createErr = errors.New("db down")
		_, err := s.CreateTicketType(context.Background(), "u-1", "p-1", "bug", "")
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
	t.Run("invalid color is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateTicketType(context.Background(), "u-1", "p-1", "bug", colors.Color("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty color is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		tt, err := s.CreateTicketType(context.Background(), "u-1", "p-1", "bug", "")
		require.NoError(t, err)
		assert.Equal(t, colors.Color(""), tt.Color)
	})
	t.Run("color from suggested list is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		tt, err := s.CreateTicketType(context.Background(), "u-1", "p-1", "bug", colors.Cyan)
		require.NoError(t, err)
		assert.Equal(t, colors.Cyan, tt.Color)
	})
}

func TestListTicketTypesByProject(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.ListTicketTypesByProject(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("filters by project", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", ProjectID: "p-1", Name: "bug"}
		typeRepo(s).types["t-2"] = &TicketType{ID: "t-2", ProjectID: "p-2", Name: "task"}
		types, err := s.ListTicketTypesByProject(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, types, 1)
		assert.Equal(t, "t-1", types[0].ID)
	})
}

func TestRenameTicketType(t *testing.T) {
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameTicketType(context.Background(), "u-1", "t-1", "  ", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing type is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameTicketType(context.Background(), "u-1", "nope", "bug", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("renames and keeps identity", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug"}
		tt, err := s.RenameTicketType(context.Background(), "u-1", "t-1", "defect", "")
		require.NoError(t, err)
		assert.Equal(t, "defect", tt.Name)
		assert.Equal(t, "t-1", tt.ID)
	})
	t.Run("invalid color is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug"}
		_, err := s.RenameTicketType(context.Background(), "u-1", "t-1", "bug", colors.Color("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("sets and clears color", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug", Color: colors.Cyan}
		tt, err := s.RenameTicketType(context.Background(), "u-1", "t-1", "bug", colors.Emerald)
		require.NoError(t, err)
		assert.Equal(t, colors.Emerald, tt.Color)

		tt, err = s.RenameTicketType(context.Background(), "u-1", "t-1", "bug", "")
		require.NoError(t, err)
		assert.Equal(t, colors.Color(""), tt.Color)
	})
}

func TestSetTicketTypeTemplate(t *testing.T) {
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.SetTicketTypeTemplate(context.Background(), "u-1", "t-1", "## Why")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("missing type is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.SetTicketTypeTemplate(context.Background(), "u-1", "nope", "## Why")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("replaces the template and keeps name and color", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug", Color: colors.Cyan, BodyTemplate: "## Old"}
		tt, err := s.SetTicketTypeTemplate(context.Background(), "u-1", "t-1", "## Steps to reproduce\n\n")
		require.NoError(t, err)
		assert.Equal(t, "## Steps to reproduce\n\n", tt.BodyTemplate)
		assert.Equal(t, "bug", tt.Name)
		assert.Equal(t, colors.Cyan, tt.Color)
	})
	t.Run("rename keeps the template", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug", BodyTemplate: "## Steps"}
		tt, err := s.RenameTicketType(context.Background(), "u-1", "t-1", "defect", "")
		require.NoError(t, err)
		assert.Equal(t, "## Steps", tt.BodyTemplate)
	})
}

func TestDeleteTicketType(t *testing.T) {
	t.Run("in-use type is conflict", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.ticketPro["t-1"] = "p-1"
		typeRepo(s).count = 1
		err := s.DeleteTicketType(context.Background(), "u-1", "t-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
	t.Run("deletes unused type", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1"}
		require.NoError(t, s.DeleteTicketType(context.Background(), "u-1", "t-1"))
		_, ok := typeRepo(s).types["t-1"]
		assert.False(t, ok)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.DeleteTicketType(context.Background(), "u-1", "t-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestReorderTicketTypes(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.ReorderTicketTypes(context.Background(), "u-1", " ", []string{"a"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("duplicate ids are invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.ReorderTicketTypes(context.Background(), "u-1", "p-1", []string{"a", "a"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("every type must be listed", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["a"] = &TicketType{ID: "a", ProjectID: "p-1"}
		typeRepo(s).types["b"] = &TicketType{ID: "b", ProjectID: "p-1"}
		err := s.ReorderTicketTypes(context.Background(), "u-1", "p-1", []string{"a"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("reorders", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["a"] = &TicketType{ID: "a", ProjectID: "p-1"}
		typeRepo(s).types["b"] = &TicketType{ID: "b", ProjectID: "p-1"}
		require.NoError(t, s.ReorderTicketTypes(context.Background(), "u-1", "p-1", []string{"b", "a"}))
		assert.Equal(t, 0, typeRepo(s).types["b"].Position)
		assert.Equal(t, 1, typeRepo(s).types["a"].Position)
	})
}

func TestCreateStatus(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateStatus(context.Background(), "u-1", "  ", "Blocked", StatusKindProgress, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateStatus(context.Background(), "u-1", "p-1", "  ", StatusKindProgress, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("invalid kind is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateStatus(context.Background(), "u-1", "p-1", "Blocked", StatusKind("bogus"), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("kind defaults to backlog", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		st, err := s.CreateStatus(context.Background(), "u-1", "p-1", "Blocked", "", "")
		require.NoError(t, err)
		assert.Equal(t, StatusKindBacklog, st.Kind)
	})
	t.Run("creates with next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1", Position: 0}
		st, err := s.CreateStatus(context.Background(), "u-1", "p-1", "Blocked", StatusKindProgress, "")
		require.NoError(t, err)
		assert.Equal(t, 1, st.Position)
		assert.Equal(t, "p-1", st.ProjectID)
		assert.Equal(t, StatusKindProgress, st.Kind)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		_, err := s.CreateStatus(context.Background(), "u-1", "p-1", "Blocked", StatusKindProgress, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("invalid icon is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.CreateStatus(context.Background(), "u-1", "p-1", "Blocked", StatusKindProgress, StatusIcon("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty icon is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		st, err := s.CreateStatus(context.Background(), "u-1", "p-1", "Blocked", StatusKindProgress, "")
		require.NoError(t, err)
		assert.Equal(t, StatusIcon(""), st.Icon)
	})
	t.Run("icon from suggested list is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		st, err := s.CreateStatus(context.Background(), "u-1", "p-1", "Blocked", StatusKindProgress, StatusIconInReview)
		require.NoError(t, err)
		assert.Equal(t, StatusIconInReview, st.Icon)
	})
}

func TestListStatusesByProject(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.ListStatusesByProject(context.Background(), "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("filters by project", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1", Name: "Blocked"}
		statusRepo(s).statuses["s-2"] = &Status{ID: "s-2", ProjectID: "p-2", Name: "Open"}
		statuses, err := s.ListStatusesByProject(context.Background(), "p-1")
		require.NoError(t, err)
		require.Len(t, statuses, 1)
		assert.Equal(t, "s-1", statuses[0].ID)
	})
}

func TestRenameStatus(t *testing.T) {
	t.Run("empty name is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameStatus(context.Background(), "u-1", "s-1", "  ", StatusKindProgress, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("invalid kind is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameStatus(context.Background(), "u-1", "s-1", "Blocked", StatusKind("bogus"), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("invalid icon is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
		_, err := s.RenameStatus(context.Background(), "u-1", "s-1", "Blocked", StatusKindProgress, StatusIcon("bogus"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("renames and keeps identity", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
		st, err := s.RenameStatus(context.Background(), "u-1", "s-1", "In review", StatusKindDone, StatusIconInReview)
		require.NoError(t, err)
		assert.Equal(t, "In review", st.Name)
		assert.Equal(t, StatusKindDone, st.Kind)
		assert.Equal(t, StatusIconInReview, st.Icon)
		assert.Equal(t, "s-1", st.ID)
	})
	t.Run("missing status is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.RenameStatus(context.Background(), "u-1", "nope", "Blocked", StatusKindProgress, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestReorderStatuses(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.ReorderStatuses(context.Background(), "u-1", " ", []string{"a"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("duplicate ids are invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.ReorderStatuses(context.Background(), "u-1", "p-1", []string{"a", "a"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("every status must be listed", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["a"] = &Status{ID: "a", ProjectID: "p-1"}
		statusRepo(s).statuses["b"] = &Status{ID: "b", ProjectID: "p-1"}
		err := s.ReorderStatuses(context.Background(), "u-1", "p-1", []string{"a"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("reorders", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["a"] = &Status{ID: "a", ProjectID: "p-1"}
		statusRepo(s).statuses["b"] = &Status{ID: "b", ProjectID: "p-1"}
		require.NoError(t, s.ReorderStatuses(context.Background(), "u-1", "p-1", []string{"b", "a"}))
		assert.Equal(t, 0, statusRepo(s).statuses["b"].Position)
		assert.Equal(t, 1, statusRepo(s).statuses["a"].Position)
	})
}

func TestDeleteStatus(t *testing.T) {
	t.Run("in-use status is conflict", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.ticketPro["t-1"] = "p-1"
		statusRepo(s).count = 1
		err := s.DeleteStatus(context.Background(), "u-1", "s-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
	t.Run("deletes unused status", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1"}
		require.NoError(t, s.DeleteStatus(context.Background(), "u-1", "s-1"))
		_, ok := statusRepo(s).statuses["s-1"]
		assert.False(t, ok)
	})
	t.Run("missing status is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		err := s.DeleteStatus(context.Background(), "u-1", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, false)
		err := s.DeleteStatus(context.Background(), "u-1", "s-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestGetTicketType(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.GetTicketType(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing type is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.GetTicketType(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("returns a type", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}
		tt, err := s.GetTicketType(context.Background(), "tt-1")
		require.NoError(t, err)
		assert.Equal(t, "bug", tt.Name)
	})
}

func TestGetStatus(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.GetStatus(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing status is not found", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		_, err := s.GetStatus(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("returns a status", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
		st, err := s.GetStatus(context.Background(), "s-1")
		require.NoError(t, err)
		assert.Equal(t, "Blocked", st.Name)
	})
}

func TestDeleteTicketType_NotFound(t *testing.T) {
	s, _, _ := newOwnerRepo(t, true)
	err := s.DeleteTicketType(context.Background(), "u-1", "nope")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}
