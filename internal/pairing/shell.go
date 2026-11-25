package pairing

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Shell job limits (the personal runners spec, Later: shell jobs).
const (
	DefaultShellTimeout = 60 * time.Second
	MaxShellTimeout     = time.Hour
	// shellJobsAtOnce is how many jobs run on one computer at a time; the rest wait here, in memory.
	shellJobsAtOnce = 4
	// maxShellOutput is the output a job keeps, its last MiB.
	maxShellOutput  = 1 << 20
	maxShellCommand = 64 << 10
	// ShellOutputRetention is how long a finished job keeps its output; the rest of the record is kept forever.
	ShellOutputRetention = 30 * 24 * time.Hour
	shellPurgeBatch      = 500
	shellPurgeInterval   = 24 * time.Hour
	// shellAnswerGrace is how long past its timeout a job waits for its runner's answer before it fails.
	shellAnswerGrace = 30 * time.Second
)

// Shell job statuses: queued and running are live; the rest are how it ended.
const (
	ShellQueued    = "queued"
	ShellRunning   = "running"
	ShellSucceeded = "succeeded"
	ShellFailed    = "failed"
	ShellTimedOut  = "timed_out"
	ShellCancelled = "cancelled"
	ShellRefused   = "refused"
)

// Adapters a shell job is started through.
const (
	AdapterWeb = "web"
	AdapterMCP = "mcp"
)

// shellNames are the shells a job may name; empty is the computer's default.
var shellNames = []string{"", "bash", "sh", "pwsh", "powershell"}

// ShellJob is one command run on a computer through its runner.
type ShellJob struct {
	ID          string `json:"id"`
	ComputerID  string `json:"computer_id"`
	OwnerID     string `json:"owner_id"`
	RequesterID string `json:"requester_id"`
	// RequesterName is who ran it as the list shows them; set on a list only.
	RequesterName   string     `json:"requester_name,omitempty"`
	Adapter         string     `json:"adapter"`
	Shell           string     `json:"shell"`
	Command         string     `json:"command"`
	Cwd             string     `json:"cwd"`
	TimeoutSeconds  int        `json:"timeout_seconds"`
	Status          string     `json:"status"`
	ExitCode        *int       `json:"exit_code"`
	Error           string     `json:"error"`
	Output          string     `json:"output,omitempty"`
	OutputTruncated bool       `json:"output_truncated"`
	OutputPurged    bool       `json:"output_purged"`
	CreatedAt       time.Time  `json:"created_at"`
	StartedAt       *time.Time `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at"`
}

// RunCommandInput is what a person asks to run on a computer.
type RunCommandInput struct {
	Command string
	// Shell is bash or sh on Linux and macOS, pwsh or powershell on Windows; empty is the computer's default.
	Shell string
	// Cwd is the folder it starts in; empty is the home folder of the person the runner runs as.
	Cwd     string
	Timeout time.Duration
	Adapter string
}

// ShellRun is one command for a computer's runner to run.
type ShellRun struct {
	ID      string
	Shell   string
	Command string
	Cwd     string
	Timeout time.Duration
}

// ShellEnd is how a job ended on its runner, or why the server ended it.
type ShellEnd struct {
	Status   string
	ExitCode int
	Error    string
}

// ShellSink receives one running job's output and then its end, once.
type ShellSink interface {
	Output(lines []containerlog.Line)
	Done(end ShellEnd)
}

// Shells runs commands on a computer through its runner.
type Shells interface {
	// RunShell sends run to the computer's runner; sink gets what it prints and how it ended.
	RunShell(ctx context.Context, computerID string, run ShellRun, sink ShellSink) error
	// CancelShell asks the runner running jobID to stop it; its end still arrives through the sink.
	CancelShell(ctx context.Context, jobID string) error
}

