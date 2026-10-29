package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/collab"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/tickets"
)

func ticketHistoryRequest(t *testing.T, f permFixture, user, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/tickets/"+f.ticket.ID+path, nil).WithContext(as(user))
	rec := httptest.NewRecorder()
	tickets.NewHandler(f.svc.ticketsSvc).Routes().ServeHTTP(rec, req)
	return rec
}

func jsonKeys(t *testing.T, raw map[string]any) []string {
	t.Helper()
	out := make([]string, 0, len(raw))
	for k := range raw {
		out = append(out, k)
	}
	return out
}

// TestTicketHistoryRoutes_ReadersGetTheSharedShape_AndOnlyWritersRestore holds the three routes to the shape every
// history answers, and to the permission each needs: tickets:read to look, tickets:write to restore.
func TestTicketHistoryRoutes_ReadersGetTheSharedShape_AndOnlyWritersRestore(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	_, err := f.svc.ticketsSvc.UpdateTicket(as(uWriter), f.ticket.ID, f.ticket.Title, "second")
	require.NoError(t, err)
	_, err = f.svc.ticketsSvc.SetDueOn(as(uWriter), f.ticket.ID, "2026-10-30")
	require.NoError(t, err)

	rec := ticketHistoryRequest(t, f, uReader, http.MethodGet, "/history")
	require.Equal(t, http.StatusOK, rec.Code)
	var hist struct {
		Versions []map[string]any `json:"versions"`
		Changes  []map[string]any `json:"changes"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &hist))
	require.Len(t, hist.Versions, 2)
	for _, v := range hist.Versions {
		assert.ElementsMatch(t, []string{"version", "created_at", "name", "editor_ids", "via", "current", "words_added", "words_removed", "restored_from"}, jsonKeys(t, v), "no body")
		assert.Empty(t, v["name"], "tickets keep no named versions")
	}
	assert.EqualValues(t, 2, hist.Versions[0]["version"], "newest first")
	assert.Equal(t, true, hist.Versions[0]["current"])
	assert.Equal(t, false, hist.Versions[1]["current"])
	require.Len(t, hist.Changes, 2)
	for _, c := range hist.Changes {
		assert.ElementsMatch(t, []string{"id", "field", "from", "to", "actor", "created_at"}, jsonKeys(t, c))
	}
	assert.Equal(t, "due_on", hist.Changes[0]["field"], "newest first")
	assert.Equal(t, "created", hist.Changes[1]["field"])
	assert.Equal(t, map[string]any{"kind": "user", "user_id": uWriter}, hist.Changes[0]["actor"])

	rec = ticketHistoryRequest(t, f, uReader, http.MethodGet, "/versions/2")
	require.Equal(t, http.StatusOK, rec.Code)
	var version map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &version))
	assert.ElementsMatch(t, []string{"ticket_id", "version", "body", "editor_ids", "via", "created_at"}, jsonKeys(t, version))
	assert.Equal(t, "second", version["body"])
	assert.Equal(t, http.StatusBadRequest, ticketHistoryRequest(t, f, uReader, http.MethodGet, "/versions/0").Code)
	assert.Equal(t, http.StatusNotFound, ticketHistoryRequest(t, f, uReader, http.MethodGet, "/versions/99").Code)

	assert.Equal(t, http.StatusNotFound, ticketHistoryRequest(t, f, uOutsider, http.MethodGet, "/history").Code, "no access reads as no such ticket")
	assert.Equal(t, http.StatusForbidden, ticketHistoryRequest(t, f, uPlain, http.MethodGet, "/history").Code)
	assert.Equal(t, http.StatusForbidden, ticketHistoryRequest(t, f, uReader, http.MethodPost, "/versions/1/restore").Code, "a reader may look, not restore")
	got, err := f.svc.ticketsSvc.Get(as(uOwner), f.ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, "second", got.Body, "the refused restore changed nothing")

	rec = ticketHistoryRequest(t, f, uWriter, http.MethodPost, "/versions/1/restore")
	require.Equal(t, http.StatusOK, rec.Code)
	got, err = f.svc.ticketsSvc.Get(as(uOwner), f.ticket.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Body, "version 1 held the empty description the ticket was filed with")
}

// TestTicketRestore_SavesTheOpenSittingBeforeTheRestoreReplacesTheRoom files what an editor typed as its own version,
// below the version of the restore that then replaces it.
func TestTicketRestore_SavesTheOpenSittingBeforeTheRestoreReplacesTheRoom(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	s := f.svc
	ticket, err := s.ticketsSvc.Create(as(uWriter), "project-general", "Plan", paragraphBody("first draft"), "", "")
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle("GET /ws/collab/tickets/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.ticketsHub.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: r.URL.Query().Get("as")})))
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	editor := dialTicket(t, srv, ticket.ID, uWriter, "edit")
	init := readCollab(t, editor)
	require.Equal(t, "seed", readCollab(t, editor).Type)
	sendCollab(t, editor, collab.ClientMsg{Type: "update", Update: "dXBkYXRl"})
	sendCollab(t, editor, collab.ClientMsg{Type: "commit", Update: "c25hcA==", BaseSeq: init.Seq, Body: paragraphBody("browser edit")})
	require.Eventually(t, func() bool {
		got, err := s.ticketsSvc.Get(as(uWriter), ticket.ID)
		return err == nil && got.Body == paragraphBody("browser edit")
	}, 5*time.Second, 20*time.Millisecond)

	_, err = s.ticketsSvc.RestoreVersion(as(uWriter), ticket.ID, 1)
	require.NoError(t, err)

	hist, err := s.ticketsSvc.History(as(uWriter), ticket.ID)
	require.NoError(t, err)
	require.Len(t, hist.Versions, 3)
	assert.Equal(t, []string{uWriter}, hist.Versions[1].EditorIDs)
	sitting, err := s.ticketsSvc.GetVersion(as(uWriter), ticket.ID, hist.Versions[1].Version)
	require.NoError(t, err)
	assert.Equal(t, paragraphBody("browser edit"), sitting.Body, "the version below the restore holds what the editor typed")
	restored, err := s.ticketsSvc.GetVersion(as(uWriter), ticket.ID, hist.Versions[0].Version)
	require.NoError(t, err)
	assert.Equal(t, paragraphBody("first draft"), restored.Body)
}
