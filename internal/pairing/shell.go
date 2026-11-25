package pairing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/auditnote"
	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

const (
	shellPurgeBatch    = 500
	shellPurgeInterval = 24 * time.Hour
)

// Shells runs commands on a computer through its runner.
type Shells interface {
	// RunShell sends run to the computer's runner; sink gets what it prints and how it ended.
	RunShell(ctx context.Context, computerID string, run shelljob.Run, sink shelljob.Sink) error
	// CancelShell asks the runner running jobID to stop it; its end still arrives through the sink.
	CancelShell(ctx context.Context, jobID string) error
}

// ShellJobStore persists shell jobs; each change carries its event for the outbox in the same transaction.
type ShellJobStore interface {
	shelljob.Store
	GetShellJob(ctx context.Context, id string) (*shelljob.Job, error)
	// ListShellJobs pages a computer's jobs newest first, without output; a requester id keeps only theirs.
	ListShellJobs(ctx context.Context, computerID, requesterID string, w paging.Window) ([]shelljob.Job, int, error)
	// PurgeShellOutput empties the output of up to limit jobs that ended before before.
	PurgeShellOutput(ctx context.Context, before time.Time, limit int) (int64, error)
	// FailUnfinishedShellJobs ends every job still queued or running, which a restart left with nobody waiting on it.
	FailUnfinishedShellJobs(ctx context.Context, at time.Time) (int64, error)
	// SetComputerShell sets one of userID's own computers' switch for commands from Nexul, or ErrNotFound.
	SetComputerShell(ctx context.Context, userID, computerID string, enabled bool, at time.Time) error
}

// newShellQueue is the queue of commands on computers, handed to each computer's runner.
func newShellQueue(s *Service) *shelljob.Queue {
	return shelljob.NewQueue(shelljob.Config{
		Store: s.repo, Runners: computerRunners{s}, Event: shellJobEvent, Now: s.now,
		StartError: shellStartError, Silent: "the computer's Nexul app did not say how the command ended",
	})
}

// computerRunners hands queued commands to their computers' runners.
type computerRunners struct{ s *Service }

func (r computerRunners) Start(ctx context.Context, job shelljob.Job, run shelljob.Run, sink shelljob.Sink) error {
	return r.s.shells.RunShell(ctx, job.ComputerID, run, sink)
}

func (r computerRunners) Cancel(ctx context.Context, jobID string) error {
	return r.s.shells.CancelShell(ctx, jobID)
}

func (r computerRunners) StillAllowed(ctx context.Context, job shelljob.Job) (shelljob.End, bool) {
	return r.s.stillAllowed(ctx, job)
}

// RunCommand records a command on one of the caller's computers and queues it; it returns the job at once, queued or
// already running. The computer's own opt-in is its runner's to check, so a refusal there is a recorded job.
func (s *Service) RunCommand(ctx context.Context, userID, computerID string, in shelljob.Input) (*shelljob.Job, error) {
	if err := shelljob.Validate(&in); err != nil {
		return nil, err
	}
	c, err := s.commandsComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if !c.ShellEnabled {
		return nil, fmt.Errorf("%w: commands from Nexul are turned off for %s; its owner turns them on under Commands", apperrs.ErrConflict, c.Name)
	}
	job := shelljob.Job{
		ID: ids.New(), ComputerID: c.ID, OwnerID: c.UserID, RequesterID: userID, Adapter: in.Adapter, Shell: in.Shell,
		Command: in.Command, Cwd: in.Cwd, TimeoutSeconds: int(in.Timeout / time.Second), Status: shelljob.Queued,
		CreatedAt: s.now().UTC(),
	}
	if err := s.shellQueue.Add(ctx, job); err != nil {
		return nil, fmt.Errorf("record command on %s: %w", c.Name, err)
	}
	auditnote.Job(ctx, c.ID, c.UserID, job.ID)
	return &job, nil
}

// commandsComputer is a computer the caller owns, or one shared with them with Run commands; any other is not found.
func (s *Service) commandsComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	if s.shells == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: commands on computers are not wired", apperrs.ErrFatal))
	}
	return s.computerWith(ctx, userID, computerID, permissions.ComputersCommands)
}

// stillAllowed checks again, as a queued job reaches the front, that its computer is there and takes commands.
func (s *Service) stillAllowed(ctx context.Context, job shelljob.Job) (shelljob.End, bool) {
	c, err := s.commandsComputer(ctx, job.RequesterID, job.ComputerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return shelljob.End{Status: shelljob.Cancelled, Error: "access revoked"}, false
	}
	if err != nil {
		return shelljob.End{Status: shelljob.Failed, Error: "could not check the computer before running"}, false
	}
	if !c.ShellEnabled {
		return shelljob.End{Status: shelljob.Cancelled, Error: "commands from Nexul were turned off for this computer"}, false
	}
	return shelljob.End{}, true
}

// shellStartError is what a job that never reached its runner says: the runner's own reason, or that it is offline.
func shellStartError(err error) string {
	if errors.Is(err, apperrs.ErrConflict) {
		return strings.TrimPrefix(err.Error(), apperrs.ErrConflict.Error()+": ")
	}
	return "the computer's Nexul app is not connected"
}

