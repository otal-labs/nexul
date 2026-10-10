package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// ClientConfig wires the runner host binary's connection to the server.
type ClientConfig struct {
	// URL is the runner WebSocket endpoint.
	URL string
	// Credential is the runner's own credential; the server knows its id, name and machine from it.
	Credential string
	// Name is the runner's unit name, passed to `nexul uninstall runner` once the runner is removed.
	Name string
	// Version is the runner binary's stamped build version ("dev" for local
	// builds); the server persists it so the UI can flag stale runners.
	Version           string
	Logger            *slog.Logger
	Executor          Executor
	HeartbeatInterval time.Duration
	ConnectTimeout    time.Duration
	WriteTimeout      time.Duration
	BackoffBase       time.Duration
	BackoffMax        time.Duration
	// Personal runs this runner for one person's computer (ADR 0146): it opens streams to T3 Code and refuses deploy work.
	Personal bool
	// StreamURL is the ws(s) base a stream opens under, as <StreamURL>/<id>.
	StreamURL string
	// T3Home is T3 Code's base directory, whose runtime file names the port it listens on.
	T3Home string
}

// Client connects a runner over WS, executes assigned jobs, and reconnects with exponential backoff.
type Client struct {
	cfg ClientConfig
	log *slog.Logger
	hb  time.Duration

	mu        sync.Mutex
	jobID     string
	jobCancel context.CancelFunc
	// pendingUpdate holds an update frame that arrived mid-job; applied once the job's result frame is sent.
	pendingUpdate *Frame
	// writeMu serializes writes: a websocket.Conn allows only one writer at a time.
	writeMu sync.Mutex
	// logsMu/logs hold the cancel of every open container log stream, keyed by stream id.
	logsMu sync.Mutex
	logs   map[string]context.CancelFunc
	// streams joins every stream, openStreams counts them against maxStreams, and dial reaches T3 Code.
	streams     sync.WaitGroup
	openStreams atomic.Int32
	dial        func(ctx context.Context, network, address string) (net.Conn, error)
	// findT3 locates T3 Code's command for a home, "" when it is not installed.
	findT3 func(home string) string
	// t3 keeps a personal runner's T3 Code background service running.
	t3 *t3Keeper
}

// NewClient wires the runner client with sane defaults for unset durations.
func NewClient(cfg ClientConfig) *Client {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 10 * time.Second
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.BackoffBase <= 0 {
		cfg.BackoffBase = time.Second
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = 30 * time.Second
	}
	c := &Client{cfg: cfg, log: cfg.Logger, hb: cfg.HeartbeatInterval, logs: map[string]context.CancelFunc{}, dial: (&net.Dialer{}).DialContext, findT3: findT3}
	c.t3 = newT3Keeper(c.t3Probe, c.restartT3, cfg.Logger)
	return c
}

// errRemoved ends the connection loop: the server removed this runner, so reconnecting can never succeed.
var errRemoved = errors.New("runner removed")

// Run dials the server and keeps the runner connected; a dropped connection retries with exponential backoff.
// A removed runner uninstalls its own service and returns instead.
func (c *Client) Run(ctx context.Context) error {
	ctx, stopStreams := context.WithCancel(ctx)
	defer c.streams.Wait()
	defer stopStreams()
	c.cleanupOldBinary()
	c.log.Info("runner starting", "runner", c.cfg.Name, "server", c.cfg.URL)
	if c.cfg.Personal {
		c.streams.Go(func() { c.t3.run(ctx) })
	}
	backoff := NewBackoff(c.cfg.BackoffBase, c.cfg.BackoffMax,
		rand.New(rand.NewSource(time.Now().UnixNano())))
	attempt := 0
	for {
		err := c.runOnce(ctx)
		if errors.Is(err, errRemoved) {
			c.uninstall(ctx)
			return nil
		}
		if ctx.Err() != nil {
			return nil
		}
		attempt++
		c.log.Warn("runner connection lost; reconnecting", "error", err, "attempt", attempt)
		if !backoff.Wait(ctx, attempt) {
			return nil
		}
	}
}

