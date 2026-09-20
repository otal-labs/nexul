package voice

import (
	"context"
	"log/slog"
	"time"

	"github.com/otal-labs/nexul/internal/livekit"
)

// pollInterval is the reconciliation poll's cadence; the sole presence feed on localhost/dev.
const pollInterval = 30 * time.Second

// RunReconciliation polls LiveKit and corrects occupancy drift until ctx is done.
// ponytail: polls unconditionally, no subscriber-presence gate; live.Hub has no cheap count and volume is negligible.
func (s *Service) RunReconciliation(ctx context.Context, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcileOnce(ctx, logger)
		}
	}
}

func (s *Service) reconcileOnce(ctx context.Context, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	client, err := s.credentials.LiveKit(ctx)
	if err != nil {
		return // no LiveKit connector configured yet; nothing to reconcile
	}
	rooms, err := client.ListRooms(ctx)
	if err != nil {
		logger.Warn("voice occupancy reconciliation poll failed", "error", err)
		return
	}
	changed := s.occupancy.reconcile(fromLiveKitRooms(rooms))
	for _, room := range changed {
		if err := s.publish(ctx, room, s.occupancy.room(room)); err != nil {
			logger.Warn("voice occupancy change publish failed", "conversation_id", room, "error", err)
		}
	}
}

// fromLiveKitRooms converts the room listing into occupancy's input shape; ListRooms already omits empty rooms.
func fromLiveKitRooms(rooms []livekit.Room) map[string][]Occupant {
	out := make(map[string][]Occupant, len(rooms))
	for _, r := range rooms {
		if len(r.Participants) == 0 {
			continue
		}
		occupants := make([]Occupant, len(r.Participants))
		for i, p := range r.Participants {
			occupants[i] = Occupant{Identity: p.Identity, Name: p.Name}
		}
		out[r.Name] = occupants
	}
	return out
}