// ShellJobStore persists shell jobs; each change carries its event for the outbox in the same transaction.
type ShellJobStore interface {
	InsertShellJob(ctx context.Context, job ShellJob, evt eventbus.OutboxEvent) error
	// StartShellJob marks a queued job running; false when it was no longer queued.
	StartShellJob(ctx context.Context, id string, at time.Time, evt eventbus.OutboxEvent) (bool, error)
	// FinishShellJob stores a live job's end and output; a job that already ended is left as it was.
	FinishShellJob(ctx context.Context, job ShellJob, evt eventbus.OutboxEvent) error
	GetShellJob(ctx context.Context, id string) (*ShellJob, error)
	// ListShellJobs pages a computer's jobs newest first, without output; a requester id keeps only theirs.
	ListShellJobs(ctx context.Context, computerID, requesterID string, w paging.Window) ([]ShellJob, int, error)
	// PurgeShellOutput empties the output of up to limit jobs that ended before before.
	PurgeShellOutput(ctx context.Context, before time.Time, limit int) (int64, error)
	// FailUnfinishedShellJobs ends every job still queued or running, which a restart left with nobody waiting on it.
	FailUnfinishedShellJobs(ctx context.Context, at time.Time) (int64, error)
	// SetComputerShell sets one of userID's own computers' switch for commands from Nexul, or ErrNotFound.
	SetComputerShell(ctx context.Context, userID, computerID string, enabled bool, at time.Time) error
}

// liveJob is a queued or running job: its output so far, who watches it, and its end once it has one.
type liveJob struct {
	job  ShellJob
	svc  *Service
	done chan struct{}
	// claimed is set, under the queue's lock, once the job got a place on its computer.
	claimed bool
	// requester is who ran it, never changed, so the queue reads it without the job's lock.
	requester string

	mu        sync.Mutex
	timer     *time.Timer
	lines     []containerlog.Line
	size      int
	truncated bool
	watchers  map[*jobFeed]struct{}
	ended     bool
}

// shellQueue holds every live job per computer, in the order they were asked for.
type shellQueue struct {
	mu   sync.Mutex
	jobs map[string]*liveJob
	// order lists each computer's live job ids, oldest first; the first shellJobsAtOnce of them run.
	order map[string][]string
}

func newShellQueue() *shellQueue {
	return &shellQueue{jobs: map[string]*liveJob{}, order: map[string][]string{}}
}

// RunCommand records a command on one of the caller's computers and queues it; it returns the job at once, queued or
// already running. The computer's own opt-in is its runner's to check, so a refusal there is a recorded job.
func (s *Service) RunCommand(ctx context.Context, userID, computerID string, in RunCommandInput) (*ShellJob, error) {
	if err := validateCommand(&in); err != nil {
		return nil, err
	}
	c, err := s.commandsComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	if !c.ShellEnabled {
		return nil, fmt.Errorf("%w: commands from Nexul are turned off for %s; its owner turns them on under Commands", apperrs.ErrConflict, c.Name)
	}
	job := ShellJob{
		ID: ids.New(), ComputerID: c.ID, OwnerID: c.UserID, RequesterID: userID, Adapter: in.Adapter, Shell: in.Shell,
		Command: in.Command, Cwd: in.Cwd, TimeoutSeconds: int(in.Timeout / time.Second), Status: ShellQueued,
		CreatedAt: s.now().UTC(),
	}
	if err := s.repo.InsertShellJob(ctx, job, shellJobEvent(job)); err != nil {
		return nil, fmt.Errorf("record command on %s: %w", c.Name, err)
	}
	live := &liveJob{job: job, svc: s, requester: userID, done: make(chan struct{}), watchers: map[*jobFeed]struct{}{}}
	s.shellQueue.mu.Lock()
	s.shellQueue.jobs[job.ID] = live
	s.shellQueue.order[job.ComputerID] = append(s.shellQueue.order[job.ComputerID], job.ID)
	s.shellQueue.mu.Unlock()
	s.startDueJobs(job.ComputerID)
	return &job, nil
}

func validateCommand(in *RunCommandInput) error {
	in.Shell = strings.TrimSpace(in.Shell)
	in.Cwd = strings.TrimSpace(in.Cwd)
	if strings.TrimSpace(in.Command) == "" {
		return fmt.Errorf("%w: a command is required", apperrs.ErrInvalid)
	}
	if len(in.Command) > maxShellCommand {
		return fmt.Errorf("%w: a command is at most %d KB", apperrs.ErrInvalid, maxShellCommand>>10)
	}
	if !slices.Contains(shellNames, in.Shell) {
		return fmt.Errorf("%w: shell must be bash, sh, pwsh or powershell, or empty for the computer's default", apperrs.ErrInvalid)
	}
	if in.Timeout == 0 {
		in.Timeout = DefaultShellTimeout
	}
	if in.Timeout < time.Second || in.Timeout > MaxShellTimeout {
		return fmt.Errorf("%w: timeout must be 1 second to 1 hour", apperrs.ErrInvalid)
	}
	in.Timeout = in.Timeout.Truncate(time.Second)
	if in.Adapter == "" {
		in.Adapter = AdapterWeb
	}
	return nil
}

