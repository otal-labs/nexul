package main

import (
	"context"
	"encoding/json"
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

func paragraphBody(text string) string {
	return `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"` + text + `"}]}]}`
}

func dialCollab(t *testing.T, srv *httptest.Server, docID, mode string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, srv.URL+"/ws/collab/"+docID+"?mode="+mode, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })
	return conn
}

func readCollab(t *testing.T, conn *websocket.Conn) collab.ServerMsg {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	var m collab.ServerMsg
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func sendCollab(t *testing.T, conn *websocket.Conn, m collab.ClientMsg) {
	t.Helper()
	data, err := json.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, conn.Write(t.Context(), websocket.MessageText, data))
}

// TestCollabReset_ServerWriteWinsOverLiveSession saves a doc through the use-case while an editor holds a live
// session: a commit the editor sent before hearing of the write must not overwrite it, and a later joiner must start
// from the written body rather than replay the session's older state.
func TestCollabReset_ServerWriteWinsOverLiveSession(t *testing.T) {
	svc, store := newWired(t)
	seedMentionPeople(t, store)
	doc, err := svc.docsSvc.Create(as("u-onik"), "project-general", "Plan", paragraphBody("first draft"))
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle("GET /ws/collab/{docID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc.collabHub.ServeHTTP(w, r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: "u-onik"})))
	}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	editor := dialCollab(t, srv, doc.ID, "edit")
	readCollab(t, editor) // init
	viewer := dialCollab(t, srv, doc.ID, "view")
	readCollab(t, viewer) // init

	sendCollab(t, editor, collab.ClientMsg{Type: "hello", ClientID: 7})
	sendCollab(t, editor, collab.ClientMsg{Type: "update", Update: "dXBkYXRl"})
	relayed := readCollab(t, viewer)
	require.Equal(t, "update", relayed.Type)
	sendCollab(t, editor, collab.ClientMsg{Type: "commit", Update: "c25hcA==", BaseSeq: relayed.Seq, Body: paragraphBody("browser edit")})
	require.Equal(t, "commit", readCollab(t, viewer).Type)

	_, err = svc.docsSvc.Update(as("u-onik"), doc.ID, "Plan", paragraphBody("agent rewrite"))
	require.NoError(t, err)

	sendCollab(t, editor, collab.ClientMsg{Type: "commit", Update: "c3RhbGU=", BaseSeq: relayed.Seq, Body: paragraphBody("stale browser edit")})
	// The editor's frames are handled in order, so its presence relay proves the stale commit was handled first.
	sendCollab(t, editor, collab.ClientMsg{Type: "presence", ClientID: 7, Payload: "cA=="})
	var seen []string
	for frame := readCollab(t, viewer); frame.Type != "presence"; frame = readCollab(t, viewer) {
		seen = append(seen, frame.Type)
	}
	assert.Equal(t, []string{"reset"}, seen, "the viewer hears the reset and never a relay of the stale commit")

	saved, err := svc.docsSvc.Get(as("u-onik"), doc.ID)
	require.NoError(t, err)
	require.Contains(t, saved.Body, "agent rewrite")

	late := dialCollab(t, srv, doc.ID, "edit")
	init := readCollab(t, late)
	assert.Nil(t, init.Snapshot, "a joiner after the write must not replay the session's pre-write snapshot")
	assert.Empty(t, init.Updates)
	assert.Equal(t, "seed", readCollab(t, late).Type, "the first editor to join the cleared room seeds it from the body")

	sendCollab(t, late, collab.ClientMsg{Type: "commit", Update: "bmV3", BaseSeq: init.Seq, Body: paragraphBody("edit after the write")})
	for readCollab(t, viewer).Type != "commit" {
	}
	saved, err = svc.docsSvc.Get(as("u-onik"), doc.ID)
	require.NoError(t, err)
	assert.Contains(t, saved.Body, "edit after the write", "a commit based on the post-reset state still saves")
}
