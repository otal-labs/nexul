package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newTestProject(id, name string, position int) *workspace.Project {
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	return &workspace.Project{ID: id, Name: name, Position: position, WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now}
}

func TestProjectsRepo_Create_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestProject("p-1", "Backend", 0)
	require.NoError(t, s.Projects.Create(context.Background(), want))

	got, err := s.Projects.Get(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, want.Name, got.Name)
	assert.Equal(t, 0, got.Position)
	assert.Equal(t, workspace.ProjectIcon(""), got.Icon)
}

func TestProjectsRepo_IconRoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := newTestProject("p-1", "Backend", 0)
	want.Icon = workspace.ProjectIconRocket
	require.NoError(t, s.Projects.Create(context.Background(), want))

	got, err := s.Projects.Get(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, workspace.ProjectIconRocket, got.Icon)

	updated := *got
	updated.Icon = workspace.ProjectIconGlobe
	require.NoError(t, s.Projects.Update(context.Background(), &updated))
	got, err = s.Projects.Get(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, workspace.ProjectIconGlobe, got.Icon)
}

func TestProjectsRepo_Create_SeedsDefaultBoard(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))

	statuses, err := s.Statuses.ListByProject(context.Background(), "p-1")
	require.NoError(t, err)
	require.Len(t, statuses, 5, "one column per stage, seeded exactly once")
	byName := map[string]*workspace.Status{}
	for _, st := range statuses {
		byName[st.Name] = st
	}
	assert.Equal(t, workspace.StatusKindBacklog, byName["Backlog"].Kind)
	assert.Equal(t, workspace.StatusKindProgress, byName["In progress"].Kind)
	assert.Equal(t, workspace.StatusKindReview, byName["In review"].Kind)
	assert.Equal(t, workspace.StatusKindTesting, byName["Testing"].Kind)
	assert.Equal(t, workspace.StatusKindDone, byName["Done"].Kind)

	types, err := s.TicketTypes.ListByProject(context.Background(), "p-1")
	require.NoError(t, err)
	require.Len(t, types, 3, "task/bug/feature, seeded exactly once")
	var names []string
	for _, tt := range types {
		names = append(names, tt.Name)
	}
	assert.ElementsMatch(t, []string{"task", "bug", "feature"}, names)
}

func TestProjectsRepo_Create_SeedsBodyTemplates(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Backend", 0)))

	types, err := s.TicketTypes.ListByProject(context.Background(), "p-1")
	require.NoError(t, err)
	require.Len(t, types, len(workspace.DefaultTicketTypes))
	for i, want := range workspace.DefaultTicketTypes {
		assert.Equal(t, want.Name, types[i].Name)
		assert.Equal(t, want.BodyTemplate, types[i].BodyTemplate)
	}
	assert.Contains(t, types[1].BodyTemplate, "## Steps to reproduce")
}

func TestMigration_BackfillsGeneralProjectTemplates_MatchingTheSeed(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)

	types, err := s.TicketTypes.ListByProject(context.Background(), "project-general")
	require.NoError(t, err)
	byName := map[string]string{}
	for _, tt := range types {
		byName[tt.Name] = tt.BodyTemplate
	}
	for _, want := range workspace.DefaultTicketTypes {
		assert.Equal(t, want.BodyTemplate, byName[want.Name], "migration 0009 and DefaultTicketTypes drifted for %s", want.Name)
	}
}

func TestProjectsRepo_Create_Duplicate_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	err := s.Projects.Create(context.Background(), newTestProject("p-1", "B", 1))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestProjectsRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Projects.Get(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestProjectsRepo_List_OrderedByPosition(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-2", "B", 2)))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 1)))

	got, err := s.Projects.List(context.Background(), "workspace-default")
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, "project-general", got[0].ID)
	assert.Equal(t, "p-1", got[1].ID)
	assert.Equal(t, "p-2", got[2].ID)
}

func TestProjectsRepo_List_ScopesToWorkspace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), &tenancy.Workspace{ID: "ws-2", Name: "Other", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	p := newTestProject("p-1", "A", 0)
	p.WorkspaceID = "ws-2"
	require.NoError(t, s.Projects.Create(context.Background(), p))

	got, err := s.Projects.List(context.Background(), "ws-2")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "p-1", got[0].ID)

	got, err = s.Projects.List(context.Background(), "workspace-default")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "project-general", got[0].ID)
}

