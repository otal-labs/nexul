package deploy

// Topics the deploy domain publishes onto the bus; shapes mirror the runner's contracts (ADR 0017).
// Topic strings keep their original "service.*" names so existing bus consumers keep matching.
const (
	TopicDeployRequested       = "deploy.requested"
	TopicDeployCancelRequested = "deploy.cancel_requested"
	TopicDeployStatusChanged   = "deploy.status_changed"
	TopicStackCreated          = "service.created"
	TopicStackUpdated          = "service.updated"
	TopicStackDeleted          = "service.deleted"
)

// Topics returns every topic the deploy domain publishes.
func Topics() []string {
	return []string{
		TopicDeployRequested,
		TopicDeployCancelRequested,
		TopicDeployStatusChanged,
		TopicStackCreated,
		TopicStackUpdated,
		TopicStackDeleted,
	}
}

// DeployCancelRequestedEvent asks the runner to stop a queued or running job.
type DeployCancelRequestedEvent struct {
	ID string `json:"id"`
}

// DeployRequestedEvent is resolved from the stack; the runner ignores fields it doesn't need (ADR 0017).
// Wire field names are unchanged from the ServiceDef era: the runner protocol (internal/runner) decodes this
// independently and is out of this ticket's scope, so the JSON tags stay put even where the Go field renamed.
type DeployRequestedEvent struct {
	ID       string            `json:"id"`
	Kind     string            `json:"kind"`
	Service  string            `json:"service,omitempty"`
	Target   string            `json:"target,omitempty"`
	Image    string            `json:"image,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Strategy string            `json:"strategy,omitempty"`
	Repo     string            `json:"repo,omitempty"`
	Ref      string            `json:"ref,omitempty"`
	// ComposePath is the stack's compose file, relative to the checkout.
	ComposePath string   `json:"compose_path,omitempty"`
	Network     string   `json:"network,omitempty"`
	Ports       []string `json:"ports,omitempty"`
	Mounts      []string `json:"mounts,omitempty"`
	Command     []string `json:"command,omitempty"`
	// Dockerfile is the repo-relative file a run-strategy build builds from the cloned checkout.
	Dockerfile string `json:"dockerfile,omitempty"`
	// StackSlug names the compose project / run container and the checkout under StackRoot.
	StackSlug string `json:"stack_slug,omitempty"`
	// StackRoot is the machine-wide checkout root; the runner handler replaces it with the machine's own setting at dispatch.
	StackRoot string `json:"stack_root,omitempty"`
	// GatewayContainer/JoinNetworks make this deploy's last step "docker network connect" the gateway serving
	// this stack onto JoinNetworks, so a compose stack that recreates its network on every deploy stays
	// reachable; empty GatewayContainer skips the step entirely.
	GatewayContainer string   `json:"gateway_container,omitempty"`
	JoinNetworks     []string `json:"join_networks,omitempty"`
}

// DeployStatusChangedEvent is the deploy.status_changed payload: a deploy's terminal state, reported by the runner.
type DeployStatusChangedEvent struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Address string `json:"address,omitempty"`
	// Services is the observation report: one entry per container
	// the stack started, reconciled into the services table by HandleStatusChanged.
	Services []ObservedService `json:"services,omitempty"`
}

// ObservedService is one container in a deploy_result's observation report; mirrors runner.Service
// independently (ADR 0017: deploy never imports runner).
type ObservedService struct {
	Name          string    `json:"name"`
	ContainerName string    `json:"container_name"`
	Image         string    `json:"image,omitempty"`
	Status        string    `json:"status"`
	Networks      []Network `json:"networks,omitempty"`
	Ports         []string  `json:"ports,omitempty"`
}

// StackEvent is the shared stack.created/stack.updated payload (still published on the "service.*" topics).
// Services is only populated on creation, so topology can anchor one node per pending container without a
// second round trip.
type StackEvent struct {
	Stack    Stack       `json:"stack"`
	Services []Container `json:"services,omitempty"`
}

// StackDeletedEvent is the stack.deleted payload; consumers drop their own state keyed on the stack id.
// ServiceIDs carries the stack's container ids at the moment of deletion, since the rows themselves are
// already gone by the time a subscriber (e.g. topology, dropping their nodes) sees this event.
// Renamed from ServiceDeletedEvent.
type StackDeletedEvent struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ServiceIDs []string `json:"service_ids,omitempty"`
}

// PushTrigger mirrors gitprovider's PushEvent wire shape so deploy never imports gitprovider (ADR 0017).
type PushTrigger struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	SHA    string `json:"sha"`
}

// BranchDeletedTrigger mirrors gitprovider's BranchDeletedEvent wire shape (ADR 0017).
type BranchDeletedTrigger struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// redactEnv keeps env keys but blanks values, since deploy.requested is a bus event, no safe place for a secret.
func redactEnv(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}
	redacted := make(map[string]string, len(env))
	for k := range env {
		redacted[k] = ""
	}
	return redacted
}

// toEvent builds the deploy.requested payload; a Ref deploy (kind build) also carries the repo/ref.
// gatewayContainer/joinNetworks are resolved by the caller (a dns seam), since deploy never imports dns (ADR 0017).
func (r DeployRequest) toEvent(id string, stack *Stack, gatewayContainer string, joinNetworks []string) DeployRequestedEvent {
	kind := "deploy"
	if r.Ref != "" {
		kind = "build"
	}
	ev := DeployRequestedEvent{
		ID:               id,
		Kind:             kind,
		Service:          stack.Name,
		Target:           stack.Machine,
		Image:            r.Image,
		Env:              redactEnv(stack.Env),
		Strategy:         string(stack.Strategy),
		Ref:              r.Ref,
		ComposePath:      stack.ComposePath,
		Network:          stack.DockerNetwork,
		Ports:            stack.Ports,
		Mounts:           stack.Mounts,
		Command:          stack.Command,
		StackSlug:        stack.Slug,
		StackRoot:        defaultStackRoot,
		GatewayContainer: gatewayContainer,
		JoinNetworks:     joinNetworks,
	}
	if bs := stack.BuildSource; bs != nil {
		ev.Repo = bs.RepoOwner + "/" + bs.RepoName
		ev.Dockerfile = bs.Dockerfile
	}
	return ev
}
