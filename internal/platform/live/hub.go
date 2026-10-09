// Package live fans bus events out to connected browser WebSockets; callers decide which topics to bridge.
package live

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/jsonx"
)

// Frame is the wire shape pushed to browser clients. The SPA parses
// {topic, type, payload} and dispatches on topic (web/src/api/ws.tsx).
type Frame struct {
	Topic   string `json:"topic"`
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// WriteTimeout bounds each frame write to a browser. Slow consumers drop.
const WriteTimeout = 10 * time.Second

// sendBuffer is how many frames a socket may fall behind before it is dropped; it reconnects and refetches.
const sendBuffer = 64

// Audience decides whether a frame may reach the person behind one socket; ctx carries them as the actor.
type Audience func(ctx context.Context, topic string, payload any) bool

// Hub manages browser WS connections and pushes each frame to the ones its audience allows.
type Hub struct {
	log      *slog.Logger
	mu       sync.Mutex
	clients  map[*client]struct{}
	audience Audience
}

type client struct {
	ws     *websocket.Conn
	userID string
	send   chan []byte
}

// New wires a hub. A nil logger defaults to slog.Default().
func New(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{log: logger, clients: make(map[*client]struct{})}
}

// SetAudience wires the per-socket check every frame passes; unset, every socket receives every frame.
func (h *Hub) SetAudience(a Audience) {
	h.audience = a
}

// ServeHTTP upgrades a browser connection and pushes to it until it closes.
// Authentication is the caller's job (mount behind a RequireWS-style guard that attaches the actor).
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log.Warn("browser ws upgrade failed", "error", err)
		return
	}
	actor, _ := identity.ActorFromCtx(r.Context())
	c := &client{ws: conn, userID: actor.ID, send: make(chan []byte, sendBuffer)}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go h.writeLoop(ctx, c)
	h.add(c)
	h.log.Info("browser ws connected", "remote", r.RemoteAddr)

	// Drain inbound until the peer closes; browsers send nothing, but reading
	// is what detects a closed connection.
	for {
		if _, _, err := conn.Read(r.Context()); err != nil {
			break
		}
	}
	h.remove(c)
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

// Publish queues a frame for every browser its audience allows; a client too far behind is dropped and reconnects
// with backoff, so one stalled socket never holds up the others.
func (h *Hub) Publish(ctx context.Context, topic string, payload any) error {
	data, err := jsonx.Marshal(Frame{Topic: topic, Type: "event", Payload: payload})
	if err != nil {
		return err
	}
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	// A person's tabs and phone share one read check per frame; agent streams publish at token rate.
	allowed := make(map[string]bool, len(clients))
	for _, c := range clients {
		ok, seen := allowed[c.userID]
		if !seen {
			ok = h.audience == nil || h.audience(identity.WithActor(ctx, identity.Actor{ID: c.userID}), topic, payload)
			allowed[c.userID] = ok
		}
		if !ok {
			continue
		}
		select {
		case c.send <- data:
		default:
			h.log.Warn("browser ws too far behind; dropping client", "user_id", c.userID)
			h.remove(c)
			_ = c.ws.CloseNow() // the read loop sees the close and ends the connection
		}
	}
	return nil
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

// writeLoop sends a client's queued frames in order until its connection ends; a failed write drops the client.
func (h *Hub) writeLoop(ctx context.Context, c *client) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-c.send:
			wctx, cancel := context.WithTimeout(ctx, WriteTimeout)
			err := c.ws.Write(wctx, websocket.MessageText, data)
			cancel()
			if err != nil {
				h.log.Warn("browser ws write failed; dropping client", "error", err)
				h.remove(c)
				_ = c.ws.CloseNow() // the read loop sees the close and ends the connection
				return
			}
		}
	}
}
