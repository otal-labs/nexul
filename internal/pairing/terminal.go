package pairing

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/auditnote"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// OpenTerminal records a terminal on one of the caller's computers, or one shared with them by Run commands, and opens
// it on the computer's runner; the caller then attaches to it within shelljob.AttachWindow (ADR 0212).
func (s *Service) OpenTerminal(ctx context.Context, userID, computerID string, size shelljob.TerminalSize) (*shelljob.Job, error) {
	c, err := s.commandsComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if !c.ShellEnabled {
		return nil, fmt.Errorf("%w: commands from Nexul are turned off for %s; its owner turns them on under Commands", apperrs.ErrConflict, c.Name)
	}
	job := shelljob.Job{ID: ids.New(), ComputerID: c.ID, OwnerID: c.UserID, RequesterID: userID, CreatedAt: s.now().UTC()}
	auditnote.Job(ctx, c.ID, c.UserID, job.ID)
	return s.terminals.Open(ctx, job, size, func(ctx context.Context, job shelljob.Job, size shelljob.TerminalSize) (shelljob.TerminalStream, error) {
		return s.shells.OpenTerminal(ctx, job.ComputerID, job.ID, size)
	})
}

// AttachTerminal relays the viewer accept gives to a terminal the caller opened until either side closes it.
func (s *Service) AttachTerminal(ctx context.Context, userID, jobID string, accept func() (shelljob.TerminalStream, error)) error {
	return s.terminals.Attach(ctx, jobID, userID, accept)
}
