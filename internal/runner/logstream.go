package runner

import (
	"strings"
	"sync"
	"time"
)

// A batch goes out once its first line is logFlushAfter old, once logFlushBytes wait, or when a step ends; a
// job's whole output is capped at logMaxBytes so a runaway build cannot flood the socket or the database.
const (
	logFlushAfter    = 250 * time.Millisecond
	logFlushBytes    = 32 << 10
	logMaxBytes      = 2 << 20
	logTruncatedLine = "… output truncated\n"
)

// logStream batches one job's output into deploy_log frames. Lines arrive from the step's command goroutine and
// the flush timer fires on its own, so every method takes the mutex; a batch never mixes two phases.
type logStream struct {
	id   string
	send func(Frame)

	mu        sync.Mutex
	phase     string
	buf       strings.Builder
	timer     *time.Timer
	total     int
	truncated bool
	closed    bool
}

func newLogStream(id string, send func(Frame)) *logStream {
	return &logStream{id: id, send: send}
}

// step announces a step with its label line and returns the logf its command output streams through.
func (l *logStream) step(phase, label string) func(string) {
	l.write(phase, label+"\n")
	return func(s string) { l.write(phase, s) }
}

func (l *logStream) write(phase, s string) {
	if s == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.truncated || l.closed {
		return
	}
	if phase != l.phase {
		l.flushLocked()
		l.phase = phase
	}
	if l.total+len(s) > logMaxBytes {
		l.truncated = true
		s = logTruncatedLine
	}
	l.buf.WriteString(s)
	l.total += len(s)
	if l.buf.Len() >= logFlushBytes || l.truncated {
		l.flushLocked()
		return
	}
	if l.timer == nil {
		l.timer = time.AfterFunc(logFlushAfter, l.flush)
	}
}

// flush sends whatever is buffered; every step end calls it so the step's tail never waits on the timer.
func (l *logStream) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushLocked()
}

// close sends the last batch and stops the timer; a write after close is dropped rather than sent after the
// terminal result frame.
func (l *logStream) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flushLocked()
	l.closed = true
}

func (l *logStream) flushLocked() {
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
	if l.buf.Len() == 0 {
		return
	}
	l.send(Frame{Type: FrameDeployLog, ID: l.id, Phase: l.phase, Log: l.buf.String(), TS: time.Now().UnixMilli()})
	l.buf.Reset()
}
