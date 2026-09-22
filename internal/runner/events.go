package runner

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Topics the runner domain publishes onto the bus; deploy.requested is the inverse, consumed not published.
const (
	TopicRunnerConnected       = "runner.connected"
	TopicRunnerDisconnected    = "runner.disconnected"
	TopicRunnerHeartbeat       = "runner.heartbeat"
	TopicDeployRequested       = "deploy.requested"
	TopicDeployCancelRequested = "deploy.cancel_requested"
	TopicDeployBuildStarted    = "deploy.build_started"
	TopicDeployBuildProgress   = "deploy.build_progress"
	TopicDeployBuildCompleted  = "deploy.build_completed"
	TopicDeployDeployProgress  = "deploy.deploy_progress"
	TopicDeployLog             = "deploy.log"
	TopicDeployStatusChanged   = "deploy.status_changed"
	// TopicInstanceUpgradeRequested is the Service -> Handler dispatch edge (instance-upgrade spec): mirrors
	// deploy.requested's own bus-topic handoff instead of a direct method call, so Run's existing Subscribe
	// loop is the one place inbound work reaches a connection, matching deploy's plumbing exactly.
	TopicInstanceUpgradeRequested = "instance.upgrade_requested"
	TopicInstanceUpgradeChanged   = "instance.upgrade_changed"
)

// Topics returns every topic the runner domain publishes.
func Topics() []string {
	return []string{
		TopicRunnerConnected,
		TopicRunnerDisconnected,
		TopicRunnerHeartbeat,
		TopicDeployBuildStarted,
		TopicDeployBuildProgress,
		TopicDeployBuildCompleted,
		TopicDeployDeployProgress,
		TopicDeployLog,
		TopicDeployStatusChanged,
		TopicInstanceUpgradeRequested,
		TopicInstanceUpgradeChanged,
	}
}

// Publisher is the narrow bus seam Service needs to announce upgrade state (ADR 0019); Handler's own Bus
// interface adds Subscribe, which use-cases never need.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

// InstanceUpgradeRequestedEvent asks the handler to dispatch assign_upgrade to the connected instance runner.
type InstanceUpgradeRequestedEvent struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// RequestKind selects the assign frame a deploy.requested event becomes.
type RequestKind string

const (
	RequestBuild   RequestKind = "build"
	RequestDeploy  RequestKind = "deploy"
	RequestUpgrade RequestKind = "upgrade"
)

// RunnerConnectedEvent is published when a runner authenticates and connects.
type RunnerConnectedEvent struct {
	RunnerID string `json:"runner_id"`
	Name     string `json:"name,omitempty"`
}

// RunnerDisconnectedEvent is published when a runner disconnects or is dropped for a heartbeat timeout.
type RunnerDisconnectedEvent struct {
	RunnerID string `json:"runner_id"`
	Reason   string `json:"reason,omitempty"`
}

// RunnerHeartbeatEvent carries a runner's liveness pulse.
type RunnerHeartbeatEvent struct {
	RunnerID string `json:"runner_id"`
	TS       int64  `json:"ts"`
}

