package shelljob

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Terminal limits (ADR 0212).
const (
	// MaxTerminalInput is the largest message a terminal takes, such as one paste.
	MaxTerminalInput = 1 << 20
	// AdapterTerminal marks a job that is a terminal, not one command.
	AdapterTerminal = "terminal"
	// TerminalsAtOnce is how many terminals one runner holds open, apart from its commands.
	TerminalsAtOnce = 4
	// AttachWindow is how long an opened terminal waits for the socket of the person who opened it.
	AttachWindow = 30 * time.Second
	// MaxTerminalSide bounds a terminal's columns and rows.
	MaxTerminalSide = 1000
)

// TerminalSize is a terminal's columns and rows.
type TerminalSize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// Valid reports whether both sides are 1 to MaxTerminalSide.
func (s TerminalSize) Valid() bool {
	return s.Cols >= 1 && s.Cols <= MaxTerminalSide && s.Rows >= 1 && s.Rows <= MaxTerminalSide
}

// TerminalExit is the text message a runner sends when the terminal's shell exits, before it closes.
type TerminalExit struct {
	Exit int `json:"exit"`
}

// TerminalStream is one end of a terminal: binary messages carry its bytes, text messages a TerminalSize towards the
// runner or a TerminalExit from it.
type TerminalStream interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error
	Close()
}

// WebSocketStream is a TerminalStream over a WebSocket; Close drops it at once.
func WebSocketStream(conn *websocket.Conn) TerminalStream {
	return wsStream{conn}
}

type wsStream struct{ conn *websocket.Conn }

func (s wsStream) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return s.conn.Read(ctx)
}

func (s wsStream) Write(ctx context.Context, typ websocket.MessageType, p []byte) error {
	return s.conn.Write(ctx, typ, p)
}

func (s wsStream) Close() { _ = s.conn.CloseNow() }

// TerminalOpener opens job's terminal on its runner.
type TerminalOpener func(ctx context.Context, job Job, size TerminalSize) (TerminalStream, error)

// Terminals holds every open terminal: recorded as a job, waiting for its viewer, then relayed until either side ends.
type Terminals struct {
	store Store
	event func(Job) eventbus.OutboxEvent
	now   func() time.Time
	// startError is what a terminal its runner never opened says, from the opener's error.
	startError func(error) string

	mu   sync.Mutex
	open map[string]*terminal
}

type terminal struct {
	job    Job
	stream TerminalStream
	timer  *time.Timer
	// attached is set once its viewer took it; cancel ends its relay, why says why the server ended it.
	attached bool
	cancel   context.CancelFunc
	why      string
	done     chan struct{}
}

// NewTerminals builds an empty set of terminals recorded in cfg's store with cfg's events.
func NewTerminals(cfg Config) *Terminals {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Terminals{store: cfg.Store, event: cfg.Event, now: cfg.Now, startError: cfg.StartError, open: map[string]*terminal{}}
}

// Open records job as a terminal and opens it on its runner; its viewer then has AttachWindow to Attach. A runner that
// refuses or cannot be reached leaves a recorded job and an error saying why.
func (t *Terminals) Open(ctx context.Context, job Job, size TerminalSize, open TerminalOpener) (*Job, error) {
	if !size.Valid() {
		return nil, fmt.Errorf("%w: a terminal is 1 to %d columns and rows", apperrs.ErrInvalid, MaxTerminalSide)
	}
	key := job.runnerKey()
	t.mu.Lock()
	n := 0
	for _, o := range t.open {
		if o.job.runnerKey() == key {
			n++
		}
	}
	t.mu.Unlock()
	if n >= TerminalsAtOnce {
		return nil, fmt.Errorf("%w: %d terminals are already open there; close one first", apperrs.ErrConflict, TerminalsAtOnce)
	}
	job.Adapter, job.Status, job.Command, job.TimeoutSeconds = AdapterTerminal, Queued, "terminal", 0
	if err := t.store.InsertShellJob(ctx, job, t.event(job)); err != nil {
		return nil, err
	}
	stream, err := open(ctx, job, size)
	if err != nil {
		why := t.startError(err)
		t.finish(job, End{Status: Refused, Error: why}, nil, false)
		return nil, fmt.Errorf("%w: %s", apperrs.ErrConflict, why)
	}
	at := t.now().UTC()
	job.Status, job.StartedAt = Running, &at
	if _, err := t.store.StartShellJob(ctx, job.ID, at, t.event(job)); err != nil {
		stream.Close()
		t.finish(job, End{Status: Failed, Error: "could not record the terminal's start"}, nil, false)
		return nil, err
	}
	term := &terminal{job: job, stream: stream, done: make(chan struct{})}
	term.timer = time.AfterFunc(AttachWindow, func() { t.Cancel(job.ID, "nobody opened the terminal") })
	t.mu.Lock()
	t.open[job.ID] = term
	t.mu.Unlock()
	return &job, nil
}

// Attach relays the viewer accept gives to the terminal jobID until either side ends, for the person who opened it and
// only once; anyone else, or a second viewer, finds no terminal and accept is never called.
func (t *Terminals) Attach(ctx context.Context, jobID, userID string, accept func() (TerminalStream, error)) error {
	t.mu.Lock()
	term := t.open[jobID]
	if term == nil || term.attached || term.why != "" || term.job.RequesterID != userID {
		t.mu.Unlock()
		return fmt.Errorf("%w: terminal %s", apperrs.ErrNotFound, jobID)
	}
	term.timer.Stop()
	term.attached = true
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	term.cancel = cancel
	t.mu.Unlock()
	viewer, err := accept()
	if err != nil {
		t.Cancel(jobID, "the terminal could not be opened in the browser")
		return nil
	}
	t.relay(ctx, term, viewer)
	return nil
}

