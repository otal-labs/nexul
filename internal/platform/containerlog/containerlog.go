// Package containerlog serves container logs read live from Docker through a runner (ADR 0091): a snapshot of the
// last lines over HTTP, or a live tail over a WebSocket per viewer.
package containerlog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// DefaultTail is how many lines a read starts from when the caller names none.
const DefaultTail = 200

const (
	// agentLineChars cuts a longer line an agent reads, so one runaway line never floods its context.
	agentLineChars  = 2000
	snapshotTimeout = 30 * time.Second
	// writeTimeout drops a viewer whose socket has not taken one message in this long.
	writeTimeout = 10 * time.Second
)

const (
	mask       = "••••"
	maskMinLen = 6
)

// Line is one line a container printed, with its timestamp and stream.
type Line struct {
	TS     string `json:"ts"`
	Stream string `json:"stream"`
	Line   string `json:"line"`
}

// Feed is one open stream: Next returns the lines since the last call, io.EOF once it ended; Close stops it.
type Feed interface {
	Next(ctx context.Context) ([]Line, error)
	Close()
}

// Open starts a stream from the last tail lines; follow keeps it open for new ones.
type Open func(ctx context.Context, tail int, follow bool) (Feed, error)

// Lines is the body of the snapshot route and of every live socket message.
type Lines struct {
	Lines []Line `json:"lines"`
}

// Snapshot reads the last tail lines without following them; what names the container in an error.
func Snapshot(ctx context.Context, open Open, tail int, what string) ([]Line, error) {
	ctx, cancel := context.WithTimeout(ctx, snapshotTimeout)
	defer cancel()
	feed, err := open(ctx, tail, false)
	if err != nil {
		return nil, err
	}
	defer feed.Close()
	out := []Line{}
	for {
		lines, err := feed.Next(ctx)
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: the runner did not send the logs of %s in time", apperrs.ErrRetryable, what)
		}
		if err != nil {
			return nil, fmt.Errorf("logs of %s: %w", what, err)
		}
		out = append(out, lines...)
	}
}

// EnvMask masks env values of six or more characters, longest first so one holding another goes whole (ADR 0091).
func EnvMask(env map[string]string) *strings.Replacer {
	var values []string
	for _, v := range env {
		if utf8.RuneCountInString(v) >= maskMinLen {
			values = append(values, v)
		}
	}
	slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
	pairs := make([]string, 0, 2*len(values))
	for _, v := range values {
		pairs = append(pairs, v, mask)
	}
	return strings.NewReplacer(pairs...)
}

// CutForAgent cuts every line past agentLineChars, in place.
func CutForAgent(lines []Line) {
	for i := range lines {
		if utf8.RuneCountInString(lines[i].Line) > agentLineChars {
			lines[i].Line = string([]rune(lines[i].Line)[:agentLineChars]) + "…"
		}
	}
}

// Tail reads ?tail=, defaulting to DefaultTail; the runner caps it.
func Tail(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("tail")
	if raw == "" {
		return DefaultTail, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: tail must be a whole number of lines", apperrs.ErrInvalid)
	}
	return n, nil
}

// ServeSnapshot answers the snapshot route with the last ?tail= lines.
func ServeSnapshot(w http.ResponseWriter, r *http.Request, open Open, what string) {
	tail, err := Tail(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	lines, err := Snapshot(r.Context(), open, tail, what)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, Lines{Lines: lines})
}

// ServeSocket serves one viewer's live tail; a refusal answers before the upgrade, and closing the socket stops the
// stream on the runner.
func ServeSocket(w http.ResponseWriter, r *http.Request, open Open) {
	tail, err := Tail(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	feed, err := open(r.Context(), tail, true)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	defer feed.Close()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	ctx := conn.CloseRead(r.Context())
	for {
		lines, err := feed.Next(ctx)
		if err != nil {
			closeSocket(conn, err)
			return
		}
		writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
		err = wsjson.Write(writeCtx, conn, Lines{Lines: lines})
		cancel()
		if err != nil {
			_ = conn.CloseNow() // the viewer is gone or stuck; nothing is left to tell it
			return
		}
	}
}

// closeSocket tells the viewer why its stream ended; a close frame's reason is capped at 123 bytes.
func closeSocket(conn *websocket.Conn, err error) {
	if errors.Is(err, io.EOF) {
		_ = conn.Close(websocket.StatusNormalClosure, "the stream ended")
		return
	}
	if errors.Is(err, context.Canceled) {
		_ = conn.CloseNow() // the viewer closed first
		return
	}
	reason := err.Error()
	if len(reason) > 120 {
		reason = reason[:120]
	}
	_ = conn.Close(websocket.StatusTryAgainLater, reason)
}
