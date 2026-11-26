package runner

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// MachineShellJobs records the commands run on machines' runners.
type MachineShellJobs interface {
	shelljob.Store
	GetShellJob(ctx context.Context, id string) (*shelljob.Job, error)
	// ListMachineShellJobs pages a machine's jobs newest first, without output.
	ListMachineShellJobs(ctx context.Context, machineID string, w paging.Window) ([]shelljob.Job, int, error)
}

// MachineShells runs commands on a machine's runners: the WebSocket handler.
type MachineShells interface {
	RunMachineShell(ctx context.Context, runnerID string, run ShellRun, sink ShellSink) error
	CancelShell(ctx context.Context, jobID string) error
	OpenMachineTerminal(ctx context.Context, runnerID, id string, size shelljob.TerminalSize) (shelljob.TerminalStream, error)
}

// WithShells wires commands on machines' runners (ADR 0209); unset, none run.
func (s *Service) WithShells(jobs MachineShellJobs, shells MachineShells) *Service {
	s.shellJobs, s.shells = jobs, shells
	s.shellQueue = shelljob.NewQueue(shelljob.Config{
		Store: jobs, Runners: machineRunners{s}, Event: machineShellJobEvent, Now: func() time.Time { return s.now() },
		StartError: machineShellStartError, Silent: "the runner did not say how the command ended",
	})
	s.terminals = shelljob.NewTerminals(shelljob.Config{
		Store: jobs, Event: machineShellJobEvent, Now: func() time.Time { return s.now() }, StartError: machineShellStartError,
	})
	return s
}

// DrainShells waits for the goroutines that hand queued commands to their runners.
func (s *Service) DrainShells() {
	if s.shellQueue != nil {
		s.shellQueue.Drain()
	}
}

// RunCommand records a command on runnerID, a machine's runner, and queues it; it returns the job at once, queued or
// already running. The runner's own opt-in, its install's --allow-shell, is the runner's to check, so a refusal there
// is a recorded job.
func (s *Service) RunCommand(ctx context.Context, runnerID string, in shelljob.Input) (*shelljob.Job, error) {
	if err := shelljob.Validate(&in); err != nil {
		return nil, err
	}
	rn, m, err := s.shellRunner(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	if !m.ShellEnabled {
		return nil, fmt.Errorf("%w: commands from Nexul are off on machine %s; someone with Run commands on it turns them on under Runners", apperrs.ErrConflict, m.Name)
	}
	actor, _ := identity.ActorFromCtx(ctx)
	job := shelljob.Job{
		ID: ids.New(), RunnerID: rn.ID, MachineID: m.ID, RequesterID: actor.ID, Adapter: in.Adapter, Shell: in.Shell,
		Command: in.Command, Cwd: in.Cwd, TimeoutSeconds: int(in.Timeout.Seconds()), Status: shelljob.Queued,
		CreatedAt: s.now().UTC(),
	}
	if err := s.shellQueue.Add(ctx, job); err != nil {
		return nil, fmt.Errorf("record command on runner %s: %w", rn.Name, err)
	}
	return &job, nil
}

// shellRunner is runnerID with its machine, for a caller holding Run commands there. A personal runner, and one on a
// machine the caller can't see, is not found.
func (s *Service) shellRunner(ctx context.Context, runnerID string) (*Runner, *Machine, error) {
	if s.shellQueue == nil || s.machines == nil {
		return nil, nil, apperrs.Fatal(fmt.Errorf("%w: commands on machines are not wired", apperrs.ErrFatal))
	}
	notFound := fmt.Errorf("%w: runner %s; machine_list lists the runners", apperrs.ErrNotFound, runnerID)
	rn, err := s.repo.GetByID(ctx, runnerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil, notFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get runner %s: %w", runnerID, err)
	}
	if rn.Personal() || rn.MachineID == "" {
		return nil, nil, notFound
	}
	err = s.requireMachine(ctx, rn.MachineID, permissions.MachinesShell)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil, notFound
	}
	if err != nil {
		return nil, nil, err
	}
	m, err := s.machines.Get(ctx, rn.MachineID)
	if err != nil {
		return nil, nil, fmt.Errorf("get the machine of runner %s: %w", rn.Name, err)
	}
	return rn, m, nil
}

// ListShellJobs pages a machine's commands newest first, without output, for a caller holding Run commands there.
func (s *Service) ListShellJobs(ctx context.Context, machineID string, w paging.Window) ([]shelljob.Job, int, error) {
	if s.shellJobs == nil {
		return nil, 0, apperrs.Fatal(fmt.Errorf("%w: commands on machines are not wired", apperrs.ErrFatal))
	}
	if err := s.requireMachine(ctx, machineID, permissions.MachinesShell); err != nil {
		return nil, 0, err
	}
	jobs, total, err := s.shellJobs.ListMachineShellJobs(ctx, machineID, w.Clamped())
	if err != nil {
		return nil, 0, fmt.Errorf("list commands on machine %s: %w", machineID, err)
	}
	return jobs, total, nil
}

// GetShellJob returns one command on a machine with its output, to a caller holding Run commands there; a computer's
// command, or one on a machine they can't see, is not found.
func (s *Service) GetShellJob(ctx context.Context, jobID string) (*shelljob.Job, error) {
	if s.shellJobs == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: commands on machines are not wired", apperrs.ErrFatal))
	}
	notFound := fmt.Errorf("%w: command %s", apperrs.ErrNotFound, jobID)
	job, err := s.shellJobs.GetShellJob(ctx, jobID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, notFound
	}
	if err != nil {
		return nil, fmt.Errorf("get command %s: %w", jobID, err)
	}
	if job.MachineID == "" {
		return nil, notFound
	}
	err = s.requireMachine(ctx, job.MachineID, permissions.MachinesShell)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, notFound
	}
	if err != nil {
		return nil, err
	}
	s.shellQueue.Overlay(job)
	return job, nil
}

