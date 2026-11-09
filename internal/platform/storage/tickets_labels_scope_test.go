package storage

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// readerGate lets the caller read tickets only in its projects.
type readerGate struct{ projects []string }

func (g readerGate) RequireProject(_ context.Context, projectID string, _ permissions.Action) error {
	if slices.Contains(g.projects, projectID) {
		return nil
	}
	return fmt.Errorf("project %s: %w", projectID, apperrs.ErrForbidden)
}

func (g readerGate) CallerProjects(context.Context, permissions.Action) ([]string, bool, error) {
	return g.projects, false, nil
}

func (g readerGate) CallerProjectsIn(context.Context, string, permissions.Action) ([]string, bool, error) {
	return g.projects, false, nil
}

func (g readerGate) Require(context.Context, string, permissions.Action) error { return nil }

// seedLabelledTickets files n long-bodied tickets across three projects; each carries one label shared by all
// projects and one only its project uses.
func seedLabelledTickets(t *testing.T, s *Store, n int) {
	t.Helper()
	ctx := t.Context()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{"p-a", "p-b", "p-c"} {
		require.NoError(t, s.Projects.Create(ctx, &workspace.Project{
			ID: id, Name: id, Prefix: strings.ToUpper(strings.ReplaceAll(id, "-", "")), WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now,
		}))
	}
	body := strings.Repeat("spec line\n", 200)
	for i := range n {
		project := []string{"p-a", "p-b", "p-c"}[i%3]
		tk := newTestTicket(fmt.Sprintf("t-%05d", i), "")
		tk.ProjectID, tk.Body = project, body
		tk.Labels = []string{"shared", "only-" + project}
		require.NoError(t, s.Tickets.Create(ctx, tk))
	}
}

func TestTicketsService_ListAllLabels_ReadsOnlyTheLabelsHoweverManyTicketsThereAre(t *testing.T) {
	read := func(n int) (labels []string, statements, rows int64) {
		s, st := newCountedStore(t)
		seedLabelledTickets(t, s, n)
		svc := tickets.NewService(s.Tickets, nil, nil)
		svc.SetGate(readerGate{projects: []string{"p-a", "p-b"}})
		st.Reset()
		labels, err := svc.ListAllLabels(t.Context(), "")
		require.NoError(t, err)
		return labels, st.Count(), st.Rows()
	}

	fewLabels, fewStatements, fewRows := read(30)
	manyLabels, manyStatements, manyRows := read(600)

	assert.Equal(t, []string{"only-p-a", "only-p-b", "shared"}, manyLabels, "a project the caller cannot read adds no label")
	assert.Equal(t, fewLabels, manyLabels)
	assert.Equal(t, fewStatements, manyStatements, "statements at 30 and 600 tickets")
	assert.Equal(t, fewRows, manyRows, "rows read at 30 and 600 tickets")
}

func TestTicketsRepo_ListLabelNames_MatchesTheLabelsOfEveryReadableTicket(t *testing.T) {
	s := newTestStore(t)
	seedLabelledTickets(t, s, 12)
	all, err := s.Tickets.List(t.Context())
	require.NoError(t, err)

	for name, scope := range map[string]tickets.TicketScope{
		"all projects":  {All: true},
		"two projects":  {ProjectIDs: []string{"p-a", "p-c"}},
		"no projects":   {},
		"unknown named": {ProjectIDs: []string{"p-none"}},
	} {
		var want []string
		for _, tk := range all {
			if scope.All || slices.Contains(scope.ProjectIDs, tk.ProjectID) {
				want = append(want, tk.Labels...)
			}
		}
		slices.Sort(want)
		want = slices.Compact(want)

		got, err := s.Tickets.ListLabelNames(t.Context(), scope)
		require.NoError(t, err, name)
		assert.Equal(t, want, got, name)
	}
}