// GetShellJob returns one job with its output, to its computer's owner or the person who ran it.
func (s *Service) GetShellJob(ctx context.Context, userID, jobID string) (*shelljob.Job, error) {
	job, err := s.repo.GetShellJob(ctx, jobID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("%w: command %s", apperrs.ErrNotFound, jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("get command %s: %w", jobID, err)
	}
	if !s.mayReadJob(ctx, userID, *job) {
		return nil, fmt.Errorf("%w: command %s", apperrs.ErrNotFound, jobID)
	}
	s.shellQueue.Overlay(job)
	return job, nil
}

// mayReadJob is whether userID owns the job's computer or ran the job; nobody else reads it.
func (s *Service) mayReadJob(_ context.Context, userID string, job shelljob.Job) bool {
	return userID != "" && job.ComputerID != "" && (userID == job.OwnerID || userID == job.RequesterID)
}

// ListShellJobs pages a computer's commands newest first, without output: every job to its owner, and to a person it
// is shared with by Run commands only their own.
func (s *Service) ListShellJobs(ctx context.Context, userID, computerID string, w paging.Window) ([]shelljob.Job, int, error) {
	c, err := s.computerWith(ctx, userID, computerID, permissions.ComputersCommands)
	if err != nil {
		return nil, 0, err
	}
	requester := ""
	if c.UserID != userID {
		requester = userID
	}
	jobs, total, err := s.repo.ListShellJobs(ctx, c.ID, requester, w.Clamped())
	if err != nil {
		return nil, 0, fmt.Errorf("list commands on %s: %w", c.Name, err)
	}
	return jobs, total, nil
}

// WaitShellJob waits for a job to end and returns it with its output; a ctx that ends first returns it as it is.
func (s *Service) WaitShellJob(ctx context.Context, userID, jobID string) (*shelljob.Job, error) {
	s.shellQueue.Wait(ctx, jobID)
	return s.GetShellJob(context.WithoutCancel(ctx), userID, jobID)
}

// CancelShellJob stops a queued or running job, for its computer's owner or the person who ran it.
func (s *Service) CancelShellJob(ctx context.Context, userID, jobID string) (*shelljob.Job, error) {
	job, err := s.GetShellJob(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	s.shellQueue.Cancel(ctx, job.ID, "")
	return s.WaitShellJob(ctx, userID, jobID)
}

// SetShellEnabled turns commands from Nexul on or off for one of the caller's own computers; off stops every job on it.
func (s *Service) SetShellEnabled(ctx context.Context, userID, computerID string, enabled bool) (*Computer, error) {
	c, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetComputerShell(ctx, userID, c.ID, enabled, s.now().UTC()); err != nil {
		return nil, err
	}
	c.ShellEnabled = enabled
	c.BearerToken = ""
	if !enabled {
		s.cancelComputerJobs(ctx, c.ID, "", "commands from Nexul were turned off for this computer")
	}
	return c, nil
}

// cancelComputerJobs stops every live job on computerID, or only requesterID's when it is set, saying why.
func (s *Service) cancelComputerJobs(ctx context.Context, computerID, requesterID, why string) {
	s.shellQueue.CancelWhere(ctx, func(job shelljob.Job) bool {
		return job.ComputerID == computerID && (requesterID == "" || job.RequesterID == requesterID)
	}, why)
}

// HandleComputerGrantChanged stops what a person ran on a computer shared with them once its rule no longer gives them
// Run commands: queued jobs are dropped and running ones killed, both as access revoked.
func (s *Service) HandleComputerGrantChanged(ctx context.Context, ev eventbus.Event) error {
	var p struct {
		ComputerID  string   `json:"computer_id"`
		UserID      string   `json:"user_id"`
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.UserID == "" || slices.Contains(p.Permissions, string(permissions.ComputersCommands)) {
		return nil
	}
	s.cancelComputerJobs(ctx, p.ComputerID, p.UserID, "access revoked")
	return nil
}

// OpenShellOutput streams a job's output from its first line: live while it runs, what was kept once it ended.
func (s *Service) OpenShellOutput(ctx context.Context, userID, jobID string) (containerlog.Feed, error) {
	job, err := s.GetShellJob(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	return s.shellQueue.Open(*job), nil
}

func shellJobEvent(job shelljob.Job) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicShellJobChanged, Payload: ShellJobChangedEvent{
		JobID: job.ID, ComputerID: job.ComputerID, OwnerID: job.OwnerID, UserID: job.RequesterID, Status: job.Status,
		MembersOnly: true,
	}}
}

// RunShellUpkeep ends the jobs a restart left unfinished, a machine's too, then empties output older than
// shelljob.OutputRetention at start and daily after that, until ctx is cancelled.
func (s *Service) RunShellUpkeep(ctx context.Context, log *slog.Logger) {
	if n, err := s.repo.FailUnfinishedShellJobs(ctx, s.now().UTC()); err != nil || n > 0 {
		log.Info("ended the commands a restart left unfinished", "jobs", n, "error", err)
	}
	ticker := time.NewTicker(shellPurgeInterval)
	defer ticker.Stop()
	for {
		n, err := s.PurgeShellOutput(ctx)
		if err != nil {
			log.Warn("command output purge failed", "emptied", n, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// PurgeShellOutput empties the output of every job, a computer's or a machine's, that ended more than
// shelljob.OutputRetention ago, a batch at a time.
func (s *Service) PurgeShellOutput(ctx context.Context) (int64, error) {
	before := s.now().UTC().Add(-shelljob.OutputRetention)
	var total int64
	for {
		n, err := s.repo.PurgeShellOutput(ctx, before, shellPurgeBatch)
		total += n
		if err != nil {
			return total, fmt.Errorf("empty command output before %s: %w", before, err)
		}
		if n < shellPurgeBatch {
			return total, nil
		}
	}
}
