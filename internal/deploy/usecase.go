package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// ProjectStore validates a stack's project and build-source repo against the workspace domain (ADR 0017).
type ProjectStore interface {
	ProjectExists(ctx context.Context, projectID string) (bool, error)
	RepoInProject(ctx context.Context, projectID, owner, name string) (bool, error)
	// LinkRepo attaches a repository to a project; linking one already attached is a no-op.
	LinkRepo(ctx context.Context, projectID, owner, name string) error
}

// GatewayLookup is the dns seam checking a rule's network has a gateway before its hostname template resolves (ADR 0017).
type GatewayLookup interface {
	NetworkHasGateway(ctx context.Context, dockerNetwork string) (bool, error)
}

// ExposureManager is the dns seam for exposing/unexposing a hostname; EnsureExposure must be idempotent, not additive.
type ExposureManager interface {
	EnsureExposure(ctx context.Context, hostname, service string, port int, dockerNetwork string) error
	// RemoveExposures releases every hostname routed to a stack, whether the exposure was keyed by the stack
	// name (branch-deploy) or by one of its container ids (wizard/UI) — a stack can hold several hostnames.
	RemoveExposures(ctx context.Context, stackName string, containerIDs []string) error
}

// GatewayJoin is the dns seam resolving the gateway already serving a stack's own network, so a deploy's
// last step can rejoin it; found is false when no gateway currently serves it.
type GatewayJoin interface {
	GatewayForStackNetwork(ctx context.Context, network string) (gatewayContainer string, found bool, err error)
}

// Service is the deploy use-case layer (ADR 0019): stacks, their observed containers, deploy executions, and the
// status_changed consumer.
type Service struct {
	repo        Repo
	stacks      StackRepo
	services    ServiceRepo
	projects    ProjectStore
	gateways    GatewayLookup
	exposures   ExposureManager
	machines    MachineLookup
	discoverer  MachineDiscoverer
	adopter     GatewayAdopter
	gatewayJoin GatewayJoin
	now         func() time.Time
}

// NewService wires the deploy use-cases; GatewayLookup/ExposureManager are set later, breaking a cycle with dns.
func NewService(repo Repo, stacks StackRepo, services ServiceRepo, projects ProjectStore) *Service {
	return &Service{repo: repo, stacks: stacks, services: services, projects: projects, now: time.Now}
}

// SetGatewayLookup wires the dns gateway-existence seam; unset, the gateway check on a hostname template is skipped.
func (s *Service) SetGatewayLookup(g GatewayLookup) { s.gateways = g }

// SetExposureManager wires the dns exposure seam; unset, rules never expose a hostname or clean one up.
func (s *Service) SetExposureManager(e ExposureManager) { s.exposures = e }

// SetMachineLookup wires the runner-domain machine-name seam Import uses; set after runner is constructed (ADR 0017).
func (s *Service) SetMachineLookup(m MachineLookup) { s.machines = m }

// SetMachineDiscoverer wires the runner-domain discovery seam Import uses to fetch fresh observed facts (ADR 0017).
func (s *Service) SetMachineDiscoverer(d MachineDiscoverer) { s.discoverer = d }

// SetGatewayAdopter wires the dns seam Import uses to adopt a discovered cloudflared tunnel as a gateway (ADR 0017).
func (s *Service) SetGatewayAdopter(a GatewayAdopter) { s.adopter = a }

// SetGatewayJoin wires the dns gateway-rejoin seam; unset, a deploy never carries a gateway_container/
// join_networks last step.
func (s *Service) SetGatewayJoin(g GatewayJoin) { s.gatewayJoin = g }

