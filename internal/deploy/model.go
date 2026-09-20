package deploy

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusHealthy Status = "healthy"
	StatusFailed  Status = "failed"
)

type Strategy string

const (
	StrategyCompose Strategy = "compose"
	StrategyRun     Strategy = "run"
)

// Kind distinguishes a repo-driven build from a plain image deploy.
type Kind string

const (
	KindDeploy Kind = "deploy"
	KindBuild  Kind = "build"
)

// defaultComposePath is applied to a compose-strategy stack with no explicit compose file.
const defaultComposePath = "docker-compose.yml"

// defaultStackRoot is the checkout root deploy.requested carries; the runner handler swaps in the machine's own at dispatch.
const defaultStackRoot = "/data/nexul"

type Deploy struct {
	ID   string `json:"id"`
	Kind Kind   `json:"kind"`
	// StackID is the deploying stack's id.
	StackID string `json:"stack_id"`
	// ServiceID is a deprecated alias for StackID, kept on the wire for one release so existing consumers keep working.
	ServiceID string   `json:"service_id,omitempty"`
	Service   string   `json:"service"`
	Target    string   `json:"target"`
	Image     string   `json:"image"`
	Status    Status   `json:"status"`
	Strategy  Strategy `json:"strategy"`
	Log       string   `json:"log"`
	// Address is the container's address on its docker network, reported by the runner at start.
	Address string `json:"address,omitempty"`
	// TriggeredBy/RuleID/RuleName/TicketID/PRNumber record who/what triggered the deploy.
	TriggeredBy string    `json:"triggered_by,omitempty"`
	RuleID      string    `json:"rule_id,omitempty"`
	RuleName    string    `json:"rule_name,omitempty"`
	TicketID    string    `json:"ticket_id,omitempty"`
	PRNumber    int       `json:"pr_number,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DeployRequest is the input to Service.Deploy: the stack plus a pre-built Image or a Ref to build from.
type DeployRequest struct {
	StackID string `json:"stack_id"`
	// ServiceID is a deprecated alias for StackID, accepted so callers on the old field name keep working.
	ServiceID string `json:"service_id,omitempty"`
	Image     string `json:"image,omitempty"`
	Ref       string `json:"ref,omitempty"`
	// TriggeredBy/RuleID/RuleName/TicketID/PRNumber record who/what started this deploy.
	TriggeredBy string `json:"triggered_by,omitempty"`
	RuleID      string `json:"rule_id,omitempty"`
	RuleName    string `json:"rule_name,omitempty"`
	TicketID    string `json:"ticket_id,omitempty"`
	PRNumber    int    `json:"pr_number,omitempty"`
}

// normalize folds the deprecated ServiceID alias into StackID; called once, at the top of Service.Deploy.
func (r *DeployRequest) normalize() {
	if r.StackID == "" {
		r.StackID = r.ServiceID
	}
}

// Validate rejects a request missing the stack or both image and ref.
func (r DeployRequest) Validate() error {
	if strings.TrimSpace(r.StackID) == "" {
		return fmt.Errorf("%w: stack is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(r.Image) == "" && strings.TrimSpace(r.Ref) == "" {
		return fmt.Errorf("%w: image or ref is required", apperrs.ErrInvalid)
	}
	return nil
}

// dnsLabelMaxLen fits both a DNS label (RFC 1035) and a container name with the base stack name.
const dnsLabelMaxLen = 63

// nameToken matches a lowercase Docker/DNS-safe token, the shape a rule's explicit NameSuffix must satisfy.
var nameToken = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Slug converts arbitrary text into a deterministic, DNS/Docker-safe token; maxLen caps it for a container/DNS name.
// Used both for a branch deploy's clone suffix and for a stack's slug, derived once from its name at creation.
func Slug(text string, maxLen int) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(text) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if maxLen > 0 && len(s) > maxLen {
		s = strings.Trim(s[:maxLen], "-")
	}
	return s
}

// BranchDeployRule maps a branch pattern to a network, hostname template, and clone suffix (DerivesClone/CloneSuffix).
type BranchDeployRule struct {
	Pattern          string `json:"pattern"`
	DockerNetwork    string `json:"docker_network"`
	HostnameTemplate string `json:"hostname_template,omitempty"`
	NameSuffix       string `json:"name_suffix,omitempty"`
	// Port is the container port HostnameTemplate exposes; required whenever HostnameTemplate is set.
	Port int `json:"port,omitempty"`
}

// Validate rejects a rule that cannot be evaluated or saved.
func (r BranchDeployRule) Validate() error {
	if strings.TrimSpace(r.Pattern) == "" {
		return fmt.Errorf("%w: branch deploy rule pattern is required", apperrs.ErrInvalid)
	}
	if strings.Count(r.Pattern, "*") > 1 {
		return fmt.Errorf("%w: branch deploy rule pattern %q must have at most one wildcard", apperrs.ErrInvalid, r.Pattern)
	}
	if idx := strings.IndexByte(r.Pattern, '*'); idx >= 0 && idx != len(r.Pattern)-1 {
		return fmt.Errorf("%w: branch deploy rule pattern %q must use a single trailing wildcard", apperrs.ErrInvalid, r.Pattern)
	}
	if strings.TrimSpace(r.DockerNetwork) == "" {
		return fmt.Errorf("%w: branch deploy rule %q requires a docker network", apperrs.ErrInvalid, r.Pattern)
	}
	if r.NameSuffix != "" {
		if r.IsWildcard() {
			return fmt.Errorf("%w: wildcard branch deploy rule %q always uses the branch slug; leave name suffix empty", apperrs.ErrInvalid, r.Pattern)
		}
		if !nameToken.MatchString(r.NameSuffix) {
			return fmt.Errorf("%w: branch deploy rule name suffix %q must be lowercase alphanumeric with hyphens", apperrs.ErrInvalid, r.NameSuffix)
		}
	}
	if r.HostnameTemplate != "" && r.Port <= 0 {
		return fmt.Errorf("%w: branch deploy rule %q's hostname template requires a port", apperrs.ErrInvalid, r.Pattern)
	}
	return nil
}

// IsWildcard reports whether the pattern ends in a single trailing wildcard.
func (r BranchDeployRule) IsWildcard() bool { return strings.HasSuffix(r.Pattern, "*") }

// Matches reports whether branch satisfies the rule's pattern.
func (r BranchDeployRule) Matches(branch string) bool {
	if !r.IsWildcard() {
		return r.Pattern == branch
	}
	return strings.HasPrefix(branch, strings.TrimSuffix(r.Pattern, "*"))
}

// DerivesClone reports whether the rule spawns a derived stack record vs. redeploying the base stack in place.
func (r BranchDeployRule) DerivesClone() bool {
	return r.IsWildcard() || r.NameSuffix != ""
}

// CloneSuffix returns the "<base>-<suffix>" a derived clone is named with; only meaningful if DerivesClone.
func (r BranchDeployRule) CloneSuffix(branch string) string {
	if r.IsWildcard() {
		return Slug(branch, dnsLabelMaxLen)
	}
	return r.NameSuffix
}

// BuildSource is the repo-backed build config on a stack; RepoOwner/RepoName alone redeploys the last image.
type BuildSource struct {
	RepoOwner   string `json:"repo_owner,omitempty"`
	RepoName    string `json:"repo_name,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Dockerfile  string `json:"dockerfile,omitempty"`
	ComposePath string `json:"compose_path,omitempty"`
}