// WaitShellJob waits for a command to end and returns it with its output; a ctx that ends first returns it as it is.
func (s *Service) WaitShellJob(ctx context.Context, jobID string) (*shelljob.Job, error) {
	if _, err := s.GetShellJob(ctx, jobID); err != nil {
		return nil, err
	}
	s.shellQueue.Wait(ctx, jobID)
	return s.GetShellJob(context.WithoutCancel(ctx), jobID)
}

// CancelShellJob stops a queued or running command on a machine.
func (s *Service) CancelShellJob(ctx context.Context, jobID string) (*shelljob.Job, error) {
	if _, err := s.GetShellJob(ctx, jobID); err != nil {
		return nil, err
	}
	s.terminals.Cancel(jobID, "")
	s.shellQueue.Cancel(ctx, jobID, "")
	return s.WaitShellJob(ctx, jobID)
}

// OpenShellOutput streams a command's output from its first line: live while it runs, what was kept once it ended.
func (s *Service) OpenShellOutput(ctx context.Context, jobID string) (containerlog.Feed, error) {
	job, err := s.GetShellJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return s.shellQueue.Open(*job), nil
}

// SetMachineShell turns commands from Nexul on or off for a machine's runners; off stops every command on it.
func (s *Service) SetMachineShell(ctx context.Context, machineID string, enabled bool) (*Machine, error) {
	if s.shellQueue == nil || s.machines == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: commands on machines are not wired", apperrs.ErrFatal))
	}
	if err := s.requireMachine(ctx, machineID, permissions.MachinesShell); err != nil {
		return nil, err
	}
	if err := s.machines.SetShell(ctx, machineID, enabled); err != nil {
		return nil, err
	}
	if !enabled {
		match := func(job shelljob.Job) bool { return job.MachineID == machineID }
		why := "commands from Nexul were turned off for this machine"
		s.shellQueue.CancelWhere(ctx, match, why)
		s.terminals.CancelWhere(match, why)
	}
	return s.machines.Get(ctx, machineID)
}

// machineRunners hands queued commands to their machines' runners.
type machineRunners struct{ s *Service }

func (r machineRunners) Start(ctx context.Context, job shelljob.Job, run shelljob.Run, sink shelljob.Sink) error {
	return r.s.shells.RunMachineShell(ctx, job.RunnerID, ShellRun{ID: run.ID, Shell: run.Shell, Script: run.Command, Cwd: run.Cwd, Timeout: run.Timeout}, JobSink(sink))
}

func (r machineRunners) Cancel(ctx context.Context, jobID string) error {
	return r.s.shells.CancelShell(ctx, jobID)
}

// StillAllowed checks again, as a queued command reaches the front, that its machine takes commands and that whoever
// ran it still holds Run commands there.
func (r machineRunners) StillAllowed(ctx context.Context, job shelljob.Job) (shelljob.End, bool) {
	s := r.s
	m, err := s.machines.Get(ctx, job.MachineID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return shelljob.End{Status: shelljob.Cancelled, Error: "the machine was deleted"}, false
	}
	if err != nil {
		return shelljob.End{Status: shelljob.Failed, Error: "could not check the machine before running"}, false
	}
	if !m.ShellEnabled {
		return shelljob.End{Status: shelljob.Cancelled, Error: "commands from Nexul were turned off for this machine"}, false
	}
	if s.creators == nil || job.RequesterID == "" {
		return shelljob.End{}, true
	}
	held, err := s.creators.MachinePermissions(ctx, job.RequesterID, job.MachineID)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return shelljob.End{Status: shelljob.Failed, Error: "could not check the machine before running"}, false
	}
	if !slices.Contains(held, permissions.MachinesShell) {
		return shelljob.End{Status: shelljob.Cancelled, Error: "access revoked"}, false
	}
	return shelljob.End{}, true
}

// machineShellStartError is what a command that never reached its runner says.
func machineShellStartError(err error) string {
	if errors.Is(err, ErrCannotOpenTerminal) {
		return "the runner is too old to open a terminal; it updates itself, then try again"
	}
	if errors.Is(err, apperrs.ErrConflict) {
		return strings.TrimPrefix(err.Error(), apperrs.ErrConflict.Error()+": ")
	}
	if errors.Is(err, ErrCannotRunShell) {
		return "the runner is too old to run commands; it updates itself, then try again"
	}
	return "the runner is not connected"
}

// JobSink hands a runner's shell frames to a shell job queue in its own types.
func JobSink(sink shelljob.Sink) ShellSink {
	return jobSink{sink}
}

type jobSink struct{ sink shelljob.Sink }

func (j jobSink) Output(lines []ContainerLogLine) {
	out := make([]containerlog.Line, len(lines))
	for i, l := range lines {
		out[i] = containerlog.Line(l)
	}
	j.sink.Output(out)
}

func (j jobSink) Done(r ShellResult) {
	j.sink.Done(shelljob.End{Status: r.Status, ExitCode: r.ExitCode, Error: r.Error})
}

func machineShellJobEvent(job shelljob.Job) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicShellJobChanged, Payload: ShellJobChangedEvent{
		JobID: job.ID, RunnerID: job.RunnerID, MachineID: job.MachineID, UserID: job.RequesterID, Status: job.Status,
	}}
}