// Deploy triggers a deploy, rejecting a second active one; the runner reports state via status_changed.
func (s *Service) Deploy(ctx context.Context, req DeployRequest) (*Deploy, error) {
	req.normalize()
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("deploy %s: %w", req.StackID, err)
	}
	stack, err := s.stacks.GetByID(ctx, req.StackID)
	if err != nil {
		return nil, fmt.Errorf("deploy %s: %w", req.StackID, err)
	}
	kind := KindDeploy
	if req.Ref != "" {
		if stack.BuildSource == nil {
			return nil, fmt.Errorf("%w: stack %q has no build source to build from", apperrs.ErrInvalid, stack.Name)
		}
		kind = KindBuild
	}
	active, err := s.repo.HasActive(ctx, stack.ID)
	if err != nil {
		return nil, fmt.Errorf("deploy %s: %w", stack.Name, err)
	}
	if active {
		return nil, fmt.Errorf("%w: stack %q already has an active deploy", apperrs.ErrConflict, stack.Name)
	}
	now := s.now().UTC()
	d := &Deploy{
		ID: ids.New(), StackID: stack.ID, ServiceID: stack.ID, Service: stack.Name, Target: stack.Machine,
		Image: req.Image, Kind: kind, Status: StatusPending, Strategy: stack.Strategy,
		TriggeredBy: req.TriggeredBy, RuleID: req.RuleID, RuleName: req.RuleName,
		TicketID: req.TicketID, PRNumber: req.PRNumber,
		CreatedAt: now, UpdatedAt: now,
	}
	gatewayContainer, joinNetworks := s.resolveGatewayJoin(ctx, stack)
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeployRequested, Payload: req.toEvent(d.ID, stack, gatewayContainer, joinNetworks)}
	if err := s.repo.Create(ctx, d, evt); err != nil {
		return nil, fmt.Errorf("create deploy: %w", err)
	}
	return d, nil
}

// Rollback redeploys the last healthy image as a normal record, so provenance tracks the acting user.
func (s *Service) Rollback(ctx context.Context, stackID, triggeredBy string) (*Deploy, error) {
	if strings.TrimSpace(stackID) == "" {
		return nil, fmt.Errorf("%w: stack is required", apperrs.ErrInvalid)
	}
	last, err := s.repo.LastHealthy(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("rollback %s: %w", stackID, err)
	}
	if strings.TrimSpace(last.Image) == "" {
		return nil, fmt.Errorf("%w: stack %s has no healthy image to roll back to", apperrs.ErrConflict, last.Service)
	}
	return s.Deploy(ctx, DeployRequest{
		StackID:     stackID,
		Image:       last.Image,
		TriggeredBy: triggeredBy,
	})
}

// declaredServices resolves the pending Container rows a new stack starts with, keyed by compose service name
// (matching a compose file's own `services:` map); a run stack (or any empty declared map) defaults to exactly
// one container named after the slug.
func declaredServices(stack *Stack, declared map[string]Declared) []Container {
	if len(declared) == 0 {
		declared = map[string]Declared{stack.Slug: {Ports: stack.Ports}}
	}
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic order; map iteration isn't.
	out := make([]Container, 0, len(names))
	for _, name := range names {
		out = append(out, Container{
			ID: ids.New(), StackID: stack.ID, Name: name, Declared: declared[name], Status: ServiceStatusPending,
		})
	}
	return out
}

// CreateStack persists a new stack and its pending services, rejecting a duplicate slug per machine,
// and publishes stack.created (still on the "service.created" topic). Renamed from CreateService.
func (s *Service) CreateStack(ctx context.Context, stack Stack, declared map[string]Declared) (*Stack, error) {
	stack = applyStackDefaults(stack)
	if err := stack.Validate(); err != nil {
		return nil, err
	}
	if err := s.checkProject(ctx, &stack); err != nil {
		return nil, err
	}
	if err := s.checkBranchDeployRules(ctx, &stack); err != nil {
		return nil, err
	}
	stack.Slug = Slug(stack.Name, dnsLabelMaxLen)
	if err := s.checkSlugUnique(ctx, &stack, ""); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	stack.ID = ids.New()
	stack.Managed = true
	stack.CreatedAt = now
	stack.UpdatedAt = now
	services := declaredServices(&stack, declared)
	evt := eventbus.OutboxEvent{
		ID:      ids.New(),
		Topic:   TopicStackCreated,
		Payload: StackEvent{Stack: stack, Services: services},
	}
	if err := s.stacks.Create(ctx, &stack, evt); err != nil {
		return nil, fmt.Errorf("create stack %s: %w", stack.Name, err)
	}
	for _, svc := range services {
		if err := s.services.Create(ctx, &svc); err != nil {
			return nil, fmt.Errorf("create service %s for stack %s: %w", svc.Name, stack.Name, err)
		}
	}
	return &stack, nil
}

// GetStack returns a stack by id.
func (s *Service) GetStack(ctx context.Context, id string) (*Stack, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	stack, err := s.stacks.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get stack %s: %w", id, err)
	}
	return stack, nil
}

// GetServiceByName returns a stack by name, since dns exposures reference stacks by name, not id.
func (s *Service) GetServiceByName(ctx context.Context, name string) (*Stack, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: stack name is required", apperrs.ErrInvalid)
	}
	stack, err := s.stacks.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get stack %s: %w", name, err)
	}
	return stack, nil
}