// DeployRequestedEvent is the deploy.requested payload; fields match what assign frames need.
type DeployRequestedEvent struct {
	ID       string            `json:"id"`
	Kind     RequestKind       `json:"kind"`
	Repo     string            `json:"repo,omitempty"`
	Ref      string            `json:"ref,omitempty"`
	Steps    []string          `json:"steps,omitempty"`
	Service  string            `json:"service,omitempty"`
	Target   string            `json:"target,omitempty"`
	Image    string            `json:"image,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Strategy string            `json:"strategy,omitempty"`
	// Network is the container's docker network (run strategy).
	Network string `json:"network,omitempty"`
	// Ports are host->container mappings for the run strategy (e.g. "80:80").
	Ports []string `json:"ports,omitempty"`
	// Mounts are host->container bind mounts for the run strategy.
	Mounts []string `json:"mounts,omitempty"`
	// Command replaces the image's default command for the run strategy.
	Command []string `json:"command,omitempty"`
	// GitToken is set on the runner from the assign frame and never serialized back out.
	GitToken string `json:"-"`
	// Dockerfile is a repo-relative path, carried so a build job can build from the checkout.
	Dockerfile string `json:"dockerfile,omitempty"`
	// ComposePath is the compose file's path relative to the checkout; also the compose project's `-f` flag.
	ComposePath string `json:"compose_path,omitempty"`
	// StackSlug names the compose project / run container and the checkout under StackRoot.
	StackSlug string `json:"stack_slug,omitempty"`
	// StackRoot is the machine-wide checkout root the persistent checkout lives under.
	StackRoot string `json:"stack_root,omitempty"`
	// GatewayContainer/JoinNetworks carry this deploy's last step: join the gateway onto JoinNetworks;
	// empty GatewayContainer skips it.
	GatewayContainer string   `json:"gateway_container,omitempty"`
	JoinNetworks     []string `json:"join_networks,omitempty"`
}

// DeployCancelRequestedEvent's terminal state reports via deploy.status_changed, not this event.
type DeployCancelRequestedEvent struct {
	ID string `json:"id"`
}

// RunningJob builds the live job view for a runner executing this request.
func (r DeployRequestedEvent) RunningJob() *RunningJob {
	return &RunningJob{ID: r.ID, Kind: r.Kind, Service: r.Service}
}

// Queued builds the queue view entry for a request waiting for a runner.
func (r DeployRequestedEvent) Queued() QueuedJob {
	return QueuedJob{ID: r.ID, Kind: r.Kind, Service: r.Service, Target: r.Target}
}

// fits reports whether r may run on c: a job fits any idle connected runner whose machine name equals the
// stack's machine (Target); an empty machine still fits any runner.
func (r DeployRequestedEvent) fits(c *runnerConn) bool {
	return r.Target == "" || r.Target == c.machine
}

// toFrame converts the request into an assign frame; env is real values from EnvLookup, never the redacted r.Env.
func (r DeployRequestedEvent) toFrame(env map[string]string) (Frame, error) {
	switch r.Kind {
	case RequestBuild:
		return Frame{
			Type:        FrameAssignBuild,
			ID:          r.ID,
			Repo:        r.Repo,
			Ref:         r.Ref,
			Steps:       r.Steps,
			Service:     r.Service,
			Env:         env,
			Strategy:    r.Strategy,
			Network:     r.Network,
			Ports:       r.Ports,
			Mounts:      r.Mounts,
			Command:     r.Command,
			Dockerfile:  r.Dockerfile,
			ComposePath: r.ComposePath,
			StackSlug:   r.StackSlug,
			StackRoot:   r.StackRoot,
		}, nil
	case RequestDeploy:
		return Frame{
			Type:             FrameAssignDeploy,
			ID:               r.ID,
			Service:          r.Service,
			Image:            r.Image,
			Env:              env,
			Strategy:         r.Strategy,
			ComposePath:      r.ComposePath,
			Network:          r.Network,
			Ports:            r.Ports,
			Mounts:           r.Mounts,
			Command:          r.Command,
			StackSlug:        r.StackSlug,
			StackRoot:        r.StackRoot,
			GatewayContainer: r.GatewayContainer,
			JoinNetworks:     r.JoinNetworks,
		}, nil
	default:
		return Frame{}, fmt.Errorf("%w: unknown deploy.requested kind %q", apperrs.ErrInvalid, r.Kind)
	}
}

// BuildStartedEvent marks the first build step (the build lifecycle begins).
type BuildStartedEvent struct {
	ID    string `json:"id"`
	Total int    `json:"total"`
	Log   string `json:"log,omitempty"`
}

// BuildProgressEvent carries per-step build progress.
type BuildProgressEvent struct {
	ID    string `json:"id"`
	Step  int    `json:"step"`
	Total int    `json:"total"`
	Log   string `json:"log,omitempty"`
}

// BuildCompletedEvent is the terminal build state.
type BuildCompletedEvent struct {
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Artifacts []string `json:"artifacts,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// DeployProgressEvent carries the pull/start/health phases of a deploy.
type DeployProgressEvent struct {
	ID    string `json:"id"`
	Phase string `json:"phase"`
	Log   string `json:"log,omitempty"`
}

// DeployLogEvent is one deploy_log batch: newline-joined output lines from one phase, TS in unix milliseconds.
type DeployLogEvent struct {
	ID    string `json:"id"`
	Phase string `json:"phase"`
	Log   string `json:"log"`
	TS    int64  `json:"ts"`
}

// DeployStatusChangedEvent is the terminal deploy state; the runner is the only entity that observes container health.
type DeployStatusChangedEvent struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Address string `json:"address,omitempty"`
	// Services is the observation report (spec §4 step 3), one entry per container the stack started.
	Services []ObservedService `json:"services,omitempty"`
}
