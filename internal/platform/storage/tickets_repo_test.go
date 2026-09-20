package storage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newTestTicket(id, docID string) *tickets.Ticket {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	return &tickets.Ticket{
		ID: id, Title: "Fix storage", Body: "Write migrations", Status: tickets.StatusOpen,
		DocID: docID, ProjectID: "project-general", Assignee: "onik97", CreatedAt: now, UpdatedAt: now,
	}
}

func TestTicketsRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Tickets.GetByID(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTicketsRepo_GetByPrefixAndNumber_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	require.NoError(t, s.Projects.Create(context.Background(), &workspace.Project{
		ID: "p-erf", Name: "Engineering", Prefix: "ERF", WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	want := newTestTicket("t-1", "doc-1")
	want.ProjectID = "p-erf"
	require.NoError(t, s.Tickets.Create(context.Background(), want))

	got, err := s.Tickets.GetByPrefixAndNumber(context.Background(), "ERF", 1)
	require.NoError(t, err)
	assert.Equal(t, "t-1", got.ID)
	assert.Equal(t, "Fix storage", got.Title)
}

func TestTicketsRepo_GetByPrefixAndNumber_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Tickets.GetByPrefixAndNumber(context.Background(), "ZZZ", 1)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTicketsRepo_Create_DuplicateID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "doc-1")))
	err := s.Tickets.Create(context.Background(), newTestTicket("t-1", "doc-1"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestTicketsRepo_Create_UnknownDocID_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Tickets.Create(context.Background(), newTestTicket("t-1", "does-not-exist"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestTicketsRepo_Create_GetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	want := newTestTicket("t-1", "doc-1")
	require.NoError(t, s.Tickets.Create(context.Background(), want))

	got, err := s.Tickets.GetByID(context.Background(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, want.Title, got.Title)
	assert.Equal(t, tickets.StatusOpen, got.Status)
	assert.Equal(t, "doc-1", got.DocID)
	assert.Equal(t, "onik97", got.Assignee)
}

func TestTicketsRepo_List_ReturnsAll(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-2", "doc-1")))

	got, err := s.Tickets.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestTicketsRepo_ListByDoc_ScopesToDoc(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-2")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-2", "doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-3", "doc-2")))

	got, err := s.Tickets.ListByDoc(context.Background(), "doc-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "t-1", got[0].ID)
	assert.Equal(t, "t-2", got[1].ID)
}

func TestTicketsRepo_UpdateTicket_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Tickets.UpdateTicket(context.Background(), "missing", "New title", "New body")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTicketsRepo_UpdateTicket_Persists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "doc-1")))

	require.NoError(t, s.Tickets.UpdateTicket(context.Background(), "t-1", "New title", "New body"))
	got, err := s.Tickets.GetByID(context.Background(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, "New title", got.Title)
	assert.Equal(t, "New body", got.Body)
}

func TestTicketsRepo_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Tickets.UpdateStatus(context.Background(), "missing", tickets.StatusDone)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTicketsRepo_UpdateStatus_Persists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "doc-1")))

	require.NoError(t, s.Tickets.UpdateStatus(context.Background(), "t-1", tickets.StatusDone))
	got, err := s.Tickets.GetByID(context.Background(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, tickets.StatusDone, got.Status)
	assert.Equal(t, 0, got.Position, "first ticket in the (done, uncategorized) pair")
}

// Regression: position is computed in Create's own transaction, not a pre-read, so concurrent creates can't collide.
func TestTicketsRepo_Create_ConcurrentIntoEmptyPairAssignsDistinctPositions(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = s.Tickets.Create(context.Background(), newTestTicket(fmt.Sprintf("t-%d", i), "doc-1"))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "create %d", i)
	}

	got, err := s.Tickets.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, n)
	seen := map[int]bool{}
	for _, tk := range got {
		assert.False(t, seen[tk.Position], "position %d assigned to more than one ticket", tk.Position)
		seen[tk.Position] = true
	}
	assert.Len(t, seen, n, "all n positions are distinct")
}

// ADR 0004: ticket numbering is sequential per project, independent of any other project's tickets.
func TestTicketsRepo_Create_AssignsSequentialNumberPerProject(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("project-a", "A", 0)))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("project-b", "B", 1)))

	a1 := newTestTicket("a-1", "doc-1")
	a1.ProjectID = "project-a"
	require.NoError(t, s.Tickets.Create(context.Background(), a1))

	b1 := newTestTicket("b-1", "doc-1")
	b1.ProjectID = "project-b"
	require.NoError(t, s.Tickets.Create(context.Background(), b1))

	a2 := newTestTicket("a-2", "doc-1")
	a2.ProjectID = "project-a"
	require.NoError(t, s.Tickets.Create(context.Background(), a2))

	gotA1, err := s.Tickets.GetByID(context.Background(), "a-1")
	require.NoError(t, err)
	assert.Equal(t, 1, gotA1.Number)

	gotB1, err := s.Tickets.GetByID(context.Background(), "b-1")
	require.NoError(t, err)
	assert.Equal(t, 1, gotB1.Number, "project-b's first ticket is unaffected by project-a's count")

	gotA2, err := s.Tickets.GetByID(context.Background(), "a-2")
	require.NoError(t, err)
	assert.Equal(t, 2, gotA2.Number)
}

// Regression: number is computed inside Create's own transaction, like position, so concurrent creates can't collide.
func TestTicketsRepo_Create_ConcurrentIntoSameProjectAssignsDistinctNumbers(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Projects.Create(context.Background(), newTestProject("project-concurrent", "Concurrent", 0)))

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tk := newTestTicket(fmt.Sprintf("num-%d", i), "doc-1")
			tk.ProjectID = "project-concurrent"
			errs[i] = s.Tickets.Create(context.Background(), tk)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "create %d", i)
	}

	got, err := s.Tickets.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, n)
	seen := map[int]bool{}
	for _, tk := range got {
		assert.False(t, seen[tk.Number], "number %d assigned to more than one ticket", tk.Number)
		seen[tk.Number] = true
	}
	assert.Len(t, seen, n, "all n numbers are distinct")
}

func TestTicketsRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Tickets.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTicketsRepo_Delete_RemovesTicket(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), newTestTicket("t-1", "doc-1")))

	require.NoError(t, s.Tickets.Delete(context.Background(), "t-1"))
	_, err := s.Tickets.GetByID(context.Background(), "t-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