// ListStacks returns a project's base stacks; branch deployments list separately, via ListBranchDeployments.
// An empty projectID spans every project, which the workspace-wide topology canvas needs to wire networks and
// hostnames.
func (s *Service) ListStacks(ctx context.Context, projectID string) ([]*Stack, error) {
	stacks, err := s.stacks.ListByProject(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return nil, fmt.Errorf("list stacks for project %s: %w", projectID, err)
	}
	out := make([]*Stack, 0, len(stacks))
	for _, stack := range stacks {
		if stack.DerivedFrom == "" {
			out = append(out, stack)
		}
	}
	return out, nil
}

// ListBranchDeployments returns a base stack's live branch deployments, the accessor its API response attaches.
func (s *Service) ListBranchDeployments(ctx context.Context, baseStackID string) ([]*Stack, error) {
	if strings.TrimSpace(baseStackID) == "" {
		return nil, fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	stacks, err := s.stacks.ListByDerivedFrom(ctx, baseStackID)
	if err != nil {
		return nil, fmt.Errorf("list branch deployments for %s: %w", baseStackID, err)
	}
	return stacks, nil
}

// UpdateStack validates and persists a definition change and publishes stack.updated via the outbox.
func (s *Service) UpdateStack(ctx context.Context, stack Stack) (*Stack, error) {
	if strings.TrimSpace(stack.ID) == "" {
		return nil, fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	stack = applyStackDefaults(stack)
	if err := stack.Validate(); err != nil {
		return nil, err
	}
	existing, err := s.stacks.GetByID(ctx, stack.ID)
	if err != nil {
		return nil, fmt.Errorf("update stack %s: %w", stack.ID, err)
	}
	if bs := stack.BuildSource; existing.BuildSource == nil && bs != nil && bs.RepoOwner != "" && bs.RepoName != "" && s.projects != nil {
		if err := s.projects.LinkRepo(ctx, stack.ProjectID, bs.RepoOwner, bs.RepoName); err != nil {
			return nil, fmt.Errorf("link repository %s/%s: %w", bs.RepoOwner, bs.RepoName, err)
		}
	}
	if err := s.checkProject(ctx, &stack); err != nil {
		return nil, err
	}
	if err := s.checkBranchDeployRules(ctx, &stack); err != nil {
		return nil, err
	}
	// Slug is derived once at creation and never re-derived from a later name change (spec §2).
	stack.Slug = existing.Slug
	if err := s.checkSlugUnique(ctx, &stack, stack.ID); err != nil {
		return nil, err
	}
	stack.Managed = existing.Managed || stack.BuildSource != nil
	stack.CreatedAt = existing.CreatedAt
	stack.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{
		ID:      ids.New(),
		Topic:   TopicStackUpdated,
		Payload: StackEvent{Stack: stack},
	}
	if err := s.stacks.Update(ctx, &stack, evt); err != nil {
		return nil, fmt.Errorf("update stack %s: %w", stack.ID, err)
	}
	return &stack, nil
}

// DeleteStack releases every hostname the stack holds from the tunnel/DNS first, then tears down the
// definition; deploy history is kept, its services are dropped.
func (s *Service) DeleteStack(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	stack, err := s.stacks.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete stack %s: %w", id, err)
	}
	services, err := s.services.ListByStack(ctx, id)
	if err != nil {
		return fmt.Errorf("list services for stack %s: %w", id, err)
	}
	serviceIDs := make([]string, len(services))
	for i, svc := range services {
		serviceIDs[i] = svc.ID
	}
	if s.exposures != nil {
		if err := s.exposures.RemoveExposures(ctx, stack.Name, serviceIDs); err != nil {
			return fmt.Errorf("remove exposures for stack %s: %w", stack.Name, err)
		}
	}
	evt := eventbus.OutboxEvent{
		ID:      ids.New(),
		Topic:   TopicStackDeleted,
		Payload: StackDeletedEvent{ID: stack.ID, Name: stack.Name, ServiceIDs: serviceIDs},
	}
	if err := s.stacks.Delete(ctx, id, evt); err != nil {
		return fmt.Errorf("delete stack %s: %w", id, err)
	}
	if err := s.services.DeleteByStack(ctx, id); err != nil {
		return fmt.Errorf("delete services for stack %s: %w", id, err)
	}
	return nil
}

// ListServices returns a stack's observed containers.
func (s *Service) ListServices(ctx context.Context, stackID string) ([]*Container, error) {
	if strings.TrimSpace(stackID) == "" {
		return nil, fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	svcs, err := s.services.ListByStack(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("list services for stack %s: %w", stackID, err)
	}
	return svcs, nil
}

// GetService returns one observed container by id.
func (s *Service) GetService(ctx context.Context, id string) (*Container, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: service id is required", apperrs.ErrInvalid)
	}
	svc, err := s.services.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get service %s: %w", id, err)
	}
	return svc, nil
}

