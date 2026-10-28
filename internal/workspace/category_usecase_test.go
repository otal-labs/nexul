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
	t.Run("creates with next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Position: 0}
		c, err := s.CreateCategory(context.Background(), "p-1", "Sprint 1", "")
		require.NoError(t, err)
		assert.Equal(t, "Sprint 1", c.Name)
		assert.Equal(t, "p-1", c.ProjectID)
		assert.Equal(t, 1, c.Position)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		catRepo(s).createErr = errors.New("db down")
		_, err := s.CreateCategory(context.Background(), "p-1", "Sprint 1", "")
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
	t.Run("color from suggested list is valid", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		c, err := s.CreateCategory(context.Background(), "p-1", "Sprint 1", colors.Cyan)
		require.NoError(t, err)
		assert.Equal(t, colors.Cyan, c.Color)
	})
}

func TestGetCategory(t *testing.T) {
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
	t.Run("renames and keeps identity", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1", Name: "Sprint 1"}
		c, err := s.RenameCategory(context.Background(), "c-1", "Sprint 2", "")
		require.NoError(t, err)
		assert.Equal(t, "Sprint 2", c.Name)
		assert.Equal(t, "c-1", c.ID)
	})
	t.Run("sets and clears color", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", Name: "Sprint 1", Color: colors.Cyan}
		c, err := s.RenameCategory(context.Background(), "c-1", "Sprint 2", colors.Emerald)
		require.NoError(t, err)
		assert.Equal(t, colors.Emerald, c.Color)

		c, err = s.RenameCategory(context.Background(), "c-1", "Sprint 3", "")
		require.NoError(t, err)
		assert.Equal(t, colors.Color(""), c.Color)
	})
}

func TestReorderCategories(t *testing.T) {
	t.Run("reorders", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		catRepo(s).cats["c-2"] = &Category{ID: "c-2", ProjectID: "p-1"}
		require.NoError(t, s.ReorderCategories(context.Background(), "p-1", []string{"c-2", "c-1"}))
		assert.Equal(t, 0, catRepo(s).cats["c-2"].Position)
		assert.Equal(t, 1, catRepo(s).cats["c-1"].Position)
	})
}

func TestDeleteCategory(t *testing.T) {
	t.Run("deletes category", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		require.NoError(t, s.DeleteCategory(context.Background(), "c-1"))
		_, ok := catRepo(s).cats["c-1"]
		assert.False(t, ok)
	})
}

func TestMoveTicketToCategory(t *testing.T) {
	t.Run("moves ticket into category, naming the category and the ticket's project", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		s.SetTicketProjects(fakeTicketProjects{"t-1": "p-1"})
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		require.NoError(t, s.MoveTicketToCategory(context.Background(), "t-1", "c-1"))
		require.Len(t, catRepo(s).moved, 1)
		assert.Equal(t, TicketCategoryChangedEvent{TicketID: "t-1", CategoryID: "c-1", ProjectID: "p-1"}, catRepo(s).moved[0].Payload)
	})
	t.Run("empty category uncategorizes", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		s.SetTicketProjects(fakeTicketProjects{"t-1": "p-1"})
		repo.projects["p-1"] = &Project{ID: "p-1"}
		require.NoError(t, s.MoveTicketToCategory(context.Background(), "t-1", "  "))
		require.Len(t, catRepo(s).moved, 1)
		assert.Equal(t, TicketCategoryChangedEvent{TicketID: "t-1", CategoryID: "", ProjectID: "p-1"}, catRepo(s).moved[0].Payload)
	})
}

func TestCreateTicketType(t *testing.T) {
	t.Run("creates with next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", ProjectID: "p-1", Position: 0}
		tt, err := s.CreateTicketType(context.Background(), "p-1", "bug", "")
		require.NoError(t, err)
		assert.Equal(t, "bug", tt.Name)
		assert.Equal(t, "p-1", tt.ProjectID)
		assert.Equal(t, 1, tt.Position)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).createErr = errors.New("db down")
		_, err := s.CreateTicketType(context.Background(), "p-1", "bug", "")
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
	t.Run("color from suggested list is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		tt, err := s.CreateTicketType(context.Background(), "p-1", "bug", colors.Cyan)
		require.NoError(t, err)
		assert.Equal(t, colors.Cyan, tt.Color)
	})
}

