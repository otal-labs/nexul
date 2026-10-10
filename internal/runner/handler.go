package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/release"
	"github.com/otal-labs/nexul/internal/platform/version"
)

// Bus is the slice of the event bus the handler needs (ADR 0019).
type Bus interface {
	Publish(ctx context.Context, topic string, payload any) error
	Subscribe(ctx context.Context, topic string, h eventbus.Handler) error
}

// GitTokenLookup resolves clone credentials and whether a missing token permits the pre-key runner fallback.
type GitTokenLookup interface {
	RepoToken(ctx context.Context, repo string) (CloneCredentials, error)
}

// CloneCredentials explicitly permits the pre-key runner credential fallback; App authorization never does.
type CloneCredentials struct {
	Token         string `json:"-"`
	AllowFallback bool
}

// EnvLookup resolves real env values at dispatch time; deploy.requested only carries redacted keys.
type EnvLookup interface {
	ResolveEnv(ctx context.Context, service string) (map[string]string, error)
}

// HandlerConfig wires the server-side runner WS handler.
type HandlerConfig struct {
	Bus Bus
	// Repo tracks runner presence (heartbeat, connected state).
	Repo Repo
	// Envs is optional; nil means every assign frame carries an empty env.
	Envs EnvLookup
	// GitTokens is optional; nil means a build clones with whatever token the runner's own env provides.
	GitTokens GitTokenLookup
	// Machines is optional; nil means the handshake skips machine resolution and an assign frame's stack_root
	// stays empty (issue 05).
	Machines MachineRepo
	// Upgrades is optional; nil means an upgrade_progress/upgrade_result frame and a disconnect while pending
	// are logged but never persisted.
	Upgrades UpgradeRepo
	// Logger for lifecycle and failure logs. Defaults to slog.Default().
	Logger *slog.Logger
	// HeartbeatInterval is the expected beat cadence. Defaults to 10s.
	HeartbeatInterval time.Duration
	// MissedHeartbeats disconnects a runner after this many missed beats. Defaults to 3.
	MissedHeartbeats int
	// WriteTimeout bounds each frame write to a runner. Defaults to 10s.
	WriteTimeout time.Duration
	// ReadLimit caps a single inbound frame. Defaults to 64KiB.
	ReadLimit int64
}

// Handler is the server-side WebSocket endpoint for runners.
type Handler struct {
	log  *slog.Logger
	bus  Bus
	repo Repo
	cfg  HandlerConfig

	mu      sync.Mutex
	conns   map[string]*runnerConn
	pending []DeployRequestedEvent

	// discoverMu/discoverWaiters correlate in-flight discover jobs to the goroutine waiting on their result.
	discoverMu      sync.Mutex
	discoverWaiters map[string]chan Frame

	// logsMu/logSubs route logs_chunk and logs_end frames to the viewer's feed, keyed by stream id.
	logsMu  sync.Mutex
	logSubs map[string]logSub

	// updateMu guards updateSettings/updateRelease: set once by SetUpdateSource after runnerSvc (and its release
	// client) exist, since that happens after the handler is constructed in server/cmd/workers.go.
	updateMu       sync.Mutex
	updateSettings SettingsReader
	updateRelease  *release.Client
}

type runnerConn struct {
	id   string
	name string
	// machine is the name of the machine the runner was enrolled on; dispatch pools on it.
	machine       string
	machineID     string
	ws            *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	writeMu       sync.Mutex
	hbMu          sync.Mutex
	lastHeartbeat time.Time
	jobMu         sync.Mutex
	job           *DeployRequestedEvent
	upgradeMu     sync.Mutex
	// upgradeID is the instance_upgrades record this connection is running, mirroring job for the busy check; empty when idle.
	upgradeID string
}

// NewHandler wires the server-side runner WS handler.
func NewHandler(cfg HandlerConfig) *Handler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 10 * time.Second
	}
	if cfg.MissedHeartbeats <= 0 {
		cfg.MissedHeartbeats = 3
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.ReadLimit <= 0 {
		cfg.ReadLimit = 64 << 10
	}
	return &Handler{
		log:             cfg.Logger,
		bus:             cfg.Bus,
		repo:            cfg.Repo,
		cfg:             cfg,
		conns:           make(map[string]*runnerConn),
		discoverWaiters: make(map[string]chan Frame),
		logSubs:         make(map[string]logSub),
	}
}

