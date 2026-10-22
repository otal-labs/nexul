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

	"github.com/otal-labs/nexul/internal/platform/logging"
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
	// T3Supervised means a supervisor on the computer restarts T3 Code, so the runner never does.
	T3Supervised bool
	// RunnerLog is a personal runner's own log file, which a logs_request for the runner reads; nil serves none.
	RunnerLog *RunnerLog
	// Containers is how a machine runner reads its host's containers; nil, and any personal runner, watches none.
	Containers *ContainerWatch
	// Logs queues this runner's own log records for the server to relay to the log store; only a machine's runner
	// sets it, and a personal runner never ships even when it is set.
	Logs *logging.Queue
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
	// dockerFacts reads Docker on the computer; dockerAvailable is what the last facts report said.
	dockerFacts     func(ctx context.Context) *DockerFacts
	dockerAvailable atomic.Bool
	// logEvery is how often queued log records go to the server.
	logEvery time.Duration
	// requestEvery is how often a proxy gateway's request counters are read.
	requestEvery time.Duration
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
	c := &Client{cfg: cfg, log: cfg.Logger, hb: cfg.HeartbeatInterval, logs: map[string]context.CancelFunc{}, dial: (&net.Dialer{}).DialContext, findT3: findT3, dockerFacts: dockerFacts, logEvery: logFlushEvery, requestEvery: requestCountsEvery}
	c.t3 = newT3Keeper(c.t3Probe, c.restartT3, cfg.Logger)
	c.t3.supervised = cfg.T3Supervised
	return c
}

// logFlushEvery is how often queued log records go to the server.
const logFlushEvery = 2 * time.Second

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
		connected, err := c.runOnce(ctx)
		if errors.Is(err, errRemoved) {
			c.uninstall(ctx)
			return nil
		}
		if ctx.Err() != nil {
			return nil
		}
		// A drop after a connection that came up starts the backoff over; only failed dials in a row grow it.
		if connected {
			attempt = 0
		}
		attempt++
		c.log.Warn("runner connection lost; reconnecting", "error", err, "attempt", attempt)
		if !backoff.Wait(ctx, attempt) {
			return nil
		}
	}
}

// runOnce maintains one connection: heartbeats on a ticker, assigns run in goroutines, cancels abort the job.
// connected reports whether the dial succeeded, whatever ended the connection afterwards.
func (c *Client) runOnce(ctx context.Context) (connected bool, err error) {
	dialCtx, cancel := context.WithTimeout(ctx, c.cfg.ConnectTimeout)
	conn, resp, dialErr := websocket.Dial(dialCtx, c.wsURL(), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + c.cfg.Credential}},
	})
	cancel()
	if dialErr != nil {
		if refusedAsRemoved(resp) {
			return false, errRemoved
		}
		return false, fmt.Errorf("dial %s: %w", c.cfg.URL, dialErr)
	}
	connected = true
	defer func() {
		err = errors.Join(err, conn.CloseNow())
	}()
	// A gateway_routes frame grows with the exposures; the library's 32KiB default would drop it.
	conn.SetReadLimit(1 << 20)
	c.log.Info("runner connected", "runner", c.cfg.Name)

	streamCtx := ctx
	var loops sync.WaitGroup
	defer loops.Wait() // after cancel: a facts report still reading the computer must not outlive its connection
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	defer c.cancelJob("")

	loops.Go(func() { c.heartbeatLoop(ctx, conn) })
	loops.Go(func() { c.pingLoop(ctx, conn) })
	if c.cfg.Personal {
		loops.Go(func() { c.factsLoop(ctx, conn) })
	}
	if !c.cfg.Personal && c.cfg.Containers != nil {
		loops.Go(func() { c.watchContainers(ctx, conn) })
		if c.cfg.Containers.Fetch != nil {
			loops.Go(func() { c.watchRequests(ctx, conn) })
		}
	}
	if c.cfg.Logs != nil && !c.cfg.Personal {
		loops.Go(func() { c.logLoop(ctx, conn) })
	}

	for {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return true, err
		}
		frame, err := ParseFrame(raw)
		if err != nil {
			c.log.Warn("invalid frame from server", "runner", c.cfg.Name, "error", err)
			continue
		}
		if frame.Type == FrameUninstall {
			return true, errRemoved
		}
		c.handleFrame(streamCtx, ctx, conn, *frame)
	}
}

