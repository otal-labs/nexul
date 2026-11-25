// Package shelljob is the server side of a shell job (ADR 0209): the record, its checks on input, and the queue that
// runs four at a time per runner and keeps a running job's output for its viewers. A paired computer's jobs and a
// machine's runners' jobs share it; each domain decides who may start and read its own.
package shelljob

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Limits (the personal runners spec, Later: shell jobs).
const (
	DefaultTimeout = 60 * time.Second
	MaxTimeout     = time.Hour
	// AtOnce is how many jobs run on one runner at a time; the rest wait in the queue, in memory.
	AtOnce = 4
	// MaxOutput is the output a job keeps, its last MiB.
	MaxOutput  = 1 << 20
	maxCommand = 64 << 10
	// OutputRetention is how long a finished job keeps its output; the rest of the record is kept forever.
	OutputRetention = 30 * 24 * time.Hour
	// AnswerGrace is how long past its timeout a job waits for its runner's answer before it fails.
	AnswerGrace = 30 * time.Second
	// AgentTail is the end of a job's output an agent reads back.
	AgentTail = 64 << 10
)

// Statuses: queued and running are live; the rest are how a job ended.
const (
	Queued    = "queued"
	Running   = "running"
	Succeeded = "succeeded"
	Failed    = "failed"
	TimedOut  = "timed_out"
	Cancelled = "cancelled"
	Refused   = "refused"
)

// Adapters a job is started through.
const (
	AdapterWeb = "web"
	AdapterMCP = "mcp"
)

// shellNames are the shells a job may name; empty is the runner's default.
var shellNames = []string{"", "bash", "sh", "pwsh", "powershell"}