// SetUpdateSource wires the settings reader and release client an "update" frame needs; called once runnerSvc
// (and its release client) exist, since the wiring order in server/cmd/workers.go builds this handler first.
func (h *Handler) SetUpdateSource(settings SettingsReader, rel *release.Client) {
	h.updateMu.Lock()
	h.updateSettings = settings
	h.updateRelease = rel
	h.updateMu.Unlock()
}

// Run subscribes to deploy.requested and deploy.cancel_requested, then serves until ctx is cancelled.
func (h *Handler) Run(ctx context.Context) error {
	if err := h.bus.Subscribe(ctx, TopicDeployRequested, h.handleRequest); err != nil {
		return fmt.Errorf("subscribe %s: %w", TopicDeployRequested, err)
	}
	if err := h.bus.Subscribe(ctx, TopicDeployCancelRequested, h.handleCancelRequest); err != nil {
		return fmt.Errorf("subscribe %s: %w", TopicDeployCancelRequested, err)
	}
	if err := h.bus.Subscribe(ctx, TopicInstanceUpgradeRequested, h.handleUpgradeRequest); err != nil {
		return fmt.Errorf("subscribe %s: %w", TopicInstanceUpgradeRequested, err)
	}
	<-ctx.Done()
	h.closeAll("shutdown")
	return nil
}

// ServeHTTP authenticates a runner by its credential and upgrades the connection, then owns it until disconnect.
// Identity, name and machine come from the runner's record; the query only reports version, os and arch.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rn, machine, ok := h.admit(w, r)
	if !ok {
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.log.Warn("websocket upgrade failed", "error", err)
		return
	}
	conn.SetReadLimit(h.cfg.ReadLimit)
	q := r.URL.Query()
	c := &runnerConn{id: rn.ID, name: rn.Name, machine: machine, machineID: rn.MachineID, ws: conn}
	h.handleConn(r.Context(), c, connectInfo{version: q.Get("version"), os: q.Get("os"), arch: q.Get("arch")})
}

// admit resolves the bearer credential to its runner and machine, writing the refusal itself when it fails: a
// removed runner is told so, so it uninstalls itself rather than retrying forever.
func (h *Handler) admit(w http.ResponseWriter, r *http.Request) (rn *Runner, machine string, ok bool) {
	cred, err := authenticate(r.Context(), h.repo, bearerToken(r))
	if errors.Is(err, apperrs.ErrUnauthorized) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, "", false
	}
	if err != nil {
		h.log.Error("runner credential lookup failed", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, "", false
	}
	if cred.Revoked {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(RemovedRefusal)) // the status alone already refuses; the body is the reason
		return nil, "", false
	}
	rn, err = h.repo.GetByID(r.Context(), cred.RunnerID)
	if err != nil {
		h.log.Error("runner record lookup failed", "runner_id", cred.RunnerID, "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, "", false
	}
	return rn, h.machineName(r.Context(), rn.MachineID), true
}

// machineName resolves the runner's machine for dispatch pooling; empty when machines aren't wired or unknown.
func (h *Handler) machineName(ctx context.Context, machineID string) string {
	if h.cfg.Machines == nil || machineID == "" {
		return ""
	}
	m, err := h.cfg.Machines.Get(ctx, machineID)
	if err != nil {
		h.log.Warn("runner machine lookup failed", "machine_id", machineID, "error", err)
		return ""
	}
	return m.Name
}

func bearerToken(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}

// connectInfo is what a runner reports about its build in its connection URL.
type connectInfo struct {
	version, os, arch string
}

