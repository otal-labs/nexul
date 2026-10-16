package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/otal-labs/nexul/internal/platform/redact"
)

const (
	runnerLogName  = "runner.log"
	runnerLogLimit = 5 << 20
	// followerBuffer bounds the lines a follower holds while its stream catches up.
	followerBuffer = 256
)

// RunnerLog is the rotated file a personal runner's logger also writes to, and the source of its followers (ADR 0091).
type RunnerLog struct {
	path  string
	limit int64

	mu        sync.Mutex
	file      *os.File
	size      int64
	followers map[*LogFollower]struct{}
}

// OpenRunnerLog opens runner.log in dir, owner-only, appending to what a restart left there.
func OpenRunnerLog(dir string) (*RunnerLog, error) {
	return openRunnerLog(filepath.Join(dir, runnerLogName), runnerLogLimit)
}

func openRunnerLog(path string, limit int64) (*RunnerLog, error) {
	l := &RunnerLog{path: path, limit: limit, followers: map[*LogFollower]struct{}{}}
	if err := l.open(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *RunnerLog) open() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open runner log: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return errors.Join(fmt.Errorf("stat runner log: %w", err), f.Close())
	}
	l.file, l.size = f, info.Size()
	return nil
}

// Close closes the file; followers are left to their own streams' end.
func (l *RunnerLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// Write appends one record, rotating first when it would pass the limit, and hands it to every follower even when the file fails.
func (l *RunnerLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var rotateErr error
	if l.size > 0 && l.size+int64(len(p)) > l.limit {
		rotateErr = l.rotate()
	}
	n, writeErr := l.file.Write(p)
	l.size += int64(n)
	l.broadcast(p)
	return n, errors.Join(rotateErr, writeErr)
}

// rotate moves the file to runner.log.1, replacing the previous one, and starts a new file.
func (l *RunnerLog) rotate() error {
	closeErr := l.file.Close()
	renameErr := os.Rename(l.path, l.path+".1")
	return errors.Join(closeErr, renameErr, l.open())
}

func (l *RunnerLog) broadcast(p []byte) {
	if len(l.followers) == 0 {
		return
	}
	line := strings.TrimRight(string(p), "\n")
	for f := range l.followers {
		select {
		case f.lines <- line:
		default:
			f.dropped++
		}
	}
}

// LogFollower receives each line the log writes after Open returned it; Close stops it.
type LogFollower struct {
	log     *RunnerLog
	lines   chan string
	dropped int
}

// Lines is the raw JSON lines in the order they were written.
func (f *LogFollower) Lines() <-chan string { return f.lines }

// Dropped is how many lines a full buffer lost since the last call.
func (f *LogFollower) Dropped() int {
	f.log.mu.Lock()
	defer f.log.mu.Unlock()
	n := f.dropped
	f.dropped = 0
	return n
}

// Close stops the follower; safe to call twice.
func (f *LogFollower) Close() {
	f.log.mu.Lock()
	defer f.log.mu.Unlock()
	delete(f.log.followers, f)
}

// Open returns the last n lines and, with follow, a follower for each line after them, under one lock so none repeats or is missed.
func (l *RunnerLog) Open(n int, follow bool) ([]string, *LogFollower, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lines, err := l.lastLines(n)
	if err != nil {
		return nil, nil, err
	}
	if !follow {
		return lines, nil, nil
	}
	f := &LogFollower{log: l, lines: make(chan string, followerBuffer)}
	l.followers[f] = struct{}{}
	return lines, f, nil
}

// lastLines reads the file and, when it holds fewer than n, the previous one before it.
func (l *RunnerLog) lastLines(n int) ([]string, error) {
	var out []string
	for _, path := range []string{l.path, l.path + ".1"} {
		if len(out) >= n {
			break
		}
		lines, err := readLines(path)
		if err != nil {
			return nil, err
		}
		out = slices.Concat(lines, out)
	}
	return out[max(len(out)-n, 0):], nil
}

func readLines(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read runner log: %w", err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
}

// renderRunnerLine is one JSON record as `time level message key=value…` with tokens hidden; a line that is not JSON passes through.
func renderRunnerLine(raw string) ContainerLogLine {
	line := ContainerLogLine{Stream: LogStreamStdout, Line: raw}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var rec map[string]any
	if err := dec.Decode(&rec); err == nil {
		line = renderRecord(rec)
	}
	line.Line = redact.Tokens(line.Line)
	if len(line.Line) > logLineMaxBytes {
		line.Line = line.Line[:logLineMaxBytes] + "…"
	}
	return line
}

func renderRecord(rec map[string]any) ContainerLogLine {
	ts, _ := rec[slog.TimeKey].(string)
	level, _ := rec[slog.LevelKey].(string)
	msg, _ := rec[slog.MessageKey].(string)
	for _, k := range []string{slog.TimeKey, slog.LevelKey, slog.MessageKey} {
		delete(rec, k)
	}
	var b strings.Builder
	b.WriteString(ts + " " + level + " " + msg)
	for _, k := range slices.Sorted(maps.Keys(rec)) {
		b.WriteString(" " + k + "=" + logValue(rec[k]))
	}
	stream := LogStreamStdout
	if level == slog.LevelError.String() {
		stream = LogStreamStderr
	}
	return ContainerLogLine{TS: ts, Stream: stream, Line: b.String()}
}

// logValue writes a field's value the way slog's text handler does for a string, and as JSON for anything nested.
func logValue(v any) string {
	if s, ok := v.(string); ok {
		if s == "" || strings.ContainsAny(s, " =\"\t\n") {
			return strconv.Quote(s)
		}
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
