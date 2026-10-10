package runner

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Topics the runner domain publishes onto the bus; deploy.requested is the inverse, consumed not published.
const (
	TopicRunnerConnected    = "runner.connected"
	TopicRunnerDisconnected = "runner.disconnected"
	TopicRunnerHeartbeat    = "runner.heartbeat"
	// TopicPersonalChanged is ephemeral: a personal runner enrolling, connecting, disconnecting or removed, in place of
	// runner.connected and runner.disconnected, which every runners:read holder hears (ADR 0146).
	TopicPersonalChanged = "runner.personal_changed"
	// TopicFactsReported is ephemeral and never bridged to a socket: a personal runner's report, for pairing.
	TopicFactsReported         = "runner.facts_reported"
	TopicDeployRequested       = "deploy.requested"
	TopicDeployCancelRequested = "deploy.cancel_requested"
	TopicDeployBuildStarted    = "deploy.build_started"
	TopicDeployBuildProgress   = "deploy.build_progress"
	TopicDeployBuildCompleted  = "deploy.build_completed"
	TopicDeployDeployProgress  = "deploy.deploy_progress"
	TopicDeployLog             = "deploy.log"
	TopicDeployStatusChanged   = "deploy.status_changed"
	// TopicInstanceUpgradeRequested is the Service -> Handler dispatch edge: mirrors
	// deploy.requested's own bus-topic handoff instead of a direct method call, so Run's existing Subscribe
	// loop is the one place inbound work reaches a connection, matching deploy's plumbing exactly.
	TopicInstanceUpgradeRequested = "instance.upgrade_requested"
	TopicInstanceUpgradeChanged   = "instance.upgrade_changed"
)

// Topics returns every topic the runner domain publishes.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicRunnerConnected, Payload: RunnerConnectedEvent{}},
		{Name: TopicRunnerDisconnected, Payload: RunnerDisconnectedEvent{}},
		{Name: TopicRunnerHeartbeat, Payload: RunnerHeartbeatEvent{}},
		{Name: TopicPersonalChanged, Payload: PersonalChangedEvent{}},
		{Name: TopicFactsReported, Payload: FactsReportedEvent{}},
		{Name: TopicDeployBuildStarted, Payload: BuildStartedEvent{}},
		{Name: TopicDeployBuildProgress, Payload: BuildProgressEvent{}},
		{Name: TopicDeployBuildCompleted, Payload: BuildCompletedEvent{}},
		{Name: TopicDeployDeployProgress, Payload: DeployProgressEvent{}},
		{Name: TopicDeployLog, Payload: DeployLogEvent{}},
		{Name: TopicDeployStatusChanged, Payload: DeployStatusChangedEvent{}},
		{Name: TopicInstanceUpgradeRequested, Payload: InstanceUpgradeRequestedEvent{}},
		{Name: TopicInstanceUpgradeChanged, Payload: Upgrade{}},
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

// Personal runner states a PersonalChangedEvent reports.
const (
	PersonalEnrolled     = "enrolled"
	PersonalConnected    = "connected"
	PersonalDisconnected = "disconnected"
	PersonalRemoved      = "removed"
)

// PersonalChangedEvent is one of a person's computer's runner changing state; it reaches only its owner.
type PersonalChangedEvent struct {
	RunnerID   string `json:"runner_id"`
	ComputerID string `json:"computer_id" jsonschema:"The computer the runner reaches."`
	UserID     string `json:"user_id" jsonschema:"The person who installed the runner and owns the computer."`
	State      string `json:"state" enum:"enrolled,connected,disconnected,removed"`
	// Hostname is the computer's hostname as its installer reported it, sent only when the runner enrolls.
	Hostname    string `json:"hostname,omitempty" jsonschema:"The computer's hostname as its installer reported it; sent only with state enrolled."`
	MembersOnly bool   `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

// FactsReportedEvent is a personal runner's report about its computer, as the runner sent it.
type FactsReportedEvent struct {
	RunnerID    string `json:"runner_id"`
	ComputerID  string `json:"computer_id" jsonschema:"The computer the runner reaches."`
	UserID      string `json:"user_id" jsonschema:"The person who owns the computer."`
	Facts       Facts  `json:"facts"`
	MembersOnly bool   `json:"members_only" jsonschema:"Always true: a person's computer is never delivered to integrations or automations."`
}

func personalChanged(r *Runner, state, hostname string) PersonalChangedEvent {
	return PersonalChangedEvent{RunnerID: r.ID, ComputerID: r.ComputerID, UserID: r.OwnerUserID, State: state, Hostname: hostname, MembersOnly: true}
}

// HandleAccountClosed is the account.disabled and account.removed consumer: the account's personal runners are
// revoked, and reactivating the account later brings none back.
func (s *Service) HandleAccountClosed(ctx context.Context, ev eventbus.Event) error {
	var p struct {
		AccountID string `json:"account_id"`
	}
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	if p.AccountID == "" {
		return apperrs.Fatal(fmt.Errorf("%w: %s names no account", apperrs.ErrInvalid, ev.Topic))
	}
	return s.RevokePersonalRunners(ctx, p.AccountID)
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

// fits reports whether r may run on c: a job fits any idle connected deploy runner whose machine name equals the
// stack's machine (Target); an empty machine still fits any deploy runner, and never a personal one.
func (r DeployRequestedEvent) fits(c *runnerConn) bool {
	return !c.personal && (r.Target == "" || r.Target == c.machine)
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
	Phase string `json:"phase" enum:"checkout,build,deploy"`
	Log   string `json:"log"`
	TS    int64  `json:"ts"`
}

// DeployStatusChangedEvent is the terminal deploy state; the runner is the only entity that observes container health.
type DeployStatusChangedEvent struct {
	ID      string `json:"id"`
	Status  string `json:"status" enum:"pending,running,healthy,failed"`
	Error   string `json:"error,omitempty"`
	Address string `json:"address,omitempty"`
	// Services is the observation report (spec §4 step 3), one entry per container the stack started.
	Services []ObservedService `json:"services,omitempty"`
}
