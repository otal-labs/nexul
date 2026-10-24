package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/tickets"
)

// seedDueTickets files tickets across two workspaces and three projects; the number is the day of October they are due,
// and 0 files one with no due date.
func seedDueTickets(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.db.Exec(`INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-other', 'Other', 'other', 1, 1)`)
	require.NoError(t, err)
	seedProject(t, s, "p-private", "workspace-default", "PRV")
	seedProject(t, s, "p-other", "ws-other", "OTH")
	for _, tk := range []struct {
		id, project, due string
	}{
		{"t-a", "project-general", "2026-10-05"},
		{"t-b", "project-general", "2026-10-01"},
		{"t-c", "project-general", "2026-10-31"},
		{"t-d", "project-general", "2026-11-01"},
		{"t-e", "project-general", "2026-09-30"},
		{"t-none", "project-general", ""},
		{"t-private", "p-private", "2026-10-10"},
		{"t-other", "p-other", "2026-10-10"},
		{"t-same-day", "project-general", "2026-10-05"},
	} {
		ticket := newTestTicket(tk.id, "")
		ticket.ProjectID = tk.project
		ticket.DueOn = tk.due
		ticket.Labels = []string{"urgent"}
		require.NoError(t, s.Tickets.Create(t.Context(), ticket))
		if tk.due != "" {
			require.NoError(t, s.Tickets.UpdateDueOn(t.Context(), tk.id, tk.due), "Create leaves the due date to UpdateDueOn")
		}
	}
}

func dueIDs(ts []*tickets.Ticket) []string {
	ids := make([]string, len(ts))
	for i, tk := range ts {
		ids[i] = tk.ID
	}
	return ids
}

func TestTicketsRepo_UpdateDueOn(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Tickets.Create(t.Context(), newTestTicket("t-1", "")))

	got, err := s.Tickets.GetByID(t.Context(), "t-1")
	require.NoError(t, err)
	assert.Empty(t, got.DueOn, "a new ticket has no due date")

	evt := eventbus.OutboxEvent{ID: "e-1", Topic: tickets.TopicDueDateChanged, Payload: tickets.DueDateChangedEvent{Ticket: *got, PreviousDueOn: ""}}
	require.NoError(t, s.Tickets.UpdateDueOn(t.Context(), "t-1", "2026-10-30", evt))
	got, err = s.Tickets.GetByID(t.Context(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, "2026-10-30", got.DueOn)
	assert.Equal(t, "onik97", got.Developer, "a due date edit touches nothing else")
	payloads := outboxPayload(t, s, tickets.TopicDueDateChanged)
	require.Len(t, payloads, 1)
	assert.Equal(t, "", payloads[0]["previous_due_on"])

	require.NoError(t, s.Tickets.UpdateDueOn(t.Context(), "t-1", ""))
	got, err = s.Tickets.GetByID(t.Context(), "t-1")
	require.NoError(t, err)
	assert.Empty(t, got.DueOn)

	require.ErrorIs(t, s.Tickets.UpdateDueOn(t.Context(), "missing", "2026-10-30"), apperrs.ErrNotFound)
}

