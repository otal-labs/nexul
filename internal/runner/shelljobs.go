package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/version"
)

// Shell job limits (the personal runners spec, Later: shell jobs).
const (
	// MaxShellTimeoutSeconds is the longest a shell job may run, an hour.
	MaxShellTimeoutSeconds = 3600
	// MaxShellJobs is how many shell jobs one runner runs at a time; the server queues the rest.
	MaxShellJobs = 4
	// lastWithoutShell is the newest release whose personal runner does not answer shell_run.
	lastWithoutShell = "v0.3.100-beta"
)

// ErrCannotRunShell means the computer's runner is too old to run shell jobs.
var ErrCannotRunShell = errors.New("the runner cannot run shell jobs")

// ShellRun is one command for a computer's runner to run.
type ShellRun struct {
	ID      string
	Shell   string
	Script  string
	Cwd     string
	Timeout time.Duration
}

// ShellResult is how a shell job ended, as its runner reported it or as the server ended it.
type ShellResult struct {
	Status   string
	ExitCode int
	Error    string
}

// ShellSink receives one shell job's output batches and then its end, once.
type ShellSink interface {
	Output(lines []ContainerLogLine)
	Done(result ShellResult)
}

// ShellFacts is whether the computer itself lets its runner run shell jobs.
type ShellFacts struct {
	Enabled bool `json:"enabled" jsonschema:"Whether the computer lets its Nexul app run commands; nexul shell on or off, run there, sets it."`
}

// ---- runner side ----

// startShell runs one shell_run in its own goroutine, refusing it when the computer turned shell jobs off or runs its
// limit already; a dropped connection kills it.
func (c *Client) startShell(ctx context.Context, conn *websocket.Conn, f Frame) {
	send := func(fr Frame) { c.sendFrame(ctx, conn, fr) }
	refuse := func(msg string) {
		send(Frame{Type: FrameShellResult, ID: f.ID, Status: ShellStatusRefused, Error: msg})
	}
	if !c.cfg.ShellEnabled {
		refuse("commands are turned off on this computer; `nexul shell on`, run there, turns them on")
		return
	}
	c.shellMu.Lock()
	if len(c.shells) >= MaxShellJobs {
		c.shellMu.Unlock()
		refuse(fmt.Sprintf("this computer already runs %d commands", MaxShellJobs))
		return
	}
	jobCtx, cancel := context.WithCancel(ctx)
	c.shells[f.ID] = cancel
	c.shellMu.Unlock()
	go func() {
		defer c.shellFinished(ctx, f.ID)
		defer c.contained(f, func(msg string) {
			send(Frame{Type: FrameShellResult, ID: f.ID, Status: ShellStatusFailed, Error: msg})
		})
		send(c.runShell(jobCtx, f, send))
	}()
}

