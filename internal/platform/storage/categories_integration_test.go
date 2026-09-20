package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newTestCategory(id, projectID, name string, position int) *workspace.Category {
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	return &workspace.Category{ID: id, ProjectID: projectID, Name: name, Position: position, CreatedAt: now, UpdatedAt: now}
}

func TestCategoriesRepo_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))

	want := newTestCategory("c-1", "p-1", "Sprint 1", 0)
	require.NoError(t, s.Categories.Create(ctx, want))

	got, err := s.Categories.Get(ctx, "c-1")
	require.NoError(t, err)
	assert.Equal(t, "Sprint 1", got.Name)
	assert.Equal(t, "p-1", got.ProjectID)
	assert.Equal(t, colors.Color(""), got.Color)

	cats, err := s.Categories.ListByProject(ctx, "p-1")
	require.NoError(t, err)
	require.Len(t, cats, 1)
}

func TestCategoriesRepo_ColorRoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))

	want := newTestCategory("c-1", "p-1", "Sprint 1", 0)
	want.Color = colors.Cyan
	require.NoError(t, s.Categories.Create(ctx, want))

	got, err := s.Categories.Get(ctx, "c-1")
	require.NoError(t, err)
	assert.Equal(t, colors.Cyan, got.Color)

	updated := *got
	updated.Color = colors.Emerald
	require.NoError(t, s.Categories.Update(ctx, &updated))
	got, err = s.Categories.Get(ctx, "c-1")
	require.NoError(t, err)
	assert.Equal(t, colors.Emerald, got.Color)
}

func TestCategoriesRepo_Reorder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-1", "p-1", "A", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-2", "p-1", "B", 1)))

	require.NoError(t, s.Categories.Reorder(ctx, "p-1", []string{"c-2", "c-1"}))
	cats, err := s.Categories.ListByProject(ctx, "p-1")
	require.NoError(t, err)
	require.Len(t, cats, 2)
	assert.Equal(t, "c-2", cats[0].ID)
	assert.Equal(t, "c-1", cats[1].ID)
}

func TestCategoriesRepo_DeleteDoesNotDeleteTickets(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-1", "p-1", "Sprint 1", 0)))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", CategoryID: "c-1", Title: "x", Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, s.Categories.Delete(ctx, "c-1"))
	got, err := s.Tickets.GetByID(ctx, "t-1")
	require.NoError(t, err)
	assert.Equal(t, "", got.CategoryID, "ticket becomes uncategorized, never deleted")
}

func TestCategoriesRepo_SetTicketCategory(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-1", "p-1", "Sprint 1", 0)))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", Title: "x", Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, s.Categories.SetTicketCategory(ctx, "t-1", "c-1"))
	got, err := s.Tickets.GetByID(ctx, "t-1")
	require.NoError(t, err)
	assert.Equal(t, "c-1", got.CategoryID)

	require.NoError(t, s.Categories.SetTicketCategory(ctx, "t-1", ""))
	got, err = s.Tickets.GetByID(ctx, "t-1")
	require.NoError(t, err)
	assert.Equal(t, "", got.CategoryID)
}

func TestCategoriesRepo_SetTicketCategory_SameCategoryIsNoOpForPosition(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-1", "p-1", "Sprint 1", 0)))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", Title: "x", Status: "open", CategoryID: "c-1", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-2", ProjectID: "p-1", Title: "y", Status: "open", CategoryID: "c-1", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.Tickets.SetPosition(ctx, "t-1", 7))

	// Re-sending t-1's current category (a redundant move, e.g. a drop back
	// into the same lane) must not bump it to the end of the pair.
	require.NoError(t, s.Categories.SetTicketCategory(ctx, "t-1", "c-1"))
	got, err := s.Tickets.GetByID(ctx, "t-1")
	require.NoError(t, err)
	assert.Equal(t, "c-1", got.CategoryID)
	assert.Equal(t, 7, got.Position, "same-category move leaves manual position untouched")
}

func TestCategoriesRepo_SetTicketCategory_MissingTicket(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-1", "p-1", "Sprint 1", 0)))
	err := s.Categories.SetTicketCategory(ctx, "nope", "c-1")
	require.Error(t, err)
	assert.True(t, errorsIsNotFound(err))
}

func TestTicketTypesRepo_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	want := &workspace.TicketType{ID: "tt-1", ProjectID: "project-general", Name: "bug", Position: 0, Color: colors.Cyan, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.TicketTypes.Create(ctx, want))

	got, err := s.TicketTypes.Get(ctx, "tt-1")
	require.NoError(t, err)
	assert.Equal(t, "bug", got.Name)
	assert.Equal(t, colors.Cyan, got.Color)

	types, err := s.TicketTypes.ListByProject(ctx, "project-general")
	require.NoError(t, err)
	require.Len(t, types, 4, "seeded task/bug/feature + our type")
}

