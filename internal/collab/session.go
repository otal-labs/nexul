package collab

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
)

// errPeerGone reports a send to a client whose write pump already exited.
var errPeerGone = errors.New("collab peer gone")

// client is one joined connection; the write pump drains send and closes done on exit.
type client struct {
	id       string // actor id (auth user)
	mode     Mode
	clientID int // y client id announced in hello (presence routing)
	send     chan []byte
	done     chan struct{} // closed by the pump when the peer is gone
	hangUp   func()        // closes the socket so the read loop ends and the peer reconnects
	closed   bool
	stale    bool // joined before a reset, so its writes describe state the room no longer holds
}

// session is one document's live collaboration room; a persistence failure logs but still relays.
type session struct {
	log    *slog.Logger
	docID  string
	store  Store
	writer DocWriter

	// writeMu fences each stored write against Hub.Reset, so none lands between a server-side write and the reset.
	writeMu sync.Mutex

	mu        sync.Mutex
	clients   map[*client]struct{}
	presence  map[int]string // y client id → base64 awareness state
	seq       int64          // latest seq persisted for this doc
	lastBody  string         // last canonical body committed (write dedupe)
	lastTitle string         // last canonical title committed (write dedupe)
	resetSeq  int64          // commits whose base predates the last reset are dropped
	hasState  bool           // the room holds stored or relayed state, so nobody seeds it
	seeder    *client        // the one joiner told to seed the empty room
}

func (s *session) join(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = struct{}{}
}

// sendInit pushes the stored replay to the joined client: snapshot, increments, seq, and presence roster.
func (s *session) sendInit(ctx context.Context, c *client) error {
	replay, err := s.store.LoadReplay(ctx, s.docID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.seq = replay.Seq
	if replay.Snapshot != nil || len(replay.Increments) > 0 {
		s.hasState = true
	}
	seed := s.claimSeed(c)
	presence := make([]PresenceMsg, 0, len(s.presence))
	for id, p := range s.presence {
		presence = append(presence, PresenceMsg{ClientID: id, Payload: p})
	}
	s.mu.Unlock()
	if err := s.send(c, ServerMsg{Type: msgInit, Seq: replay.Seq, Snapshot: replay.Snapshot, Updates: replay.Increments, Presence: presence}); err != nil {
		return err
	}
	if !seed {
		return nil
	}
	return s.send(c, ServerMsg{Type: msgSeed})
}

// claimSeed makes c the empty room's one seeder, since two seeding it would show the body twice; callers hold s.mu.
func (s *session) claimSeed(c *client) bool {
	if s.hasState || s.seeder != nil || c.mode != ModeEdit || c.stale || c.closed {
		return false
	}
	s.seeder = c
	return true
}

// writable refuses a viewer's writes, a dropped peer's, and any write to a locked doc; a failed lock lookup refuses too.
func (s *session) writable(ctx context.Context, c *client) bool {
	if c.mode != ModeEdit {
		s.log.Warn("collab: dropped a write from a viewer", "doc", s.docID, "user", c.id)
		return false
	}
	s.mu.Lock()
	_, member := s.clients[c]
	stale := c.stale
	s.mu.Unlock()
	if !member {
		s.log.Info("collab: dropped a write from a peer no longer in the room", "doc", s.docID, "user", c.id)
		return false
	}
	if stale {
		s.log.Info("collab: dropped a write made before a reset", "doc", s.docID, "user", c.id)
		return false
	}
	// ponytail: one doc read per relayed update; cache the flag in the session if it ever shows in a profile.
	locked, err := s.writer.Locked(ctx, s.docID)
	if err != nil || locked {
		s.log.Info("collab: dropped a write to a locked doc", "doc", s.docID, "user", c.id, "error", err)
		return false
	}
	return true
}

// handleUpdate persists and relays one Y.js update; a persistence failure logs but still relays.
func (s *session) handleUpdate(ctx context.Context, c *client, m ClientMsg) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if !s.writable(ctx, c) {
		return
	}
	var seq int64
	stored, err := s.store.AppendUpdate(ctx, s.docID, c.id, KindUpdate, m.Update)
	if err != nil {
		s.log.Warn("collab: persist update failed; relaying anyway", "doc", s.docID, "error", err)
	}
	s.mu.Lock()
	s.hasState = true
	if err == nil {
		seq = stored
		if seq > s.seq {
			s.seq = seq
		}
	}
	s.mu.Unlock()
	s.broadcast(c, ServerMsg{Type: msgUpdate, From: c.id, Seq: seq, Payload: m.Update})
}

