package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/chat"
)

// TestIntegration_Note walks a note through the wired services: who may leave one and where, what message_list and an
// agent turn read back, and that its file goes only with its message, taking the images it points at.
func TestIntegration_Note(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	thread, err := f.store.Chat.GetTicketThread(context.Background(), f.ticket.ID)
	require.NoError(t, err)
	handoff := chat.NoteFileInput{Name: "handoff", Markdown: "# Handoff\n\nLogin retries twice."}

	t.Run("only a ticket's thread takes a note, from someone who may write the ticket", func(t *testing.T) {
		for user, want := range map[string]string{uOwner: ok, uWriter: ok, uReader: forbidden, uPlain: forbidden, uOutsider: notFound} {
			_, _, err := s.chatSvc.PostNote(as(user), thread.ID, user, "Left a handoff", handoff)
			assert.Equal(t, want, outcome(err), "a note as %s", user)
		}
		_, _, err := s.chatSvc.PostNote(as(uOwner), f.channel.ID, uOwner, "Left a handoff", handoff)
		assert.Equal(t, invalid, outcome(err), "a channel takes no note")
	})

	t.Run("the HTTP post route leaves a note as the Agent", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{"body": "Left a handoff", "file": handoff})
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/chat/conversations/"+thread.ID+"/messages", bytes.NewReader(body))
		chat.NewHandler(s.chatSvc, nil).Routes().ServeHTTP(rec, req.WithContext(chat.WithUserID(as(uOwner), uOwner)))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var m chat.Message
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m))
		assert.Equal(t, chat.AuthorAgent, m.AuthorKind)
		assert.Equal(t, uOwner, m.AuthorID)
		assert.NotEmpty(t, m.AttachmentID)
	})

	shot := uploadPNG(t, s.attachmentsSvc, thread.ID, "shot.png")
	kept := uploadPNG(t, s.attachmentsSvc, thread.ID, "kept.png")
	elsewhere := uploadPNG(t, s.attachmentsSvc, f.channel.ID, "elsewhere.png")
	markdown := "Retry twice.\n\n![shot](/api/attachments/" + shot + ")\n![elsewhere](/api/attachments/" + elsewhere + ")"
	posted := callChatTool(as(uWriter), t, s.chatSvc, "message_post", map[string]any{
		"ticket_id": f.ticket.ID, "body": "Left a handoff", "file": map[string]string{"name": "retries.md", "markdown": markdown},
	})
	noteID, fileID := posted["id"].(string), posted["file"].(map[string]any)["id"].(string)

	t.Run("message_post leaves the note as the Agent on the caller's behalf, beside the ticket's own files", func(t *testing.T) {
		assert.Equal(t, "agent", posted["author_kind"])
		assert.Equal(t, uWriter, posted["author_id"])
		assert.Equal(t, "retries.md", posted["file"].(map[string]any)["name"])
		files, err := s.attachmentsSvc.List(as(uWriter), attachments.Owner{TicketID: f.ticket.ID})
		require.NoError(t, err)
		for _, a := range files {
			assert.NotEqual(t, fileID, a.ID, "a note's file never lists under the ticket's attachments")
		}
	})

	t.Run("message_list and an agent turn's history carry the note's markdown", func(t *testing.T) {
		listed := callChatTool(as(uReader), t, s.chatSvc, "message_list", map[string]any{"ticket_id": f.ticket.ID, "limit": 1})
		newest := listed["items"].([]any)[0].(map[string]any)
		assert.Equal(t, noteID, newest["id"])
		assert.Equal(t, markdown, newest["file"].(map[string]any)["markdown"])

		history, err := agentConversations{svc: s.chatSvc}.MessagesSince(context.Background(), thread.ID, time.Time{})
		require.NoError(t, err)
		require.NotEmpty(t, history)
		last := history[len(history)-1]
		require.NotNil(t, last.Note)
		assert.Equal(t, markdown, last.Note.Markdown)
	})

	t.Run("its file is never deleted alone", func(t *testing.T) {
		assert.Equal(t, invalid, outcome(s.attachmentsSvc.Delete(as(uWriter), fileID)))
		_, err := s.attachmentsSvc.Get(as(uWriter), fileID)
		assert.NoError(t, err)
	})

	t.Run("deleting the note takes tickets:write and removes its file and its own thread's images", func(t *testing.T) {
		assert.Equal(t, forbidden, outcome(s.chatSvc.DeleteMessage(as(uReader), noteID, uReader)))
		require.NoError(t, s.chatSvc.DeleteMessage(as(uOwner), noteID, uOwner), "the poster is not the only one")
		for id, want := range map[string]string{fileID: notFound, shot: notFound, kept: ok, elsewhere: ok} {
			_, err := s.attachmentsSvc.Get(as(uOwner), id)
			assert.Equal(t, want, outcome(err), id)
		}
	})
}

func uploadPNG(t *testing.T, svc *attachments.Service, conversationID, name string) string {
	t.Helper()
	a, err := svc.Upload(as(uWriter), attachments.Owner{ConversationID: conversationID}, name, []byte("\x89PNG\r\n\x1a\n"))
	require.NoError(t, err)
	return a.ID
}

// callChatTool calls a chat MCP tool and reads its result back as JSON, the shape a client sees.
func callChatTool(ctx context.Context, t *testing.T, svc *chat.Service, name string, args map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	for _, tool := range chat.MCPTools(svc) {
		if tool.Name != name {
			continue
		}
		out, err := tool.Call(ctx, raw)
		require.NoError(t, err)
		encoded, err := json.Marshal(out)
		require.NoError(t, err)
		var result map[string]any
		require.NoError(t, json.Unmarshal(encoded, &result))
		return result
	}
	t.Fatalf("no chat tool %s", name)
	return nil
}