func (h *Handler) handleConn(ctx context.Context, c *runnerConn, info connectInfo) {
	ctx, cancel := context.WithCancel(ctx)
	c.ctx, c.cancel = ctx, cancel
	c.touchHeartbeat()

	h.mu.Lock()
	if old, ok := h.conns[c.id]; ok {
		old.cancel()
		_ = old.ws.Close(websocket.StatusGoingAway, "replaced by new connection")
	}
	h.conns[c.id] = c
	h.mu.Unlock()

	h.log.Info("runner connected", "runner_id", c.id, "name", c.name, "version", info.version, "machine", c.machine)
	if err := h.registerRunner(ctx, c.id, info.version); err != nil {
		h.log.Warn("runner repo register failed", "runner_id", c.id, "error", err)
	}
	h.sendUpdateIfNeeded(ctx, c, c.id, info.version, info.os, info.arch)
	h.touchMachine(ctx, c)
	h.publish(ctx, TopicRunnerConnected, RunnerConnectedEvent{RunnerID: c.id, Name: c.name})

	h.dispatchPending(ctx, c)

	go h.watchdog(c)
	h.readLoop(ctx, c)
}

// sendUpdateIfNeeded pushes an "update" frame when this is a release build and the connecting runner reports a
// different, real version; best-effort like resolveMachine — a lookup failure never blocks or fails the
// connection, it just skips the push.
func (h *Handler) sendUpdateIfNeeded(ctx context.Context, c *runnerConn, runnerID, reportedVersion, osName, arch string) {
	if !version.IsRelease() || reportedVersion == "" || reportedVersion == "dev" || reportedVersion == version.Version {
		return
	}
	h.updateMu.Lock()
	settings, rel := h.updateSettings, h.updateRelease
	h.updateMu.Unlock()
	if settings == nil || rel == nil {
		return
	}
	target := osName + "-" + arch
	assetName, ok := AssetName(target)
	if !ok {
		return
	}
	instanceURL, err := settings.GetInstanceURL(ctx)
	if err != nil || instanceURL == "" {
		h.log.Warn("runner update instance url lookup failed", "runner_id", runnerID, "error", err)
		return
	}
	downloadURL := fmt.Sprintf("%s/%s?version=%s", InstallDownloadURL(instanceURL, "", false), target, url.QueryEscape(version.Version))
	sha := h.updateChecksum(ctx, rel, runnerID, assetName)
	if err := h.sendFrame(ctx, c, Frame{Type: FrameUpdate, Version: version.Version, URL: downloadURL, Sha256: sha}); err != nil {
		h.log.Warn("update frame send failed", "runner_id", runnerID, "error", err)
	}
}

// updateChecksum looks up this server's own release's sha256 for assetName; any lookup failure logs at warn and
// returns "", so the runner still gets the update frame and skips verification.
func (h *Handler) updateChecksum(ctx context.Context, rel *release.Client, runnerID, assetName string) string {
	rl, err := rel.ByTag(ctx, version.Version)
	if err != nil {
		h.log.Warn("runner update release lookup failed", "runner_id", runnerID, "version", version.Version, "error", err)
		return ""
	}
	sums, err := rel.Checksums(ctx, rl)
	if err != nil {
		h.log.Warn("runner update checksum lookup failed", "runner_id", runnerID, "version", version.Version, "error", err)
		return ""
	}
	return sums[assetName]
}

// registerRunner marks the enrolled runner connected and records the build version it reported on this connect.
func (h *Handler) registerRunner(ctx context.Context, id, version string) error {
	if err := h.repo.SetVersion(ctx, id, version); err != nil {
		return err
	}
	return h.repo.SetConnected(ctx, id, true)
}

// handleRequest is the deploy.requested consumer; a malformed payload is fatal (dead-lettered).
func (h *Handler) handleRequest(ctx context.Context, ev eventbus.Event) error {
	var req DeployRequestedEvent
	if err := json.Unmarshal(ev.Payload, &req); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse deploy.requested: %w", err))
	}
	if _, err := req.toFrame(nil); err != nil {
		return apperrs.Fatal(fmt.Errorf("deploy.requested %s: %w", req.ID, err))
	}
	c := h.idleConnected(req)
	if c == nil {
		h.queue(req)
		return nil
	}
	if err := h.dispatchOne(ctx, c, req); err != nil {
		if h.failDeniedBuild(ctx, req, err) {
			return nil
		}
		h.queue(req)
	}
	return nil
}

// handleCancelRequest is the deploy.cancel_requested consumer.
func (h *Handler) handleCancelRequest(ctx context.Context, ev eventbus.Event) error {
	var p DeployCancelRequestedEvent
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse deploy.cancel_requested: %w", err))
	}
	return h.Cancel(ctx, p.ID)
}