// Job is one command run through a runner: on a paired computer (ComputerID, OwnerID) or on a machine (RunnerID,
// MachineID).
type Job struct {
	ID          string `json:"id"`
	ComputerID  string `json:"computer_id,omitempty"`
	OwnerID     string `json:"owner_id,omitempty"`
	RunnerID    string `json:"runner_id,omitempty"`
	MachineID   string `json:"machine_id,omitempty"`
	RequesterID string `json:"requester_id"`
	// RequesterName and RunnerName are who ran it and where, as a list shows them; set on a list only.
	RequesterName   string     `json:"requester_name,omitempty"`
	RunnerName      string     `json:"runner_name,omitempty"`
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

// TailForAgent cuts j's output to its last AgentTail bytes, marking it truncated when it cut any.
func (j *Job) TailForAgent() {
	if len(j.Output) > AgentTail {
		j.Output, j.OutputTruncated = j.Output[len(j.Output)-AgentTail:], true
	}
}

// runnerKey is the runner a job waits its turn on: its computer's, or its own on a machine.
func (j Job) runnerKey() string {
	return cmp.Or(j.ComputerID, j.RunnerID)
}

// Input is what a person asks to run.
type Input struct {
	Command string
	// Shell is bash or sh on Linux and macOS, pwsh or powershell on Windows; empty is the runner's default.
	Shell string
	// Cwd is the folder it starts in; empty is the home folder of the user the runner runs as.
	Cwd     string
	Timeout time.Duration
	Adapter string
}

// Validate checks in and fills its defaults: the timeout, and the web as its adapter.
func Validate(in *Input) error {
	in.Shell = strings.TrimSpace(in.Shell)
	in.Cwd = strings.TrimSpace(in.Cwd)
	if strings.TrimSpace(in.Command) == "" {
		return fmt.Errorf("%w: a command is required", apperrs.ErrInvalid)
	}
	if len(in.Command) > maxCommand {
		return fmt.Errorf("%w: a command is at most %d KB", apperrs.ErrInvalid, maxCommand>>10)
	}
	if !slices.Contains(shellNames, in.Shell) {
		return fmt.Errorf("%w: shell must be bash, sh, pwsh or powershell, or empty for the default", apperrs.ErrInvalid)
	}
	if in.Timeout == 0 {
		in.Timeout = DefaultTimeout
	}
	if in.Timeout < time.Second || in.Timeout > MaxTimeout {
		return fmt.Errorf("%w: timeout must be 1 second to 1 hour", apperrs.ErrInvalid)
	}
	in.Timeout = in.Timeout.Truncate(time.Second)
	if in.Adapter == "" {
		in.Adapter = AdapterWeb
	}
	return nil
}

// Run is one command for a runner to run.
type Run struct {
	ID      string
	Shell   string
	Command string
	Cwd     string
	Timeout time.Duration
}

// End is how a job ended on its runner, or why the server ended it.
type End struct {
	Status   string
	ExitCode int
	Error    string
}

// Sink receives one running job's output and then its end, once.
type Sink interface {
	Output(lines []containerlog.Line)
	Done(end End)
}

// Store records a queue's jobs; each change carries its event for the outbox in the same transaction.
type Store interface {
	InsertShellJob(ctx context.Context, job Job, evt eventbus.OutboxEvent) error
	// StartShellJob marks a queued job running; false when it was no longer queued.
	StartShellJob(ctx context.Context, id string, at time.Time, evt eventbus.OutboxEvent) (bool, error)
	// FinishShellJob stores a live job's end and output; a job that already ended is left as it was.
	FinishShellJob(ctx context.Context, job Job, evt eventbus.OutboxEvent) error
}

// Runners hands a queue's jobs to their runners.
type Runners interface {
	// Start sends run to job's runner; sink gets what it prints and how it ended.
	Start(ctx context.Context, job Job, run Run, sink Sink) error
	// Cancel asks the runner running jobID to stop it; its end still arrives through the sink.
	Cancel(ctx context.Context, jobID string) error
	// StillAllowed checks a queued job again as it reaches the front; false ends it with end.
	StillAllowed(ctx context.Context, job Job) (end End, ok bool)
}

// Config wires a queue.
type Config struct {
	Store   Store
	Runners Runners
	// Event is the outbox event every change of a job carries.
	Event func(Job) eventbus.OutboxEvent
	Now   func() time.Time
	// StartError is what a job that never reached its runner says, from Start's error.
	StartError func(error) string
	// Silent is what a job whose runner never said how it ended says.
	Silent string
}

// Queue holds every live job, per runner in the order they were asked for, the first AtOnce of each running.
type Queue struct {
	cfg  Config
	runs sync.WaitGroup

	mu   sync.Mutex
	jobs map[string]*live
	// order lists each runner's live job ids, oldest first.
	order map[string][]string
}

// NewQueue builds an empty queue.
func NewQueue(cfg Config) *Queue {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Queue{cfg: cfg, jobs: map[string]*live{}, order: map[string][]string{}}
}

// Drain waits for the goroutines that hand jobs to their runners.
func (q *Queue) Drain() {
	q.runs.Wait()
}

// Add records a queued job and starts it once its runner has room.
func (q *Queue) Add(ctx context.Context, job Job) error {
	if err := q.cfg.Store.InsertShellJob(ctx, job, q.cfg.Event(job)); err != nil {
		return err
	}
	l := &live{job: job, asked: job, q: q, done: make(chan struct{}), watchers: map[*feed]struct{}{}}
	key := job.runnerKey()
	q.mu.Lock()
	q.jobs[job.ID] = l
	q.order[key] = append(q.order[key], job.ID)
	q.mu.Unlock()
	q.startDue(key)
	return nil
}

// live is a queued or running job: its output so far, who watches it, and its end once it has one.
type live struct {
	job Job
	// asked is the job as it was queued, never changed, so the queue reads it without the job's lock.
	asked Job
	q     *Queue
	done  chan struct{}
	// claimed is set, under the queue's lock, once the job got a place on its runner.
	claimed bool

	mu        sync.Mutex
	timer     *time.Timer
	lines     []containerlog.Line
	size      int
	truncated bool
	watchers  map[*feed]struct{}
	ended     bool
}

// startDue starts the queued jobs on key's runner that now fit in its AtOnce.
func (q *Queue) startDue(key string) {
	q.mu.Lock()
	var due []*live
	for i, id := range q.order[key] {
		if i >= AtOnce {
			break
		}
		l := q.jobs[id]
		if !l.claimed {
			l.claimed = true
			due = append(due, l)
		}
	}
	q.mu.Unlock()
	for _, l := range due {
		q.runs.Go(func() { q.start(l) })
	}
}

// start checks the job may still run, marks it running and hands it to its runner.
func (q *Queue) start(l *live) {
	ctx := context.Background()
	l.mu.Lock()
	job, ended := l.job, l.ended
	l.mu.Unlock()
	if ended {
		return
	}
	if end, ok := q.cfg.Runners.StillAllowed(ctx, job); !ok {
		l.Done(end)
		return
	}
	at := q.cfg.Now().UTC()
	job.Status, job.StartedAt = Running, &at
	started, err := q.cfg.Store.StartShellJob(ctx, job.ID, at, q.cfg.Event(job))
	if err != nil {
		l.Done(End{Status: Failed, Error: "could not record the command's start"})
		logging.FromCtx(ctx).Error("start shell job", "job_id", job.ID, "error", err)
		return
	}
	l.mu.Lock()
	if !started || l.ended {
		l.mu.Unlock()
		return
	}
	l.job.Status, l.job.StartedAt = Running, &at
	l.mu.Unlock()
	run := Run{ID: job.ID, Shell: job.Shell, Command: job.Command, Cwd: job.Cwd, Timeout: time.Duration(job.TimeoutSeconds) * time.Second}
	if err := q.cfg.Runners.Start(ctx, job, run, l); err != nil {
		l.Done(End{Status: Failed, Error: q.cfg.StartError(err)})
		return
	}
	timer := time.AfterFunc(run.Timeout+AnswerGrace, func() {
		l.Done(End{Status: Failed, Error: q.cfg.Silent})
	})
	l.mu.Lock()
	l.timer = timer
	l.mu.Unlock()
}

// Output keeps the last MaxOutput bytes of what the job printed and passes the lines to its watchers.
func (l *live) Output(lines []containerlog.Line) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ended {
		return
	}
	for _, line := range lines {
		l.lines = append(l.lines, line)
		l.size += len(line.Line) + 1
	}
	for l.size > MaxOutput && len(l.lines) > 1 {
		l.size -= len(l.lines[0].Line) + 1
		l.lines, l.truncated = l.lines[1:], true
	}
	for w := range l.watchers {
		w.push(lines)
	}
}