func TestTicketTypesRepo_DeleteInUseConflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", TypeID: "ticket-type-bug", Title: "x", Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	count, err := s.TicketTypes.CountTickets(ctx, "ticket-type-bug")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStatusesRepo_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	want := &workspace.Status{ID: "blocked", ProjectID: "project-general", Name: "Blocked", Position: 4, Kind: workspace.StatusKindProgress, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.Statuses.Create(ctx, want))

	got, err := s.Statuses.Get(ctx, "blocked")
	require.NoError(t, err)
	assert.Equal(t, "Blocked", got.Name)
	assert.Equal(t, workspace.StatusKindProgress, got.Kind)

	statuses, err := s.Statuses.ListByProject(ctx, "project-general")
	require.NoError(t, err)
	require.Len(t, statuses, 5, "seeded open/in_progress/done/closed + ours")
}

func TestStatusesRepo_ListByProject_ActiveBeforeCompleted(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	// Board columns group by kind first, so a later active status still lands before the completed group.
	require.NoError(t, s.Statuses.Create(ctx, &workspace.Status{ID: "qa", ProjectID: "project-general", Name: "Ready for QA", Position: 9, Kind: workspace.StatusKindProgress, CreatedAt: now, UpdatedAt: now}))

	statuses, err := s.Statuses.ListByProject(ctx, "project-general")
	require.NoError(t, err)
	ids := make([]string, 0, len(statuses))
	for _, st := range statuses {
		ids = append(ids, st.ID)
	}
	assert.Equal(t, []string{"open", "in_progress", "qa", "done", "closed"}, ids)
}

func TestStatusesRepo_Exists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	ok, err := s.Statuses.Exists(ctx, "open")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = s.Statuses.Exists(ctx, "nope")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestStatusesRepo_CountTickets(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", Title: "x", Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	count, err := s.Statuses.CountTickets(ctx, "open")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTicketsRepo_TypeAndCategoryColumns(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-1", "p-1", "Sprint 1", 0)))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", CategoryID: "c-1", TypeID: "ticket-type-bug", Title: "x", Status: "open",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	got, err := s.Tickets.GetByID(ctx, "t-1")
	require.NoError(t, err)
	assert.Equal(t, "c-1", got.CategoryID)
	assert.Equal(t, "ticket-type-bug", got.TypeID)

	require.NoError(t, s.Tickets.UpdateType(ctx, "t-1", "ticket-type-task"))
	got, err = s.Tickets.GetByID(ctx, "t-1")
	require.NoError(t, err)
	assert.Equal(t, "ticket-type-task", got.TypeID)
}

func TestTicketsRepo_Labels(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", Title: "x", Status: "open", Labels: []string{"bug", "urgent"},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-2", ProjectID: "p-1", Title: "y", Status: "open", Labels: []string{"bug"},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	got, err := s.Tickets.GetByID(ctx, "t-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"bug", "urgent"}, got.Labels)

	all, err := s.Tickets.ListAllLabels(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"bug", "urgent"}, all)

	require.NoError(t, s.Tickets.AddLabel(ctx, "t-2", "urgent"))
	got, err = s.Tickets.GetByID(ctx, "t-2")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"bug", "urgent"}, got.Labels)

	require.NoError(t, s.Tickets.RemoveLabel(ctx, "t-2", "bug"))
	got, err = s.Tickets.GetByID(ctx, "t-2")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"urgent"}, got.Labels)

	list, err := s.Tickets.ListLabels(ctx, "t-2")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"urgent"}, list)
}

func TestTicketsRepo_LabelColors(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-2", "Mobile", 1)))

	require.NoError(t, s.Tickets.SetLabelColor(ctx, "p-1", "bug", colors.Cyan))
	require.NoError(t, s.Tickets.SetLabelColor(ctx, "p-1", "urgent", colors.Orange))

	got, err := s.Tickets.LabelColors(ctx, "p-1", []string{"bug", "urgent", "unconfigured"})
	require.NoError(t, err)
	assert.Equal(t, colors.Cyan, got["bug"])
	assert.Equal(t, colors.Orange, got["urgent"])
	_, ok := got["unconfigured"]
	assert.False(t, ok, "unconfigured label has no row, not a zero value")

	// create-or-update: setting a color again for the same label updates in
	// place rather than erroring or duplicating the row.
	require.NoError(t, s.Tickets.SetLabelColor(ctx, "p-1", "bug", colors.Emerald))
	got, err = s.Tickets.LabelColors(ctx, "p-1", []string{"bug"})
	require.NoError(t, err)
	assert.Equal(t, colors.Emerald, got["bug"])

	// project scoping: the same label name in a different project carries
	// its own color, keyed by (project_id, label) per spec.md section 1.
	require.NoError(t, s.Tickets.SetLabelColor(ctx, "p-2", "bug", colors.Fuchsia))
	gotP2, err := s.Tickets.LabelColors(ctx, "p-2", []string{"bug"})
	require.NoError(t, err)
	assert.Equal(t, colors.Fuchsia, gotP2["bug"], "p-2's bug color is independent of p-1's")
}