// idleConnected returns one connected runner with no running job that req fits, or nil when none is free.
func (h *Handler) idleConnected(req DeployRequestedEvent) *runnerConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.conns {
		if c.jobSnapshot() == nil && c.upgradeSnapshot() == "" && req.fits(c) {
			return c
		}
	}
	return nil
}

// queue appends a request to the pending list for the next free runner.
func (h *Handler) queue(req DeployRequestedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending = append(h.pending, req)
}

// dispatchOne assigns one request to a runner; a send/env failure leaves the job unowned to requeue.
func (h *Handler) dispatchOne(ctx context.Context, c *runnerConn, req DeployRequestedEvent) error {
	frame, err := h.buildFrame(ctx, req)
	if err != nil {
		return err
	}
	// Owned before the frame goes out, so a cancel racing the runner's start finds the job instead of dropping.
	c.setJob(&req)
	if err := h.sendFrame(ctx, c, frame); err != nil {
		c.clearJob(req.ID)
		return err
	}
	return nil
}

// buildFrame resolves the secrets a job needs at dispatch time: real env values and, for builds, a clone token.
func (h *Handler) buildFrame(ctx context.Context, req DeployRequestedEvent) (Frame, error) {
	env, err := h.resolveEnv(ctx, req)
	if err != nil {
		return Frame{}, err
	}
	frame, err := req.toFrame(env)
	if err != nil {
		return Frame{}, err
	}
	if req.Kind == RequestBuild && h.cfg.GitTokens != nil {
		token, err := h.cloneToken(ctx, req.Repo)
		if err != nil {
			return Frame{}, err
		}
		frame.GitToken = token
	}
	frame.StackRoot = h.stackRootFor(ctx, req.Target)
	return frame, nil
}

func (h *Handler) cloneToken(ctx context.Context, repo string) (string, error) {
	credentials, err := h.cfg.GitTokens.RepoToken(ctx, repo)
	if credentials.AllowFallback {
		if err != nil {
			h.log.Warn("legacy clone uses runner credentials", "repo", repo, "error", err)
		}
		return credentials.Token, nil
	}
	if err != nil {
		err = fmt.Errorf("authorize repository clone: %w", err)
		if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) || errors.Is(err, apperrs.ErrUnauthorized) || errors.Is(err, apperrs.ErrInvalid) {
			return "", apperrs.Fatal(err)
		}
		return "", err
	}
	if credentials.Token == "" {
		return "", apperrs.Fatal(fmt.Errorf("%w: the App returned no clone credential", apperrs.ErrUnauthorized))
	}
	return credentials.Token, nil
}

func (h *Handler) failDeniedBuild(ctx context.Context, req DeployRequestedEvent, err error) bool {
	if req.Kind != RequestBuild || !errors.Is(err, apperrs.ErrFatal) {
		return false
	}
	h.publish(ctx, TopicDeployBuildCompleted, BuildCompletedEvent{ID: req.ID, Status: BuildStatusFailed, Error: err.Error()})
	return true
}

// stackRootFor looks up the target machine's stack root at dispatch time (spec §2/§4); empty when Machines
// isn't wired, the job has no target machine, or the lookup fails (the runner falls back to its own default).
func (h *Handler) stackRootFor(ctx context.Context, machine string) string {
	if h.cfg.Machines == nil || machine == "" {
		return ""
	}
	m, err := h.cfg.Machines.GetByName(ctx, machine)
	if err != nil {
		h.log.Warn("stack root lookup failed", "machine", machine, "error", err)
		return ""
	}
	return m.StackRoot
}

// resolveEnv fetches real env values for dispatch; never logged, never republished.
func (h *Handler) resolveEnv(ctx context.Context, req DeployRequestedEvent) (map[string]string, error) {
	if h.cfg.Envs == nil || req.Service == "" {
		return nil, nil
	}
	env, err := h.cfg.Envs.ResolveEnv(ctx, req.Service)
	if err != nil {
		return nil, fmt.Errorf("resolve env for service %s: %w", req.Service, err)
	}
	return env, nil
}

