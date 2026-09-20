package collab

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Mode gates the join permission: edit needs the doc's edit bit, view needs the read bit (ADR 0042).
type Mode string

const (
	ModeEdit Mode = "edit"
	ModeView Mode = "view"
)

// WriteTimeout bounds each frame write to a browser. Slow consumers drop.
const WriteTimeout = 10 * time.Second

// sendBuffer is the per-client frame queue; the write pump drains it.
const sendBuffer = 64

var errUnknownMode = errors.New("mode must be edit or view")

// Hub relays Y.js updates and presence, replays state to joiners, and commits state via the docs seam.
type Hub struct {
	log    *slog.Logger
	store  Store
	access AccessChecker
	writer DocWriter

	mu       sync.Mutex
	sessions map[string]*session
}

// NewHub wires a collab hub. A nil logger defaults to slog.Default().
func NewHub(logger *slog.Logger, store Store, access AccessChecker, writer DocWriter) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		log:      logger,
		store:    store,
		access:   access,
		writer:   writer,
		sessions: make(map[string]*session),
	}
}

// ServeHTTP assumes the caller already authenticated (mount behind RequireWS).
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	actor, ok := identity.ActorFromCtx(r.Context())
	if !ok || actor.ID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	docID := strings.TrimSpace(r.PathValue("docID"))
	if docID == "" {
		http.Error(w, "missing doc id", http.StatusBadRequest)
		return
	}
	mode, action, err := modeAction(r.URL.Query().Get("mode"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	allowed, err := h.access.Can(r.Context(), actor.ID, docID, action)
	if err != nil || !allowed {
		// Fail closed: a broken checker denies, never allows (the same rule the docs use-case layer applies).
		h.log.Warn("collab join denied", "doc", docID, "user", actor.ID, "mode", mode, "error", err)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log.Warn("collab ws upgrade failed", "error", err)
		return
	}
	conn.SetReadLimit(MaxFrameBytes)

	c := &client{id: actor.ID, send: make(chan []byte, sendBuffer), done: make(chan struct{})}
	s := h.session(docID)
	s.join(c)
	h.log.Info("collab joined", "doc", docID, "user", actor.ID, "mode", mode)
	defer func() {
		s.leave(c)
		h.removeSession(docID)
	}()

	go h.pump(c, conn, s)

	if err := s.sendInit(r.Context(), c); err != nil {
		h.log.Warn("collab replay failed", "doc", docID, "user", actor.ID, "error", err)
		_ = conn.Close(websocket.StatusInternalError, "replay failed")
		return
	}

	for {
		_, data, err := conn.Read(r.Context())
		if err != nil {
			break
		}
		h.dispatch(r.Context(), c, s, data)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

// pump closes the connection and done on a write failure, so blocking session sends give up on the dead peer.
func (h *Hub) pump(c *client, conn *websocket.Conn, s *session) {
	for data := range c.send {
		if err := writeFrame(conn, data); err != nil {
			h.log.Warn("collab write failed; dropping client", "user", c.id, "error", err)
			s.leave(c)
			close(c.done)
			_ = conn.Close(websocket.StatusPolicyViolation, "write failed")
			return
		}
	}
}

func writeFrame(conn *websocket.Conn, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), WriteTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, data)
}

// dispatch routes one inbound frame; malformed frames are dropped with a log rather than killing the session.
func (h *Hub) dispatch(ctx context.Context, c *client, s *session, data []byte) {
	m, err := parseClientMsg(data)
	if err != nil {
		h.log.Warn("collab dropped frame", "user", c.id, "error", err)
		return
	}
	switch m.Type {
	case msgHello:
		s.setClientID(c, m.ClientID)
	case msgUpdate:
		s.handleUpdate(ctx, c, m)
	case msgCommit:
		s.handleCommit(ctx, c, m)
	case msgPresence:
		if c.clientID == 0 {
			h.log.Warn("collab presence before hello", "user", c.id)
			return
		}
		s.handlePresence(c, m)
	default:
		h.log.Warn("collab dropped unknown frame", "user", c.id, "type", m.Type)
	}
}

// session rooms are reclaimed when their last client leaves; the store keeps the persisted state.
func (h *Hub) session(docID string) *session {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.sessions[docID]
	if !ok {
		s = &session{
			log:      h.log,
			docID:    docID,
			store:    h.store,
			writer:   h.writer,
			clients:  make(map[*client]struct{}),
			presence: make(map[int]string),
		}
		h.sessions[docID] = s
	}
	return s
}

// removeSession reclaims an empty room (called after leave).
func (h *Hub) removeSession(docID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[docID]; ok && s.empty() {
		delete(h.sessions, docID)
	}
}

// modeAction maps the requested mode to its permission bit.
func modeAction(mode string) (Mode, permissions.Action, error) {
	switch Mode(mode) {
	case ModeEdit:
		return ModeEdit, permissions.DocsWrite, nil
	case ModeView:
		return ModeView, permissions.DocsRead, nil
	default:
		return "", "", errUnknownMode
	}
}