// stopShell kills one shell job's process tree; an id that already ended is a no-op.
func (c *Client) stopShell(id string) {
	c.shellMu.Lock()
	cancel := c.shells[id]
	c.shellMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// shellFinished forgets a shell job, and applies an update that waited for the last one.
func (c *Client) shellFinished(ctx context.Context, id string) {
	c.shellMu.Lock()
	cancel := c.shells[id]
	delete(c.shells, id)
	c.shellMu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.applyDeferredUpdate(ctx)
}

// shellsRunning reports whether a shell job is running, which defers an update (ADR 0052).
func (c *Client) shellsRunning() bool {
	c.shellMu.Lock()
	defer c.shellMu.Unlock()
	return len(c.shells) > 0
}

// runShell runs f's command to its end and returns the shell_result that says how it ended.
func (c *Client) runShell(ctx context.Context, f Frame, send func(Frame)) Frame {
	result := Frame{Type: FrameShellResult, ID: f.ID, Status: ShellStatusFailed}
	name, args, err := shellCommand(f.Shell, f.Script)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	dir, err := shellDir(f.Cwd)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	timeout := time.Duration(f.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Dir, cmd.Env = dir, shellEnv(os.Environ())
	out := &containerLogStream{id: f.ID, send: send, capped: true, frame: FrameShellOutput, stamped: true}
	stdout, stderr := out.writer(LogStreamStdout), out.writer(LogStreamStderr)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	killTree(cmd)
	// A child still holding the pipes must not keep a killed job open.
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	stdout.flush()
	stderr.flush()
	out.close()
	if ctx.Err() != nil {
		result.Status = ShellStatusCancelled
		return result
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		result.Status, result.Error = ShellStatusTimedOut, fmt.Sprintf("stopped after %s", timeout)
		return result
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		result.ExitCode = exit.ExitCode()
		return result
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Status = ShellStatusSucceeded
	return result
}

// shellDir is where a job starts: the home folder when none is named, with a leading ~ read as the home folder.
func shellDir(cwd string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the home folder: %w", err)
	}
	if cwd == "" || cwd == "~" {
		return home, nil
	}
	if rest, ok := strings.CutPrefix(cwd, "~/"); ok {
		return filepath.Join(home, rest), nil
	}
	return cwd, nil
}

// shellEnv is the runner's environment without its own NEXUL_ settings, which carry its credential's path.
func shellEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(strings.ToUpper(kv), "NEXUL_") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// firstOnPath is the first of names found on PATH.
func firstOnPath(names ...string) (string, error) {
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("none of %s is installed", strings.Join(names, ", "))
}

// ---- server side ----

type shellSub struct {
	sink ShellSink
	conn *runnerConn
}

// RunShell sends run to computerID's runner; sink gets its output and its end, which a dropped connection also ends.
func (h *Handler) RunShell(ctx context.Context, computerID string, run ShellRun, sink ShellSink) error {
	c, err := h.computerConn(ctx, computerID)
	if err != nil {
		return err
	}
	if c.version == "" || !version.Newer(c.version, lastWithoutShell) {
		return ErrCannotRunShell
	}
	h.shellMu.Lock()
	h.shellSubs[run.ID] = shellSub{sink: sink, conn: c}
	h.shellMu.Unlock()
	f := Frame{Type: FrameShellRun, ID: run.ID, Shell: run.Shell, Script: run.Script, Cwd: run.Cwd, TimeoutSeconds: int(run.Timeout / time.Second)}
	if err := h.sendFrame(ctx, c, f); err != nil {
		h.takeShellSub(run.ID)
		return apperrs.Retryable(fmt.Errorf("send shell_run: %w", err))
	}
	return nil
}

// CancelShell asks the runner running jobID to stop it; its shell_result then ends it.
func (h *Handler) CancelShell(ctx context.Context, jobID string) error {
	h.shellMu.Lock()
	sub, ok := h.shellSubs[jobID]
	h.shellMu.Unlock()
	if !ok {
		return nil
	}
	if err := h.sendFrame(ctx, sub.conn, Frame{Type: FrameShellCancel, ID: jobID}); err != nil {
		return apperrs.Retryable(fmt.Errorf("send shell_cancel: %w", err))
	}
	return nil
}

// deliverShell routes a shell_output or shell_result to its job, only from the runner that was asked.
func (h *Handler) deliverShell(c *runnerConn, f Frame) {
	if f.Type == FrameShellOutput {
		h.shellMu.Lock()
		sub, ok := h.shellSubs[f.ID]
		h.shellMu.Unlock()
		if ok && sub.conn == c {
			sub.sink.Output(f.Lines)
		}
		return
	}
	h.shellMu.Lock()
	sub, ok := h.shellSubs[f.ID]
	if ok && sub.conn == c {
		delete(h.shellSubs, f.ID)
	}
	h.shellMu.Unlock()
	if !ok || sub.conn != c {
		h.log.Warn("shell result refused", "runner_id", c.id, "id", f.ID)
		return
	}
	sub.sink.Done(ShellResult{Status: f.Status, ExitCode: f.ExitCode, Error: f.Error})
}

func (h *Handler) takeShellSub(id string) {
	h.shellMu.Lock()
	delete(h.shellSubs, id)
	h.shellMu.Unlock()
}

// endShellSubs fails every job a dropped runner connection was running.
func (h *Handler) endShellSubs(c *runnerConn) {
	h.shellMu.Lock()
	var ended []ShellSink
	for id, sub := range h.shellSubs {
		if sub.conn == c {
			ended = append(ended, sub.sink)
			delete(h.shellSubs, id)
		}
	}
	h.shellMu.Unlock()
	for _, sink := range ended {
		sink.Done(ShellResult{Status: ShellStatusFailed, Error: "the computer's Nexul app disconnected"})
	}
}

// handleShell starts a shell_run or stops the job a shell_cancel names.
func (c *Client) handleShell(ctx context.Context, conn *websocket.Conn, f Frame) {
	if f.Type == FrameShellCancel {
		c.stopShell(f.ID)
		return
	}
	c.startShell(ctx, conn, f)
}
