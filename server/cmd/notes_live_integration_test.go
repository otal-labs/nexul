package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/collab"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// TestIntegration_LiveNote edits one note's file live through the wired notes hub, from joining to deleting the note.
func TestIntegration_LiveNote(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	thread, err := f.store.Chat.GetTicketThread(context.Background(), f.ticket.ID)
	require.NoError(t, err)
	note, _, err := s.chatSvc.PostNote(as(uWriter), thread.ID, uWriter, "Left a handoff", chat.NoteFileInput{Name: "handoff", Markdown: "first draft"})
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle("GET /ws/collab/notes/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.notesHub.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: r.URL.Query().Get("as")})))
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	file := func() string {
		files, err := s.chatSvc.NoteFiles(context.Background(), []*chat.Message{note})
		require.NoError(t, err)
		return files[note.AttachmentID].Markdown
	}

	t.Run("someone who may not write the ticket neither joins the room nor replaces the file", func(t *testing.T) {
		for _, user := range []string{uReader, uPlain, uOutsider} {
			for _, mode := range []string{"edit", "view"} {
				assert.Equal(t, http.StatusForbidden, dialNoteStatus(t, srv, note.ID, user, mode), "%s joining to %s", user, mode)
			}
		}
		_, _, err := s.chatSvc.ReplaceNote(as(uReader), note.ID, uReader, "taken over")
		assert.Equal(t, forbidden, outcome(err))
		assert.Equal(t, http.StatusForbidden, putNote(t, s.chatSvc, uReader, note.ID, "taken over"))
		assert.Equal(t, "first draft", file())
	})

	t.Run("a replace needs a note and its markdown", func(t *testing.T) {
		plain, err := s.chatSvc.PostMessage(as(uWriter), thread.ID, uWriter, "just a reply")
		require.NoError(t, err)
		_, _, err = s.chatSvc.ReplaceNote(as(uWriter), plain.ID, uWriter, "now a note")
		assert.Equal(t, invalid, outcome(err), "a message without a file has nothing to replace")
		_, _, err = s.chatSvc.ReplaceNote(as(uWriter), note.ID, uWriter, "  ")
		assert.Equal(t, invalid, outcome(err))
		_, err = callChatToolErr(as(uWriter), t, s.chatSvc, "message_post", map[string]any{"body": "also this", "file": map[string]string{"note_id": note.ID, "markdown": "x"}})
		assert.Equal(t, invalid, outcome(err), "a replace leaves the message itself alone")
		assert.Equal(t, "first draft", file())
	})

	t.Run("opening the note without typing writes nothing", func(t *testing.T) {
		before := noteUpdates(t, f, note.ID)
		idle := dialNote(t, srv, note.ID, uWriter)
		readCollab(t, idle) // init
		require.Equal(t, "seed", readCollab(t, idle).Type, "the first joiner loads the file into the empty room")
		sendCollab(t, idle, collab.ClientMsg{Type: "hello", ClientID: 3})
		sendCollab(t, idle, collab.ClientMsg{Type: "update", Update: "c2VlZA=="})
		require.NoError(t, idle.Close(websocket.StatusNormalClosure, ""))
		assert.Equal(t, "first draft", file())
		assert.Equal(t, before, noteUpdates(t, f, note.ID), "no chat.message.updated for a visit")
	})

	alice := dialNote(t, srv, note.ID, uWriter)
	readCollab(t, alice) // init
	bob := dialNote(t, srv, note.ID, uOwner)
	readCollab(t, bob) // init
	sendCollab(t, alice, collab.ClientMsg{Type: "hello", ClientID: 7})

	t.Run("two editors see each other's changes and a commit writes the file as markdown", func(t *testing.T) {
		before := noteUpdates(t, f, note.ID)
		sendCollab(t, alice, collab.ClientMsg{Type: "update", Update: "dXBkYXRl"})
		relayed := readCollab(t, bob)
		require.Equal(t, "update", relayed.Type)
		sendCollab(t, alice, collab.ClientMsg{Type: "commit", Update: "c25hcA==", BaseSeq: relayed.Seq, Body: paragraphBody("second draft")})
		require.Equal(t, "commit", readCollab(t, bob).Type)
		assert.Equal(t, "second draft", strings.TrimSpace(file()))
		assert.Equal(t, before+1, noteUpdates(t, f, note.ID), "the commit publishes chat.message.updated")
	})

	t.Run("an agent's replace wins over a commit sent before it", func(t *testing.T) {
		sendCollab(t, alice, collab.ClientMsg{Type: "update", Update: "bWlk"})
		midEdit := readCollab(t, bob)
		require.Equal(t, "update", midEdit.Type)

		replaced := callChatTool(as(uWriter), t, s.chatSvc, "message_post", map[string]any{"file": map[string]string{"note_id": note.ID, "markdown": "agent rewrite"}})
		assert.Equal(t, "agent rewrite", replaced["file"].(map[string]any)["markdown"])
		assert.Equal(t, "reset", readCollab(t, alice).Type)

		sendCollab(t, alice, collab.ClientMsg{Type: "commit", Update: "c3RhbGU=", BaseSeq: midEdit.Seq, Body: paragraphBody("stale edit")})
		// Alice's frames are handled in order, so her presence relay proves the stale commit was handled first.
		sendCollab(t, alice, collab.ClientMsg{Type: "presence", ClientID: 7, Payload: "cA=="})
		var seen []string
		for frame := readCollab(t, bob); frame.Type != "presence"; frame = readCollab(t, bob) {
			seen = append(seen, frame.Type)
		}
		assert.Equal(t, []string{"reset"}, seen, "bob hears the reset and never a relay of the stale commit")
		assert.Equal(t, "agent rewrite", file())

		late := dialNote(t, srv, note.ID, uOwner)
		init := readCollab(t, late)
		assert.Nil(t, init.Snapshot, "the room's pre-replace state is gone")
		assert.Empty(t, init.Updates)
		assert.Equal(t, "seed", readCollab(t, late).Type, "the next editor reloads the room from the replaced file")

		assert.Equal(t, http.StatusOK, putNote(t, s.chatSvc, uOwner, note.ID, "browser replace"))
		assert.Equal(t, "reset", readCollab(t, late).Type, "the HTTP replace resets the room too")
		assert.Equal(t, "browser replace", file())
	})

	t.Run("a deleted note's room refuses commits and new joiners", func(t *testing.T) {
		carol := dialNote(t, srv, note.ID, uOwner)
		init := readCollab(t, carol)
		readCollab(t, carol) // seed
		sendCollab(t, carol, collab.ClientMsg{Type: "hello", ClientID: 9})
		require.NoError(t, s.chatSvc.DeleteMessage(as(uOwner), note.ID, uOwner))
		before := noteUpdates(t, f, note.ID)

		sendCollab(t, carol, collab.ClientMsg{Type: "commit", Update: "bGF0ZQ==", BaseSeq: init.Seq, Body: paragraphBody("after delete")})
		sendCollab(t, carol, collab.ClientMsg{Type: "presence", ClientID: 9, Payload: "cA=="})
		for frame := readCollab(t, bob); frame.Type != "presence"; frame = readCollab(t, bob) {
			assert.NotEqual(t, "commit", frame.Type, "a commit to a deleted note is never relayed")
		}
		assert.Equal(t, before, noteUpdates(t, f, note.ID))
		assert.Equal(t, http.StatusForbidden, dialNoteStatus(t, srv, note.ID, uOwner, "edit"))
		_, _, err := s.chatSvc.ReplaceNote(as(uOwner), note.ID, uOwner, "revived")
		assert.Equal(t, notFound, outcome(err))
	})
}

// callChatToolErr calls a chat MCP tool and returns its error, for the calls a client sees refused.
func callChatToolErr(ctx context.Context, t *testing.T, svc *chat.Service, name string, args map[string]any) (any, error) {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	for _, tool := range chat.MCPTools(svc) {
		if tool.Name == name {
			return tool.Call(ctx, raw)
		}
	}
	t.Fatalf("no chat tool %s", name)
	return nil, nil
}

func noteRoomURL(srv *httptest.Server, noteID, user, mode string) string {
	return srv.URL + "/ws/collab/notes/" + noteID + "?mode=" + mode + "&as=" + user
}

func dialNote(t *testing.T, srv *httptest.Server, noteID, user string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, noteRoomURL(srv, noteID, user, "edit"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })
	return conn
}

func dialNoteStatus(t *testing.T, srv *httptest.Server, noteID, user, mode string) int {
	t.Helper()
	_, resp, err := websocket.Dial(t.Context(), noteRoomURL(srv, noteID, user, mode), nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func putNote(t *testing.T, svc *chat.Service, user, noteID, markdown string) int {
	t.Helper()
	body, err := json.Marshal(map[string]string{"markdown": markdown})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/chat/messages/"+noteID+"/note", bytes.NewReader(body))
	chat.NewHandler(svc, nil).Routes().ServeHTTP(rec, req.WithContext(chat.WithUserID(as(user), user)))
	return rec.Code
}

// noteUpdates counts the chat.message.updated events waiting in the outbox for the note.
func noteUpdates(t *testing.T, f permFixture, noteID string) int {
	t.Helper()
	entries, err := f.store.Outbox.Unpublished(context.Background(), 500)
	require.NoError(t, err)
	n := 0
	for _, e := range entries {
		if e.Topic == chat.TopicMessageUpdated && strings.Contains(string(e.Payload), noteID) {
			n++
		}
	}
	return n
}