func errorsIsNotFound(err error) bool {
	return errors.Is(err, apperrs.ErrNotFound)
}

func TestStatusesRepo_UpdateDeleteReorder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	st := &workspace.Status{ID: "s-9", ProjectID: "project-general", Name: "Blocked", Position: 4, Kind: workspace.StatusKindProgress, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.Statuses.Create(ctx, st))

	updated := *st
	updated.Name = "In review"
	updated.Kind = workspace.StatusKindDone
	require.NoError(t, s.Statuses.Update(ctx, &updated))
	got, err := s.Statuses.Get(ctx, "s-9")
	require.NoError(t, err)
	assert.Equal(t, "In review", got.Name)
	assert.Equal(t, workspace.StatusKindDone, got.Kind)

	other := &workspace.Status{ID: "s-10", ProjectID: "project-general", Name: "Backlog", Position: 5, Kind: workspace.StatusKindProgress, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.Statuses.Create(ctx, other))
	require.NoError(t, s.Statuses.Reorder(ctx, "project-general", []string{"s-10", "s-9"}))
	got10, err := s.Statuses.Get(ctx, "s-10")
	require.NoError(t, err)
	assert.Equal(t, 0, got10.Position, "reorder assigns position by index")
	got9, err := s.Statuses.Get(ctx, "s-9")
	require.NoError(t, err)
	assert.Equal(t, 1, got9.Position)

	require.NoError(t, s.Statuses.Delete(ctx, "s-9"))
	_, err = s.Statuses.Get(ctx, "s-9")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))

	require.NoError(t, s.Statuses.Update(ctx, &workspace.Status{ID: "s-10", Name: "x", Kind: workspace.StatusKindProgress, UpdatedAt: now}))
	require.NoError(t, s.Statuses.Delete(ctx, "s-10"))
	err = s.Statuses.Delete(ctx, "s-10")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestTicketTypesRepo_UpdateDeleteReorder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	tt := &workspace.TicketType{ID: "tt-9", ProjectID: "project-general", Name: "chore", Position: 5, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.TicketTypes.Create(ctx, tt))

	updated := *tt
	updated.Name = "ops"
	updated.Color = colors.Emerald
	require.NoError(t, s.TicketTypes.Update(ctx, &updated))
	got, err := s.TicketTypes.Get(ctx, "tt-9")
	require.NoError(t, err)
	assert.Equal(t, "ops", got.Name)
	assert.Equal(t, colors.Emerald, got.Color)

	other := &workspace.TicketType{ID: "tt-10", ProjectID: "project-general", Name: "docs", Position: 6, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, s.TicketTypes.Create(ctx, other))
	require.NoError(t, s.TicketTypes.Reorder(ctx, "project-general", []string{"tt-10", "tt-9"}))
	got10, err := s.TicketTypes.Get(ctx, "tt-10")
	require.NoError(t, err)
	assert.Equal(t, 0, got10.Position)
	got9, err := s.TicketTypes.Get(ctx, "tt-9")
	require.NoError(t, err)
	assert.Equal(t, 1, got9.Position)

	require.NoError(t, s.TicketTypes.Delete(ctx, "tt-9"))
	_, err = s.TicketTypes.Get(ctx, "tt-9")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))

	require.NoError(t, s.TicketTypes.Update(ctx, &workspace.TicketType{ID: "tt-10", Name: "x", UpdatedAt: now}))
	require.NoError(t, s.TicketTypes.Delete(ctx, "tt-10"))
	err = s.TicketTypes.Delete(ctx, "tt-10")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestCategoriesRepo_UpdateListCount(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-1", "Backend", 0)))
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-2", "Frontend", 1)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-1", "p-1", "Sprint 1", 0)))
	require.NoError(t, s.Categories.Create(ctx, newTestCategory("c-2", "p-2", "Sprint 1", 0)))

	updated := newTestCategory("c-1", "p-1", "Sprint 2", 0)
	require.NoError(t, s.Categories.Update(ctx, updated))
	got, err := s.Categories.Get(ctx, "c-1")
	require.NoError(t, err)
	assert.Equal(t, "Sprint 2", got.Name)

	all, err := s.Categories.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)

	require.NoError(t, s.Tickets.Create(ctx, &tickets.Ticket{
		ID: "t-1", ProjectID: "p-1", CategoryID: "c-1", Title: "x", Status: "open", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	count, err := s.Categories.CountTickets(ctx, "c-1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	err = s.Categories.Update(ctx, &workspace.Category{ID: "nope", Name: "x", UpdatedAt: time.Now()})
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}
