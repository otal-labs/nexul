package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/aymanbagabas/go-pty"
	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
	"github.com/otal-labs/nexul/internal/platform/version"
)

// lastWithoutTerminal is the newest release whose runner does not answer terminal_open.
const lastWithoutTerminal = "v0.3.101-beta"

// ErrCannotOpenTerminal means the runner is too old to open a terminal.
var ErrCannotOpenTerminal = errors.New("the runner cannot open a terminal")

// exitGrace is how long a terminal whose shell exited waits for the rest of its output before it closes.
const exitGrace = time.Second

// ---- server side ----

// OpenTerminal opens a terminal on computerID through its runner; the stream carries it until either end closes it.
func (h *Handler) OpenTerminal(ctx context.Context, computerID, id string, size shelljob.TerminalSize) (shelljob.TerminalStream, error) {
	c, err := h.computerConn(ctx, computerID)
	if err != nil {
		return nil, err
	}
	return h.openTerminal(ctx, c, id, size)
}

// OpenMachineTerminal opens a terminal on runnerID, a machine's runner.
func (h *Handler) OpenMachineTerminal(ctx context.Context, runnerID, id string, size shelljob.TerminalSize) (shelljob.TerminalStream, error) {
	h.mu.Lock()
	c := h.conns[runnerID]
	h.mu.Unlock()
	if c == nil || c.personal {
		return nil, apperrs.Retryable(fmt.Errorf("runner %s is not connected", runnerID))
	}
	return h.openTerminal(ctx, c, id, size)
}

