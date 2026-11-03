package pairing

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// AppRestarter restarts the Nexul app on a computer through its runner.
type AppRestarter interface {
	// RestartApp asks computerID's runner to restart itself, and returns once it agreed, before it is back.
	RestartApp(ctx context.Context, computerID string) error
}

// RestartApp restarts the Nexul app on one of the caller's own computers, so it starts again with the PATH the computer
// saved since it last started; the computer reads Reconnecting until the app is back and reports its facts again.
func (s *Service) RestartApp(ctx context.Context, userID, computerID string) error {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return err
	}
	if s.restarter == nil || s.runners == nil {
		return apperrs.Fatal(fmt.Errorf("%w: restarting the Nexul app is not wired", apperrs.ErrFatal))
	}
	runner, err := s.runners.ComputerRunner(ctx, c.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w: %s has no Nexul app to restart", apperrs.ErrConflict, c.Name)
	}
	if err != nil {
		return fmt.Errorf("read the runner of computer %s: %w", c.ID, err)
	}
	if !runner.Connected {
		return fmt.Errorf("%w: %s is offline, so its Nexul app can't be restarted from here", apperrs.ErrConflict, c.Name)
	}
	if err := s.restarter.RestartApp(ctx, c.ID); err != nil {
		return fmt.Errorf("restart the Nexul app on %s: %w", c.Name, err)
	}
	return nil
}