// handleFrame acts on one server frame; streamCtx outlives this connection, ctx does not.
func (c *Client) handleFrame(streamCtx, ctx context.Context, conn *websocket.Conn, frame Frame) {
	if reply, refused := refusal(frame); refused && c.cfg.Personal && !c.servesDocker(frame) {
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
		go c.joinNetworks(ctx, conn, frame)
	case FrameGatewayRoutes:
		// Inline, not a goroutine: two changes in a row must land in the order they were sent.
		c.setRoutes(ctx, conn, frame)
	case FrameStackTeardown:
		// Inline for the same reason: a teardown must finish before the deploy that reuses its slug starts.
		c.teardownStack(ctx, conn, frame)
	case FrameUpdate:
		go c.handleUpdate(ctx, frame)
	case FrameLogsRequest:
		c.startLogs(ctx, conn, frame)
	case FrameLogsCancel:
		c.stopLogs(frame.ID)
	case FrameHarnessDial:
		c.startStream(streamCtx, ctx, conn, frame.ID)
	case FrameT3PairTokenRequest, FrameGitClone:
		c.answerForComputer(ctx, conn, frame)
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
		go func() {
			defer c.contained(f, func(msg string) { _ = send(upgradeFailed(f, msg)) })
			_ = c.cfg.Executor.Upgrade(jobCtx, f, send)
		}()
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

func (c *Client) joinNetworks(ctx context.Context, conn *websocket.Conn, f Frame) {
	defer c.contained(f, func(msg string) {
		c.sendFrame(ctx, conn, Frame{Type: FrameJoinNetworksResult, GatewayContainer: f.GatewayContainer, Status: BuildStatusFailed, Error: msg})
	})
	c.cfg.Executor.JoinNetworks(ctx, f.GatewayContainer, f.JoinNetworks, func(fr Frame) { c.sendFrame(ctx, conn, fr) })
}

func (c *Client) setRoutes(ctx context.Context, conn *websocket.Conn, f Frame) {
	defer c.contained(f, func(msg string) {
		c.sendFrame(ctx, conn, Frame{Type: FrameGatewayRoutesResult, GatewayContainer: f.GatewayContainer, Status: BuildStatusFailed, Error: msg})
	})
	ctx, cancel := context.WithTimeout(ctx, setRoutesTimeout)
	defer cancel()
	c.cfg.Executor.SetRoutes(ctx, f.GatewayContainer, f.GatewayRoutes, func(fr Frame) { c.sendFrame(ctx, conn, fr) })
}

func (c *Client) teardownStack(ctx context.Context, conn *websocket.Conn, f Frame) {
	defer c.contained(f, func(msg string) {
		c.sendFrame(ctx, conn, Frame{Type: FrameStackTeardownResult, StackSlug: f.StackSlug, Status: BuildStatusFailed, Error: msg})
	})
	ctx, cancel := context.WithTimeout(ctx, stackTeardownTimeout)
	defer cancel()
	c.cfg.Executor.Teardown(ctx, f.StackSlug, f.Strategy, func(fr Frame) { c.sendFrame(ctx, conn, fr) })
}

// contained is deferred first in a goroutine the client starts: a panic there is logged and answered with reply
// (nil when no one waits), because it would otherwise end the whole runner and every stream on it.
func (c *Client) contained(f Frame, reply func(msg string)) {
	r := recover()
	if r == nil {
		return
	}
	c.log.Error("runner goroutine panicked", "runner", c.cfg.Name, "id", f.ID, "type", f.Type, "panic", r)
	if reply != nil {
		reply(fmt.Sprint("the runner hit an unexpected error: ", r))
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

// pingMisses is how many heartbeat intervals a ping waits for its pong before the connection counts as dead.
const pingMisses = 3

// pingLoop ends a connection whose server stopped answering, such as one a sleeping host left half open: nothing else
// arrives to time the read out, so without it the runner would wait on a dead socket forever.
func (c *Client) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(c.hb)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		pingCtx, cancel := context.WithTimeout(ctx, pingMisses*c.hb)
		err := conn.Ping(pingCtx)
		cancel()
		if err == nil || ctx.Err() != nil {
			continue
		}
		c.log.Warn("runner connection dead: the server stopped answering pings; reconnecting", "runner", c.cfg.Name, "waited", (pingMisses * c.hb).String(), "error", err)
		_ = conn.CloseNow() // ends the read in runOnce, whose loop dials again
		return
	}
}

// logLoop sends the queued log records on connect, which is how a reconnect replays what piled up, and every
// logEvery after.
func (c *Client) logLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(c.logEvery)
	defer ticker.Stop()
	for {
		c.flushLogs(ctx, conn)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// flushLogs sends batches until the queue is empty or one fails; a failed batch stays queued and is not logged,
// since the lost connection is.
func (c *Client) flushLogs(ctx context.Context, conn *websocket.Conn) {
	for {
		recs, next, dropped := c.cfg.Logs.Peek(logging.MaxBatch, logging.MaxBatchBytes)
		if len(recs) == 0 {
			return
		}
		if err := c.writeFrame(ctx, conn, Frame{Type: FrameLog, Records: recs}); err != nil {
			return
		}
		c.cfg.Logs.Ack(next, dropped)
	}
}

// sendFrame logs write failures instead of failing the caller; the connection drop surfaces on the next read.
func (c *Client) sendFrame(ctx context.Context, conn *websocket.Conn, f Frame) {
	if err := c.writeFrame(ctx, conn, f); err != nil {
		c.log.Warn("frame send failed", "runner", c.cfg.Name, "type", f.Type, "error", err)
	}
}

func (c *Client) writeFrame(ctx context.Context, conn *websocket.Conn, f Frame) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.WriteTimeout)
	defer cancel()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return wsjson.Write(ctx, conn, f)
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
