package collab

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeStore is an in-memory Store: per-doc update log with a shared seq counter, snapshot selection, and trimming, mirroring the storage repo's contract so session tests stay hermetic.
type fakeStore struct {
	mu         sync.Mutex
	seq        int64
	byDoc      map[string][]StoredUpdate
	failAppend bool
	loadErr    error
	trimErr    error
}

func newFakeStore() *fakeStore {
	return &fakeStore{byDoc: make(map[string][]StoredUpdate)}
}

func (f *fakeStore) AppendUpdate(_ context.Context, docID, actorID string, kind UpdateKind, payload string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failAppend {
		return 0, errors.New("disk full")
	}
	f.seq++
	row := StoredUpdate{Seq: f.seq, Kind: kind, ActorID: actorID, Payload: payload, CreatedAt: time.Now().UTC()}
	f.byDoc[docID] = append(f.byDoc[docID], row)
	return f.seq, nil
}

func (f *fakeStore) LoadReplay(_ context.Context, docID string) (Replay, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.loadErr != nil {
		return Replay{}, f.loadErr
	}
	rows := f.byDoc[docID]
	var replay Replay
	replay.Seq = f.seq
	for i := range rows {
		r := rows[i]
		if r.Kind == KindSnapshot {
			r := r
			replay.Snapshot = &r
			replay.Increments = nil
			continue
		}
		replay.Increments = append(replay.Increments, r)
	}
	return replay, nil
}

func (f *fakeStore) TrimUpdates(_ context.Context, docID string, baseSeq int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.trimErr != nil {
		return f.trimErr
	}
	rows := f.byDoc[docID]
	kept := rows[:0]
	for _, r := range rows {
		if r.Kind == KindUpdate && r.Seq <= baseSeq {
			continue
		}
		kept = append(kept, r)
	}
	f.byDoc[docID] = kept
	return nil
}

// fakeWriter records canonical-body commits.
type fakeWriter struct {
	mu    sync.Mutex
	calls []commitCall
	fail  bool
}

type commitCall struct {
	docID, title, body string
}

func (f *fakeWriter) CommitCollab(_ context.Context, docID, title, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("commit rejected")
	}
	f.calls = append(f.calls, commitCall{docID: docID, title: title, body: body})
	return nil
}

