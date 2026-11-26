package plays

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_AutoPlaysRoundTrip(t *testing.T) {
	s, _, ps := autoPlayFixture(t)
	routes := NewHandler(s).Routes()
	base := "/api/workspaces/" + workspaceID + "/plays/" + ps[TypeTicket].ID + "/auto-plays"

	rec := do(t, routes, http.MethodPost, base, `{"enabled":true,"moment":"ticket.created"}`, "editor")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created AutoPlay
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.False(t, created.Enabled, "a create starts off whatever enabled says")

	rec = do(t, routes, http.MethodPatch, base+"/"+created.ID, `{"enabled":true,"moment":"ticket.unblocked"}`, "editor")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = do(t, routes, http.MethodGet, base, "", "editor")
	require.Equal(t, http.StatusOK, rec.Code)
	var list []AutoPlay
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list, 1)
	assert.True(t, list[0].Enabled)
	assert.Equal(t, MomentTicketUnblocked, list[0].Moment)

	assert.Equal(t, http.StatusForbidden, do(t, routes, http.MethodDelete, base+"/"+created.ID, "", "writer").Code)
	assert.Equal(t, http.StatusNoContent, do(t, routes, http.MethodDelete, base+"/"+created.ID, "", "editor").Code)
	assert.Equal(t, http.StatusNotFound, do(t, routes, http.MethodPatch, base+"/"+created.ID, `{"moment":"ticket.created"}`, "editor").Code)
	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPost, base, `{`, "editor").Code)
	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, base+"/x", `{`, "editor").Code)
	assert.Equal(t, http.StatusForbidden, do(t, routes, http.MethodGet, base, "", "player").Code)
}

func TestHandler_AutoPlayLimits(t *testing.T) {
	s, _, _ := autoPlayFixture(t)
	routes := NewHandler(s).Routes()
	path := "/api/workspaces/" + workspaceID + "/plays/auto-play-limits"

	rec := do(t, routes, http.MethodGet, path, "", "editor")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"daily_cap_per_ticket":5}`, rec.Body.String())

	rec = do(t, routes, http.MethodPatch, path, `{"daily_cap_per_ticket":8}`, "editor")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"daily_cap_per_ticket":8}`, rec.Body.String())

	rec = do(t, routes, http.MethodPatch, path, `{"runs_at_once_per_person":3}`, "editor")
	assert.Equal(t, http.StatusBadRequest, rec.Code, "runs at once is set on each computer now")

	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, path, `{"daily_cap_per_ticket":0}`, "editor").Code)
	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, path, `{`, "editor").Code)
	assert.Equal(t, http.StatusForbidden, do(t, routes, http.MethodGet, path, "", "player").Code)
}

func TestHandler_AutoPlayRunsAsAPersonAndRunAsMePicksTheCaller(t *testing.T) {
	s, repo, ps := personFixture(t)
	routes := NewHandler(s).Routes()
	base := "/api/workspaces/" + workspaceID + "/plays/" + ps[TypeStack].ID + "/auto-plays"
	read := func(rec *httptest.ResponseRecorder) AutoPlay {
		t.Helper()
		require.Less(t, rec.Code, 300, rec.Body.String())
		var a AutoPlay
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &a))
		return a
	}

	rec := do(t, routes, http.MethodPost, base, `{"moment":"stack.service_crashed"}`, "editor")
	created := read(rec)
	assert.Equal(t, []any{RunOnPerson, "editor"}, []any{created.RunOn, created.RunAs}, "a new one runs as its creator")
	assert.Contains(t, rec.Body.String(), `"computer_ids":[]`, "the retired list stays empty, never null")
	path := base + "/" + created.ID
	repo.autoPlays[0].RunAs = "someone-else"

	older := read(do(t, routes, http.MethodPost, base, `{"moment":"stack.service_crashed","run_on":"computer"}`, "editor"))
	assert.Equal(t, RunOnPerson, older.RunOn, "run_on computer reads as person")

	kept := read(do(t, routes, http.MethodPatch, path, `{"enabled":true,"moment":"stack.service_recovered"}`, "editor"))
	assert.Equal(t, "someone-else", kept.RunAs, "no run_as_me keeps the person")

	mine := read(do(t, routes, http.MethodPatch, path, `{"enabled":true,"moment":"stack.service_recovered","run_as_me":true}`, "editor"))
	assert.Equal(t, "editor", mine.RunAs, "run_as_me makes the caller the person")

	ticketBase := "/api/workspaces/" + workspaceID + "/plays/" + ps[TypeTicket].ID + "/auto-plays"
	ticket := read(do(t, routes, http.MethodPost, ticketBase, `{"moment":"ticket.created"}`, "editor"))
	tpath := ticketBase + "/" + ticket.ID
	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, tpath, `{"enabled":true,"moment":"ticket.created","run_on":"person"}`, "editor").Code,
		"switching on needs a person picked")
	assert.Equal(t, http.StatusOK, do(t, routes, http.MethodPatch, tpath, `{"enabled":true,"moment":"ticket.created","run_on":"person","run_as_me":true}`, "editor").Code)
}

func TestHandler_ListCountsTheAutoPlaysSwitchedOn(t *testing.T) {
	s, _, ps := autoPlayFixture(t)
	routes := NewHandler(s).Routes()
	path := "/api/workspaces/" + workspaceID + "/plays"
	ticket := ps[TypeTicket].ID
	for _, enabled := range []bool{true, true, false} {
		a, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ticket, AutoPlayInput{Moment: MomentTicketCreated})
		require.NoError(t, err)
		_, err = s.UpdateAutoPlay(ctxAs("editor"), workspaceID, ticket, a.ID, enabled, AutoPlayInput{Moment: MomentTicketCreated})
		require.NoError(t, err)
	}

	tests := []struct {
		name string
		user string
		want map[string]*int
	}{
		{"a reader of auto plays gets the enabled ones counted, 0 for a play with none", "editor", map[string]*int{ticket: ptr(2), ps[TypeDoc].ID: ptr(0)}},
		{"a caller without autoplays:read gets no field", "player", map[string]*int{ticket: nil, ps[TypeDoc].ID: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, routes, http.MethodGet, path, "", tt.user)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var rows []struct {
				ID          string `json:"id"`
				AutoPlaysOn *int   `json:"auto_plays_on"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rows))
			got := map[string]*int{}
			for _, row := range rows {
				got[row.ID] = row.AutoPlaysOn
			}
			for id, want := range tt.want {
				assert.Equal(t, want, got[id], id)
			}
		})
	}

	raw := do(t, routes, http.MethodGet, path, "", "player").Body.String()
	assert.NotContains(t, raw, "auto_plays_on", "the key is left out, not null")
}

func ptr(n int) *int { return &n }
