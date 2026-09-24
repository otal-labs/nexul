package tickets

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeLinkSet(t *testing.T, code int, body []byte) LinkSet {
	t.Helper()
	require.Equal(t, http.StatusOK, code, string(body))
	var set LinkSet
	require.NoError(t, json.Unmarshal(body, &set))
	return set
}

func TestTicketsHandler_TicketLinks(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	ts := seedTickets(t, s, "frontend", "backend")
	h := NewHandler(s).Routes()
	a, b := ts[0].ID, ts[1].ID

	t.Run("error paths", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPost, "/api/tickets/"+a+"/blocked-by", `{`).Code)
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPost, "/api/tickets/"+a+"/blocked-by", `{"blocker_id":"`+a+`"}`).Code)
		assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodPost, "/api/tickets/"+a+"/blocked-by", `{"blocker_id":"nope"}`).Code)
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPut, "/api/tickets/"+a+"/found-in", `{`).Code)
		assert.Equal(t, http.StatusBadRequest, serve(t, h, http.MethodPut, "/api/tickets/"+a+"/found-in", `{}`).Code)
		assert.Equal(t, http.StatusNotFound, serve(t, h, http.MethodGet, "/api/tickets/nope/ticket-links", "").Code)
	})

	t.Run("blocked-by round trip with cycle refused", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/tickets/"+a+"/blocked-by", `{"blocker_id":"`+b+`"}`)
		set := decodeLinkSet(t, rec.Code, rec.Body.Bytes())
		assert.True(t, set.Blocked)

		rec = serve(t, h, http.MethodPost, "/api/tickets/"+b+"/blocked-by", `{"blocker_id":"`+a+`"}`)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "cycle")

		rec = serve(t, h, http.MethodGet, "/api/tickets/blockers", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var blockers map[string][]LinkedTicket
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &blockers))
		assert.Len(t, blockers[a], 1)

		rec = serve(t, h, http.MethodGet, "/api/tickets/"+b+"/ticket-links", "")
		assert.Len(t, decodeLinkSet(t, rec.Code, rec.Body.Bytes()).Blocks, 1)

		rec = serve(t, h, http.MethodDelete, "/api/tickets/"+a+"/blocked-by/"+b, "")
		assert.Empty(t, decodeLinkSet(t, rec.Code, rec.Body.Bytes()).BlockedBy)
	})

	t.Run("found-in round trip", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/tickets/"+a+"/found-in", `{"origin_unknown":true}`)
		assert.True(t, decodeLinkSet(t, rec.Code, rec.Body.Bytes()).OriginUnknown)

		rec = serve(t, h, http.MethodPut, "/api/tickets/"+a+"/found-in", `{"origin_id":"`+b+`"}`)
		set := decodeLinkSet(t, rec.Code, rec.Body.Bytes())
		require.NotNil(t, set.FoundIn)
		assert.Equal(t, b, set.FoundIn.ID)

		rec = serve(t, h, http.MethodDelete, "/api/tickets/"+a+"/found-in", "")
		assert.Nil(t, decodeLinkSet(t, rec.Code, rec.Body.Bytes()).FoundIn)
	})

	t.Run("uncleared blockers failure is 500", func(t *testing.T) {
		repo.ticketLinkErr = assert.AnError
		t.Cleanup(func() { repo.ticketLinkErr = nil })
		assert.Equal(t, http.StatusInternalServerError, serve(t, h, http.MethodGet, "/api/tickets/blockers", "").Code)
	})
}
