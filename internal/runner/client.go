package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// ClientConfig wires the runner host binary's connection to the server.
type ClientConfig struct {
	URL      string
	Token    string
	RunnerID string
	Name     string
	// Machine is the machine this runner reports on connect (NEXUL_MACHINE, issue 05).
	Machine string
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
	return &Client{cfg: cfg, log: cfg.Logger, hb: cfg.HeartbeatInterval}
}

// Run dials the server and keeps the runner connected; a dropped connection retries with exponential backoff.
func (c *Client) Run(ctx context.Context) error {
	c.cleanupOldBinary()
	c.log.Info("runner starting", "runner_id", c.cfg.RunnerID, "server", c.cfg.URL)
	backoff := NewBackoff(c.cfg.BackoffBase, c.cfg.BackoffMax,
		rand.New(rand.NewSource(time.Now().UnixNano())))
	attempt := 0
	for {
		err := c.runOnce(ctx)
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
	conn, _, dialErr := websocket.Dial(dialCtx, c.wsURL(), nil)
	cancel()
	if dialErr != nil {
		return fmt.Errorf("dial %s: %w", c.cfg.URL, dialErr)
	}
	defer func() {
		err = errors.Join(err, conn.CloseNow())
	}()
	c.log.Info("runner connected", "runner_id", c.cfg.RunnerID)

	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	defer c.cancelJob("")

	go c.heartbeatLoop(ctx, conn)

	for {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return err
		}
		frame, err := ParseFrame(raw)
		if err != nil {
			c.log.Warn("invalid frame from server", "runner_id", c.cfg.RunnerID, "error", err)
			continue
		}
		switch frame.Type {
		case FrameAssignBuild, FrameAssignDeploy, FrameAssignUpgrade:
			c.startJob(ctx, conn, *frame)
		case FrameCancel:
			c.cancelJob(frame.ID)
		case FrameDiscover:
			go c.handleDiscover(ctx, conn, *frame)
		case FrameJoinNetworks:
			jn := *frame
			go c.cfg.Executor.JoinNetworks(ctx, jn.GatewayContainer, jn.JoinNetworks, func(fr Frame) { c.sendFrame(ctx, conn, fr) })
		case FrameUpdate:
			go c.handleUpdate(ctx, *frame)
		default:
			c.log.Warn("unexpected server frame", "runner_id", c.cfg.RunnerID, "type", frame.Type)
		}
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
		c.sendFrame(ctx, conn, Frame{Type: FrameHeartbeat, RunnerID: c.cfg.RunnerID, TS: time.Now().Unix()})
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
		c.log.Warn("frame send failed", "runner_id", c.cfg.RunnerID, "type", f.Type, "error", err)
	}
}

// wsURL builds the connection URL with the auth token and identity query params.
func (c *Client) wsURL() string {
	q := url.Values{}
	q.Set("token", c.cfg.Token)
	q.Set("runner_id", c.cfg.RunnerID)
	q.Set("name", c.cfg.Name)
	q.Set("machine", c.cfg.Machine)
	if c.cfg.Version != "" {
		q.Set("version", c.cfg.Version)
	}
	q.Set("os", runtime.GOOS)
	q.Set("arch", runtime.GOARCH)
	return c.cfg.URL + "?" + q.Encode()
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