// Buildable reports a Dockerfile/compose file is set, vs. an "image-only" source matching branch-deploy pushes only.
func (bs *BuildSource) Buildable() bool {
	return bs != nil && (bs.Dockerfile != "" || bs.ComposePath != "")
}

// Stack is a deploy stack definition: a workload owned by one project, on one machine, with a strategy.
// A stack owns the observed Container rows its deploys started.
type Stack struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	// Slug is derived once from Name at creation (lowercase, [a-z0-9-], runs collapsed, trimmed); unique per Machine.
	// It is the compose project name and the default network is "<slug>_default"; a run stack's container is "<slug>".
	Slug string `json:"slug"`
	// Machine names the machine whose runner pool dispatch routes this stack to.
	Machine  string   `json:"machine"`
	Strategy Strategy `json:"strategy"`
	// ComposePath is the compose file's path relative to the repo (compose strategy only); defaults to docker-compose.yml.
	ComposePath string            `json:"compose_path,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	// DockerNetwork is the network the container joins (run strategy only; compose reads its own).
	DockerNetwork string `json:"docker_network,omitempty"`
	// Ports are host→container mappings for the run strategy, used by entry-path services like a reverse proxy.
	Ports []string `json:"ports,omitempty"`
	// Mounts are host→container bind mounts for the run strategy, used by entry-path services needing host access.
	Mounts []string `json:"mounts,omitempty"`
	// Command overrides the image's default command for the run strategy (cloudflared needs `tunnel run`).
	Command []string `json:"command,omitempty"`
	// BuildSource, when set, makes this a repo-driven stack.
	BuildSource *BuildSource `json:"build_source,omitempty"`
	// BranchDeployRules only apply on a base stack; a clone never carries its own rules.
	BranchDeployRules []BranchDeployRule `json:"branch_deploy_rules,omitempty"`
	// DerivedFrom is the base stack's id for a branch deployment's clone; empty for a base stack.
	DerivedFrom string `json:"derived_from,omitempty"`
	// Branch is the source branch this clone tracks; empty unless DerivedFrom is set.
	Branch string `json:"branch,omitempty"`
	// Managed is false until a build source is attached (UpdateStack flips it) — an unmanaged stack has no repo to build or deploy from.
	Managed   bool      `json:"managed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DefaultNetwork returns the docker network the stack's own containers join: DockerNetwork for a run
