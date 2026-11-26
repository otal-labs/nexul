package runner

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// OpenTerminal records a terminal on runnerID, a machine's runner, and opens it there for a caller holding Run commands
// on the machine; the caller then attaches to it within shelljob.AttachWindow (ADR 0212).
func (s *Service) OpenTerminal(ctx context.Context, runnerID string, size shelljob.TerminalSize) (*shelljob.Job, error) {
	rn, m, err := s.shellRunner(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	if !m.ShellEnabled {
		return nil, fmt.Errorf("%w: commands from Nexul are off on machine %s; someone with Run commands on it turns them on under Runners", apperrs.ErrConflict, m.Name)
	}
	actor, _ := identity.ActorFromCtx(ctx)
	job := shelljob.Job{ID: ids.New(), RunnerID: rn.ID, MachineID: m.ID, RequesterID: actor.ID, CreatedAt: s.now().UTC()}
	return s.terminals.Open(ctx, job, size, func(ctx context.Context, job shelljob.Job, size shelljob.TerminalSize) (shelljob.TerminalStream, error) {
		return s.shells.OpenMachineTerminal(ctx, job.RunnerID, job.ID, size)
	})
}

// AttachTerminal relays the viewer accept gives to a terminal the caller opened until either side closes it.
func (s *Service) AttachTerminal(ctx context.Context, jobID string, accept func() (shelljob.TerminalStream, error)) error {
	if s.terminals == nil {
		return apperrs.Fatal(fmt.Errorf("%w: commands on machines are not wired", apperrs.ErrFatal))
	}
	actor, _ := identity.ActorFromCtx(ctx)
	return s.terminals.Attach(ctx, jobID, actor.ID, accept)
}