// runOnce maintains one connection: heartbeats on a ticker, assigns run in goroutines, cancels abort the job.
func (c *Client) runOnce(ctx context.Context) (err error) {
	dialCtx, cancel := context.WithTimeout(ctx, c.cfg.ConnectTimeout)
	conn, resp, dialErr := websocket.Dial(dialCtx, c.wsURL(), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + c.cfg.Credential}},
	})
	cancel()
	if dialErr != nil {
		if refusedAsRemoved(resp) {
			return errRemoved
		}
		return fmt.Errorf("dial %s: %w", c.cfg.URL, dialErr)
	}
	defer func() {
		err = errors.Join(err, conn.CloseNow())
	}()
	c.log.Info("runner connected", "runner", c.cfg.Name)

	streamCtx := ctx
	var loops sync.WaitGroup
	defer loops.Wait() // after cancel: a facts report still reading the computer must not outlive its connection
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	defer c.cancelJob("")

	loops.Go(func() { c.heartbeatLoop(ctx, conn) })
	if c.cfg.Personal {
		loops.Go(func() { c.factsLoop(ctx, conn) })
	}

	for {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return err
		}
		frame, err := ParseFrame(raw)
		if err != nil {
			c.log.Warn("invalid frame from server", "runner", c.cfg.Name, "error", err)
			continue
		}
		if frame.Type == FrameUninstall {
			return errRemoved
		}
		c.handleFrame(streamCtx, ctx, conn, *frame)
	}
}

// handleFrame acts on one server frame; streamCtx outlives this connection, ctx does not.
func (c *Client) handleFrame(streamCtx, ctx context.Context, conn *websocket.Conn, frame Frame) {
	if reply, refused := refusal(frame); refused && c.cfg.Personal {
		c.log.Warn("personal runner refused a frame", "runner", c.cfg.Name, "type", frame.Type)
		c.sendFrame(ctx, conn, reply)
		return
	}
	switch frame.Type {
	case FrameAssignBuild, FrameAssignDeploy, FrameAssignUpgrade:
		c.startJob(ctx, conn, frame)
	case FrameCancel:
		c.cancelJob(frame.ID)
	case FrameDiscover:
		go c.handleDiscover(ctx, conn, frame)
	case FrameJoinNetworks:
		go c.cfg.Executor.JoinNetworks(ctx, frame.GatewayContainer, frame.JoinNetworks, func(fr Frame) { c.sendFrame(ctx, conn, fr) })
	case FrameUpdate:
		go c.handleUpdate(ctx, frame)
	case FrameLogsRequest:
		c.startLogs(ctx, conn, frame)
	case FrameLogsCancel:
		c.stopLogs(frame.ID)
	case FrameHarnessDial:
		c.startStream(streamCtx, ctx, conn, frame.ID)
	case FrameT3PairTokenRequest:
		c.answerPairToken(ctx, conn, frame.ID)
	default:
		c.log.Warn("unexpected server frame", "runner", c.cfg.Name, "type", frame.Type)
	}
}

