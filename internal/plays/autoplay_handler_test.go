package plays

import (
	"encoding/json"
	"net/http"
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

	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, path, `{"daily_cap_per_ticket":0}`, "editor").Code)
	assert.Equal(t, http.StatusBadRequest, do(t, routes, http.MethodPatch, path, `{`, "editor").Code)
	assert.Equal(t, http.StatusForbidden, do(t, routes, http.MethodGet, path, "", "player").Code)
}