// dispatchPending hands queued work to a runner, one job at a time; a send failure requeues and stops.
func (h *Handler) dispatchPending(ctx context.Context, c *runnerConn) {
	for {
		if c.jobSnapshot() != nil || c.upgradeSnapshot() != "" {
			return
		}
		req, ok := h.popFitting(c)
		if !ok {
			return
		}

		if err := h.dispatchOne(ctx, c, req); err != nil {
			if h.failDeniedBuild(ctx, req, err) {
				continue
			}
			h.log.Warn("queued dispatch failed", "runner_id", c.id, "id", req.ID, "error", err)
			h.mu.Lock()
			h.pending = append([]DeployRequestedEvent{req}, h.pending...)
			h.mu.Unlock()
			return
		}
	}
}

// popFitting removes and returns the first pending job that fits c, leaving the order of the rest untouched.
func (h *Handler) popFitting(c *runnerConn) (DeployRequestedEvent, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, req := range h.pending {
		if req.fits(c) {
			h.pending = append(h.pending[:i], h.pending[i+1:]...)
			return req, true
		}
	}
	return DeployRequestedEvent{}, false
}

// readLoop drains inbound frames until close; a dropped mid-job runner fails the job, never requeues it.
func (h *Handler) readLoop(ctx context.Context, c *runnerConn) {
	defer func() {
		c.cancel()
		// c.cancel() just ended ctx; SQLite refuses a cancelled ctx, so the teardown writes get one that outlives it.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.cfg.WriteTimeout)
		defer cancel()
		h.removeConn(c)
		h.endLogSubs(c)
		if job := c.jobSnapshot(); job != nil {
			h.publish(ctx, TopicDeployStatusChanged, DeployStatusChangedEvent{
				ID: job.ID, Status: DeployStatusFailed, Error: "runner disconnected",
			})
		}
		if id := c.upgradeSnapshot(); id != "" {
			h.failPendingUpgrade(ctx, id)
		}
		// A removed runner's row is already gone, so there is nothing to mark disconnected.
		if err := h.repo.SetConnected(ctx, c.id, false); err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			h.log.Warn("runner disconnect repo update failed", "runner_id", c.id, "error", err)
		}
		h.publish(ctx, TopicRunnerDisconnected, RunnerDisconnectedEvent{RunnerID: c.id, Reason: "disconnected"})
		_ = c.ws.Close(websocket.StatusNormalClosure, "")
	}()
	for {
		var raw json.RawMessage
		if err := wsjson.Read(ctx, c.ws, &raw); err != nil {
			return
		}
		frame, err := ParseFrame(raw)
		if err != nil {
			h.log.Warn("invalid frame from runner", "runner_id", c.id, "error", err)
			_ = c.ws.Close(websocket.StatusPolicyViolation, "invalid frame")
			return
		}
		if err := h.dispatchFrame(ctx, c, *frame); err != nil {
			h.log.Warn("frame dispatch failed", "runner_id", c.id, "type", frame.Type, "error", err)
		}
	}
}