func (h *Handler) openTerminal(ctx context.Context, c *runnerConn, id string, size shelljob.TerminalSize) (shelljob.TerminalStream, error) {
	if c.version == "" || !version.Newer(c.version, lastWithoutTerminal) {
		return nil, ErrCannotOpenTerminal
	}
	if id == "" {
		id = ids.New()
	}
	w := &streamWaiter{runnerID: c.id, raw: true, result: make(chan dialResult), done: make(chan struct{})}
	if err := h.addWaiter(id, w); err != nil {
		return nil, err
	}
	defer h.dropWaiter(id, w)
	if err := h.sendFrame(ctx, c, Frame{Type: FrameTerminalOpen, ID: id, Cols: size.Cols, Rows: size.Rows}); err != nil {
		return nil, apperrs.Retryable(fmt.Errorf("send terminal_open: %w", err))
	}
	timer := time.NewTimer(streamTTL)
	defer timer.Stop()
	select {
	case r := <-w.result:
		return r.terminal, r.err
	case <-timer.C:
		return nil, apperrs.Retryable(fmt.Errorf("the runner opened no terminal within %s", streamTTL))
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// terminalStream is a terminal's stream from its runner; closing it ends the ServeStream holding it.
type terminalStream struct {
	conn   *websocket.Conn
	cancel context.CancelFunc
}

func (s *terminalStream) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return s.conn.Read(ctx)
}

func (s *terminalStream) Write(ctx context.Context, typ websocket.MessageType, p []byte) error {
	return s.conn.Write(ctx, typ, p)
}

func (s *terminalStream) Close() {
	_ = s.conn.CloseNow()
	s.cancel()
}

// ---- runner side ----

// shellOff is why this runner refuses commands and terminals, or "" when its own switch lets them run.
func (c *Client) shellOff() string {
	if c.cfg.ShellEnabled {
		return ""
	}
	if c.cfg.Personal {
		return "commands are turned off on this computer; `nexul shell on`, run there, turns them on"
	}
	return fmt.Sprintf("commands are off on this runner: it was installed without --allow-shell; `sudo nexul shell on %s`, run on the machine, turns them on", c.cfg.Name)
}

// startTerminal answers terminal_open: the shell on a pseudo-terminal, carried by a stream of its own until the shell
// exits or the stream closes; streamCtx outlives a control reconnect.
func (c *Client) startTerminal(streamCtx, ctx context.Context, control *websocket.Conn, f Frame) {
	refuse := func(msg string) {
		c.sendFrame(ctx, control, Frame{Type: FrameHarnessDialRefused, ID: f.ID, Error: msg})
	}
	if off := c.shellOff(); off != "" {
		refuse(off)
		return
	}
	if c.terminals.Add(1) > shelljob.TerminalsAtOnce {
		c.terminals.Add(-1)
		refuse(fmt.Sprintf("this runner already has %d terminals open", shelljob.TerminalsAtOnce))
		return
	}
	c.streams.Go(func() {
		defer c.terminals.Add(-1)
		defer c.recoverStream(f.ID)
		term, err := startPty(f)
		if err != nil {
			refuse(err.Error())
			return
		}
		defer term.close()
		dialCtx, cancel := context.WithTimeout(streamCtx, c.cfg.ConnectTimeout)
		ws, _, err := websocket.Dial(dialCtx, c.cfg.StreamURL+"/"+url.PathEscape(f.ID), &websocket.DialOptions{
			HTTPHeader: http.Header{"Authorization": {"Bearer " + c.cfg.Credential}},
		})
		cancel()
		if err != nil {
			c.log.Warn("terminal stream dial failed", "runner", c.cfg.Name, "stream", f.ID, "error", err)
			return
		}
		defer func() { _ = ws.CloseNow() }()
		ws.SetReadLimit(shelljob.MaxTerminalInput)
		term.relay(streamCtx, ws)
	})
}

// ptyTerminal is a shell running on a pseudo-terminal.
type ptyTerminal struct {
	pty  pty.Pty
	cmd  *pty.Cmd
	once sync.Once
}

// startPty starts the default shell in the home folder on a pseudo-terminal of f's size.
func startPty(f Frame) (*ptyTerminal, error) {
	name, args, err := terminalShell()
	if err != nil {
		return nil, err
	}
	dir, err := shellDir("")
	if err != nil {
		return nil, err
	}
	p, err := pty.New()
	if err != nil {
		return nil, fmt.Errorf("open a pseudo-terminal: %w", err)
	}
	if err := p.Resize(f.Cols, f.Rows); err != nil {
		_ = p.Close()
		return nil, err
	}
	cmd := p.Command(name, args...)
	cmd.Dir, cmd.Env = dir, append(shellEnv(os.Environ()), "TERM=xterm-256color")
	if err := cmd.Start(); err != nil {
		_ = p.Close()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	releaseTty(p)
	return &ptyTerminal{pty: p, cmd: cmd}, nil
}

// close kills the shell and everything it started, and closes the pseudo-terminal; only the first call acts.
func (t *ptyTerminal) close() {
	t.once.Do(func() {
		killTerminal(t.cmd.Process.Pid)
		_ = t.pty.Close()
	})
}

// relay copies the terminal's output to ws and ws's keys and sizes to it until the shell exits, which it reports with
// its exit code, or ws closes, which kills the shell.
func (t *ptyTerminal) relay(ctx context.Context, ws *websocket.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	output := make(chan struct{})
	go func() {
		defer close(output)
		buf := make([]byte, 16<<10)
		for {
			n, err := t.pty.Read(buf)
			if n > 0 && ws.Write(ctx, websocket.MessageBinary, buf[:n]) != nil {
				return
			}
			if err != nil {
				return
			}
		}
	}()
	exited := make(chan int, 1)
	go func() {
		err := t.cmd.Wait()
		code := 0
		if t.cmd.ProcessState != nil {
			code = t.cmd.ProcessState.ExitCode()
		}
		if err != nil && code == 0 {
			code = -1
		}
		exited <- code
	}()
	go t.input(ctx, ws, cancel)
	select {
	case code := <-exited:
		select {
		case <-output:
		case <-time.After(exitGrace):
		}
		b, _ := json.Marshal(shelljob.TerminalExit{Exit: code})
		_ = ws.Write(ctx, websocket.MessageText, b)
		_ = ws.Close(websocket.StatusNormalClosure, "")
	case <-ctx.Done():
	}
	t.close()
	<-output
}

// input writes what ws sends to the terminal: binary messages are keys, text messages a new size.
func (t *ptyTerminal) input(ctx context.Context, ws *websocket.Conn, cancel context.CancelFunc) {
	defer cancel()
	for {
		typ, p, err := ws.Read(ctx)
		if err != nil {
			return
		}
		if typ == websocket.MessageBinary {
			if _, err := t.pty.Write(p); err != nil {
				return
			}
			continue
		}
		var s shelljob.TerminalSize
		if json.Unmarshal(p, &s) == nil && s.Valid() {
			_ = t.pty.Resize(s.Cols, s.Rows)
		}
	}
}