// commandsComputer is a computer the caller owns, or one shared with them with Run commands; any other is not found.
func (s *Service) commandsComputer(ctx context.Context, userID, computerID string) (*Computer, error) {
	if s.shells == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: commands on computers are not wired", apperrs.ErrFatal))
	}
	return s.computerWith(ctx, userID, computerID, permissions.ComputersCommands)
}

// startDueJobs starts the queued jobs on computerID that now fit in its shellJobsAtOnce.
func (s *Service) startDueJobs(computerID string) {
	s.shellQueue.mu.Lock()
	var due []*liveJob
	for i, id := range s.shellQueue.order[computerID] {
		if i >= shellJobsAtOnce {
			break
		}
		live := s.shellQueue.jobs[id]
		if !live.claimed {
			live.claimed = true
			due = append(due, live)
		}
	}
	s.shellQueue.mu.Unlock()
	for _, live := range due {
		s.shellRuns.Go(func() { s.startJob(live) })
	}
}

// startJob checks the job may still run, marks it running and hands it to the computer's runner.
func (s *Service) startJob(live *liveJob) {
	ctx := context.Background()
	live.mu.Lock()
	job, ended := live.job, live.ended
	live.mu.Unlock()
	if ended {
		return
	}
	if end, ok := s.stillAllowed(ctx, job); !ok {
		live.Done(end)
		return
	}
	at := s.now().UTC()
	job.Status, job.StartedAt = ShellRunning, &at
	started, err := s.repo.StartShellJob(ctx, job.ID, at, shellJobEvent(job))
	if err != nil {
		live.Done(ShellEnd{Status: ShellFailed, Error: "could not record the command's start"})
		logging.FromCtx(ctx).Error("start shell job", "job_id", job.ID, "error", err)
		return
	}
	live.mu.Lock()
	if !started || live.ended {
		live.mu.Unlock()
		return
	}
	live.job.Status, live.job.StartedAt = ShellRunning, &at
	live.mu.Unlock()
	run := ShellRun{ID: job.ID, Shell: job.Shell, Command: job.Command, Cwd: job.Cwd, Timeout: time.Duration(job.TimeoutSeconds) * time.Second}
	if err := s.shells.RunShell(ctx, job.ComputerID, run, live); err != nil {
		live.Done(ShellEnd{Status: ShellFailed, Error: shellStartError(err)})
		return
	}
	timer := time.AfterFunc(run.Timeout+shellAnswerGrace, func() {
		live.Done(ShellEnd{Status: ShellFailed, Error: "the computer's Nexul app did not say how the command ended"})
	})
	live.mu.Lock()
	live.timer = timer
	live.mu.Unlock()
}

// stillAllowed checks again, as a queued job reaches the front, that its computer is there and takes commands.
func (s *Service) stillAllowed(ctx context.Context, job ShellJob) (ShellEnd, bool) {
	c, err := s.commandsComputer(ctx, job.RequesterID, job.ComputerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return ShellEnd{Status: ShellCancelled, Error: "access revoked"}, false
	}
	if err != nil {
		return ShellEnd{Status: ShellFailed, Error: "could not check the computer before running"}, false
	}
	if !c.ShellEnabled {
		return ShellEnd{Status: ShellCancelled, Error: "commands from Nexul were turned off for this computer"}, false
	}
	return ShellEnd{}, true
}

// shellStartError is what a job that never reached its runner says: the runner's own reason, or that it is offline.
func shellStartError(err error) string {
	if errors.Is(err, apperrs.ErrConflict) {
		return strings.TrimPrefix(err.Error(), apperrs.ErrConflict.Error()+": ")
	}
	return "the computer's Nexul app is not connected"
}