// checkProject also requires the build source's repo to belong to the project, if one is set.
func (s *Service) checkProject(ctx context.Context, stack *Stack) error {
	if s.projects == nil {
		return nil
	}
	ok, err := s.projects.ProjectExists(ctx, stack.ProjectID)
	if err != nil {
		return fmt.Errorf("check project %s: %w", stack.ProjectID, err)
	}
	if !ok {
		return fmt.Errorf("%w: project %s does not exist", apperrs.ErrInvalid, stack.ProjectID)
	}
	if bs := stack.BuildSource; bs != nil && bs.RepoOwner != "" {
		in, err := s.projects.RepoInProject(ctx, stack.ProjectID, bs.RepoOwner, bs.RepoName)
		if err != nil {
			return fmt.Errorf("check repo %s/%s: %w", bs.RepoOwner, bs.RepoName, err)
		}
		if !in {
			return fmt.Errorf("%w: repository %s/%s is not in project %s", apperrs.ErrInvalid, bs.RepoOwner, bs.RepoName, stack.ProjectID)
		}
	}
	return nil
}

// CreateStackOptions are the wizard's extras on stack creation: link the build repository to the project first,
// and enqueue the first deploy at the given ref (default: the build source's branch) once the stack exists.
type CreateStackOptions struct {
	LinkRepository bool
	Deploy         bool
	Ref            string
	TriggeredBy    string
}

// CreateStackWithOptions is one wizard step server-side: link, create, deploy. Linking before creating is what
// lets a fresh repository pass CreateStack's repo-in-project check.
func (s *Service) CreateStackWithOptions(ctx context.Context, in Stack, declared map[string]Declared, opts CreateStackOptions) (*Stack, *Deploy, error) {
	if opts.LinkRepository && in.BuildSource != nil && in.BuildSource.RepoOwner != "" {
		if err := s.projects.LinkRepo(ctx, in.ProjectID, in.BuildSource.RepoOwner, in.BuildSource.RepoName); err != nil {
			return nil, nil, fmt.Errorf("link repository %s/%s: %w", in.BuildSource.RepoOwner, in.BuildSource.RepoName, err)
		}
	}
	stack, err := s.CreateStack(ctx, in, declared)
	if err != nil {
		return nil, nil, err
	}
	if !opts.Deploy {
		return stack, nil, nil
	}
	ref := opts.Ref
	if ref == "" && stack.BuildSource != nil {
		ref = stack.BuildSource.Branch
	}
	d, err := s.Deploy(ctx, DeployRequest{StackID: stack.ID, Ref: ref, TriggeredBy: opts.TriggeredBy})
	if err != nil {
		return stack, nil, fmt.Errorf("first deploy of %s: %w", stack.Name, err)
	}
	return stack, d, nil
}

// checkBranchDeployRules rejects a hostname template whose docker network has no gateway to ever satisfy it.
func (s *Service) checkBranchDeployRules(ctx context.Context, stack *Stack) error {
	if s.gateways == nil {
		return nil
	}
	for _, r := range stack.BranchDeployRules {
		if r.HostnameTemplate == "" {
			continue
		}
		ok, err := s.gateways.NetworkHasGateway(ctx, r.DockerNetwork)
		if err != nil {
			return fmt.Errorf("check gateway for network %s: %w", r.DockerNetwork, err)
		}
		if !ok {
			return fmt.Errorf("%w: branch deploy rule %q's network %q has no gateway to expose %q",
				apperrs.ErrInvalid, r.Pattern, r.DockerNetwork, r.HostnameTemplate)
		}
	}
	return nil
}

