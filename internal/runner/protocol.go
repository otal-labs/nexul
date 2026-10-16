package runner

import (
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// FrameType discriminates runner protocol frames.
type FrameType string

const (
	FrameHeartbeat      FrameType = "heartbeat"
	FrameAssignBuild    FrameType = "assign_build"
	FrameAssignDeploy   FrameType = "assign_deploy"
	FrameCancel         FrameType = "cancel"
	FrameBuildProgress  FrameType = "build_progress"
	FrameBuildResult    FrameType = "build_result"
	FrameDeployProgress FrameType = "deploy_progress"
	FrameDeployResult   FrameType = "deploy_result"
	// FrameDeployLog (runner -> server) streams a batch of newline-joined output lines from one phase of a
	// deploy job; its ts is unix milliseconds, unlike heartbeat's seconds.
	FrameDeployLog FrameType = "deploy_log"
	// FrameDiscover (server -> runner) and FrameDiscoverResult (runner -> server) are the machine-discovery
	// job pair (spec §3, §8): scan the host's containers and networks for the import wizard.
	FrameDiscover       FrameType = "discover"
	FrameDiscoverResult FrameType = "discover_result"
	// FrameJoinNetworks (server -> runner) asks a gateway container to join docker networks immediately,
	// outside the deploy flow: an exposure created for an already-running stack.
	FrameJoinNetworks FrameType = "join_networks"
	// FrameJoinNetworksResult (runner -> server) reports the outcome; the server treats it as best-effort log
	// fodder, not something it blocks on.
	FrameJoinNetworksResult FrameType = "join_networks_result"
	// FrameUpdate (server -> runner) carries a newer release for the runner to download and swap itself with
	// (upgrade-path spec, "Runner protocol"); sent best-effort right after registerRunner on connect.
	FrameUpdate FrameType = "update"
	// FrameAssignUpgrade (server -> runner) asks the instance runner to start `nexul upgrade` on its host (ADR 0069);
	// FrameUpgradeProgress/FrameUpgradeResult report it, and `started` is the last frame since the server restarts next.
	FrameAssignUpgrade   FrameType = "assign_upgrade"
	FrameUpgradeProgress FrameType = "upgrade_progress"
	FrameUpgradeResult   FrameType = "upgrade_result"
	// FrameUninstall (server -> runner) says the runner was removed: it uninstalls its own service and exits.
	FrameUninstall FrameType = "uninstall"
	// Container logs (ADR 0091): logs_request and logs_cancel go to the runner, logs_chunk and logs_end come back.
	FrameLogsRequest FrameType = "logs_request"
	FrameLogsChunk   FrameType = "logs_chunk"
	FrameLogsEnd     FrameType = "logs_end"
	FrameLogsCancel  FrameType = "logs_cancel"
	// Streams (ADR 0146): harness_dial opens stream id to T3 Code; no frame names a host or port, the runner picks.
	FrameHarnessDial        FrameType = "harness_dial"
	FrameHarnessDialRefused FrameType = "harness_dial_refused"
	// FrameT3PairTokenRequest asks a personal runner to mint a one-time T3 Code pairing token, answered by FrameT3PairToken.
	FrameT3PairTokenRequest FrameType = "t3_pair_token_request"
	FrameT3PairToken        FrameType = "t3_pair_token"
	// FrameContainerEvent (runner -> server) is one die, oom or start of a container on a machine runner's host, and
	// FrameContainerStates a snapshot of that host's containers sent when the event stream (re)starts. The runner
	// does not know which containers are Nexul's; the server matches them to stacks.
	FrameContainerEvent  FrameType = "container_event"
	FrameContainerStates FrameType = "container_states"
	// FrameFacts (runner -> server) is a personal runner's report about its computer, on connect and every 6 hours.
	FrameFacts FrameType = "facts"
	// FrameLog (runner -> server) carries a batch of the runner's own log records for the server to relay to the log
	// store; only a machine's runner sends it, and the server stamps where they came from.
	FrameLog FrameType = "log"
	// FrameRequestCounts (runner -> server) carries the requests each proxy router served since the runner's previous
	// read, every 30 seconds; a frame with no counts still says the read worked, and only a machine's runner sends it.
	FrameRequestCounts FrameType = "request_counts"
)

// Build and deploy statuses carried by result frames.
const (
	BuildStatusSuccess = "success"
	BuildStatusFailed  = "failed"

	DeployStatusHealthy = "healthy"
	DeployStatusFailed  = "failed"

	DeployPhasePulling  = "pulling"
	DeployPhaseStarting = "starting"
	DeployPhaseHealthy  = "healthy"

	// Log phases group a deploy_log batch: git clone/fetch, image or compose build, then start and network join.
	LogPhaseCheckout = "checkout"
	LogPhaseBuild    = "build"
	LogPhaseDeploy   = "deploy"

	UpgradeStatusStarted = "started"
	UpgradeStatusFailed  = "failed"

	ContainerActionDie   = "die"
	ContainerActionOOM   = "oom"
	ContainerActionStart = "start"

	ContainerStateRunning = "running"
	ContainerStateExited  = "exited"

	LogStreamStdout = "stdout"
	LogStreamStderr = "stderr"

	// LogTargetRunner on a logs_request asks for the runner's own log, not a container's.
	LogTargetRunner = "runner"
)

// Frame is the flat JSON wire message of the runner protocol; per-type field requirements are enforced by Validate.
type Frame struct {
	Type     FrameType         `json:"type"`
	ID       string            `json:"id,omitempty"`
	Repo     string            `json:"repo,omitempty"`
	Ref      string            `json:"ref,omitempty"`
	Steps    []string          `json:"steps,omitempty"`
	Service  string            `json:"service,omitempty"`
	Image    string            `json:"image,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Strategy string            `json:"strategy,omitempty"`
	Network  string            `json:"network,omitempty"`
	Ports    []string          `json:"ports,omitempty"`
	Mounts   []string          `json:"mounts,omitempty"`
	Command  []string          `json:"command,omitempty"`
	// GitToken authenticates the clone for this build only; it rides the runner socket, never the event bus or logs.
	GitToken string `json:"git_token,omitempty"`
	// Dockerfile and ComposePath are repo-relative paths into the checkout; ComposePath is also the compose
	// project's `-f` flag for the compose strategy.
	Dockerfile  string `json:"dockerfile,omitempty"`
	ComposePath string `json:"compose_path,omitempty"`
	// StackSlug names the compose project / run container (`-p`/`--name`) and the checkout under StackRoot.
	StackSlug string `json:"stack_slug,omitempty"`
	// StackRoot is the machine-wide checkout root; the checkout lives at "<StackRoot>/stacks/<StackSlug>/repo".
	StackRoot string   `json:"stack_root,omitempty"`
	TS        int64    `json:"ts,omitempty"`
	Step      int      `json:"step,omitempty"`
	Total     int      `json:"total,omitempty"`
	Log       string   `json:"log,omitempty"`
	Status    string   `json:"status,omitempty"`
	Artifacts []string `json:"artifacts,omitempty"`
	Phase     string   `json:"phase,omitempty"`
	Error     string   `json:"error,omitempty"`
	// Address is the primary container's address on its docker network, kept for consumers that haven't moved
	// onto Services yet; derived from the same observation report.
	Address string `json:"address,omitempty"`
	// Services is the deploy_result observation report: one entry per container the stack started.
	Services []ObservedService `json:"services,omitempty"`
	Commit   string            `json:"commit,omitempty"`
	// Containers and Networks carry a discover_result's report for the import wizard.
	Containers []DiscoveredContainer `json:"containers,omitempty"`
	Networks   []NetworkInfo         `json:"networks,omitempty"`
	// GatewayContainer/JoinNetworks: on an assign frame, the deploy's last step joins the gateway onto the stack's
	// networks; on a join_networks frame they are the whole request.
	GatewayContainer string   `json:"gateway_container,omitempty"`
	JoinNetworks     []string `json:"join_networks,omitempty"`
	// Version, URL and Sha256 carry an update frame: the release the runner should fetch and swap itself with.
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
	Sha256  string `json:"sha256,omitempty"`
	// Container, Tail and Follow carry a logs_request; Lines a logs_chunk. Target is empty for a container's logs;
	// Mask asks the runner to hide the container's own env values in them.
	Container string             `json:"container,omitempty"`
	Target    string             `json:"target,omitempty"`
	Mask      bool               `json:"mask,omitempty"`
	Tail      int                `json:"tail,omitempty"`
	Follow    bool               `json:"follow,omitempty"`
	Lines     []ContainerLogLine `json:"lines,omitempty"`
	// Token carries a t3_pair_token's one-time T3 Code pairing token; it is never logged.
	Token string `json:"token,omitempty"`
	// Facts carries a facts frame's report.
	Facts *Facts `json:"facts,omitempty"`
	// Project is a container's compose project, Action its Docker event and ExitCode the exit code a die carried;
	// they ride a container_event beside Container, Service and TS (unix seconds).
	Project  string `json:"project,omitempty"`
	Action   string `json:"action,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	// States carries a container_states snapshot.
	States []ContainerState `json:"states,omitempty"`
	// Records carries a log frame's batch.
	Records []logging.Record `json:"records,omitempty"`
	// Counts carries a request_counts frame's per-router deltas.
	Counts []RouterCount `json:"counts,omitempty"`
}

// ContainerState is one container in a container_states snapshot. State is running, or exited for a container that
// is down; only a running container and one that exited non-zero or was OOM-killed are reported.
type ContainerState struct {
	Container  string `json:"container"`
	Project    string `json:"project,omitempty"`
	Service    string `json:"service,omitempty"`
	State      string `json:"state"`
	ExitCode   int    `json:"exit_code,omitempty"`
	OOM        bool   `json:"oom,omitempty"`
	FinishedAt int64  `json:"finished_at,omitempty"`
}

// ContainerLogLine is one line a container printed: Docker's RFC 3339 timestamp, stdout or stderr, and the text.
type ContainerLogLine struct {
	TS     string `json:"ts"`
	Stream string `json:"stream"`
	Line   string `json:"line"`
}

// ObservedService is one observed container in a deploy_result's report; named
// Observed- (not Service) to avoid clashing with the runner-domain use-case type Service in this same package.
type ObservedService struct {
	// Name is the compose service label, or the slug for a run stack's single container.
	Name          string `json:"name"`
	ContainerName string `json:"container_name"`
	Image         string `json:"image,omitempty"`
	// Status is one of running|healthy|exited, derived from docker inspect's State.Status and State.Health.Status.
	Status   string            `json:"status"`
	Networks []ObservedNetwork `json:"networks,omitempty"`
	// Ports are published bindings as "host:container/proto" and network-only ports as "container/proto".
	Ports []string `json:"ports,omitempty"`
}

// ObservedNetwork is one docker network a reported container joined, with its address on it.
type ObservedNetwork struct {
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
	// GatewayContainer/JoinNetworks carry a network-join step: on assign_deploy, the deploy's last step
	// (guarded on GatewayContainer being set); on join_networks/join_networks_result, a standalone request.
	GatewayContainer string   `json:"gateway_container,omitempty"`
	JoinNetworks     []string `json:"join_networks,omitempty"`
}

// frameValidators maps each frame type to the check for the fields it requires.
var frameValidators = map[FrameType]func(*Frame) error{
	FrameHeartbeat:          (*Frame).validateNothing,
	FrameAssignBuild:        (*Frame).validateAssignBuild,
	FrameAssignDeploy:       (*Frame).validateAssignDeploy,
	FrameCancel:             (*Frame).validateCancel,
	FrameBuildProgress:      (*Frame).validateBuildProgress,
	FrameBuildResult:        (*Frame).validateBuildResult,
	FrameDeployProgress:     (*Frame).validateDeployProgress,
	FrameDeployResult:       (*Frame).validateDeployResult,
	FrameDeployLog:          (*Frame).validateDeployLog,
	FrameDiscover:           (*Frame).validateDiscover,
	FrameDiscoverResult:     (*Frame).validateDiscoverResult,
	FrameJoinNetworks:       (*Frame).validateJoinNetworks,
	FrameJoinNetworksResult: (*Frame).validateJoinNetworksResult,
	FrameUpdate:             (*Frame).validateUpdate,
	FrameAssignUpgrade:      (*Frame).validateAssignUpgrade,
	FrameUpgradeProgress:    (*Frame).validateUpgradeProgress,
	FrameUpgradeResult:      (*Frame).validateUpgradeResult,
	FrameUninstall:          (*Frame).validateNothing,
	FrameLogsRequest:        (*Frame).validateLogsRequest,
	FrameLogsChunk:          (*Frame).validateLogsChunk,
	FrameLogsEnd:            (*Frame).validateID,
	FrameLogsCancel:         (*Frame).validateID,
	FrameHarnessDial:        (*Frame).validateID,
	FrameHarnessDialRefused: (*Frame).validateHarnessDialRefused,
	FrameT3PairTokenRequest: (*Frame).validateID,
	FrameT3PairToken:        (*Frame).validateT3PairToken,
	FrameFacts:              (*Frame).validateFacts,
	FrameContainerEvent:     (*Frame).validateContainerEvent,
	FrameContainerStates:    (*Frame).validateContainerStates,
	FrameLog:                (*Frame).validateLog,
	FrameRequestCounts:      (*Frame).validateRequestCounts,
}

// Validate checks the fields required by the frame's type; unknown types and malformed values return ErrInvalid.
func (f *Frame) Validate() error {
	if f.Type == "" {
		return fmt.Errorf("%w: frame type is required", apperrs.ErrInvalid)
	}
	validate, ok := frameValidators[f.Type]
	if !ok {
		return fmt.Errorf("%w: unknown frame type %q", apperrs.ErrInvalid, f.Type)
	}
	return validate(f)
}

// validateNothing accepts frames whose type is the whole message: the connection already names the runner.
func (f *Frame) validateNothing() error {
	return nil
}

func (f *Frame) validateAssignBuild() error {
	if f.ID == "" || f.Repo == "" || f.Ref == "" {
		return fmt.Errorf("%w: assign_build requires id, repo and ref", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateAssignDeploy() error {
	if f.ID == "" || f.Service == "" || f.Image == "" {
		return fmt.Errorf("%w: assign_deploy requires id, service and image", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateCancel() error {
	if f.ID == "" {
		return fmt.Errorf("%w: cancel requires id", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateBuildProgress() error {
	if f.ID == "" || f.Step < 1 || f.Total < f.Step {
		return fmt.Errorf("%w: build_progress requires id and 1 <= step <= total", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateBuildResult() error {
	if f.ID == "" || !oneOf(f.Status, BuildStatusSuccess, BuildStatusFailed) {
		return fmt.Errorf("%w: build_result requires id and a valid status", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateDeployProgress() error {
	if f.ID == "" || !oneOf(f.Phase, DeployPhasePulling, DeployPhaseStarting, DeployPhaseHealthy) {
		return fmt.Errorf("%w: deploy_progress requires id and a valid phase", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateDeployResult() error {
	if f.ID == "" || !oneOf(f.Status, DeployStatusHealthy, DeployStatusFailed) {
		return fmt.Errorf("%w: deploy_result requires id and a valid status", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateDeployLog() error {
	if f.ID == "" || f.Log == "" || !oneOf(f.Phase, LogPhaseCheckout, LogPhaseBuild, LogPhaseDeploy) {
		return fmt.Errorf("%w: deploy_log requires id, log and a valid phase", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateDiscover() error {
	if f.ID == "" {
		return fmt.Errorf("%w: discover requires id", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateJoinNetworks() error {
	if f.GatewayContainer == "" || len(f.JoinNetworks) == 0 {
		return fmt.Errorf("%w: join_networks requires gateway_container and at least one network", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateDiscoverResult() error {
	if f.ID == "" {
		return fmt.Errorf("%w: discover_result requires id", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateJoinNetworksResult() error {
	if f.GatewayContainer == "" {
		return fmt.Errorf("%w: join_networks_result requires gateway_container", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateUpdate() error {
	if f.Version == "" || f.URL == "" {
		return fmt.Errorf("%w: update requires version and url", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateAssignUpgrade() error {
	if f.ID == "" || f.Version == "" {
		return fmt.Errorf("%w: assign_upgrade requires id and version", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateUpgradeProgress() error {
	if f.ID == "" || f.Log == "" {
		return fmt.Errorf("%w: upgrade_progress requires id and log", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateUpgradeResult() error {
	if f.ID == "" || !oneOf(f.Status, UpgradeStatusStarted, UpgradeStatusFailed) {
		return fmt.Errorf("%w: upgrade_result requires id and a valid status", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateLogsRequest() error {
	if f.ID == "" || f.Tail < 0 {
		return fmt.Errorf("%w: logs_request requires id and a tail of zero or more", apperrs.ErrInvalid)
	}
	if f.Target == LogTargetRunner && (f.Container != "" || f.Mask) {
		return fmt.Errorf("%w: a logs_request for the runner's own log names no container and masks nothing", apperrs.ErrInvalid)
	}
	if f.Target != "" && f.Target != LogTargetRunner {
		return fmt.Errorf("%w: logs_request target %q", apperrs.ErrInvalid, f.Target)
	}
	if f.Target == "" && f.Container == "" {
		return fmt.Errorf("%w: logs_request requires a container", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateLogsChunk() error {
	if f.ID == "" || len(f.Lines) == 0 {
		return fmt.Errorf("%w: logs_chunk requires id and lines", apperrs.ErrInvalid)
	}
	for _, l := range f.Lines {
		if !oneOf(l.Stream, LogStreamStdout, LogStreamStderr) {
			return fmt.Errorf("%w: logs_chunk line stream %q", apperrs.ErrInvalid, l.Stream)
		}
	}
	return nil
}

func (f *Frame) validateHarnessDialRefused() error {
	if f.ID == "" || f.Error == "" {
		return fmt.Errorf("%w: harness_dial_refused requires id and error", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateT3PairToken() error {
	if f.ID == "" || (f.Token == "") == (f.Error == "") {
		return fmt.Errorf("%w: t3_pair_token requires id and either token or error", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateFacts() error {
	if f.Facts == nil || !oneOf(f.Facts.T3.State, T3Answering, T3NotRunning, T3Missing, T3NotLoopback) {
		return fmt.Errorf("%w: facts requires facts with a valid T3 Code state", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateContainerEvent() error {
	if f.Container == "" || !oneOf(f.Action, ContainerActionDie, ContainerActionOOM, ContainerActionStart) {
		return fmt.Errorf("%w: container_event requires container and a die, oom or start action", apperrs.ErrInvalid)
	}
	return nil
}

func (f *Frame) validateContainerStates() error {
	if len(f.States) == 0 {
		return fmt.Errorf("%w: container_states requires states", apperrs.ErrInvalid)
	}
	for _, st := range f.States {
		if st.Container == "" || !oneOf(st.State, ContainerStateRunning, ContainerStateExited) {
			return fmt.Errorf("%w: container_states entry %q needs a container and a running or exited state", apperrs.ErrInvalid, st.Container)
		}
	}
	return nil
}

func (f *Frame) validateLog() error {
	if len(f.Records) == 0 || len(f.Records) > logging.MaxBatch {
		return fmt.Errorf("%w: log requires 1..%d records", apperrs.ErrInvalid, logging.MaxBatch)
	}
	return nil
}

func (f *Frame) validateRequestCounts() error {
	for _, rc := range f.Counts {
		if rc.Router == "" || rc.Requests < 0 || rc.Errors < 0 || rc.Errors > rc.Requests {
			return fmt.Errorf("%w: request_counts entry %q needs a router and 0 <= errors <= requests", apperrs.ErrInvalid, rc.Router)
		}
	}
	return nil
}

func (f *Frame) validateID() error {
	if f.ID == "" {
		return fmt.Errorf("%w: %s requires id", apperrs.ErrInvalid, f.Type)
	}
	return nil
}

// ParseFrame decodes and validates a frame; the single deserialization entry point on both sides.
func ParseFrame(data []byte) (*Frame, error) {
	var f Frame
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decode frame: %w", err)
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return &f, nil
}

// Encode validates and serializes the frame, so protocol bugs surface at the call site, not on the peer.
func (f *Frame) Encode() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("encode frame: %w", err)
	}
	return data, nil
}

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}
