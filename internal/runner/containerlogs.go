package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Container log limits (ADR 0091); logLineMaxBytes keeps any chunk under the server's 64KiB frame read limit.
const (
	maxLogTail      = 1000
	maxLogStreams   = 16
	logRateBytes    = 64 << 10
	logLineMaxBytes = 8 << 10
	// logFeedBuffer bounds the lines the server holds for a viewer that reads slower than the runner sends.
	logFeedBuffer = 2 * maxLogTail
)

// skippedLine stands in for lines a rate cap or a slow viewer dropped.
func skippedLine(n int) ContainerLogLine {
	return ContainerLogLine{TS: time.Now().UTC().Format(time.RFC3339Nano), Stream: LogStreamStdout, Line: fmt.Sprintf("… %d lines skipped", n)}
}

// ---- runner-side: running docker logs ----

// StreamRunner runs one command with stdout and stderr on separate writers, so every line keeps its stream.
type StreamRunner func(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error

// ShellStreamRunner is the host's StreamRunner; cancelling ctx kills the process.
func ShellStreamRunner(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	// A child still holding the pipes must not keep a killed stream open.
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", cmdString(name, args), err)
	}
	return nil
}

// Logs runs `docker logs` for one logs_request and ends with logs_end; a cancelled ctx kills the process.
func (e *ShellExecutor) Logs(ctx context.Context, req Frame, send func(Frame)) {
	args := []string{"logs", "--timestamps", "--tail", strconv.Itoa(min(req.Tail, maxLogTail))}
	if req.Follow {
		args = append(args, "--follow")
	}
	args = append(args, req.Container)
	out := &containerLogStream{id: req.ID, send: send, capped: req.Follow}
	stdout, stderr := out.writer(LogStreamStdout), out.writer(LogStreamStderr)
	err := e.streams(ctx, stdout, stderr, "docker", args...)
	stdout.flush()
	stderr.flush()
	lastStderr := out.close()
	end := Frame{Type: FrameLogsEnd, ID: req.ID}
	if err != nil && ctx.Err() == nil {
		// docker's own complaint ("No such container") is its last stderr line; the exit status says less.
		end.Error = err.Error()
		if lastStderr != "" {
			end.Error = lastStderr
		}
	}
	send(end)
}

// containerLogStream batches lines like the deploy log and holds a follow to logRateBytes a second.
type containerLogStream struct {
	id     string
	send   func(Frame)
	capped bool

	mu          sync.Mutex
	batch       []ContainerLogLine
	batchBytes  int
	timer       *time.Timer
	windowStart time.Time
	windowBytes int
	skipped     int
	lastStderr  string
}

func (l *containerLogStream) writer(stream string) *lineSplitter {
	return &lineSplitter{emit: func(raw string) { l.add(stream, raw) }}
}

func (l *containerLogStream) add(stream, raw string) {
	line := parseLogLine(stream, raw)
	size := encodedSize(line)
	l.mu.Lock()
	defer l.mu.Unlock()
	if stream == LogStreamStderr {
		l.lastStderr = line.Line
	}
	if l.capped {
		l.rollLocked()
		if l.windowBytes+size > logRateBytes {
			l.skipped++
			l.armLocked(time.Until(l.windowStart.Add(time.Second)))
			return
		}
		l.windowBytes += size
	}
	if l.batchBytes+size > logFlushBytes {
		l.flushLocked()
	}
	l.batch = append(l.batch, line)
	l.batchBytes += size
	l.armLocked(logFlushAfter)
}

// rollLocked opens a new rate window once the current one is a second old, reporting what the last one dropped.
func (l *containerLogStream) rollLocked() {
	now := time.Now()
	if now.Sub(l.windowStart) < time.Second {
		return
	}
	l.windowStart, l.windowBytes = now, 0
	l.appendSkippedLocked()
}

func (l *containerLogStream) appendSkippedLocked() {
	if l.skipped == 0 {
		return
	}
	l.batch = append(l.batch, skippedLine(l.skipped))
	l.skipped = 0
}