// resolveGatewayJoin looks up whether a gateway already serves stack's own network, so the deploy's last step
// can rejoin it; a lookup failure just skips the step, since it must never block a
// deploy the same way a routing nicety shouldn't.
func (s *Service) resolveGatewayJoin(ctx context.Context, stack *Stack) (string, []string) {
	if s.gatewayJoin == nil {
		return "", nil
	}
	network := stack.DefaultNetwork()
	if network == "" {
		return "", nil
	}
	container, found, err := s.gatewayJoin.GatewayForStackNetwork(ctx, network)
	if err != nil || !found {
		return "", nil
	}
	return container, []string{network}
}

// checkSlugUnique rejects a duplicate (machine, slug) pair; excludingID lets an update keep its own slug.
func (s *Service) checkSlugUnique(ctx context.Context, stack *Stack, excludingID string) error {
	existing, err := s.stacks.GetBySlugAndMachine(ctx, stack.Slug, stack.Machine)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("check stack slug %s: %w", stack.Slug, err)
	}
	if existing.ID == excludingID {
		return nil
	}
	return fmt.Errorf("%w: stack name %q already exists on machine %s", apperrs.ErrConflict, stack.Name, stack.Machine)
}

// HandleStatusChanged is the status_changed consumer that lands the runner's terminal state on the recorded deploy.
func (s *Service) HandleStatusChanged(ctx context.Context, ev eventbus.Event) error {
	var p DeployStatusChangedEvent
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse deploy.status_changed: %w", err))
	}
	status := Status(p.Status)
	if status != StatusHealthy && status != StatusFailed {
		return apperrs.Fatal(fmt.Errorf("%w: unknown deploy status %q", apperrs.ErrInvalid, p.Status))
	}
	if p.Error != "" {
		line := LogLine{TS: s.now().UnixMilli(), Text: "deploy failed: " + p.Error}
		if err := s.repo.AppendLogLines(ctx, p.ID, []LogLine{line}); err != nil {
			return fmt.Errorf("append deploy %s log: %w", p.ID, err)
		}
	}
	if err := s.transition(ctx, p.ID, status); err != nil {
		return err
	}
	if p.Address != "" {
		if err := s.repo.SetAddress(ctx, p.ID, p.Address); err != nil {
			return fmt.Errorf("set deploy %s address: %w", p.ID, err)
		}
	}
	if len(p.Services) > 0 {
		if err := s.reconcileServices(ctx, p.ID, p.Services); err != nil {
			return fmt.Errorf("reconcile services for deploy %s: %w", p.ID, err)
		}
	}
	return nil
}

// HandleLog is the deploy.log consumer: one row per line of the runner's batch, all stamped with the batch's
// phase and time. A missing deploy is fatal, since the row is written before deploy.requested ever leaves the outbox.
func (s *Service) HandleLog(ctx context.Context, ev eventbus.Event) error {
	var p DeployLogEvent
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse deploy.log: %w", err))
	}
	if p.ID == "" {
		return apperrs.Fatal(fmt.Errorf("%w: deploy.log requires id", apperrs.ErrInvalid))
	}
	lines := p.lines()
	if len(lines) == 0 {
		return nil
	}
	err := s.repo.AppendLogLines(ctx, p.ID, lines)
	if errors.Is(err, apperrs.ErrNotFound) {
		return apperrs.Fatal(fmt.Errorf("append deploy %s log: %w", p.ID, err))
	}
	if err != nil {
		return fmt.Errorf("append deploy %s log: %w", p.ID, err)
	}
	return nil
}

// Log returns a deploy's output ordered by time, never nil, so the gateway serialises an empty log as [].
func (s *Service) Log(ctx context.Context, id string) ([]LogLine, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	lines, err := s.repo.ListLogLines(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list deploy %s log: %w", id, err)
	}
	if lines == nil {
		return []LogLine{}, nil
	}
	return lines, nil
}

