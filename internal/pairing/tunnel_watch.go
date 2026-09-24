package pairing

import (
	"context"
	"errors"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Publisher is the bus seam for ephemeral events that are never written to the outbox.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

const (
	// tunnelWatchInterval paces the Cloudflare status reads while a computer waits for its connector.
	tunnelWatchInterval = 3 * time.Second
	// tunnelWatchWindow bounds an unattended wait; reading the status again restarts it.
	tunnelWatchWindow = 30 * time.Minute
)

// tunnelWatch is one computer whose tunnel checks are pushed live until both pass.
type tunnelWatch struct {
	userID string
	until  time.Time
	last   TunnelStatus
}

// watchTunnel starts or extends the live watch on a computer's tunnel, keeping the last status it pushed.
func (s *Service) watchTunnel(userID, computerID string) {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	w := s.watching[computerID]
	w.userID = userID
	w.until = s.now().Add(tunnelWatchWindow)
	s.watching[computerID] = w
}

// RunTunnelWatch polls every watched tunnel and publishes each change to its checks until ctx ends.
func (s *Service) RunTunnelWatch(ctx context.Context) {
	ticker := time.NewTicker(tunnelWatchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollTunnels(ctx)
		}
	}
}

// pollTunnels reads each watched tunnel once; a changed status is published, a connected or gone one stops being watched.
func (s *Service) pollTunnels(ctx context.Context) {
	for id, w := range s.watchedTunnels() {
		status, err := s.tunnelStatus(ctx, w.userID, id)
		if errors.Is(err, apperrs.ErrNotFound) {
			s.stopWatching(id)
			continue
		}
		if err != nil {
			logging.FromCtx(ctx).Debug("computer tunnel status read failed", "computer_id", id, "error", err)
			continue
		}
		if status == w.last {
			continue
		}
		s.recordWatched(id, status)
		s.publishTunnelStatus(ctx, id, w.userID, status)
	}
}

// watchedTunnels copies the live watches and drops the expired ones, so polling never holds the lock over I/O.
func (s *Service) watchedTunnels() map[string]tunnelWatch {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	now := s.now()
	out := make(map[string]tunnelWatch, len(s.watching))
	for id, w := range s.watching {
		if now.After(w.until) {
			delete(s.watching, id)
			continue
		}
		out[id] = w
	}
	return out
}

func (s *Service) recordWatched(computerID string, status TunnelStatus) {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	w, ok := s.watching[computerID]
	if !ok {
		return
	}
	if status.Connected() {
		delete(s.watching, computerID)
		return
	}
	w.last = status
	s.watching[computerID] = w
}

func (s *Service) stopWatching(computerID string) {
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	delete(s.watching, computerID)
}

func (s *Service) publishTunnelStatus(ctx context.Context, computerID, userID string, status TunnelStatus) {
	if s.bus == nil {
		return
	}
	err := s.bus.Publish(ctx, TopicTunnelStatusChanged, TunnelStatusChangedEvent{
		ComputerID: computerID, UserID: userID, Tunnel: status.Tunnel,
		HarnessReachable: status.HarnessReachable, HarnessVersion: status.HarnessVersion,
	})
	if err != nil {
		logging.FromCtx(ctx).Warn("publish computer tunnel status", "computer_id", computerID, "error", err)
	}
}