// Output keeps the last maxShellOutput bytes of what the job printed and passes the lines to its watchers.
func (l *liveJob) Output(lines []containerlog.Line) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ended {
		return
	}
	for _, line := range lines {
		l.lines = append(l.lines, line)
		l.size += len(line.Line) + 1
	}
	for l.size > maxShellOutput && len(l.lines) > 1 {
		l.size -= len(l.lines[0].Line) + 1
		l.lines, l.truncated = l.lines[1:], true
	}
	for w := range l.watchers {
		w.push(lines)
	}
}

// Done stores how the job ended, frees its place on the computer and starts the next; only the first end counts.
func (l *liveJob) Done(end ShellEnd) {
	l.mu.Lock()
	if l.ended {
		l.mu.Unlock()
		return
	}
	l.ended = true
	if l.timer != nil {
		l.timer.Stop()
	}
	job := l.job
	job.Status, job.Error = end.Status, end.Error
	if end.Status == ShellSucceeded || (end.Status == ShellFailed && end.Error == "") {
		code := end.ExitCode
		job.ExitCode = &code
	}
	job.Output, job.OutputTruncated = joinLines(l.lines), l.truncated
	at := l.svc.now().UTC()
	job.EndedAt = &at
	l.job = job
	watchers := l.watchers
	l.watchers = nil
	l.mu.Unlock()
	ctx := context.Background()
	if err := l.svc.repo.FinishShellJob(ctx, job, shellJobEvent(job)); err != nil {
		logging.FromCtx(ctx).Error("finish shell job", "job_id", job.ID, "error", err)
	}
	for w := range watchers {
		w.finish()
	}
	close(l.done)
	l.svc.forgetJob(job)
}

func (s *Service) forgetJob(job ShellJob) {
	s.shellQueue.mu.Lock()
	delete(s.shellQueue.jobs, job.ID)
	order := slices.DeleteFunc(s.shellQueue.order[job.ComputerID], func(id string) bool { return id == job.ID })
	s.shellQueue.order[job.ComputerID] = order
	if len(order) == 0 {
		delete(s.shellQueue.order, job.ComputerID)
	}
	s.shellQueue.mu.Unlock()
	s.startDueJobs(job.ComputerID)
}

func joinLines(lines []containerlog.Line) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Line)
		b.WriteByte('\n')
	}
	return b.String()
}

// liveJob returns a queued or running job by id, nil once it ended.
func (s *Service) liveJob(id string) *liveJob {
	s.shellQueue.mu.Lock()
	defer s.shellQueue.mu.Unlock()
	return s.shellQueue.jobs[id]
}

// GetShellJob returns one job with its output, to its computer's owner or the person who ran it.
func (s *Service) GetShellJob(ctx context.Context, userID, jobID string) (*ShellJob, error) {
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
	if live := s.liveJob(jobID); live != nil {
		live.mu.Lock()
		if !live.ended {
			job.Output, job.OutputTruncated = joinLines(live.lines), live.truncated
		}
		live.mu.Unlock()
	}
	return job, nil
}

// mayReadJob is whether userID owns the job's computer or ran the job; nobody else reads it.
func (s *Service) mayReadJob(_ context.Context, userID string, job ShellJob) bool {
	return userID != "" && (userID == job.OwnerID || userID == job.RequesterID)
}