func TestProjectsRepo_Create_UnknownWorkspace_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	p := newTestProject("p-1", "A", 0)
	p.WorkspaceID = "missing"
	err := s.Projects.Create(context.Background(), p)
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestProjectsRepo_Rename_Persists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "Old", 0)))
	updated := newTestProject("p-1", "New", 0)
	require.NoError(t, s.Projects.Update(context.Background(), updated))

	got, err := s.Projects.Get(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, "New", got.Name)
}

func TestProjectsRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Projects.Update(context.Background(), newTestProject("missing", "X", 0))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestProjectsRepo_Reorder_AssignsPositions(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-2", "B", 1)))

	require.NoError(t, s.Projects.Reorder(context.Background(), []string{"p-2", "p-1"}))
	got, err := s.Projects.List(context.Background(), "workspace-default")
	require.NoError(t, err)
	assert.Equal(t, "p-2", got[0].ID)
	assert.Equal(t, "p-1", got[2].ID)
}

func TestProjectsRepo_Delete_RemovesProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	require.NoError(t, s.Projects.Delete(context.Background(), "p-1"))
	_, err := s.Projects.Get(context.Background(), "p-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestProjectsRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Projects.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestProjectsRepo_Counts(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	require.NoError(t, s.Projects.AddRepo(context.Background(), "p-1", workspace.RepoRef{Owner: "acme", Name: "app", FullName: "acme/app"}))

	tickets, err := s.Projects.CountTickets(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, 0, tickets)

	repos, err := s.Projects.CountRepos(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, 1, repos)
}

func TestProjectsRepo_AddRepo_UnknownProject_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Projects.AddRepo(context.Background(), "missing", workspace.RepoRef{Owner: "acme", Name: "app"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestProjectsRepo_AddRepo_RepoBelongsToOneProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-2", "B", 1)))
	require.NoError(t, s.Projects.AddRepo(context.Background(), "p-1", workspace.RepoRef{Owner: "acme", Name: "app"}))

	err := s.Projects.AddRepo(context.Background(), "p-2", workspace.RepoRef{Owner: "acme", Name: "app"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestProjectsRepo_RemoveRepo_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Projects.RemoveRepo(context.Background(), "acme", "app")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestProjectsRepo_ListRepos_ScopesToProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-2", "B", 1)))
	require.NoError(t, s.Projects.AddRepo(context.Background(), "p-1", workspace.RepoRef{Owner: "acme", Name: "app", FullName: "acme/app"}))
	require.NoError(t, s.Projects.AddRepo(context.Background(), "p-2", workspace.RepoRef{Owner: "acme", Name: "other", FullName: "acme/other"}))

	got, err := s.Projects.ListRepos(context.Background(), "p-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "acme/app", got[0].FullName)
}

// AddRepo/ListRepos must round-trip whatever ConnectorID they're given; defaulting an empty one is the use-case's job.
func TestProjectsRepo_AddRepo_PersistsConnectorID(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	require.NoError(t, s.Projects.AddRepo(context.Background(), "p-1",
		workspace.RepoRef{Owner: "acme", Name: "app", FullName: "acme/app", ConnectorID: "gitlab-self-hosted"}))

	got, err := s.Projects.ListRepos(context.Background(), "p-1")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "gitlab-self-hosted", got[0].ConnectorID)
}

func TestProjectsRepo_MoveTicket_PersistsAndNotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	tk := newTestTicket("t-1", "")
	tk.ProjectID = "project-general"
	require.NoError(t, s.Tickets.Create(context.Background(), tk))

	require.NoError(t, s.Projects.MoveTicket(context.Background(), "t-1", "p-1"))
	got, err := s.Tickets.GetByID(context.Background(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, "p-1", got.ProjectID)

	err = s.Projects.MoveTicket(context.Background(), "missing", "p-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestProjectsRepo_MoveTicket_UnknownProject_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "")))
	err := s.Projects.MoveTicket(context.Background(), "t-1", "missing")
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestTicketsRepo_ListByProject_ScopesToProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-1", "A", 0)))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("p-2", "B", 1)))

	a := newTestTicket("t-1", "")
	a.ProjectID = "p-1"
	b := newTestTicket("t-2", "")
	b.ProjectID = "p-2"
	c := newTestTicket("t-3", "")
	c.ProjectID = "p-1"
	require.NoError(t, s.Tickets.Create(context.Background(), a))
	require.NoError(t, s.Tickets.Create(context.Background(), b))
	require.NoError(t, s.Tickets.Create(context.Background(), c))

	got, err := s.Tickets.ListByProject(context.Background(), "p-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "t-1", got[0].ID)
	assert.Equal(t, "t-3", got[1].ID)
}
