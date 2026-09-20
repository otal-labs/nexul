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
	clientID int    // y client id announced in hello (presence routing)
	send     chan []byte
	done     chan struct{} // closed by the pump when the peer is gone
	closed   bool
}

// session is one document's live collaboration room; a persistence failure logs but still relays.
type session struct {
	log    *slog.Logger
	docID  string
	store  Store
	writer DocWriter

	mu        sync.Mutex
	clients   map[*client]struct{}
	presence  map[int]string // y client id → base64 awareness state
	seq       int64          // latest seq persisted for this doc
	lastBody  string         // last canonical body committed (write dedupe)
	lastTitle string         // last canonical title committed (write dedupe)
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
	presence := make([]PresenceMsg, 0, len(s.presence))
	for id, p := range s.presence {
		presence = append(presence, PresenceMsg{ClientID: id, Payload: p})
	}
	s.mu.Unlock()
	return s.send(c, ServerMsg{Type: msgInit, Seq: replay.Seq, Snapshot: replay.Snapshot, Updates: replay.Increments, Presence: presence})
}

// handleUpdate persists and relays one Y.js update; a persistence failure logs but still relays.
func (s *session) handleUpdate(ctx context.Context, c *client, m ClientMsg) {
	var seq int64
	stored, err := s.store.AppendUpdate(ctx, s.docID, c.id, KindUpdate, m.Update)
	if err != nil {
		s.log.Warn("collab: persist update failed; relaying anyway", "doc", s.docID, "error", err)
	}
	if err == nil {
		seq = stored
		s.mu.Lock()
		if seq > s.seq {
			s.seq = seq
		}
		s.mu.Unlock()
	}
	s.broadcast(c, ServerMsg{Type: msgUpdate, From: c.id, Seq: seq, Payload: m.Update})
}

// handleCommit persists a snapshot, trims already-applied increments, writes the canonical body, and broadcasts.
func (s *session) handleCommit(ctx context.Context, c *client, m ClientMsg) {
	seq, err := s.store.AppendUpdate(ctx, s.docID, c.id, KindSnapshot, m.Update)
	if err != nil {
		s.log.Warn("collab: persist snapshot failed", "doc", s.docID, "error", err)
		return
	}
	if err := s.store.TrimUpdates(ctx, s.docID, m.BaseSeq); err != nil {
		s.log.Warn("collab: trim updates failed", "doc", s.docID, "error", err)
	}
	s.mu.Lock()
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
	s.mu.Unlock()
	s.broadcast(nil, ServerMsg{Type: msgLeave, ClientID: c.clientID})
}

// broadcast sends to every participant but one; a stuck peer is dropped, never blocking the session.
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
			delete(s.clients, c)
			dropped = append(dropped, c)
		}
	}
	s.mu.Unlock()
	for _, c := range dropped {
		c.closed = true
		s.broadcast(nil, ServerMsg{Type: msgLeave, ClientID: c.clientID})
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