// Done stores how the job ended, frees its place on its runner and starts the next; only the first end counts.
func (l *live) Done(end End) {
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
	if end.Status == Succeeded || (end.Status == Failed && end.Error == "") {
		code := end.ExitCode
		job.ExitCode = &code
	}
	job.Output, job.OutputTruncated = joinLines(l.lines), l.truncated
	at := l.q.cfg.Now().UTC()
	job.EndedAt = &at
	l.job = job
	watchers := l.watchers
	l.watchers = nil
	l.mu.Unlock()
	ctx := context.Background()
	if err := l.q.cfg.Store.FinishShellJob(ctx, job, l.q.cfg.Event(job)); err != nil {
		logging.FromCtx(ctx).Error("finish shell job", "job_id", job.ID, "error", err)
	}
	for w := range watchers {
		w.finish()
	}
	close(l.done)
	l.q.forget(job)
}

func (q *Queue) forget(job Job) {
	key := job.runnerKey()
	q.mu.Lock()
	delete(q.jobs, job.ID)
	order := slices.DeleteFunc(q.order[key], func(id string) bool { return id == job.ID })
	q.order[key] = order
	if len(order) == 0 {
		delete(q.order, key)
	}
	q.mu.Unlock()
	q.startDue(key)
}

func joinLines(lines []containerlog.Line) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Line)
		b.WriteByte('\n')
	}
	return b.String()
}

