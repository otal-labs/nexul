package runner

import (
	"context"
	"time"
)

// graceHold is one dropped personal runner waiting out the reconnect grace; its timer reports it disconnected.
type graceHold struct {
	timer *time.Timer
}

// holdOffline reports a personal runner's dropped connection as reconnecting, and as disconnected only once
// ReconnectGrace passes without it coming back: a paused computer misses a beat and is back seconds later.
func (h *Handler) holdOffline(ctx context.Context, c *runnerConn) {
	h.graceMu.Lock()
	if h.graceClosed {
		h.graceMu.Unlock()
		return
	}
	if old, ok := h.grace[c.id]; ok {
		old.timer.Stop()
	}
	g := &graceHold{}
	g.timer = time.AfterFunc(h.cfg.ReconnectGrace, func() { h.endGrace(c, g) })
	h.grace[c.id] = g
	h.graceMu.Unlock()
	h.publish(ctx, TopicPersonalChanged, personalChanged(c.record, PersonalReconnecting, ""))
}

// endGrace reports c disconnected unless it reconnected, or dropped again and started a newer hold, meanwhile.
func (h *Handler) endGrace(c *runnerConn, g *graceHold) {
	h.graceMu.Lock()
	if h.grace[c.id] != g {
		h.graceMu.Unlock()
		return
	}
	delete(h.grace, c.id)
	h.graceMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), h.cfg.WriteTimeout)
	defer cancel()
	h.publish(ctx, TopicPersonalChanged, personalChanged(c.record, PersonalDisconnected, ""))
}

// releaseHold ends runnerID's grace without reporting it disconnected, because it connected again.
func (h *Handler) releaseHold(runnerID string) {
	h.graceMu.Lock()
	defer h.graceMu.Unlock()
	if g, ok := h.grace[runnerID]; ok {
		g.timer.Stop()
		delete(h.grace, runnerID)
	}
}

// stopHolds drops every pending grace at shutdown, so no timer publishes on a bus that is going away.
func (h *Handler) stopHolds() {
	h.graceMu.Lock()
	defer h.graceMu.Unlock()
	h.graceClosed = true
	for id, g := range h.grace {
		g.timer.Stop()
		delete(h.grace, id)
	}
}

// Reconnecting reports whether runnerID dropped less than ReconnectGrace ago and has not connected again.
func (h *Handler) Reconnecting(runnerID string) bool {
	h.graceMu.Lock()
	defer h.graceMu.Unlock()
	_, ok := h.grace[runnerID]
	return ok
}