// dispatchFrame translates a runner frame into a bus event; a terminal result frees the runner.
func (h *Handler) dispatchFrame(ctx context.Context, c *runnerConn, f Frame) error {
	switch f.Type {
	case FrameHeartbeat:
		c.touchHeartbeat()
		if err := h.repo.Heartbeat(ctx, c.id, time.Unix(f.TS, 0).UTC()); err != nil {
			return fmt.Errorf("runner heartbeat repo: %w", err)
		}
		h.touchMachine(ctx, c)
		return h.bus.Publish(ctx, TopicRunnerHeartbeat, RunnerHeartbeatEvent{RunnerID: c.id, TS: f.TS})
	case FrameBuildProgress:
		if f.Step == 1 {
			return h.bus.Publish(ctx, TopicDeployBuildStarted, BuildStartedEvent{ID: f.ID, Total: f.Total, Log: f.Log})
		}
		return h.bus.Publish(ctx, TopicDeployBuildProgress, BuildProgressEvent{ID: f.ID, Step: f.Step, Total: f.Total, Log: f.Log})
	case FrameBuildResult:
		// build_result is a phase terminal, not a job terminal: the runner deploys next, freed by deploy_result.
		return h.bus.Publish(ctx, TopicDeployBuildCompleted, BuildCompletedEvent{ID: f.ID, Status: f.Status, Artifacts: f.Artifacts, Error: f.Error})
	case FrameDeployProgress:
		return h.bus.Publish(ctx, TopicDeployDeployProgress, DeployProgressEvent{ID: f.ID, Phase: f.Phase, Log: f.Log})
	case FrameDeployLog:
		return h.bus.Publish(ctx, TopicDeployLog, DeployLogEvent{ID: f.ID, Phase: f.Phase, Log: f.Log, TS: f.TS})
	case FrameDeployResult:
		c.clearJob(f.ID)
		h.dispatchPending(ctx, c)
		return h.bus.Publish(ctx, TopicDeployStatusChanged, DeployStatusChangedEvent{
			ID: f.ID, Status: f.Status, Error: f.Error, Address: f.Address, Services: f.Services,
		})
	case FrameDiscoverResult:
		h.deliverDiscoverResult(f)
		return nil
	case FrameLogsChunk, FrameLogsEnd:
		h.deliverLogs(f)
		return nil
	case FrameJoinNetworksResult:
		if f.Status == BuildStatusFailed {
			h.log.Warn("join_networks failed on runner", "runner_id", c.id, "gateway_container", f.GatewayContainer, "error", f.Error)
		}
		return nil
	case FrameUpgradeProgress:
		h.log.Info("instance upgrade progress", "runner_id", c.id, "upgrade_id", f.ID, "log", f.Log)
		return nil
	case FrameUpgradeResult:
		return h.handleUpgradeResult(ctx, c, f)
	default:
		return nil
	}
}

// handleUpgradeRequest is the instance.upgrade_requested consumer (TopicInstanceUpgradeRequested's own doc
// comment explains the bus-topic choice): it dispatches assign_upgrade to the connected runner named instance.
// RequestUpgrade already checked the runner was free; a disconnect between that check and this call is not an
// error here — UpgradeStatus's lazy resolution fails the record after upgradeResolveWindow either way.
func (h *Handler) handleUpgradeRequest(ctx context.Context, ev eventbus.Event) error {
	var req InstanceUpgradeRequestedEvent
	if err := json.Unmarshal(ev.Payload, &req); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse instance.upgrade_requested: %w", err))
	}
	c := h.connNamed(instanceRunnerName)
	if c == nil {
		h.log.Warn("instance upgrade requested but the instance runner is not connected", "upgrade_id", req.ID)
		return nil
	}
	frame := Frame{Type: FrameAssignUpgrade, ID: req.ID, Version: req.Version}
	if err := h.sendFrame(ctx, c, frame); err != nil {
		h.log.Warn("assign_upgrade send failed", "upgrade_id", req.ID, "error", err)
		return nil
	}
	c.setUpgrade(req.ID)
	return nil
}

// handleUpgradeResult persists the runner's started/failed verdict and announces it; a failed result frees the
// runner's slot, while started leaves it held since the runner (and this connection) is about to be recreated.
func (h *Handler) handleUpgradeResult(ctx context.Context, c *runnerConn, f Frame) error {
	if f.Status == UpgradeStatusFailed {
		c.clearUpgrade(f.ID)
	}
	if h.cfg.Upgrades == nil {
		return nil
	}
	if err := h.cfg.Upgrades.SetStatus(ctx, f.ID, f.Status, f.Error); err != nil {
		return fmt.Errorf("set instance upgrade %s status: %w", f.ID, err)
	}
	u, err := h.cfg.Upgrades.GetByID(ctx, f.ID)
	if err != nil {
		return fmt.Errorf("get instance upgrade %s: %w", f.ID, err)
	}
	h.publish(ctx, TopicInstanceUpgradeChanged, u)
	return nil
}

