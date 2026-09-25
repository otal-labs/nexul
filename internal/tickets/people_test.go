package tickets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

type fakeUserLogins struct {
	logins map[string]string
	err    error
}

func (f fakeUserLogins) LoginForUserID(_ context.Context, userID string) (string, error) {
	return f.logins[userID], f.err
}

func personOf(t *Ticket, role Role) string {
	if role == RoleTester {
		return t.Tester
	}
	return t.Developer
}

func TestSetPerson(t *testing.T) {
	for _, role := range []Role{RoleDeveloper, RoleTester} {
		t.Run(string(role), func(t *testing.T) {
			t.Run("empty id is invalid", func(t *testing.T) {
				_, err := newTestService(newFakeRepo()).SetPerson(t.Context(), "", role, "onik97")
				assert.True(t, errors.Is(err, apperrs.ErrInvalid))
			})
			t.Run("missing ticket is not found", func(t *testing.T) {
				_, err := newTestService(newFakeRepo()).SetPerson(t.Context(), "nope", role, "onik97")
				assert.True(t, errors.Is(err, apperrs.ErrNotFound))
			})
			t.Run("sets, publishes, and clears", func(t *testing.T) {
				repo := newFakeRepo()
				s := newTestService(repo)
				created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
				require.NoError(t, err)

				got, err := s.SetPerson(t.Context(), created.ID, role, " onik97 ")
				require.NoError(t, err)
				assert.Equal(t, "onik97", personOf(got, role))
				evts := repo.eventsFor(personTopic(role))
				require.Len(t, evts, 1)
				e, ok := evts[0].Payload.(PersonChangedEvent)
				require.True(t, ok)
				assert.Equal(t, "", e.From)
				assert.Equal(t, "onik97", e.To)

				got, err = s.SetPerson(t.Context(), created.ID, role, "")
				require.NoError(t, err)
				assert.Equal(t, "", personOf(got, role))
				assert.Len(t, repo.eventsFor(personTopic(role)), 2)
			})
			t.Run("same login is a no-op without an event", func(t *testing.T) {
				repo := newFakeRepo()
				s := newTestService(repo)
				created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "onik97", CreateOptions{Tester: "onik97"})
				require.NoError(t, err)
				_, err = s.SetPerson(t.Context(), created.ID, role, "onik97")
				require.NoError(t, err)
				assert.Empty(t, repo.eventsFor(personTopic(role)))
			})
		})
	}
	t.Run("a developer change still publishes the deprecated ticket.assignee_changed", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "lena")
		require.NoError(t, err)
		_, err = s.SetPerson(t.Context(), created.ID, RoleDeveloper, "onik97")
		require.NoError(t, err)
		evts := repo.eventsFor(TopicAssigneeChanged)
		require.Len(t, evts, 1)
		e := evts[0].Payload.(AssigneeChangedEvent)
		assert.Equal(t, "lena", e.From)
		assert.Equal(t, "onik97", e.To)
		assert.Equal(t, "onik97", e.Ticket.Developer)

		_, err = s.SetPerson(t.Context(), created.ID, RoleTester, "lena")
		require.NoError(t, err)
		assert.Len(t, repo.eventsFor(TopicAssigneeChanged), 1, "a tester change is not an assignee change")
	})
	t.Run("unknown role is invalid", func(t *testing.T) {
		_, err := newTestService(newFakeRepo()).SetPerson(t.Context(), "t-1", Role("owner"), "onik97")
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("developer and tester are independent", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "onik97")
		require.NoError(t, err)
		got, err := s.SetPerson(t.Context(), created.ID, RoleTester, "lena")
		require.NoError(t, err)
		assert.Equal(t, "onik97", got.Developer)
		assert.Equal(t, "lena", got.Tester)
	})
}

