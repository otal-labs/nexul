// Package live fans bus events out to connected browser WebSockets; callers decide which topics to bridge.
package live

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
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

// Hub manages browser WS connections and broadcasts frames to all of them.
type Hub struct {
	log     *slog.Logger
	mu      sync.Mutex
	clients map[*client]struct{}
}

type client struct {
	writeMu sync.Mutex
	ws      *websocket.Conn
}

// New wires a hub. A nil logger defaults to slog.Default().
func New(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{log: logger, clients: make(map[*client]struct{})}
}

// ServeHTTP upgrades a browser connection and pushes to it until it closes.
// Authentication is the caller's job (mount behind a RequireWS-style guard).
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log.Warn("browser ws upgrade failed", "error", err)
		return
	}
	c := &client{ws: conn}
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

// Publish broadcasts a frame to every browser; a write failure just drops that client, who reconnects with backoff.
func (h *Hub) Publish(_ context.Context, topic string, payload any) error {
	data, err := json.Marshal(Frame{Topic: topic, Type: "event", Payload: payload})
	if err != nil {
		return err
	}
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	for _, c := range clients {
		if err := c.write(data); err != nil {
			h.log.Warn("browser ws write failed; dropping client", "error", err)
			h.remove(c)
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

func (c *client) write(data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), WriteTimeout)
	defer cancel()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.ws.Write(ctx, websocket.MessageText, data)
}