func TestListTicketTypesByProject(t *testing.T) {
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
	t.Run("renames and keeps identity", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug"}
		tt, err := s.RenameTicketType(context.Background(), "t-1", "defect", "")
		require.NoError(t, err)
		assert.Equal(t, "defect", tt.Name)
		assert.Equal(t, "t-1", tt.ID)
	})
	t.Run("sets and clears color", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug", Color: colors.Cyan}
		tt, err := s.RenameTicketType(context.Background(), "t-1", "bug", colors.Emerald)
		require.NoError(t, err)
		assert.Equal(t, colors.Emerald, tt.Color)

		tt, err = s.RenameTicketType(context.Background(), "t-1", "bug", "")
		require.NoError(t, err)
		assert.Equal(t, colors.Color(""), tt.Color)
	})
}

func TestSetTicketTypeTemplate(t *testing.T) {
	t.Run("replaces the template and keeps name and color", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug", Color: colors.Cyan, BodyTemplate: "## Old"}
		tt, err := s.SetTicketTypeTemplate(context.Background(), "t-1", "## Steps to reproduce\n\n")
		require.NoError(t, err)
		assert.Equal(t, "## Steps to reproduce\n\n", tt.BodyTemplate)
		assert.Equal(t, "bug", tt.Name)
		assert.Equal(t, colors.Cyan, tt.Color)
	})
	t.Run("rename keeps the template", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", Name: "bug", BodyTemplate: "## Steps"}
		tt, err := s.RenameTicketType(context.Background(), "t-1", "defect", "")
		require.NoError(t, err)
		assert.Equal(t, "## Steps", tt.BodyTemplate)
	})
}

func TestDeleteTicketType(t *testing.T) {
	t.Run("deletes unused type", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1"}
		require.NoError(t, s.DeleteTicketType(context.Background(), "t-1"))
		_, ok := typeRepo(s).types["t-1"]
		assert.False(t, ok)
	})
}

func TestReorderTicketTypes(t *testing.T) {
	t.Run("reorders", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["a"] = &TicketType{ID: "a", ProjectID: "p-1"}
		typeRepo(s).types["b"] = &TicketType{ID: "b", ProjectID: "p-1"}
		require.NoError(t, s.ReorderTicketTypes(context.Background(), "p-1", []string{"b", "a"}))
		assert.Equal(t, 0, typeRepo(s).types["b"].Position)
		assert.Equal(t, 1, typeRepo(s).types["a"].Position)
	})
}

func TestCreateStatus(t *testing.T) {
	t.Run("kind defaults to backlog", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		st, err := s.CreateStatus(context.Background(), "p-1", "Blocked", "", "")
		require.NoError(t, err)
		assert.Equal(t, StatusKindBacklog, st.Kind)
	})
	t.Run("creates with next position", func(t *testing.T) {
		s, repo, _ := newOwnerRepo(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1", Position: 0}
		st, err := s.CreateStatus(context.Background(), "p-1", "Blocked", StatusKindProgress, "")
		require.NoError(t, err)
		assert.Equal(t, 1, st.Position)
		assert.Equal(t, "p-1", st.ProjectID)
		assert.Equal(t, StatusKindProgress, st.Kind)
	})
	t.Run("icon from suggested list is valid", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		st, err := s.CreateStatus(context.Background(), "p-1", "Blocked", StatusKindProgress, StatusIconInReview)
		require.NoError(t, err)
		assert.Equal(t, StatusIconInReview, st.Icon)
	})
}

func TestListStatusesByProject(t *testing.T) {
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
	t.Run("renames and keeps identity", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
		st, err := s.RenameStatus(context.Background(), "s-1", "In review", StatusKindDone, StatusIconInReview)
		require.NoError(t, err)
		assert.Equal(t, "In review", st.Name)
		assert.Equal(t, StatusKindDone, st.Kind)
		assert.Equal(t, StatusIconInReview, st.Icon)
		assert.Equal(t, "s-1", st.ID)
		require.Len(t, statusRepo(s).updated, 1)
		assert.Equal(t, StatusKindProgress, statusRepo(s).updated[0].Payload.(StatusEvent).PreviousKind, "the frame says the stage moved from progress")
	})
}

func TestReorderStatuses(t *testing.T) {
	t.Run("reorders", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["a"] = &Status{ID: "a", ProjectID: "p-1"}
		statusRepo(s).statuses["b"] = &Status{ID: "b", ProjectID: "p-1"}
		require.NoError(t, s.ReorderStatuses(context.Background(), "p-1", []string{"b", "a"}))
		assert.Equal(t, 0, statusRepo(s).statuses["b"].Position)
		assert.Equal(t, 1, statusRepo(s).statuses["a"].Position)
	})
}