func (f *fakeWriter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

type fakeAccess struct {
	allowed bool
	err     error
}

func (f *fakeAccess) Can(context.Context, string, string, permissions.Action) (bool, error) {
	return f.allowed, f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestClient(id string) *client {
	return &client{id: id, send: make(chan []byte, sendBuffer), done: make(chan struct{})}
}

func newTestSession(store Store) *session {
	return &session{
		log:      testLogger(),
		docID:    "doc-1",
		store:    store,
		writer:   &fakeWriter{},
		clients:  make(map[*client]struct{}),
		presence: make(map[int]string),
	}
}

// drain reads one frame from a client's queue with a bound, so tests never hang.
func drain(t *testing.T, c *client) ServerMsg {
	t.Helper()
	select {
	case raw := <-c.send:
		var m ServerMsg
		require.NoError(t, json.Unmarshal(raw, &m))
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for frame")
		return ServerMsg{}
	}
}

func TestSessionUpdateRelaysToOthersOnly(t *testing.T) {
	s := newTestSession(newFakeStore())
	a := newTestClient("alice")
	b := newTestClient("bob")
	s.join(a)
	s.join(b)

	s.handleUpdate(context.Background(), a, ClientMsg{Type: msgUpdate, Update: "dXNlcg=="})

	got := drain(t, b)
	assert.Equal(t, msgUpdate, got.Type)
	assert.Equal(t, "alice", got.From)
	assert.Equal(t, "dXNlcg==", got.Payload)
	assert.Equal(t, int64(1), got.Seq)

	select {
	case raw := <-a.send:
		t.Fatalf("sender received its own relay: %s", raw)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSessionUpdatePersistenceFailureStillRelays(t *testing.T) {
	store := newFakeStore()
	store.failAppend = true
	s := newTestSession(store)
	a := newTestClient("alice")
	b := newTestClient("bob")
	s.join(a)
	s.join(b)

	s.handleUpdate(context.Background(), a, ClientMsg{Type: msgUpdate, Update: "dXNlcg=="})

	got := drain(t, b)
	assert.Equal(t, msgUpdate, got.Type)
	assert.Equal(t, int64(0), got.Seq)
}

func TestSessionCommitWritesBodyAndTrims(t *testing.T) {
	store := newFakeStore()
	writer := &fakeWriter{}
	s := newTestSession(store)
	s.writer = writer
	alice := newTestClient("alice")
	bob := newTestClient("bob")
	s.join(alice)
	s.join(bob)

	// bob's update lands at seq 1 and is relayed to alice, who applies it (baseSeq 1) then commits a full state that includes it.
	s.handleUpdate(context.Background(), bob, ClientMsg{Type: msgUpdate, Update: "Ym9iLXVwZGF0ZQ=="})
	relay := drain(t, alice)
	assert.Equal(t, msgUpdate, relay.Type)
	s.handleCommit(context.Background(), alice, ClientMsg{
		Type: msgCommit, Update: "YWxpY2Utc25hcHNob3Q=", BaseSeq: 1, Title: "T", Body: `{"type":"doc"}`,
	})

	require.Equal(t, 1, writer.count())
	call := writer.calls[0]
	assert.Equal(t, "doc-1", call.docID)
	assert.Equal(t, "T", call.title)
	assert.Equal(t, `{"type":"doc"}`, call.body)

	replay, err := store.LoadReplay(context.Background(), "doc-1")
	require.NoError(t, err)
	require.NotNil(t, replay.Snapshot)
	assert.Equal(t, KindSnapshot, replay.Snapshot.Kind)
	assert.Empty(t, replay.Increments, "trim must drop increments covered by the snapshot")

	got := drain(t, bob)
	assert.Equal(t, msgCommit, got.Type)
	assert.Equal(t, "alice", got.From)
	assert.Equal(t, int64(2), got.Seq)
}

func TestSessionCommitDedupesIdenticalBody(t *testing.T) {
	writer := &fakeWriter{}
	s := newTestSession(newFakeStore())
	s.writer = writer
	c := newTestClient("alice")
	s.join(c)

	for range 3 {
		s.handleCommit(context.Background(), c, ClientMsg{
			Type: msgCommit, Update: "YWxpY2Utc25hcHNob3Q=", BaseSeq: 0, Title: "T", Body: `{"type":"doc"}`,
		})
	}
	require.Equal(t, 1, writer.count(), "identical commits must write the canonical body once")
}

func TestSessionCommitWriterFailureKeepsSessionAlive(t *testing.T) {
	writer := &fakeWriter{fail: true}
	s := newTestSession(newFakeStore())
	s.writer = writer
	c := newTestClient("alice")
	b := newTestClient("bob")
	s.join(c)
	s.join(b)

	s.handleCommit(context.Background(), c, ClientMsg{
		Type: msgCommit, Update: "YWxpY2Utc25hcHNob3Q=", BaseSeq: 0, Title: "T", Body: `{"type":"doc"}`,
	})
	got := drain(t, b)
	assert.Equal(t, msgCommit, got.Type)
}

func TestSessionPresenceRelaysAndJoinerGetsRoster(t *testing.T) {
	s := newTestSession(newFakeStore())
	alice := newTestClient("alice")
	bob := newTestClient("bob")
	s.join(alice)
	s.join(bob)
	alice.clientID = 11

	s.handlePresence(alice, ClientMsg{Type: msgPresence, ClientID: 11, Payload: "YXdhcmVuZXNz"})

	got := drain(t, bob)
	assert.Equal(t, msgPresence, got.Type)
	assert.Equal(t, 11, got.ClientID)
	assert.Equal(t, "alice", got.From)
	assert.Equal(t, "YXdhcmVuZXNz", got.Payload)

	carol := newTestClient("carol")
	s.join(carol)
	require.NoError(t, s.sendInit(context.Background(), carol))
	init := drain(t, carol)
	require.Len(t, init.Presence, 1)
	assert.Equal(t, 11, init.Presence[0].ClientID)
}

func TestSessionLeaveBroadcastsAndClearsPresence(t *testing.T) {
	s := newTestSession(newFakeStore())
	alice := newTestClient("alice")
	bob := newTestClient("bob")
	s.join(alice)
	s.join(bob)
	alice.clientID = 11
	s.handlePresence(alice, ClientMsg{Type: msgPresence, ClientID: 11, Payload: "YXdhcmVuZXNz"})
	drain(t, bob)

	s.leave(alice)
	got := drain(t, bob)
	assert.Equal(t, msgLeave, got.Type)
	assert.Equal(t, 11, got.ClientID)
	assert.False(t, s.empty())

	s.leave(bob)
	assert.True(t, s.empty())
}

func TestSessionInitReplayShape(t *testing.T) {
	store := newFakeStore()
	ctx := context.Background()
	_, _ = store.AppendUpdate(ctx, "doc-1", "bob", KindUpdate, "dXA9MQ==")
	_, _ = store.AppendUpdate(ctx, "doc-1", "alice", KindSnapshot, "c25hcA==")
	_, _ = store.AppendUpdate(ctx, "doc-1", "carol", KindUpdate, "dXA9Mg==")

	s := newTestSession(store)
	c := newTestClient("dave")
	s.join(c)
	require.NoError(t, s.sendInit(ctx, c))
	init := drain(t, c)

	require.NotNil(t, init.Snapshot)
	assert.Equal(t, int64(2), init.Snapshot.Seq)
	assert.Equal(t, KindSnapshot, init.Snapshot.Kind)
	require.Len(t, init.Updates, 1)
	assert.Equal(t, "dXA9Mg==", init.Updates[0].Payload)
	assert.Equal(t, int64(3), init.Seq)
}

func TestSendToDeadPeerErrors(t *testing.T) {
	s := newTestSession(newFakeStore())
	c := newTestClient("alice")
	s.join(c)
	close(c.done)

	err := s.sendInit(context.Background(), c)
	require.ErrorIs(t, err, errPeerGone)
}

func TestSessionInitStoreErrorPropagates(t *testing.T) {
	store := newFakeStore()
	store.loadErr = errors.New("db down")
	s := newTestSession(store)
	c := newTestClient("alice")
	s.join(c)

	err := s.sendInit(context.Background(), c)
	require.ErrorIs(t, err, store.loadErr)
}

func TestSessionCommitPersistFailureSkipsWriteAndRelay(t *testing.T) {
	store := newFakeStore()
	store.failAppend = true
	writer := &fakeWriter{}
	s := newTestSession(store)
	s.writer = writer
	c := newTestClient("alice")
	b := newTestClient("bob")
	s.join(c)
	s.join(b)

	s.handleCommit(context.Background(), c, ClientMsg{
		Type: msgCommit, Update: "YWxpY2Utc25hcHNob3Q=", BaseSeq: 0, Title: "T", Body: `{"type":"doc"}`,
	})
	assert.Zero(t, writer.count(), "a failed snapshot must not reach the canonical doc")
	select {
	case raw := <-b.send:
		t.Fatalf("no commit relay after a failed snapshot: %s", raw)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSessionCommitTrimFailureStillWritesBody(t *testing.T) {
	store := newFakeStore()
	store.trimErr = errors.New("db down")
	writer := &fakeWriter{}
	s := newTestSession(store)
	s.writer = writer
	c := newTestClient("alice")
	s.join(c)

	s.handleCommit(context.Background(), c, ClientMsg{
		Type: msgCommit, Update: "YWxpY2Utc25hcHNob3Q=", BaseSeq: 1, Title: "T", Body: `{"type":"doc"}`,
	})
	assert.Equal(t, 1, writer.count(), "a failed trim must not block the canonical write")
}

func TestSessionBroadcastDropsStuckPeerAndLeaveIsIdempotent(t *testing.T) {
	s := newTestSession(newFakeStore())
	a := newTestClient("alice")
	b := newTestClient("bob")
	s.join(a)
	s.join(b)

	// Fill bob's queue so the broadcast drop path fires.
	for range sendBuffer {
		b.send <- []byte(`{"type":"commit","from":"alice"}`)
	}
	s.broadcast(a, ServerMsg{Type: msgCommit, From: "alice", Seq: 1})

	got := drain(t, a)
	assert.Equal(t, msgLeave, got.Type)
	assert.Equal(t, 0, got.ClientID, "bob never announced a client id")
	assert.False(t, s.empty())

	// The drop marked bob closed: a second leave is a no-op (no extra frame).
	s.leave(b)
	select {
	case raw := <-a.send:
		t.Fatalf("duplicate leave after closed guard: %s", raw)
	case <-time.After(50 * time.Millisecond):
	}
}