func (l *containerLogStream) armLocked(d time.Duration) {
	if l.timer == nil {
		l.timer = time.AfterFunc(d, l.flush)
	}
}

func (l *containerLogStream) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.capped {
		l.rollLocked()
	}
	l.flushLocked()
	if l.skipped > 0 {
		l.armLocked(time.Until(l.windowStart.Add(time.Second)))
	}
}

func (l *containerLogStream) flushLocked() {
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
	if len(l.batch) == 0 {
		return
	}
	l.send(Frame{Type: FrameLogsChunk, ID: l.id, Lines: l.batch})
	l.batch, l.batchBytes = nil, 0
}

// close sends what is left, the skipped count included, and returns the last stderr line for logs_end.
func (l *containerLogStream) close() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.appendSkippedLocked()
	l.flushLocked()
	return l.lastStderr
}

// parseLogLine splits the RFC 3339 timestamp `docker logs --timestamps` puts in front of every line.
func parseLogLine(stream, raw string) ContainerLogLine {
	line := ContainerLogLine{Stream: stream, Line: raw}
	if ts, rest, ok := strings.Cut(raw, " "); ok {
		if _, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			line.TS, line.Line = ts, rest
		}
	}
	if len(line.Line) > logLineMaxBytes {
		line.Line = line.Line[:logLineMaxBytes] + "…"
	}
	return line
}

func encodedSize(line ContainerLogLine) int {
	b, err := json.Marshal(line)
	if err != nil {
		return len(line.Line)
	}
	return len(b) + 1
}

// lineSplitter turns one pipe's writes into whole lines; each pipe has its own, so it needs no lock.
type lineSplitter struct {
	emit func(string)
	buf  []byte
}

func (w *lineSplitter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		w.emit(string(bytes.TrimSuffix(w.buf[:i], []byte("\r"))))
		w.buf = w.buf[i+1:]
	}
	if len(w.buf) > logLineMaxBytes {
		w.flush()
	}
	return len(p), nil
}

func (w *lineSplitter) flush() {
	if len(w.buf) == 0 {
		return
	}
	w.emit(string(w.buf))
	w.buf = nil
}

// ---- client-side: one goroutine per stream ----

// startLogs serves one logs_request until it ends or is cancelled; past maxLogStreams it is refused at once.
func (c *Client) startLogs(ctx context.Context, conn *websocket.Conn, f Frame) {
	send := func(fr Frame) { c.sendFrame(ctx, conn, fr) }
	c.logsMu.Lock()
	if len(c.logs) >= maxLogStreams {
		c.logsMu.Unlock()
		send(Frame{Type: FrameLogsEnd, ID: f.ID, Error: fmt.Sprintf("this runner already serves %d log streams; close one and try again", maxLogStreams)})
		return
	}
	streamCtx, cancel := context.WithCancel(ctx)
	c.logs[f.ID] = cancel
	c.logsMu.Unlock()
	go func() {
		defer c.stopLogs(f.ID)
		c.cfg.Executor.Logs(streamCtx, f, send)
	}()
}