// ListShellJobs pages a computer's commands newest first, without output: every job to its owner, and to a person it
// is shared with by Run commands only their own.
func (s *Service) ListShellJobs(ctx context.Context, userID, computerID string, w paging.Window) ([]ShellJob, int, error) {
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
func (s *Service) WaitShellJob(ctx context.Context, userID, jobID string) (*ShellJob, error) {
	if live := s.liveJob(jobID); live != nil {
		select {
		case <-live.done:
		case <-ctx.Done():
		}
	}
	return s.GetShellJob(context.WithoutCancel(ctx), userID, jobID)
}

// CancelShellJob stops a queued or running job, for its computer's owner or the person who ran it.
func (s *Service) CancelShellJob(ctx context.Context, userID, jobID string) (*ShellJob, error) {
	job, err := s.GetShellJob(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	s.cancelLive(ctx, job.ID, "")
	return s.WaitShellJob(ctx, userID, jobID)
}

// cancelLive ends a queued job here, or asks the runner to stop a running one; why replaces the runner's reason.
func (s *Service) cancelLive(ctx context.Context, jobID, why string) {
	live := s.liveJob(jobID)
	if live == nil {
		return
	}
	live.mu.Lock()
	started := live.job.StartedAt != nil
	live.mu.Unlock()
	if !started {
		live.Done(ShellEnd{Status: ShellCancelled, Error: why})
		return
	}
	if err := s.shells.CancelShell(ctx, jobID); err != nil {
		live.Done(ShellEnd{Status: ShellCancelled, Error: cmp.Or(why, "the stop could not reach the computer")})
		return
	}
	if why != "" {
		live.Done(ShellEnd{Status: ShellCancelled, Error: why})
	}
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
	s.shellQueue.mu.Lock()
	var jobIDs []string
	for _, id := range s.shellQueue.order[computerID] {
		if requesterID == "" || s.shellQueue.jobs[id].requester == requesterID {
			jobIDs = append(jobIDs, id)
		}
	}
	s.shellQueue.mu.Unlock()
	for _, id := range jobIDs {
		s.cancelLive(ctx, id, why)
	}
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
	feed := &jobFeed{ready: make(chan struct{}, 1)}
	live := s.liveJob(jobID)
	if live != nil {
		live.mu.Lock()
		if !live.ended {
			feed.lines = slices.Clone(live.lines)
			live.watchers[feed] = struct{}{}
			feed.stop = func() {
				live.mu.Lock()
				delete(live.watchers, feed)
				live.mu.Unlock()
			}
			live.mu.Unlock()
			return feed, nil
		}
		live.mu.Unlock()
	}
	for line := range strings.Lines(job.Output) {
		feed.lines = append(feed.lines, containerlog.Line{Stream: "stdout", Line: strings.TrimSuffix(line, "\n")})
	}
	feed.done = true
	return feed, nil
}

// jobFeed is one viewer's stream of a job's output.
type jobFeed struct {
	ready chan struct{}
	stop  func()
	once  sync.Once

	mu      sync.Mutex
	lines   []containerlog.Line
	skipped int
	done    bool
}

// jobFeedBuffer bounds the lines held for one viewer of a running job.
const jobFeedBuffer = 4000

// push queues lines for the viewer, past jobFeedBuffer counting them as skipped so a slow viewer never holds a job up.
func (f *jobFeed) push(lines []containerlog.Line) {
	f.mu.Lock()
	keep := min(max(jobFeedBuffer-len(f.lines), 0), len(lines))
	f.lines = append(f.lines, lines[:keep]...)
	f.skipped += len(lines) - keep
	f.mu.Unlock()
	f.signal()
}

func (f *jobFeed) finish() {
	f.mu.Lock()
	f.done = true
	f.mu.Unlock()
	f.signal()
}

func (f *jobFeed) signal() {
	select {
	case f.ready <- struct{}{}:
	default:
	}
}

// Next returns the lines since the last call, io.EOF once the job ended and every line was read.
func (f *jobFeed) Next(ctx context.Context) ([]containerlog.Line, error) {
	for {
		f.mu.Lock()
		if len(f.lines) > 0 || f.skipped > 0 {
			out := f.lines
			if f.skipped > 0 {
				out = append(out, containerlog.Line{Stream: "stdout", Line: fmt.Sprintf("… %d lines skipped", f.skipped)})
			}
			f.lines, f.skipped = nil, 0
			f.mu.Unlock()
			return out, nil
		}
		done := f.done
		f.mu.Unlock()
		if done {
			return nil, io.EOF
		}
		select {
		case <-f.ready:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Close stops following the job; safe to call twice.
func (f *jobFeed) Close() {
	f.once.Do(func() {
		if f.stop != nil {
			f.stop()
		}
	})
}

func shellJobEvent(job ShellJob) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicShellJobChanged, Payload: ShellJobChangedEvent{
		JobID: job.ID, ComputerID: job.ComputerID, OwnerID: job.OwnerID, UserID: job.RequesterID, Status: job.Status,
		MembersOnly: true,
	}}
}

// RunShellUpkeep ends the jobs a restart left unfinished, then empties output older than ShellOutputRetention at start
// and daily after that, until ctx is cancelled.
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

// PurgeShellOutput empties the output of every job that ended more than ShellOutputRetention ago, a batch at a time.
func (s *Service) PurgeShellOutput(ctx context.Context) (int64, error) {
	before := s.now().UTC().Add(-ShellOutputRetention)
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
