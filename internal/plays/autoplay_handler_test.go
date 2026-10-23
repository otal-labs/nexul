package plays

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestHandler_StackAutoPlayPicksAComputerAsTheCaller(t *testing.T) {
	s, _, ps := computerFixture(t)
	routes := NewHandler(s).Routes()
	base := "/api/workspaces/" + workspaceID + "/plays/" + ps[TypeStack].ID + "/auto-plays"

	rec := do(t, routes, http.MethodPost, base, `{"moment":"stack.service_crashed","computer_id":"c-own"}`, "editor")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created AutoPlay
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, []any{RunOnComputer, []string{"c-own"}, "c-own", "editor"}, []any{created.RunOn, created.ComputerIDs, created.ComputerID, created.RunAs}, "the older computer_id is a list of one")

	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, base+"/"+created.ID, `{"enabled":true,"moment":"stack.service_crashed","computer_id":"c-see"}`, "editor").Code)
	unpicked := do(t, routes, http.MethodPost, base, `{"moment":"stack.service_crashed"}`, "editor")
	require.Equal(t, http.StatusCreated, unpicked.Code, unpicked.Body.String())
	var waiting AutoPlay
	require.NoError(t, json.Unmarshal(unpicked.Body.Bytes(), &waiting))
	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, base+"/"+waiting.ID, `{"enabled":true,"moment":"stack.service_crashed"}`, "editor").Code,
		"switching on needs the computers picked")
	rec = do(t, routes, http.MethodPatch, base+"/"+created.ID, `{"enabled":true,"moment":"stack.service_crashed","computer_id":"c-run"}`, "editor")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestHandler_StackAutoPlayKeepsOrderedComputersUnlessAPickIsSent(t *testing.T) {
	s, _, ps := computerFixture(t)
	routes := NewHandler(s).Routes()
	base := "/api/workspaces/" + workspaceID + "/plays/" + ps[TypeStack].ID + "/auto-plays"
	read := func(rec *httptest.ResponseRecorder) AutoPlay {
		t.Helper()
		require.Less(t, rec.Code, 300, rec.Body.String())
		var a AutoPlay
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &a))
		return a
	}

	rec := do(t, routes, http.MethodPost, base, `{"moment":"stack.service_crashed","computer_ids":["c-run","c-own","c-run"]}`, "editor")
	created := read(rec)
	assert.Equal(t, []any{[]string{"c-run", "c-own"}, "c-run", "editor"}, []any{created.ComputerIDs, created.ComputerID, created.RunAs}, "in the order picked, once each")
	assert.Contains(t, rec.Body.String(), `"computer_ids":["c-run","c-own"]`)
	path := base + "/" + created.ID

	kept := read(do(t, routes, http.MethodPatch, path, `{"enabled":true,"moment":"stack.service_recovered"}`, "editor"))
	assert.Equal(t, []any{[]string{"c-run", "c-own"}, "editor"}, []any{kept.ComputerIDs, kept.RunAs}, "no computer_ids key keeps the list and who picked it")

	yours := read(do(t, routes, http.MethodPatch, path, `{"enabled":true,"moment":"stack.service_recovered","computer_ids":[]}`, "editor"))
	assert.Equal(t, []any{[]string{}, "", "editor"}, []any{yours.ComputerIDs, yours.ComputerID, yours.RunAs}, "an empty list is a pick of the caller's own order")
	assert.Contains(t, do(t, routes, http.MethodGet, base, "", "editor").Body.String(), `"computer_ids":[]`, "never null")

	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, path, `{"enabled":true,"moment":"stack.service_recovered","computer_ids":["c-own","c-see"]}`, "editor").Code,
		"one computer in the list the caller may not run agents on refuses the pick")
	tooMany := do(t, routes, http.MethodPatch, path, `{"moment":"stack.service_recovered","computer_ids":["`+strings.Join(manyComputers(MaxAutoPlayComputers+1), `","`)+`"]}`, "editor")
	assert.Equal(t, http.StatusBadRequest, tooMany.Code)
	assert.Contains(t, tooMany.Body.String(), "at most 20 computers", "the list is bounded before any right is read")
}

func manyComputers(n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, "c-"+strconv.Itoa(i))
	}
	return out
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
