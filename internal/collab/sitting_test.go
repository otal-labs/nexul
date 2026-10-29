package collab

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func typeUpdate(s *session, c *client) {
	s.handleUpdate(context.Background(), c, ClientMsg{Type: msgUpdate, Update: "dXNlcg=="})
}

func newSittingRoom(w *fakeWriter, names ...string) (*session, []*client) {
	s := newTestSession(newFakeStore())
	s.writer = w
	clients := make([]*client, 0, len(names))
	for _, n := range names {
		c := newTestClient(n)
		s.join(c)
		clients = append(clients, c)
	}
	return s, clients
}

// TestSitting_IdleRoomWritesOneVersionCreditingEveryEditor guards the history from either filling with a version per
// update or losing a co-editor's credit.
func TestSitting_IdleRoomWritesOneVersionCreditingEveryEditor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &fakeWriter{}
		s, c := newSittingRoom(w, "alice", "bob")

		typeUpdate(s, c[0])
		typeUpdate(s, c[1])
		typeUpdate(s, c[0])
		time.Sleep(SittingIdle - time.Second)
		assert.Empty(t, w.sittingsSoFar(), "nothing is written while the room is still inside the idle window")

		time.Sleep(time.Second)
		synctest.Wait()
		assert.Equal(t, [][]string{{"alice", "bob"}}, w.sittingsSoFar(), "one version, editors in first-edit order")

		time.Sleep(10 * SittingIdle)
		synctest.Wait()
		assert.Len(t, w.sittingsSoFar(), 1, "the sitting is over, so a quiet room writes no second version")
	})
}

// TestSitting_EachUpdateRestartsTheIdleClock guards a timer that runs from the first update instead of the last,
// which would cut a long edit into several versions.
func TestSitting_EachUpdateRestartsTheIdleClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &fakeWriter{}
		s, c := newSittingRoom(w, "alice")

		typeUpdate(s, c[0])
		time.Sleep(SittingIdle - time.Minute)
		typeUpdate(s, c[0])
		time.Sleep(SittingIdle - time.Minute)
		synctest.Wait()
		assert.Empty(t, w.sittingsSoFar(), "four minutes after the last update, the sitting is still open")

		time.Sleep(time.Minute)
		synctest.Wait()
		assert.Len(t, w.sittingsSoFar(), 1, "five minutes after the last update it ends")
	})
}

// TestSitting_UpdatesTheRoomRefusedStartNoSitting guards a version credited to someone whose edit never landed.
func TestSitting_UpdatesTheRoomRefusedStartNoSitting(t *testing.T) {
	tests := []struct {
		name   string
		mode   Mode
		locked bool
	}{
		{"a viewer's update", ModeView, false},
		{"an update to a locked doc", ModeEdit, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := &fakeWriter{locked: tt.locked}
				s, c := newSittingRoom(w, "alice")
				c[0].mode = tt.mode

				typeUpdate(s, c[0])
				time.Sleep(2 * SittingIdle)
				synctest.Wait()
				s.leave(c[0])
				s.endSittingIfUnattended(context.Background())

				assert.Empty(t, w.sittingsSoFar())
			})
		})
	}
}

// TestSitting_AFailedWriteDoesNotStallTheRoom guards a sitting that keeps its editors after the write fails, which
// would credit them again on the next version.
func TestSitting_AFailedWriteDoesNotStallTheRoom(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &fakeWriter{failSitting: true}
		s, c := newSittingRoom(w, "alice", "bob")

		typeUpdate(s, c[0])
		time.Sleep(SittingIdle)
		synctest.Wait()
		assert.Empty(t, w.sittingsSoFar())

		w.mu.Lock()
		w.failSitting = false
		w.mu.Unlock()
		typeUpdate(s, c[1])
		time.Sleep(SittingIdle)
		synctest.Wait()
		assert.Equal(t, [][]string{{"bob"}}, w.sittingsSoFar(), "the next sitting starts clean")
	})
}

// TestSitting_APIWriteResetsTheRoomAfterTheSittingIsSaved guards an API save landing before what the editors typed,
// which would file their text above the save that replaced it.
func TestSitting_APIWriteResetsTheRoomAfterTheSittingIsSaved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &fakeWriter{}
		hub := NewHub(testLogger(), newFakeStore(), &fakeAccess{allowed: true}, w, permissions.DocsWrite, permissions.DocsRead)
		s := hub.session("doc-1")
		alice := newTestClient("alice")
		s.join(alice)
		typeUpdate(s, alice)

		err := hub.Reset(context.Background(), "doc-1", func(context.Context) error {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.order = append(w.order, "save")
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"sitting", "save"}, w.order)

		time.Sleep(2 * SittingIdle)
		synctest.Wait()
		assert.Len(t, w.sittingsSoFar(), 1, "the reset ended the sitting, so the idle timer writes nothing more")
	})
}

// TestSitting_EndsWhenTheLastWriterLeavesNotBefore guards a version cut while a co-editor is still typing, and one
// never cut because a viewer is still in the room.
func TestSitting_EndsWhenTheLastWriterLeavesNotBefore(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &fakeWriter{}
		s, c := newSittingRoom(w, "alice", "bob", "viewer")
		c[2].mode = ModeView

		typeUpdate(s, c[0])
		s.leave(c[0])
		s.endSittingIfUnattended(context.Background())
		assert.Empty(t, w.sittingsSoFar(), "bob can still edit")

		s.leave(c[1])
		s.endSittingIfUnattended(context.Background())
		assert.Equal(t, [][]string{{"alice"}}, w.sittingsSoFar(), "only a viewer remains")
	})
}

// TestHubConnectionEndingWritesTheSitting covers the wiring a session test cannot: a closed connection is what ends
// the sitting of the room's last writer.
func TestHubConnectionEndingWritesTheSitting(t *testing.T) {
	sat := make(chan []string, 4)
	w := &fakeWriter{sat: sat}
	hub := NewHub(testLogger(), newFakeStore(), &fakeAccess{allowed: true}, w, permissions.DocsWrite, permissions.DocsRead)
	srv := serveHub(t, hub, "alice")

	alice := dial(t, srv, "/ws/collab/doc-1?mode=edit")
	recv(t, alice) // init
	bob := dial(t, srv, "/ws/collab/doc-1?mode=view")
	recv(t, bob) // init
	sendJSON(t, alice, ClientMsg{Type: msgUpdate, Update: "dXNlcg=="})
	require.Equal(t, msgUpdate, recv(t, bob).Type, "the update is handled before alice leaves")

	require.NoError(t, alice.Close(websocket.StatusNormalClosure, ""))
	select {
	case got := <-sat:
		assert.Equal(t, []string{"alice"}, got)
	case <-time.After(5 * time.Second):
		t.Fatal("the last writer left and no sitting was written")
	}
}