// stopLogs kills one stream's process; an id that already ended is a no-op.
func (c *Client) stopLogs(id string) {
	c.logsMu.Lock()
	cancel := c.logs[id]
	delete(c.logs, id)
	c.logsMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// ---- server-side: one feed per viewer ----

// LogFeed is one viewer's stream; its bounded buffer turns a slow viewer into a skipped line, never a stalled runner.
type LogFeed struct {
	ready chan struct{}
	stop  func()
	once  sync.Once

	mu      sync.Mutex
	lines   []ContainerLogLine
	skipped int
	done    bool
	err     error
}

// Next blocks until lines arrive and returns every one since the last call; io.EOF once the stream ended cleanly.
func (f *LogFeed) Next(ctx context.Context) ([]ContainerLogLine, error) {
	for {
		f.mu.Lock()
		if len(f.lines) > 0 || f.skipped > 0 {
			out := f.lines
			if f.skipped > 0 {
				out = append(out, skippedLine(f.skipped))
			}
			f.lines, f.skipped = nil, 0
			f.mu.Unlock()
			return out, nil
		}
		done, err := f.done, f.err
		f.mu.Unlock()
		if done && err == nil {
			return nil, io.EOF
		}
		if done {
			return nil, err
		}
		select {
		case <-f.ready:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Close stops the stream on the runner; safe to call twice and after the stream ended.
func (f *LogFeed) Close() {
	f.once.Do(f.stop)
}

func (f *LogFeed) push(lines []ContainerLogLine) {
	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		return
	}
	keep := min(max(logFeedBuffer-len(f.lines), 0), len(lines))
	f.lines = append(f.lines, lines[:keep]...)
	f.skipped += len(lines) - keep
	f.mu.Unlock()
	f.signal()
}

func (f *LogFeed) finish(err error) {
	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		return
	}
	f.done, f.err = true, err
	f.mu.Unlock()
	f.signal()
}

func (f *LogFeed) signal() {
	select {
	case f.ready <- struct{}{}:
	default:
	}
}

type logSub struct {
	feed *LogFeed
	conn *runnerConn
}

// OpenLogs asks a runner connected on machine to stream container's logs; the caller must Close the feed.
func (h *Handler) OpenLogs(ctx context.Context, machine, container string, tail int, follow bool) (*LogFeed, error) {
	c := h.connOnMachine(machine)
	if c == nil {
		return nil, fmt.Errorf("%w: no runner is connected on machine %q", apperrs.ErrRetryable, machine)
	}
	id := ids.New()
	feed := &LogFeed{ready: make(chan struct{}, 1)}
	feed.stop = func() {
		sub, ok := h.takeLogSub(id)
		if !ok {
			return
		}
		sub.feed.finish(nil)
		if err := h.sendFrame(c.ctx, c, Frame{Type: FrameLogsCancel, ID: id}); err != nil {
			h.log.Warn("logs_cancel send failed", "runner_id", c.id, "id", id, "error", err)
		}
	}
	h.logsMu.Lock()
	h.logSubs[id] = logSub{feed: feed, conn: c}
	h.logsMu.Unlock()
	if err := h.sendFrame(ctx, c, Frame{Type: FrameLogsRequest, ID: id, Container: container, Tail: tail, Follow: follow}); err != nil {
		h.takeLogSub(id)
		return nil, fmt.Errorf("send logs_request: %w", err)
	}
	return feed, nil
}

// deliverLogs routes a logs_chunk or logs_end to its feed; a stream the viewer already closed is dropped.
func (h *Handler) deliverLogs(f Frame) {
	if f.Type == FrameLogsChunk {
		h.logsMu.Lock()
		sub, ok := h.logSubs[f.ID]
		h.logsMu.Unlock()
		if ok {
			sub.feed.push(f.Lines)
		}
		return
	}
	sub, ok := h.takeLogSub(f.ID)
	if !ok {
		return
	}
	if f.Error != "" {
		sub.feed.finish(fmt.Errorf("%w: %s", apperrs.ErrConflict, f.Error))
		return
	}
	sub.feed.finish(nil)
}

func (h *Handler) takeLogSub(id string) (logSub, bool) {
	h.logsMu.Lock()
	defer h.logsMu.Unlock()
	sub, ok := h.logSubs[id]
	delete(h.logSubs, id)
	return sub, ok
}

// endLogSubs fails every stream a dropped runner connection was serving.
func (h *Handler) endLogSubs(c *runnerConn) {
	h.logsMu.Lock()
	var ended []*LogFeed
	for id, sub := range h.logSubs {
		if sub.conn == c {
			ended = append(ended, sub.feed)
			delete(h.logSubs, id)
		}
	}
	h.logsMu.Unlock()
	for _, feed := range ended {
		feed.finish(fmt.Errorf("%w: the runner disconnected", apperrs.ErrRetryable))
	}
}