// handleCommit persists a snapshot, trims already-applied increments, writes the canonical body, and broadcasts.
func (s *session) handleCommit(ctx context.Context, c *client, m ClientMsg) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if !s.writable(ctx, c) {
		return
	}
	s.mu.Lock()
	predatesReset := m.BaseSeq < s.resetSeq
	s.mu.Unlock()
	if predatesReset {
		s.log.Info("collab: dropped a commit based on state from before a reset", "doc", s.docID, "user", c.id)
		return
	}
	seq, err := s.store.AppendUpdate(ctx, s.docID, c.id, KindSnapshot, m.Update)
	if err != nil {
		s.log.Warn("collab: persist snapshot failed", "doc", s.docID, "error", err)
		return
	}
	if err := s.store.TrimUpdates(ctx, s.docID, m.BaseSeq); err != nil {
		s.log.Warn("collab: trim updates failed", "doc", s.docID, "error", err)
	}
	s.mu.Lock()
	s.hasState = true
	if seq > s.seq {
		s.seq = seq
	}
	// Write only on an actual title/body change; a peer's stale mount-time title would otherwise clobber a rename.
	titleChanged := m.Title != "" && m.Title != s.lastTitle
	skip := m.Body == "" || (m.Body == s.lastBody && !titleChanged)
	if !skip {
		s.lastBody = m.Body
		if titleChanged {
			s.lastTitle = m.Title
		}
	}
	s.mu.Unlock()
	if !skip {
		s.writeBody(ctx, m.Title, m.Body)
	}
	// Broadcast carries the rename since peers hold the title in plain component state, not the CRDT.
	out := ServerMsg{Type: msgCommit, From: c.id, Seq: seq}
	if titleChanged {
		out.Title = m.Title
	}
	s.broadcast(c, out)
}

// writeBody pushes a converged state into the canonical doc, firing doc.updated per commit, not per keystroke.
func (s *session) writeBody(ctx context.Context, title, body string) {
	if err := s.writer.CommitCollab(ctx, s.docID, title, body); err != nil {
		s.log.Warn("collab: commit body failed", "doc", s.docID, "error", err)
	}
}

// handlePresence relays a participant's awareness state and remembers it for late joiners.
func (s *session) handlePresence(c *client, m ClientMsg) {
	s.mu.Lock()
	if _, member := s.clients[c]; !member {
		s.mu.Unlock()
		return
	}
	s.presence[m.ClientID] = m.Payload
	s.mu.Unlock()
	s.broadcast(c, ServerMsg{Type: msgPresence, From: c.id, ClientID: m.ClientID, Payload: m.Payload})
}

// leave drops a participant: its awareness states are removed for the rest and its presence entry is forgotten.
func (s *session) leave(c *client) {
	s.mu.Lock()
	if c.closed {
		s.mu.Unlock()
		return
	}
	delete(s.presence, c.clientID)
	delete(s.clients, c)
	c.closed = true
	next := s.handOffSeed(c)
	s.mu.Unlock()
	s.broadcast(nil, ServerMsg{Type: msgLeave, ClientID: c.clientID})
	if next != nil {
		_ = s.send(next, ServerMsg{Type: msgSeed}) // a peer gone mid-handoff leaves the next join to claim the seed
	}
}

// handOffSeed passes the seed on when the seeder leaves an empty room unseeded; callers hold s.mu.
func (s *session) handOffSeed(gone *client) *client {
	if s.seeder != gone {
		return nil
	}
	s.seeder = nil
	for c := range s.clients {
		if s.claimSeed(c) {
			return c
		}
	}
	return nil
}

// reset forgets the room's state after a server-side write: current participants turn stale and are told to rejoin.
func (s *session) reset(seq int64) {
	s.mu.Lock()
	s.resetSeq, s.seq = seq, seq
	s.lastBody, s.lastTitle = "", ""
	s.hasState, s.seeder = false, nil
	for c := range s.clients {
		c.stale = true
	}
	s.mu.Unlock()
	s.broadcast(nil, ServerMsg{Type: msgReset, Seq: seq})
}

// broadcast sends to every participant but one; a stuck peer leaves and is hung up on, never blocking the session.
func (s *session) broadcast(except *client, m ServerMsg) {
	data, err := json.Marshal(m)
	if err != nil {
		s.log.Warn("collab: marshal broadcast failed", "doc", s.docID, "error", err)
		return
	}
	s.mu.Lock()
	var dropped []*client
	for c := range s.clients {
		if c == except {
			continue
		}
		select {
		case c.send <- data:
		default:
			s.log.Warn("collab: dropping slow peer", "doc", s.docID, "client", c.id)
			dropped = append(dropped, c)
		}
	}
	s.mu.Unlock()
	for _, c := range dropped {
		s.leave(c)
		c.hangUp()
	}
}

// send gives up when the peer's pump died; the done check runs first so a dead peer never swallows a frame.
func (s *session) send(c *client, m ServerMsg) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	select {
	case <-c.done:
		return errPeerGone
	default:
	}
	select {
	case c.send <- data:
		return nil
	case <-c.done:
		return errPeerGone
	}
}

// empty reports whether the session still has live participants (hub cleanup).
func (s *session) empty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients) == 0
}

// setClientID remembers the y client id announced in hello so leave frames can name it.
func (s *session) setClientID(c *client, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.clientID = id
}