// failPendingUpgrade fails id's record on disconnect, but only while it is still pending — a disconnect after
// upgrade_result "started" is the expected restart of the runner, not a failure.
func (h *Handler) failPendingUpgrade(ctx context.Context, id string) {
	if h.cfg.Upgrades == nil {
		return
	}
	u, err := h.cfg.Upgrades.GetByID(ctx, id)
	if err != nil {
		h.log.Warn("instance upgrade lookup on disconnect failed", "upgrade_id", id, "error", err)
		return
	}
	if u.Status != UpgradeStatusPending {
		return
	}
	if err := h.cfg.Upgrades.SetStatus(ctx, id, UpgradeStatusFailed, "runner disconnected"); err != nil {
		h.log.Warn("instance upgrade fail-on-disconnect failed", "upgrade_id", id, "error", err)
		return
	}
	u.Status = UpgradeStatusFailed
	u.Error = "runner disconnected"
	h.publish(ctx, TopicInstanceUpgradeChanged, u)
}

// touchMachine refreshes the connected runner's machine last_seen at connect and on every heartbeat; best-effort.
func (h *Handler) touchMachine(ctx context.Context, c *runnerConn) {
	if h.cfg.Machines == nil || c.machineID == "" {
		return
	}
	if err := h.cfg.Machines.Touch(ctx, c.machineID, c.machine, time.Now().UTC()); err != nil {
		h.log.Warn("machine touch failed", "runner_id", c.id, "machine_id", c.machineID, "error", err)
	}
}

// watchdog disconnects a runner that misses MissedHeartbeats beats.
func (h *Handler) watchdog(c *runnerConn) {
	ticker := time.NewTicker(h.cfg.HeartbeatInterval)
	defer ticker.Stop()
	window := h.cfg.HeartbeatInterval * time.Duration(h.cfg.MissedHeartbeats)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if time.Since(c.lastBeat()) > window {
				h.log.Warn("runner heartbeat timeout", "runner_id", c.id)
				_ = c.ws.Close(websocket.StatusPolicyViolation, "heartbeat timeout")
				return
			}
		}
	}
}

func (h *Handler) closeAll(reason string) {
	h.mu.Lock()
	conns := make([]*runnerConn, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		c.cancel()
		_ = c.ws.Close(websocket.StatusGoingAway, reason)
	}
}

func (h *Handler) removeConn(c *runnerConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.id] == c {
		delete(h.conns, c.id)
	}
}

// Runners returns the live connection set with each runner's running job.
func (h *Handler) Runners() []RunnerStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]RunnerStatus, 0, len(h.conns))
	for id, c := range h.conns {
		var job *RunningJob
		if req := c.jobSnapshot(); req != nil {
			job = req.RunningJob()
		}
		out = append(out, RunnerStatus{RunnerID: id, Name: c.name, RunningJob: job})
	}
	return out
}

// Queue returns the deploys waiting for a runner, FIFO order.
func (h *Handler) Queue() []QueuedJob {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]QueuedJob, 0, len(h.pending))
	for _, req := range h.pending {
		out = append(out, req.Queued())
	}
	return out
}

// Cancel stops a queued or running job; unknown ids are a no-op.
func (h *Handler) Cancel(ctx context.Context, id string) error {
	h.mu.Lock()
	removed := false
	kept := h.pending[:0]
	for _, req := range h.pending {
		if req.ID == id {
			removed = true
			continue
		}
		kept = append(kept, req)
	}
	h.pending = kept
	var c *runnerConn
	for _, conn := range h.conns {
		if job := conn.jobSnapshot(); job != nil && job.ID == id {
			c = conn
			break
		}
	}
	h.mu.Unlock()

	if removed {
		h.publish(ctx, TopicDeployStatusChanged, DeployStatusChangedEvent{ID: id, Status: DeployStatusFailed, Error: "cancelled"})
		return nil
	}
	if c != nil {
		// clearJob only drops the job if it still matches, guarding against a racing reassignment.
		c.clearJob(id)
		if err := h.sendFrame(ctx, c, Frame{Type: FrameCancel, ID: id}); err != nil {
			h.log.Warn("cancel frame send failed", "runner_id", c.id, "id", id, "error", err)
		}
		h.publish(ctx, TopicDeployStatusChanged, DeployStatusChangedEvent{ID: id, Status: DeployStatusFailed, Error: "cancelled"})
		h.dispatchPending(ctx, c)
		return nil
	}
	return nil
}

