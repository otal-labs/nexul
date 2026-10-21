package pairing

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// onRunner reports a computer whose session is through its personal runner, not a tunnel or an address.
func onRunner(c Computer) bool {
	return c.ServerURL == runnerAddress(c.ID)
}

// AnnounceMoves tells each owner of a computer still paired through a tunnel or by URL, once, to move it to the Nexul
// app. Its stored receipt keeps the notice to once across restarts (ADR 0146).
func (s *Service) AnnounceMoves(ctx context.Context) error {
	owners, err := s.repo.ComputersToMove(ctx)
	if err != nil {
		return err
	}
	at := s.now().UTC()
	for _, c := range owners {
		evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMoveRequested, Payload: MoveRequestedEvent{ComputerID: c.ID, UserID: c.UserID, MembersOnly: true}}
		if err := s.repo.RecordMoveNotice(ctx, c.UserID, at, evt); err != nil {
			return fmt.Errorf("announce the move to %s: %w", c.UserID, err)
		}
	}
	return nil
}
