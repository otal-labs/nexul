package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/collab"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// TestIntegration_LiveTicket edits one ticket's title and body live through the wired tickets hub.
func TestIntegration_LiveTicket(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	ticketID := f.ticket.ID
	mux := http.NewServeMux()
	mux.Handle("GET /ws/collab/tickets/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.ticketsHub.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: r.URL.Query().Get("as")})))
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	stored := func() (string, string) {
		got, err := s.ticketsSvc.Get(as(uWriter), ticketID)
		require.NoError(t, err)
		return got.Title, got.Body
	}
	title, _ := stored()

	t.Run("a reader joins to view only and an outsider not at all", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, dialTicketStatus(t, srv, ticketID, uReader, "edit"))
		assert.Equal(t, http.StatusForbidden, dialTicketStatus(t, srv, ticketID, uOutsider, "view"))
		viewer := dialTicket(t, srv, ticketID, uReader, "view")
		assert.Equal(t, "init", readCollab(t, viewer).Type)
	})

	editor := dialTicket(t, srv, ticketID, uWriter, "edit")
	init := readCollab(t, editor)
	require.Equal(t, "seed", readCollab(t, editor).Type, "the first editor loads the body into the empty room")
	sendCollab(t, editor, collab.ClientMsg{Type: "hello", ClientID: 7})

	t.Run("a commit without a title writes the body and keeps the title", func(t *testing.T) {
		sendCollab(t, editor, collab.ClientMsg{Type: "commit", Update: "c25hcA==", BaseSeq: init.Seq, Body: paragraphBody("typed live")})
		assert.Eventually(t, func() bool { _, body := stored(); return body == paragraphBody("typed live") }, 5*time.Second, 20*time.Millisecond)
		got, _ := stored()
		assert.Equal(t, title, got)
	})

	t.Run("the renaming client's commit carries the title", func(t *testing.T) {
		sendCollab(t, editor, collab.ClientMsg{Type: "commit", Update: "cmVuYW1l", BaseSeq: init.Seq, Title: "Renamed live", Body: paragraphBody("typed live")})
		assert.Eventually(t, func() bool { got, _ := stored(); return got == "Renamed live" }, 5*time.Second, 20*time.Millisecond)
	})

	t.Run("a body written from outside resets the room and drops the stale commit", func(t *testing.T) {
		_, err := s.ticketsSvc.UpdateTicket(as(uWriter), ticketID, "Renamed live", paragraphBody("from an agent"))
		require.NoError(t, err)
		for frame := readCollab(t, editor); frame.Type != "reset"; frame = readCollab(t, editor) {
		}
		sendCollab(t, editor, collab.ClientMsg{Type: "commit", Update: "c3RhbGU=", BaseSeq: init.Seq, Body: paragraphBody("stale edit")})
		sendCollab(t, editor, collab.ClientMsg{Type: "presence", ClientID: 7, Payload: "cA=="})
		time.Sleep(100 * time.Millisecond)
		_, body := stored()
		assert.Equal(t, paragraphBody("from an agent"), body)
	})
}

func ticketRoomURL(srv *httptest.Server, ticketID, user, mode string) string {
	return srv.URL + "/ws/collab/tickets/" + ticketID + "?mode=" + mode + "&as=" + user
}

func dialTicket(t *testing.T, srv *httptest.Server, ticketID, user, mode string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, ticketRoomURL(srv, ticketID, user, mode), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })
	return conn
}

func dialTicketStatus(t *testing.T, srv *httptest.Server, ticketID, user, mode string) int {
	t.Helper()
	_, resp, err := websocket.Dial(t.Context(), ticketRoomURL(srv, ticketID, user, mode), nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	return resp.StatusCode
}