// startJob cancels any running job and launches the assigned one; a connection drop aborts execution.
func (c *Client) startJob(ctx context.Context, conn *websocket.Conn, f Frame) {
	jobCtx, jobCancel := context.WithCancel(ctx)
	c.mu.Lock()
	if c.jobCancel != nil {
		c.jobCancel()
	}
	c.jobID = f.ID
	c.jobCancel = jobCancel
	c.mu.Unlock()

	if f.Type == FrameAssignUpgrade {
		send := func(fr Frame) error {
			c.sendFrame(ctx, conn, fr)
			if fr.Type == FrameUpgradeResult {
				c.jobFinished(ctx, fr.ID)
			}
			return nil
		}
		// sendFrame logs and swallows write failures rather than returning them, so Upgrade always runs to its
		// terminal frame; the error return exists for Upgrade's own early-return control flow, not for this call.
		go func() { _ = c.cfg.Executor.Upgrade(jobCtx, f, send) }()
		return
	}

	// Every run-strategy field the executor reads must be copied here; a field left out silently never reaches docker.
	req := DeployRequestedEvent{
		ID:               f.ID,
		Repo:             f.Repo,
		Ref:              f.Ref,
		Steps:            f.Steps,
		Service:          f.Service,
		Image:            f.Image,
		Env:              f.Env,
		Strategy:         f.Strategy,
		Network:          f.Network,
		Ports:            f.Ports,
		Mounts:           f.Mounts,
		Command:          f.Command,
		GitToken:         f.GitToken,
		Dockerfile:       f.Dockerfile,
		ComposePath:      f.ComposePath,
		StackSlug:        f.StackSlug,
		StackRoot:        f.StackRoot,
		GatewayContainer: f.GatewayContainer,
		JoinNetworks:     f.JoinNetworks,
	}
	// deploy_result is the job terminal even for a build (build_result is only a phase terminal, per
	// handler.go's dispatchFrame comment) — that's the one point where the client can safely go idle and apply
	// any update frame that arrived mid-job.
	send := func(fr Frame) {
		c.sendFrame(ctx, conn, fr)
		if fr.Type == FrameDeployResult {
			c.jobFinished(ctx, fr.ID)
		}
	}
	switch f.Type {
	case FrameAssignBuild:
		req.Kind = RequestBuild
		go c.cfg.Executor.Build(jobCtx, req, send)
	case FrameAssignDeploy:
		req.Kind = RequestDeploy
		go c.cfg.Executor.Deploy(jobCtx, req, send)
	}
}

// cancelJob aborts the running job, if any and if its id matches.
func (c *Client) cancelJob(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.jobCancel == nil {
		return
	}
	if id != "" && id != c.jobID {
		return
	}
	c.jobCancel()
	c.jobCancel = nil
	c.jobID = ""
}

func (c *Client) heartbeatLoop(ctx context.Context, conn *websocket.Conn) {
	send := func() {
		c.sendFrame(ctx, conn, Frame{Type: FrameHeartbeat, TS: time.Now().Unix()})
	}
	send()
	ticker := time.NewTicker(c.hb)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}

// sendFrame logs write failures instead of failing the caller; the connection drop surfaces on the next read.
func (c *Client) sendFrame(ctx context.Context, conn *websocket.Conn, f Frame) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.WriteTimeout)
	defer cancel()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := wsjson.Write(ctx, conn, f); err != nil {
		c.log.Warn("frame send failed", "runner", c.cfg.Name, "type", f.Type, "error", err)
	}
}

// wsURL carries what the server needs to offer an update: this build's version and platform.
func (c *Client) wsURL() string {
	q := url.Values{}
	if c.cfg.Version != "" {
		q.Set("version", c.cfg.Version)
	}
	q.Set("os", runtime.GOOS)
	q.Set("arch", runtime.GOARCH)
	return c.cfg.URL + "?" + q.Encode()
}

// refusedAsRemoved reports whether a failed handshake was the server saying this runner was removed.
func refusedAsRemoved(resp *http.Response) bool {
	if resp == nil || resp.StatusCode != http.StatusUnauthorized || resp.Body == nil {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return err == nil && strings.Contains(string(body), `"runner_removed"`)
}

// uninstall asks `nexul` to remove this runner's service from outside it; the service stopping ends this process.
func (c *Client) uninstall(ctx context.Context) {
	c.log.Warn("runner was removed from the instance; uninstalling", "runner", c.cfg.Name)
	kind, name := "runner", c.cfg.Name
	if c.cfg.Personal {
		kind, name = "computer", ""
		c.revokeNexulSessions(ctx)
	}
	if err := c.cfg.Executor.Uninstall(ctx, kind, name); err != nil {
		c.log.Error("uninstall failed; remove the service by hand", "runner", c.cfg.Name, "error", err)
	}
}

// cleanupOldBinary best-effort removes the previous binary a completed update left behind.
func (c *Client) cleanupOldBinary() {
	exe, err := runnerExecutable()
	if err != nil {
		return
	}
	if err := os.Remove(exe + ".old"); err != nil && !os.IsNotExist(err) {
		c.log.Warn("remove old runner binary failed", "path", exe+".old", "error", err)
	}
}