// stack, or compose's own project network ("<slug>_default") for a compose stack (spec §2).
func (s *Stack) DefaultNetwork() string {
	if s.Strategy == StrategyRun {
		return s.DockerNetwork
	}
	return s.Slug + "_default"
}

// Validate enforces the stack contract.
func (s *Stack) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("%w: stack name is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(s.ProjectID) == "" {
		return fmt.Errorf("%w: project is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(s.Machine) == "" {
		return fmt.Errorf("%w: machine is required", apperrs.ErrInvalid)
	}
	if err := s.validateStrategy(); err != nil {
		return err
	}
	if bs := s.BuildSource; bs != nil && (strings.TrimSpace(bs.RepoOwner) == "" || strings.TrimSpace(bs.RepoName) == "") {
		return fmt.Errorf("%w: build source requires a repository", apperrs.ErrInvalid)
	}
	return s.validateBranchDeployRules()
}

// validateStrategy checks the fields the chosen deploy Strategy requires.
func (s *Stack) validateStrategy() error {
	switch s.Strategy {
	case StrategyCompose:
		// ComposePath defaults to docker-compose.yml (applyStackDefaults), so nothing further is required here.
	case StrategyRun:
		if strings.TrimSpace(s.DockerNetwork) == "" {
			return fmt.Errorf("%w: docker network is required for the run strategy", apperrs.ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported deploy strategy %q", apperrs.ErrInvalid, s.Strategy)
	}
	return nil
}

// validateBranchDeployRules rejects a wildcard rule on a stack with no build source to deploy from.
func (s *Stack) validateBranchDeployRules() error {
	for _, r := range s.BranchDeployRules {
		if err := r.Validate(); err != nil {
			return err
		}
		if r.IsWildcard() && s.BuildSource == nil {
			return fmt.Errorf("%w: wildcard branch deploy rule %q requires the stack to have a build source", apperrs.ErrInvalid, r.Pattern)
		}
	}
	return nil
}

// ServiceStatus is a container's observed lifecycle state.
type ServiceStatus string

const (
	ServiceStatusPending ServiceStatus = "pending"
	ServiceStatusRunning ServiceStatus = "running"
	ServiceStatusHealthy ServiceStatus = "healthy"
	ServiceStatusExited  ServiceStatus = "exited"
	ServiceStatusStopped ServiceStatus = "stopped"
)

// Declared is the compose/run parse's view of a service, before anything has been observed running.
type Declared struct {
	Image   string   `json:"image,omitempty"`
	Build   string   `json:"build,omitempty"`
	Ports   []string `json:"ports,omitempty"`
	EnvKeys []string `json:"env_keys,omitempty"`
}

// Network is one docker network a container joined, with its address on it (reported by the runner).
type Network struct {
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
}

// Container is one observed container belonging to a Stack: a compose service, or a run stack's single
// container. This is the domain's "service" entity in spec vocabulary and on the wire (JSON, HTTP routes);
// it is named Container in Go to avoid clashing with the use-case type Service in this same package.
// Created pending at stack creation from the parse (Declared); updated from every observation report.
type Container struct {
	ID      string `json:"id"`
	StackID string `json:"stack_id"`
	// Name is the compose service name; a run stack's single service is named after the stack's slug.
	Name          string        `json:"name"`
	Declared      Declared      `json:"declared"`
	ContainerName string        `json:"container_name,omitempty"`
	Image         string        `json:"image,omitempty"`
	Status        ServiceStatus `json:"status"`
	Networks      []Network     `json:"networks,omitempty"`
	Ports         []string      `json:"ports,omitempty"`
	ObservedAt    time.Time     `json:"observed_at,omitempty"`
}