// reconcileServices lands the runner's observation report onto the deploying stack's containers (spec §4 step 3):
// upsert by (stack, name) with the observed facts, and mark any existing container missing from the report as
// stopped — never deleted. Declared (the compose parse's own data) is preserved from the existing row; the report
// only ever carries what docker observed.
func (s *Service) reconcileServices(ctx context.Context, deployID string, observed []ObservedService) error {
	d, err := s.repo.GetByID(ctx, deployID)
	if err != nil {
		return err
	}
	existing, err := s.services.ListByStack(ctx, d.StackID)
	if err != nil {
		return err
	}
	byName := make(map[string]*Container, len(existing))
	for _, c := range existing {
		byName[c.Name] = c
	}
	now := s.now().UTC()
	seen := make(map[string]bool, len(observed))
	for _, o := range observed {
		seen[o.Name] = true
		c := &Container{
			ID: ids.New(), StackID: d.StackID, Name: o.Name,
			ContainerName: o.ContainerName, Image: o.Image, Status: ServiceStatus(o.Status),
			Networks: o.Networks, Ports: o.Ports, ObservedAt: now,
		}
		if prev, ok := byName[o.Name]; ok {
			c.ID = prev.ID
			c.Declared = prev.Declared
		}
		if err := s.services.Upsert(ctx, c); err != nil {
			return fmt.Errorf("upsert service %s: %w", o.Name, err)
		}
	}
	for _, c := range existing {
		if seen[c.Name] || c.Status == ServiceStatusStopped {
			continue
		}
		stopped := *c
		stopped.Status = ServiceStatusStopped
		stopped.ObservedAt = now
		if err := s.services.Upsert(ctx, &stopped); err != nil {
			return fmt.Errorf("mark service %s stopped: %w", c.Name, err)
		}
	}
	return nil
}

// transition validates and applies a single state-machine step.
func (s *Service) transition(ctx context.Context, id string, to Status) error {
	cur, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("transition deploy %s: %w", id, err)
	}
	if !CanTransition(cur.Status, to) {
		if cur.Status == to {
			// Already in the target state: a redelivered status event must be a no-op, not an error (at-least-once bus).
			return nil
		}
		return fmt.Errorf("%w: cannot transition deploy %s from %s to %s", apperrs.ErrInvalid, id, cur.Status, to)
	}
	if err := s.repo.UpdateStatus(ctx, id, to); err != nil {
		return fmt.Errorf("transition deploy %s: %w", id, err)
	}
	return nil
}

// Cancel's terminal state still arrives via status_changed, not from this call.
func (s *Service) Cancel(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("cancel deploy %s: %w", id, err)
	}
	switch d.Status {
	case StatusPending, StatusRunning:
	default:
		return fmt.Errorf("%w: deploy %s is %s, cannot cancel", apperrs.ErrConflict, id, d.Status)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeployCancelRequested, Payload: DeployCancelRequestedEvent{ID: id}}
	if err := s.repo.CancelRequested(ctx, evt); err != nil {
		return fmt.Errorf("enqueue deploy.cancel_requested: %w", err)
	}
	return nil
}

// Get returns a deploy by id.
func (s *Service) Get(ctx context.Context, id string) (*Deploy, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get deploy %s: %w", id, err)
	}
	return d, nil
}

// List returns all deploys, oldest first.
func (s *Service) List(ctx context.Context) ([]*Deploy, error) {
	ds, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list deploys: %w", err)
	}
	return ds, nil
}

// ListByService returns the deploy history for one stack name, newest first.
func (s *Service) ListByService(ctx context.Context, service string) ([]*Deploy, error) {
	if strings.TrimSpace(service) == "" {
		return nil, fmt.Errorf("%w: stack is required", apperrs.ErrInvalid)
	}
	ds, err := s.repo.ListByService(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("list deploys for %s: %w", service, err)
	}
	return ds, nil
}

// ListByStackID returns the deploy history for one stack id, newest first.
func (s *Service) ListByStackID(ctx context.Context, stackID string) ([]*Deploy, error) {
	if strings.TrimSpace(stackID) == "" {
		return nil, fmt.Errorf("%w: stack is required", apperrs.ErrInvalid)
	}
	ds, err := s.repo.ListByStackID(ctx, stackID)
	if err != nil {
		return nil, fmt.Errorf("list deploys for stack %s: %w", stackID, err)
	}
	return ds, nil
}

// ListByStatus returns deploys in one state, newest first.
func (s *Service) ListByStatus(ctx context.Context, status Status) ([]*Deploy, error) {
	switch status {
	case StatusPending, StatusRunning, StatusHealthy, StatusFailed:
	default:
		return nil, fmt.Errorf("%w: unknown deploy status %q", apperrs.ErrInvalid, status)
	}
	ds, err := s.repo.ListByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("list deploys with status %s: %w", status, err)
	}
	return ds, nil
}

// applyStackDefaults fills the compose-strategy default compose path so a stack always carries a concrete value.
func applyStackDefaults(stack Stack) Stack {
	if stack.Strategy == StrategyCompose && strings.TrimSpace(stack.ComposePath) == "" {
		stack.ComposePath = defaultComposePath
	}
	return stack
}