// JoinNetworks pushes an ad hoc join_networks frame to the connected runner for machine: an exposure created
// for an already-running stack. Best-effort — a runner that isn't currently
// connected is not an error, since the stack's next deploy carries the same join step (assign_deploy's own
// gateway_container/join_networks fields).
func (h *Handler) JoinNetworks(ctx context.Context, machine, gatewayContainer string, networks []string) error {
	c := h.connOnMachine(machine)
	if c == nil {
		return nil
	}
	return h.sendFrame(ctx, c, Frame{Type: FrameJoinNetworks, GatewayContainer: gatewayContainer, JoinNetworks: networks})
}

// connOnMachine returns any connected runner on machine, busy or not, or nil.
func (h *Handler) connOnMachine(machine string) *runnerConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.conns {
		if c.machine == machine {
			return c
		}
	}
	return nil
}

// connNamed returns the connected runner named name, or nil.
func (h *Handler) connNamed(name string) *runnerConn {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.conns {
		if c.name == name {
			return c
		}
	}
	return nil
}

// Uninstall tells a connected runner it was removed and drops it from dispatch at once; the runner uninstalls
// itself and exits, and the close completes in the background so removal never waits on it.
func (h *Handler) Uninstall(ctx context.Context, runnerID string) {
	h.mu.Lock()
	c := h.conns[runnerID]
	if c != nil {
		delete(h.conns, runnerID)
	}
	h.mu.Unlock()
	if c == nil {
		return
	}
	if err := h.sendFrame(ctx, c, Frame{Type: FrameUninstall}); err != nil {
		h.log.Warn("uninstall frame send failed", "runner_id", runnerID, "error", err)
	}
	go func() { _ = c.ws.Close(websocket.StatusNormalClosure, "runner removed") }() // the read loop sees the close either way
}

// sendFrame writes one frame; safe for concurrent callers (writeMu serializes frames).
func (h *Handler) sendFrame(ctx context.Context, c *runnerConn, f Frame) error {
	ctx, cancel := context.WithTimeout(ctx, h.cfg.WriteTimeout)
	defer cancel()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return wsjson.Write(ctx, c.ws, f)
}

func (h *Handler) publish(ctx context.Context, topic string, payload any) {
	if err := h.bus.Publish(ctx, topic, payload); err != nil {
		h.log.Warn("bus publish failed", "topic", topic, "error", err)
	}
}

func (c *runnerConn) touchHeartbeat() {
	c.hbMu.Lock()
	c.lastHeartbeat = time.Now()
	c.hbMu.Unlock()
}

func (c *runnerConn) lastBeat() time.Time {
	c.hbMu.Lock()
	defer c.hbMu.Unlock()
	return c.lastHeartbeat
}

// setJob records the job this runner is executing, or clears it with nil.
func (c *runnerConn) setJob(job *DeployRequestedEvent) {
	c.jobMu.Lock()
	c.job = job
	c.jobMu.Unlock()
}

// jobSnapshot returns the current job, or nil when the runner is idle.
func (c *runnerConn) jobSnapshot() *DeployRequestedEvent {
	c.jobMu.Lock()
	defer c.jobMu.Unlock()
	return c.job
}

// clearJob drops the job when it matches id (terminal result, cancel, or disconnect).
func (c *runnerConn) clearJob(id string) {
	c.jobMu.Lock()
	defer c.jobMu.Unlock()
	if c.job != nil && c.job.ID == id {
		c.job = nil
	}
}

// setUpgrade records the instance_upgrades id this connection is running.
func (c *runnerConn) setUpgrade(id string) {
	c.upgradeMu.Lock()
	c.upgradeID = id
	c.upgradeMu.Unlock()
}

// upgradeSnapshot returns the running upgrade id, or "" when idle.
func (c *runnerConn) upgradeSnapshot() string {
	c.upgradeMu.Lock()
	defer c.upgradeMu.Unlock()
	return c.upgradeID
}

// clearUpgrade drops the upgrade id when it matches id (a failed result; a started result leaves it set since
// the connection is about to be torn down by the runner's own restart).
func (c *runnerConn) clearUpgrade(id string) {
	c.upgradeMu.Lock()
	defer c.upgradeMu.Unlock()
	if c.upgradeID == id {
		c.upgradeID = ""
	}
}