func TestDeleteStatus(t *testing.T) {
	t.Run("deletes unused status", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1"}
		require.NoError(t, s.DeleteStatus(context.Background(), "s-1"))
		_, ok := statusRepo(s).statuses["s-1"]
		assert.False(t, ok)
	})
}

func TestGetTicketType(t *testing.T) {
	t.Run("returns a type", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		typeRepo(s).types["tt-1"] = &TicketType{ID: "tt-1", Name: "bug"}
		tt, err := s.GetTicketType(context.Background(), "tt-1")
		require.NoError(t, err)
		assert.Equal(t, "bug", tt.Name)
	})
}

func TestGetStatus(t *testing.T) {
	t.Run("returns a status", func(t *testing.T) {
		s, _, _ := newOwnerRepo(t, true)
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", Name: "Blocked", Kind: StatusKindProgress}
		st, err := s.GetStatus(context.Background(), "s-1")
		require.NoError(t, err)
		assert.Equal(t, "Blocked", st.Name)
	})
}

func TestCategoryTypeAndStatusUsecases_Refusals(t *testing.T) {
	ctx := context.Background()
	seedCats := func(s *Service, repo *fakeRepo) {
		repo.projects["p-1"] = &Project{ID: "p-1"}
		catRepo(s).cats["c-1"] = &Category{ID: "c-1", ProjectID: "p-1"}
		catRepo(s).cats["c-2"] = &Category{ID: "c-2", ProjectID: "p-1"}
	}
	seedTypes := func(s *Service, repo *fakeRepo) {
		repo.projects["p-1"] = &Project{ID: "p-1"}
		typeRepo(s).types["t-1"] = &TicketType{ID: "t-1", ProjectID: "p-1", Name: "bug"}
		typeRepo(s).types["t-2"] = &TicketType{ID: "t-2", ProjectID: "p-1"}
	}
	seedStatuses := func(s *Service, repo *fakeRepo) {
		repo.projects["p-1"] = &Project{ID: "p-1"}
		statusRepo(s).statuses["s-1"] = &Status{ID: "s-1", ProjectID: "p-1", Name: "Blocked", Kind: StatusKindProgress}
		statusRepo(s).statuses["s-2"] = &Status{ID: "s-2", ProjectID: "p-1"}
	}
	bogusColor := colors.Color("bogus")
	tests := []struct {
		name  string
		owner bool
		seed  func(s *Service, repo *fakeRepo)
		call  func(s *Service) error
		want  error
	}{
		{"create a category without a project", true, nil, func(s *Service) error { _, err := s.CreateCategory(ctx, "  ", "Sprint 1", ""); return err }, apperrs.ErrInvalid},
		{"create a category without a name", true, nil, func(s *Service) error { _, err := s.CreateCategory(ctx, "p-1", "  ", ""); return err }, apperrs.ErrInvalid},
		{"create a category as a non-owner", false, nil, func(s *Service) error { _, err := s.CreateCategory(ctx, "p-1", "Sprint 1", ""); return err }, apperrs.ErrForbidden},
		{"create a category with a bogus color", true, seedCats, func(s *Service) error { _, err := s.CreateCategory(ctx, "p-1", "Sprint 1", bogusColor); return err }, apperrs.ErrInvalid},
		{"get a category without an id", true, nil, func(s *Service) error { _, err := s.GetCategory(ctx, " "); return err }, apperrs.ErrInvalid},
		{"get a missing category", true, nil, func(s *Service) error { _, err := s.GetCategory(ctx, "nope"); return err }, apperrs.ErrNotFound},
		{"list categories without a project", true, nil, func(s *Service) error { _, err := s.ListCategoriesByProject(ctx, "  "); return err }, apperrs.ErrInvalid},
		{"rename a category without a name", true, nil, func(s *Service) error { _, err := s.RenameCategory(ctx, "c-1", "  ", ""); return err }, apperrs.ErrInvalid},
		{"rename a missing category", true, nil, func(s *Service) error { _, err := s.RenameCategory(ctx, "nope", "Sprint 2", ""); return err }, apperrs.ErrNotFound},
		{"rename a category as a non-owner", false, seedCats, func(s *Service) error { _, err := s.RenameCategory(ctx, "c-1", "Sprint 2", ""); return err }, apperrs.ErrForbidden},
		{"rename a category with a bogus color", true, nil, func(s *Service) error { _, err := s.RenameCategory(ctx, "c-1", "Sprint 2", bogusColor); return err }, apperrs.ErrInvalid},
		{"reorder categories without a project", true, nil, func(s *Service) error { return s.ReorderCategories(ctx, " ", []string{"c-1"}) }, apperrs.ErrInvalid},
		{"reorder categories with duplicate ids", true, seedCats, func(s *Service) error { return s.ReorderCategories(ctx, "p-1", []string{"c-1", "c-1"}) }, apperrs.ErrInvalid},
		{"reorder categories leaving one out", true, seedCats, func(s *Service) error { return s.ReorderCategories(ctx, "p-1", []string{"c-1"}) }, apperrs.ErrInvalid},
		{"reorder categories as a non-owner", false, seedCats, func(s *Service) error { return s.ReorderCategories(ctx, "p-1", []string{"c-1", "c-2"}) }, apperrs.ErrForbidden},
		{"delete a missing category", true, nil, func(s *Service) error { return s.DeleteCategory(ctx, "nope") }, apperrs.ErrNotFound},
		{"delete a category as a non-owner", false, seedCats, func(s *Service) error { return s.DeleteCategory(ctx, "c-1") }, apperrs.ErrForbidden},
		{"move a ticket without an id", true, nil, func(s *Service) error { return s.MoveTicketToCategory(ctx, " ", "c-1") }, apperrs.ErrInvalid},
		{"move a ticket into a missing category", true, nil, func(s *Service) error { return s.MoveTicketToCategory(ctx, "t-1", "nope") }, apperrs.ErrNotFound},
		{"move a missing ticket", true, func(s *Service, repo *fakeRepo) {
			seedCats(s, repo)
			catRepo(s).moveErr = apperrs.ErrNotFound
		}, func(s *Service) error { return s.MoveTicketToCategory(ctx, "nope", "c-1") }, apperrs.ErrNotFound},

		{"create a ticket type without a project", true, nil, func(s *Service) error { _, err := s.CreateTicketType(ctx, "  ", "bug", ""); return err }, apperrs.ErrInvalid},
		{"create a ticket type without a name", true, nil, func(s *Service) error { _, err := s.CreateTicketType(ctx, "p-1", "  ", ""); return err }, apperrs.ErrInvalid},
		{"create a ticket type as a non-owner", false, nil, func(s *Service) error { _, err := s.CreateTicketType(ctx, "p-1", "bug", ""); return err }, apperrs.ErrForbidden},
		{"create a ticket type with a bogus color", true, nil, func(s *Service) error { _, err := s.CreateTicketType(ctx, "p-1", "bug", bogusColor); return err }, apperrs.ErrInvalid},
		{"get a ticket type without an id", true, nil, func(s *Service) error { _, err := s.GetTicketType(ctx, " "); return err }, apperrs.ErrInvalid},
		{"get a missing ticket type", true, nil, func(s *Service) error { _, err := s.GetTicketType(ctx, "nope"); return err }, apperrs.ErrNotFound},
		{"list ticket types without a project", true, nil, func(s *Service) error { _, err := s.ListTicketTypesByProject(ctx, "  "); return err }, apperrs.ErrInvalid},
		{"rename a ticket type without a name", true, nil, func(s *Service) error { _, err := s.RenameTicketType(ctx, "t-1", "  ", ""); return err }, apperrs.ErrInvalid},
		{"rename a missing ticket type", true, nil, func(s *Service) error { _, err := s.RenameTicketType(ctx, "nope", "bug", ""); return err }, apperrs.ErrNotFound},
		{"rename a ticket type with a bogus color", true, seedTypes, func(s *Service) error { _, err := s.RenameTicketType(ctx, "t-1", "bug", bogusColor); return err }, apperrs.ErrInvalid},
		{"set a template as a non-owner", false, seedTypes, func(s *Service) error { _, err := s.SetTicketTypeTemplate(ctx, "t-1", "## Why"); return err }, apperrs.ErrForbidden},
		{"set a template on a missing type", true, nil, func(s *Service) error { _, err := s.SetTicketTypeTemplate(ctx, "nope", "## Why"); return err }, apperrs.ErrNotFound},
		{"delete a ticket type in use", true, func(s *Service, repo *fakeRepo) {
			seedTypes(s, repo)
			repo.ticketPro["t-1"] = "p-1"
			typeRepo(s).count = 1
		}, func(s *Service) error { return s.DeleteTicketType(ctx, "t-1") }, apperrs.ErrConflict},
		{"delete a ticket type as a non-owner", false, seedTypes, func(s *Service) error { return s.DeleteTicketType(ctx, "t-1") }, apperrs.ErrForbidden},
		{"delete a missing ticket type", true, nil, func(s *Service) error { return s.DeleteTicketType(ctx, "nope") }, apperrs.ErrNotFound},
		{"reorder ticket types without a project", true, nil, func(s *Service) error { return s.ReorderTicketTypes(ctx, " ", []string{"t-1"}) }, apperrs.ErrInvalid},
		{"reorder ticket types with duplicate ids", true, seedTypes, func(s *Service) error { return s.ReorderTicketTypes(ctx, "p-1", []string{"t-1", "t-1"}) }, apperrs.ErrInvalid},
		{"reorder ticket types leaving one out", true, seedTypes, func(s *Service) error { return s.ReorderTicketTypes(ctx, "p-1", []string{"t-1"}) }, apperrs.ErrInvalid},

		{"create a status without a project", true, nil, func(s *Service) error {
			_, err := s.CreateStatus(ctx, "  ", "Blocked", StatusKindProgress, "")
			return err
		}, apperrs.ErrInvalid},
		{"create a status without a name", true, nil, func(s *Service) error {
			_, err := s.CreateStatus(ctx, "p-1", "  ", StatusKindProgress, "")
			return err
		}, apperrs.ErrInvalid},
		{"create a status with a bogus kind", true, nil, func(s *Service) error {
			_, err := s.CreateStatus(ctx, "p-1", "Blocked", StatusKind("bogus"), "")
			return err
		}, apperrs.ErrInvalid},
		{"create a status as a non-owner", false, nil, func(s *Service) error {
			_, err := s.CreateStatus(ctx, "p-1", "Blocked", StatusKindProgress, "")
			return err
		}, apperrs.ErrForbidden},
		{"create a status with a bogus icon", true, nil, func(s *Service) error {
			_, err := s.CreateStatus(ctx, "p-1", "Blocked", StatusKindProgress, StatusIcon("bogus"))
			return err
		}, apperrs.ErrInvalid},
		{"get a status without an id", true, nil, func(s *Service) error { _, err := s.GetStatus(ctx, " "); return err }, apperrs.ErrInvalid},
		{"get a missing status", true, nil, func(s *Service) error { _, err := s.GetStatus(ctx, "nope"); return err }, apperrs.ErrNotFound},
		{"list statuses without a project", true, nil, func(s *Service) error { _, err := s.ListStatusesByProject(ctx, "  "); return err }, apperrs.ErrInvalid},
		{"rename a status without a name", true, nil, func(s *Service) error {
			_, err := s.RenameStatus(ctx, "s-1", "  ", StatusKindProgress, "")
			return err
		}, apperrs.ErrInvalid},
		{"rename a status with a bogus kind", true, nil, func(s *Service) error {
			_, err := s.RenameStatus(ctx, "s-1", "Blocked", StatusKind("bogus"), "")
			return err
		}, apperrs.ErrInvalid},
		{"rename a status with a bogus icon", true, seedStatuses, func(s *Service) error {
			_, err := s.RenameStatus(ctx, "s-1", "Blocked", StatusKindProgress, StatusIcon("bogus"))
			return err
		}, apperrs.ErrInvalid},
		{"rename a missing status", true, nil, func(s *Service) error {
			_, err := s.RenameStatus(ctx, "nope", "Blocked", StatusKindProgress, "")
			return err
		}, apperrs.ErrNotFound},
		{"reorder statuses without a project", true, nil, func(s *Service) error { return s.ReorderStatuses(ctx, " ", []string{"s-1"}) }, apperrs.ErrInvalid},
		{"reorder statuses with duplicate ids", true, seedStatuses, func(s *Service) error { return s.ReorderStatuses(ctx, "p-1", []string{"s-1", "s-1"}) }, apperrs.ErrInvalid},
		{"reorder statuses leaving one out", true, seedStatuses, func(s *Service) error { return s.ReorderStatuses(ctx, "p-1", []string{"s-1"}) }, apperrs.ErrInvalid},
		{"delete a status in use", true, func(s *Service, repo *fakeRepo) {
			seedStatuses(s, repo)
			repo.ticketPro["t-1"] = "p-1"
			statusRepo(s).count = 1
		}, func(s *Service) error { return s.DeleteStatus(ctx, "s-1") }, apperrs.ErrConflict},
		{"delete a missing status", true, nil, func(s *Service) error { return s.DeleteStatus(ctx, "nope") }, apperrs.ErrNotFound},
		{"delete a status as a non-owner", false, seedStatuses, func(s *Service) error { return s.DeleteStatus(ctx, "s-1") }, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, _ := newOwnerRepo(t, tt.owner)
			if tt.seed != nil {
				tt.seed(s, repo)
			}
			require.ErrorIs(t, tt.call(s), tt.want)
		})
	}
}