func TestTicketsRepo_ListDue(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedDueTickets(t, s)
	all := tickets.TicketScope{All: true}

	tests := []struct {
		name      string
		workspace string
		from, to  string
		scope     tickets.TicketScope
		want      []string
	}{
		{"the range is inclusive, by due date then id, and skips tickets with no date", "workspace-default", "2026-10-01", "2026-10-31", all,
			[]string{"t-b", "t-a", "t-same-day", "t-private", "t-c"}},
		{"another workspace's tickets stay out", "ws-other", "2026-10-01", "2026-10-31", all, []string{"t-other"}},
		{"a project outside the readable set is left out", "workspace-default", "2026-10-01", "2026-10-31",
			tickets.TicketScope{ProjectIDs: []string{"project-general"}}, []string{"t-b", "t-a", "t-same-day", "t-c"}},
		{"a readable project of another workspace does not leak in", "workspace-default", "2026-10-01", "2026-10-31",
			tickets.TicketScope{ProjectIDs: []string{"p-other"}}, []string{}},
		{"an empty readable set matches nothing", "workspace-default", "2026-10-01", "2026-10-31", tickets.TicketScope{}, []string{}},
		{"a single day", "workspace-default", "2026-10-05", "2026-10-05", all, []string{"t-a", "t-same-day"}},
		{"a range holding nothing", "workspace-default", "2027-01-01", "2027-01-31", all, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Tickets.ListDue(t.Context(), tt.workspace, tt.from, tt.to, tt.scope)
			require.NoError(t, err)
			assert.Equal(t, tt.want, dueIDs(got))
		})
	}

	t.Run("rows come back whole, with their due date and labels", func(t *testing.T) {
		got, err := s.Tickets.ListDue(t.Context(), "workspace-default", "2026-10-05", "2026-10-05", all)
		require.NoError(t, err)
		require.NotEmpty(t, got)
		assert.Equal(t, "2026-10-05", got[0].DueOn)
		assert.Equal(t, []string{"urgent"}, got[0].Labels)
	})
}

func TestTicketsRepo_Page_DueFilter(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedDueTickets(t, s)
	all := tickets.TicketScope{All: true}

	tests := []struct {
		name  string
		f     tickets.TicketFilter
		scope tickets.TicketScope
		want  []string
	}{
		{"a closed range, by due date then id", tickets.TicketFilter{DueFrom: "2026-10-01", DueTo: "2026-10-05"}, all, []string{"t-b", "t-a", "t-same-day"}},
		{"an open end after", tickets.TicketFilter{DueFrom: "2026-10-31"}, all, []string{"t-c", "t-d"}},
		{"an open end before", tickets.TicketFilter{DueTo: "2026-09-30"}, all, []string{"t-e"}},
		{"combines with a project", tickets.TicketFilter{ProjectID: "p-private", DueFrom: "2026-10-01"}, all, []string{"t-private"}},
		{"combines with access", tickets.TicketFilter{DueFrom: "2026-10-01", DueTo: "2026-10-10"}, tickets.TicketScope{ProjectIDs: []string{"project-general"}}, []string{"t-b", "t-a", "t-same-day"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, total, err := s.Tickets.Page(t.Context(), tt.f, tt.scope, paging.Window{Limit: 50})
			require.NoError(t, err)
			assert.Equal(t, tt.want, dueIDs(ts))
			assert.Equal(t, len(tt.want), total, "the total counts exactly the rows the filters keep")
		})
	}

	t.Run("pages keep the due order", func(t *testing.T) {
		first, total, err := s.Tickets.Page(t.Context(), tickets.TicketFilter{DueFrom: "2026-10-01", DueTo: "2026-10-31"}, all, paging.Window{Limit: 2})
		require.NoError(t, err)
		second, _, err := s.Tickets.Page(t.Context(), tickets.TicketFilter{DueFrom: "2026-10-01", DueTo: "2026-10-31"}, all, paging.Window{Limit: 2, Offset: 2})
		require.NoError(t, err)
		assert.Equal(t, 6, total, "Page reads every workspace the scope reaches")
		assert.Equal(t, []string{"t-b", "t-a", "t-same-day", "t-other"}, append(dueIDs(first), dueIDs(second)...))
	})
}

func TestTicketsRepo_ListDue_UsesTheDueIndex(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	rows, err := s.db.QueryContext(t.Context(), `EXPLAIN QUERY PLAN SELECT id FROM tickets WHERE due_on != '' AND due_on >= '2026-10-01' AND due_on <= '2026-10-31' ORDER BY due_on, id`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var plan string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notused, &detail))
		plan += detail + "\n"
	}
	require.NoError(t, rows.Err())
	assert.Contains(t, plan, "idx_tickets_due_on", "a range of due dates reads the partial index, not every ticket")
}
