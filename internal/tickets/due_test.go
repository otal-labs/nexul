package tickets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func (f *fakeRepo) UpdateDueOn(_ context.Context, id, dueOn string, _ []Change, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tickets[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	t.DueOn = dueOn
	f.events = append(f.events, evts...)
	return nil
}

// ListDue mirrors the real read: the workspace's tickets the scope keeps, due in the range, by due date then id.
func (f *fakeRepo) ListDue(_ context.Context, workspaceID, from, to string, scope TicketScope) ([]*Ticket, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Ticket
	for _, t := range f.tickets {
		ws := f.workspaces[t.ProjectID]
		if ws[0] == "" {
			ws = [2]string{"workspace-default", "default"}
		}
		if ws[0] != workspaceID || t.DueOn == "" || t.DueOn < from || t.DueOn > to {
			continue
		}
		if !scope.All && !slices.Contains(scope.ProjectIDs, t.ProjectID) {
			continue
		}
		out = append(out, t)
	}
	slices.SortFunc(out, func(a, b *Ticket) int {
		return strings.Compare(a.DueOn+a.ID, b.DueOn+b.ID)
	})
	return out, nil
}

// dueGate refuses a write in the projects it lists as readOnly and a workspace the caller is not in.
type dueGate struct {
	allowGate
	readOnly  []string
	readable  []string
	notMember bool
}

func (g dueGate) RequireProject(_ context.Context, projectID string, action permissions.Action) error {
	if action == permissions.TicketsWrite && slices.Contains(g.readOnly, projectID) {
		return apperrs.ErrForbidden
	}
	return nil
}

func (g dueGate) Require(context.Context, string, permissions.Action) error {
	if g.notMember {
		return apperrs.ErrNotFound
	}
	return nil
}

func (g dueGate) CallerProjectsIn(context.Context, string, permissions.Action) ([]string, bool, error) {
	return g.readable, false, nil
}

func TestSetDueOn(t *testing.T) {
	t.Run("rejects what is not a date", func(t *testing.T) {
		for _, bad := range []string{"tomorrow", "2026-1-5", "2026-02-30", "30/10/2026", "2026-10-30T10:00:00Z", "2026-13-01"} {
			s := newTestService(newFakeRepo())
			created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
			require.NoError(t, err)
			_, err = s.SetDueOn(t.Context(), created.ID, bad)
			require.ErrorIs(t, err, apperrs.ErrInvalid, bad)
			assert.Contains(t, err.Error(), "YYYY-MM-DD")
		}
	})
	t.Run("an empty id is invalid", func(t *testing.T) {
		_, err := newTestService(newFakeRepo()).SetDueOn(t.Context(), " ", "2026-10-30")
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("a missing ticket is not found", func(t *testing.T) {
		_, err := newTestService(newFakeRepo()).SetDueOn(t.Context(), "nope", "2026-10-30")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("a caller who cannot write the project is refused", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		s.SetGate(dueGate{readOnly: []string{"p-1"}})
		_, err = s.SetDueOn(t.Context(), created.ID, "2026-10-30")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Empty(t, repo.eventsFor(TopicDueDateChanged))
		assert.Empty(t, repo.tickets[created.ID].DueOn)
	})
	t.Run("sets, publishes, changes, and clears", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		ctx := identity.WithActor(t.Context(), identity.Actor{ID: "u-9"})

		steps := []struct{ set, previous string }{
			{" 2026-10-30 ", ""},
			{"2026-11-02", "2026-10-30"},
			{"", "2026-11-02"},
		}
		for i, step := range steps {
			got, err := s.SetDueOn(ctx, created.ID, step.set)
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(step.set), got.DueOn)
			evts := repo.eventsFor(TopicDueDateChanged)
			require.Len(t, evts, i+1)
			e, ok := evts[i].Payload.(DueDateChangedEvent)
			require.True(t, ok)
			assert.Equal(t, step.previous, e.PreviousDueOn)
			assert.Equal(t, strings.TrimSpace(step.set), e.Ticket.DueOn)
			assert.Equal(t, Actor{Kind: ActorKindUser, UserID: "u-9"}, e.Actor)
			assert.Equal(t, strings.TrimSpace(step.set), repo.tickets[created.ID].DueOn)
		}
	})
	t.Run("the same date writes nothing", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		_, err = s.SetDueOn(t.Context(), created.ID, "")
		require.NoError(t, err)
		_, err = s.SetDueOn(t.Context(), created.ID, "2026-10-30")
		require.NoError(t, err)
		_, err = s.SetDueOn(t.Context(), created.ID, "2026-10-30")
		require.NoError(t, err)
		assert.Len(t, repo.eventsFor(TopicDueDateChanged), 1)
	})
}

func TestDueBetween(t *testing.T) {
	newDue := func(t *testing.T, repo *fakeRepo, project, due string) *Ticket {
		t.Helper()
		created, err := newTestService(repo).Create(t.Context(), project, "ticket "+due, "", "", "")
		require.NoError(t, err)
		created.DueOn = due
		return created
	}
	tests := []struct {
		name         string
		workspace    string
		from, to     string
		wantInvalid  bool
		wantNotFound bool
		gate         Gate
	}{
		{name: "a missing workspace is invalid", from: "2026-10-01", to: "2026-10-31", wantInvalid: true},
		{name: "a missing from is invalid", workspace: "workspace-default", to: "2026-10-31", wantInvalid: true},
		{name: "a missing to is invalid", workspace: "workspace-default", from: "2026-10-01", wantInvalid: true},
		{name: "a bad date is invalid", workspace: "workspace-default", from: "2026-10-01", to: "31-10-2026", wantInvalid: true},
		{name: "a reversed range is invalid", workspace: "workspace-default", from: "2026-10-31", to: "2026-10-01", wantInvalid: true},
		{name: "63 days is too wide", workspace: "workspace-default", from: "2026-08-01", to: "2026-10-02", wantInvalid: true},
		{name: "a caller outside the workspace is refused", workspace: "workspace-default", from: "2026-10-01", to: "2026-10-31", gate: dueGate{notMember: true}, wantNotFound: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(newFakeRepo())
			if tt.gate != nil {
				s.SetGate(tt.gate)
			}
			_, err := s.DueBetween(t.Context(), tt.workspace, tt.from, tt.to)
			require.Error(t, err)
			assert.Equal(t, tt.wantInvalid, errors.Is(err, apperrs.ErrInvalid))
			assert.Equal(t, tt.wantNotFound, errors.Is(err, apperrs.ErrNotFound))
		})
	}

	t.Run("62 days inclusive is the widest range", func(t *testing.T) {
		_, err := newTestService(newFakeRepo()).DueBetween(t.Context(), "workspace-default", "2026-08-01", "2026-10-01")
		assert.NoError(t, err)
	})
	t.Run("keeps the range, inclusive, by due date then id, and leaves a project the caller cannot read out", func(t *testing.T) {
		repo := newFakeRepo()
		repo.workspaces = map[string][2]string{"p-other": {"workspace-other", "other"}}
		before := newDue(t, repo, "p-1", "2026-09-30")
		first := newDue(t, repo, "p-1", "2026-10-01")
		hidden := newDue(t, repo, "p-hidden", "2026-10-05")
		elsewhere := newDue(t, repo, "p-other", "2026-10-06")
		last := newDue(t, repo, "p-1", "2026-10-31")
		none := newDue(t, repo, "p-1", "")
		after := newDue(t, repo, "p-1", "2026-11-01")
		for _, tk := range []*Ticket{before, first, hidden, elsewhere, last, none, after} {
			repo.tickets[tk.ID].DueOn = tk.DueOn
		}
		s := newTestService(repo)
		s.SetGate(dueGate{readable: []string{"p-1", "p-other"}})

		got, err := s.DueBetween(t.Context(), "workspace-default", "2026-10-01", "2026-10-31")
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, []string{first.ID, last.ID}, []string{got[0].ID, got[1].ID})
	})
}