func (q *Queue) live(id string) *live {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.jobs[id]
}

// Overlay puts a running job's output so far on job, read from storage, which holds it only once the job ends.
func (q *Queue) Overlay(job *Job) {
	l := q.live(job.ID)
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.ended {
		job.Output, job.OutputTruncated = joinLines(l.lines), l.truncated
	}
}

// Wait returns once jobID ended, at once when it is not live, or when ctx ends.
func (q *Queue) Wait(ctx context.Context, jobID string) {
	l := q.live(jobID)
	if l == nil {
		return
	}
	select {
	case <-l.done:
	case <-ctx.Done():
	}
}

// Cancel ends a queued job here, or asks its runner to stop a running one; why, when set, replaces the runner's reason.
func (q *Queue) Cancel(ctx context.Context, jobID, why string) {
	l := q.live(jobID)
	if l == nil {
		return
	}
	l.mu.Lock()
	started := l.job.StartedAt != nil
	l.mu.Unlock()
	if !started {
		l.Done(End{Status: Cancelled, Error: why})
		return
	}
	if err := q.cfg.Runners.Cancel(ctx, jobID); err != nil {
		l.Done(End{Status: Cancelled, Error: cmp.Or(why, "the stop could not reach the runner")})
		return
	}
	if why != "" {
		l.Done(End{Status: Cancelled, Error: why})
	}
}

// CancelWhere cancels every live job match picks, saying why.
func (q *Queue) CancelWhere(ctx context.Context, match func(Job) bool, why string) {
	q.mu.Lock()
	var ids []string
	for _, order := range q.order {
		for _, id := range order {
			if match(q.jobs[id].asked) {
				ids = append(ids, id)
			}
		}
	}
	q.mu.Unlock()
	for _, id := range ids {
		q.Cancel(ctx, id, why)
	}
}

// Open streams job's output from its first line: live while it runs, what was kept once it ended.
func (q *Queue) Open(job Job) containerlog.Feed {
	f := &feed{ready: make(chan struct{}, 1)}
	if l := q.live(job.ID); l != nil {
		l.mu.Lock()
		if !l.ended {
			f.lines = slices.Clone(l.lines)
			l.watchers[f] = struct{}{}
			f.stop = func() {
				l.mu.Lock()
				delete(l.watchers, f)
				l.mu.Unlock()
			}
			l.mu.Unlock()
			return f
		}
		l.mu.Unlock()
	}
	for line := range strings.Lines(job.Output) {
		f.lines = append(f.lines, containerlog.Line{Stream: "stdout", Line: strings.TrimSuffix(line, "\n")})
	}
	f.done = true
	return f
}

// feed is one viewer's stream of a job's output.
type feed struct {
	ready chan struct{}
	stop  func()
	once  sync.Once

	mu      sync.Mutex
	lines   []containerlog.Line
	skipped int
	done    bool
}

// feedBuffer bounds the lines held for one viewer of a running job.
const feedBuffer = 4000

// push queues lines for the viewer, past feedBuffer counting them as skipped so a slow viewer never holds a job up.
func (f *feed) push(lines []containerlog.Line) {
	f.mu.Lock()
	keep := min(max(feedBuffer-len(f.lines), 0), len(lines))
	f.lines = append(f.lines, lines[:keep]...)
	f.skipped += len(lines) - keep
	f.mu.Unlock()
	f.signal()
}

func (f *feed) finish() {
	f.mu.Lock()
	f.done = true
	f.mu.Unlock()
	f.signal()
}

func (f *feed) signal() {
	select {
	case f.ready <- struct{}{}:
	default:
	}
}

// Next returns the lines since the last call, io.EOF once the job ended and every line was read.
func (f *feed) Next(ctx context.Context) ([]containerlog.Line, error) {
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
func (f *feed) Close() {
	f.once.Do(func() {
		if f.stop != nil {
			f.stop()
		}
	})
}
