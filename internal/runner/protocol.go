package runner

import (
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
	// FrameAssignUpgrade (server -> runner) asks the bundled instance runner to bring the compose stack up on a
	// newer release by starting a one-shot helper container (instance-upgrade spec); it holds the runner's single
	// job slot like assign_deploy. FrameUpgradeProgress/FrameUpgradeResult (runner -> server) report it; a
	// `started` result is the last frame, because the helper recreates the runner's own container next.
	FrameAssignUpgrade   FrameType = "assign_upgrade"
	FrameUpgradeProgress FrameType = "upgrade_progress"
	FrameUpgradeResult   FrameType = "upgrade_result"
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

	UpgradeStatusStarted = "started"
	UpgradeStatusFailed  = "failed"
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
	RunnerID  string   `json:"runner_id,omitempty"`
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
	// Ports are published bindings formatted "host:container/proto".
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
	FrameHeartbeat:          (*Frame).validateHeartbeat,
	FrameAssignBuild:        (*Frame).validateAssignBuild,
	FrameAssignDeploy:       (*Frame).validateAssignDeploy,
	FrameCancel:             (*Frame).validateCancel,
	FrameBuildProgress:      (*Frame).validateBuildProgress,
	FrameBuildResult:        (*Frame).validateBuildResult,
	FrameDeployProgress:     (*Frame).validateDeployProgress,
	FrameDeployResult:       (*Frame).validateDeployResult,
	FrameDiscover:           (*Frame).validateDiscover,
	FrameDiscoverResult:     (*Frame).validateDiscoverResult,
	FrameJoinNetworks:       (*Frame).validateJoinNetworks,
	FrameJoinNetworksResult: (*Frame).validateJoinNetworksResult,
	FrameUpdate:             (*Frame).validateUpdate,
	FrameAssignUpgrade:      (*Frame).validateAssignUpgrade,
	FrameUpgradeProgress:    (*Frame).validateUpgradeProgress,
	FrameUpgradeResult:      (*Frame).validateUpgradeResult,
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

func (f *Frame) validateHeartbeat() error {
	if f.RunnerID == "" {
		return fmt.Errorf("%w: heartbeat requires runner_id", apperrs.ErrInvalid)
	}
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