func TestPageTickets_DueFilter(t *testing.T) {
	for _, tt := range []struct{ name, from, to string }{
		{"a bad from", "soon", ""},
		{"a bad to", "", "later"},
		{"a reversed range", "2026-10-31", "2026-10-01"},
	} {
		t.Run(tt.name+" is invalid", func(t *testing.T) {
			_, _, err := newTestService(newFakeRepo()).PageTickets(t.Context(), TicketFilter{DueFrom: tt.from, DueTo: tt.to}, paging.Window{})
			assert.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
	t.Run("an open end is accepted", func(t *testing.T) {
		_, _, err := newTestService(newFakeRepo()).PageTickets(t.Context(), TicketFilter{DueFrom: " 2026-10-01 "}, paging.Window{})
		assert.NoError(t, err)
	})
}

func TestDueAdapters(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
	require.NoError(t, err)
	h := NewHandler(s).Routes()

	t.Run("PATCH sets and clears the due date", func(t *testing.T) {
		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/due-date", `{"due_on":"2026-10-30"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "2026-10-30", decodeTicket(t, rec).DueOn)

		rec = serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/due-date", `{"due_on":""}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "", decodeTicket(t, rec).DueOn)
		assert.Contains(t, rec.Body.String(), `"due_on":""`)
	})
	t.Run("PATCH refuses a bad date, a missing ticket, and a bad body", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/due-date", `{"due_on":"soon"}`).Code)
		assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodPatch, "/api/tickets/nope/due-date", `{"due_on":"2026-10-30"}`).Code)
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/due-date", `{"due_on":`).Code)
	})
	t.Run("GET due returns the items in the range", func(t *testing.T) {
		_, err := s.SetDueOn(t.Context(), created.ID, "2026-10-30")
		require.NoError(t, err)

		rec := serve(t, h, http.MethodGet, "/api/tickets/due?workspace_id=workspace-default&from=2026-10-01&to=2026-10-31", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Items []Ticket `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Items, 1)
		assert.Equal(t, created.ID, body.Items[0].ID)

		rec = serve(t, h, http.MethodGet, "/api/tickets/due?workspace_id=workspace-default&from=2026-11-01&to=2026-11-30", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"items":[]}`, rec.Body.String())
	})
	t.Run("GET due refuses a missing range", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodGet, "/api/tickets/due?workspace_id=workspace-default", "").Code)
	})
}