// relay copies the runner's bytes to the viewer, keeping the last MaxOutput of them, and the viewer's keys and sizes to
// the runner, until either side ends; then it records how the terminal ended.
func (t *Terminals) relay(ctx context.Context, term *terminal, viewer TerminalStream) {
	defer term.cancel()
	var (
		out       tail
		exit      *int
		viewerEnd bool
		wg        sync.WaitGroup
	)
	wg.Go(func() {
		defer term.cancel()
		for {
			typ, p, err := term.stream.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageText {
				var e TerminalExit
				if json.Unmarshal(p, &e) == nil {
					exit = &e.Exit
				}
			}
			if typ == websocket.MessageBinary {
				out.write(p)
			}
			if viewer.Write(ctx, typ, p) != nil {
				return
			}
		}
	})
	for {
		typ, p, err := viewer.Read(ctx)
		if err != nil {
			viewerEnd = ctx.Err() == nil
			break
		}
		if typ == websocket.MessageText {
			p = resize(p)
		}
		if p != nil && term.stream.Write(ctx, typ, p) != nil {
			break
		}
	}
	term.cancel()
	wg.Wait()
	term.stream.Close()
	viewer.Close()
	t.mu.Lock()
	why := term.why
	t.mu.Unlock()
	t.finish(term.job, terminalEnd(exit, why, viewerEnd), &out, true)
}

// resize re-encodes a viewer's size message, or drops one that is not a valid size.
func resize(p []byte) []byte {
	var s TerminalSize
	if json.Unmarshal(p, &s) != nil || !s.Valid() {
		return nil
	}
	b, _ := json.Marshal(s)
	return b
}

// terminalEnd is how a relayed terminal ended: its shell's exit, the server's reason, its viewer closing it, or its
// runner going away.
func terminalEnd(exit *int, why string, viewerClosed bool) End {
	if exit != nil && *exit == 0 {
		return End{Status: Succeeded}
	}
	if exit != nil {
		return End{Status: Failed, ExitCode: *exit}
	}
	if why != "" {
		return End{Status: Cancelled, Error: why}
	}
	if viewerClosed {
		return End{Status: Cancelled, Error: "the terminal was closed"}
	}
	return End{Status: Failed, Error: "the runner closed the terminal"}
}

// finish records job's end with what it printed, and forgets it.
func (t *Terminals) finish(job Job, end End, out *tail, registered bool) {
	if registered {
		t.mu.Lock()
		term := t.open[job.ID]
		delete(t.open, job.ID)
		t.mu.Unlock()
		if term != nil {
			defer close(term.done)
		}
	}
	job.Status, job.Error = end.Status, end.Error
	if end.Status == Succeeded || (end.Status == Failed && end.Error == "") {
		code := end.ExitCode
		job.ExitCode = &code
	}
	if out != nil {
		job.Output, job.OutputTruncated = string(out.b), out.truncated
	}
	at := t.now().UTC()
	job.EndedAt = &at
	ctx := context.Background()
	if err := t.store.FinishShellJob(ctx, job, t.event(job)); err != nil {
		logging.FromCtx(ctx).Error("finish terminal", "job_id", job.ID, "error", err)
	}
}

// Live reports whether jobID is an open terminal.
func (t *Terminals) Live(jobID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.open[jobID] != nil
}

// Cancel closes the terminal jobID, saying why, and returns once it is recorded; an unknown id is a no-op.
func (t *Terminals) Cancel(jobID, why string) {
	t.CancelWhere(func(j Job) bool { return j.ID == jobID }, why)
}

// CancelWhere closes every open terminal match picks, saying why, and returns once each is recorded.
func (t *Terminals) CancelWhere(match func(Job) bool, why string) {
	why = cmp.Or(why, "stopped by a person")
	t.mu.Lock()
	var waiting, relayed []*terminal
	for _, term := range t.open {
		if !match(term.job) || term.why != "" {
			continue
		}
		term.why = why
		if term.attached {
			term.cancel()
			relayed = append(relayed, term)
			continue
		}
		term.timer.Stop()
		waiting = append(waiting, term)
	}
	t.mu.Unlock()
	for _, term := range waiting {
		term.stream.Close()
		t.finish(term.job, End{Status: Cancelled, Error: why}, nil, true)
	}
	for _, term := range relayed {
		<-term.done
	}
}

// tail keeps the last MaxOutput bytes written to it.
type tail struct {
	b         []byte
	truncated bool
}

func (t *tail) write(p []byte) {
	t.b = append(t.b, p...)
	if len(t.b) > MaxOutput {
		t.b, t.truncated = t.b[len(t.b)-MaxOutput:], true
	}
}

// ServeTerminal attaches the request's WebSocket to a terminal through attach; a refusal answers before the upgrade.
func ServeTerminal(w http.ResponseWriter, r *http.Request, attach func(ctx context.Context, accept func() (TerminalStream, error)) error) {
	accepted := false
	err := attach(r.Context(), func() (TerminalStream, error) {
		accepted = true
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return nil, err
		}
		conn.SetReadLimit(MaxTerminalInput)
		return WebSocketStream(conn), nil
	})
	if err != nil && !accepted {
		httpx.WriteError(w, err)
	}
}