func TestTicketJSON_KeepsDeprecatedAssignee(t *testing.T) {
	raw, err := json.Marshal(CreatedEvent{Ticket: Ticket{ID: "t-1", Developer: "onik97", Tester: "lena"}})
	require.NoError(t, err)
	var got struct {
		Ticket map[string]any `json:"ticket"`
	}
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "onik97", got.Ticket["assignee"])
	assert.Equal(t, "onik97", got.Ticket["developer"])
	assert.Equal(t, "lena", got.Ticket["tester"])
	assert.Equal(t, "t-1", got.Ticket["id"])
}

func TestCreate_Reporter(t *testing.T) {
	logins := fakeUserLogins{logins: map[string]string{"u-1": "onik97"}}
	automation := identity.Actor{ID: "u-9", Automation: &identity.AutomationRef{ID: "a-1", Name: "Triage"}}
	tests := []struct {
		name   string
		users  UserLogins
		actor  *identity.Actor
		viaMCP bool
		want   Reporter
	}{
		{"person through the gateway", logins, &identity.Actor{ID: "u-1"}, false, Reporter{Kind: ActorKindUser, Login: "onik97"}},
		{"Nexul for a person through MCP", logins, &identity.Actor{ID: "u-1"}, true, Reporter{Kind: ActorKindUserMCP, Login: "onik97"}},
		{"automation token", logins, &automation, false, Reporter{Kind: ActorKindAutomation, AutomationID: "a-1", AutomationName: "Triage"}},
		{"failed lookup keeps the user id", fakeUserLogins{err: errors.New("db down")}, &identity.Actor{ID: "u-1"}, false, Reporter{Kind: ActorKindUser, Login: "u-1"}},
		{"no lookup wired keeps the user id", nil, &identity.Actor{ID: "u-1"}, true, Reporter{Kind: ActorKindUserMCP, Login: "u-1"}},
		{"no actor records the kind only", logins, nil, false, Reporter{Kind: ActorKindUser}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			s := NewService(repo, fakeStatusStore{}, tt.users)
			ctx := t.Context()
			if tt.actor != nil {
				ctx = identity.WithActor(ctx, *tt.actor)
			}
			got, err := s.Create(ctx, "p-1", "ticket", "", "", "", CreateOptions{ViaMCP: tt.viaMCP})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Reporter)
			created := repo.eventsFor(TopicCreated)
			require.Len(t, created, 1)
			assert.Equal(t, tt.want, created[0].Payload.(CreatedEvent).Ticket.Reporter)
		})
	}
}

func TestPeopleAdapters(t *testing.T) {
	t.Run("HTTP create records the person and both roles", func(t *testing.T) {
		h := NewHandler(NewService(newFakeRepo(), fakeStatusStore{}, fakeUserLogins{logins: map[string]string{"u-1": "onik97"}})).Routes()
		req := `{"title":"Fix","project_id":"p-1","developer":"onik97","tester":"lena"}`
		rec := serveAs(t, h, http.MethodPost, "/api/tickets", req, identity.Actor{ID: "u-1"})
		require.Equal(t, http.StatusCreated, rec.Code)
		tk := decodeTicket(t, rec)
		assert.Equal(t, "onik97", tk.Developer)
		assert.Equal(t, "lena", tk.Tester)
		assert.Equal(t, Reporter{Kind: ActorKindUser, Login: "onik97"}, tk.Reporter)
	})
	t.Run("PATCH developer and tester routes", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(t.Context(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		h := NewHandler(s).Routes()

		rec := serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/developer", `{"login":"onik97"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "onik97", decodeTicket(t, rec).Developer)

		rec = serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/tester", `{"login":"lena"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "lena", decodeTicket(t, rec).Tester)

		assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodPatch, "/api/tickets/nope/tester", `{"login":"lena"}`).Code)
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPatch, "/api/tickets/"+created.ID+"/tester", `{"login":`).Code)
	})
}

func serveAs(t *testing.T, h http.Handler, method, path, body string, actor identity.Actor) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = req.WithContext(identity.WithActor(req.Context(), actor))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
