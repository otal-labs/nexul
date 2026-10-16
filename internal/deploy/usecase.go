package deploy

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// ProjectStore validates a stack's project and build-source repo against the workspace domain (ADR 0017).
type ProjectStore interface {
	ProjectExists(ctx context.Context, projectID string) (bool, error)
	RepoInProject(ctx context.Context, projectID, owner, name string) (bool, error)
	// LinkRepo attaches a repository to a project; linking one already attached is a no-op.
	LinkRepo(ctx context.Context, projectID, owner, name string) error
	// IsTestsRepo reports a project's tests repository, which never builds a stack or deploys.
	IsTestsRepo(ctx context.Context, owner, name string) (bool, error)
	// WorkspaceOf is the workspace a project belongs to.
	WorkspaceOf(ctx context.Context, projectID string) (string, error)
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
	logs        LogSource
	gate        Gate
	now         func() time.Time
	outages     OutageRepo
	desk        TicketDesk
	crashLinks  CrashTicketRepo
	watchers    StackWatchRepo
	requests    *requestState
	errorRates  ErrorRateRepo
	// outageMu serializes outage changes with the recovery timers; recoveries are the steady-state waits in flight.
	outageMu   sync.Mutex
	recoveries map[string]*recovery
}

// Gate is the permission check a stack or deploy passes before the caller touches it (the access domain, ADR 0042).
type Gate interface {
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
	// RequireStack checks action on a stack of projectID through the rules on stackID, a branch deployment's base.
	RequireStack(ctx context.Context, projectID, stackID string, action permissions.Action) error
	RequireAnywhere(ctx context.Context, action permissions.Action) error
	// CallerStacks is what a stack or deploy list in workspaceID, or every one when empty, filters by for action; all is
	// true for every project's stacks.
	CallerStacks(ctx context.Context, workspaceID string, action permissions.Action) (reach permissions.StackReach, all bool, err error)
	// RequireMachine checks a machine action by id; a machine the caller can't see is not found (ADR 0151).
	RequireMachine(ctx context.Context, machineID string, action permissions.Action) error
	// RequireMachineNamed checks a machine a person types by name, answering a machine they can't see as it does a name
	// no machine bears; RequireMachineOfStack checks the one a stack they found carries, not found when they can't see it.
	RequireMachineNamed(ctx context.Context, name string, action permissions.Action) error
	RequireMachineOfStack(ctx context.Context, name string, action permissions.Action) error
	// CallerMachines is the ids of the machines the caller sees, for a list of instance stacks to filter by; all is true
	// when they see every one without asking.
	CallerMachines(ctx context.Context) (machineIDs []string, all bool, err error)
}

// SetGate wires the permission check; unset, only the server's own calls pass.
func (s *Service) SetGate(g Gate) { s.gate = g }

// requireIn checks action where a stack lives: its project's workspace, or any workspace for one outside every
// project, the instance's own gateways (ADR 0079).
func (s *Service) requireIn(ctx context.Context, projectID string, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	if projectID == "" {
		return s.gate.RequireAnywhere(ctx, action)
	}
	return s.gate.RequireProject(ctx, projectID, action)
}

// requireStack checks action on stack: an instance stack anywhere (ADR 0079), a project's through its rules, a branch
// deployment's through its base stack's (ADR 0148).
func (s *Service) requireStack(ctx context.Context, stack *Stack, action permissions.Action) error {
	if s.gate == nil || stack.ProjectID == "" {
		if err := s.requireIn(ctx, stack.ProjectID, action); err != nil {
			return err
		}
		return s.requireInstanceStack(ctx, stack, action)
	}
	return s.gate.RequireStack(ctx, stack.ProjectID, ruleStackID(stack), action)
}

// requireInstanceStack adds what a stack outside every project takes beside its area: seeing it is See on its machine
// and dns:read, and a stack missing either is not found; editing or deploying it is Deploy to and dns:write (ADR 0151).
func (s *Service) requireInstanceStack(ctx context.Context, stack *Stack, action permissions.Action) error {
	if s.gate == nil || stack.ProjectID != "" {
		return nil
	}
	if !reads(action) {
		return s.requireAim(ctx, "", stack.Machine)
	}
	if err := s.gate.RequireMachineOfStack(ctx, stack.Machine, permissions.MachinesSee); err != nil {
		return err
	}
	err := s.gate.RequireAnywhere(ctx, permissions.DNSRead)
	if permissions.Refused(err) {
		return fmt.Errorf("%w: stack %s", apperrs.ErrNotFound, stack.ID)
	}
	return err
}

// reads is whether action only looks at a stack or a deploy.
func reads(action permissions.Action) bool {
	return action == permissions.StacksRead || action == permissions.StacksLogs || action == permissions.DeploysRead
}

// requireAim takes Deploy to on the machine something is aimed at, and dns:write too for an instance stack, beside the
// stack checks the caller already passed.
func (s *Service) requireAim(ctx context.Context, projectID, machine string) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	if err := s.gate.RequireMachineNamed(ctx, machine, permissions.MachinesDeploy); err != nil {
		return err
	}
	if projectID != "" {
		return nil
	}
	return s.gate.RequireAnywhere(ctx, permissions.DNSWrite)
}

// ruleStackID is the stack whose rules reach stack: its base for a branch deployment, else itself.
func ruleStackID(stack *Stack) string {
	if stack.DerivedFrom != "" {
		return stack.DerivedFrom
	}
	return stack.ID
}

// loadStack fetches a stack the caller may act on with action.
func (s *Service) loadStack(ctx context.Context, id string, action permissions.Action) (*Stack, error) {
	stack, err := s.stacks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireStack(ctx, stack, action); err != nil {
		return nil, err
	}
	return stack, nil
}

// loadForUpdate fetches the stack an update replaces; the caller needs stacks:write where it is and where it goes.
func (s *Service) loadForUpdate(ctx context.Context, stack Stack) (*Stack, error) {
	existing, err := s.loadStack(ctx, stack.ID, permissions.StacksWrite)
	if err != nil {
		return nil, err
	}
	if err := s.requireIn(ctx, stack.ProjectID, permissions.StacksWrite); err != nil {
		return nil, err
	}
	if existing.Machine == stack.Machine {
		return existing, nil
	}
	if err := s.requireAim(ctx, stack.ProjectID, stack.Machine); err != nil {
		return nil, err
	}
	return existing, nil
}

// requireDeploy checks action on a deploy through its stack; a deleted stack's kept history is instance-level.
func (s *Service) requireDeploy(ctx context.Context, stackID string, action permissions.Action) error {
	if identity.Internal(ctx) {
		return nil
	}
	stack, err := s.stacks.GetByID(ctx, stackID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return s.requireIn(ctx, "", action)
	}
	if err != nil {
		return fmt.Errorf("stack %s: %w", stackID, err)
	}
	return s.requireStack(ctx, stack, action)
}

// readableDeploys keeps the deploys the caller may read.
func (s *Service) readableDeploys(ctx context.Context, ds []*Deploy) ([]*Deploy, error) {
	return permissions.Filter(ds, func(d *Deploy) string { return d.StackID }, func(stackID string) error {
		return s.requireDeploy(ctx, stackID, permissions.DeploysRead)
	})
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
	return s.start(ctx, req, "")
}

// start creates the deploy and its deploy.requested row; a retry of retryOf also writes deploy.retried beside it.
func (s *Service) start(ctx context.Context, req DeployRequest, retryOf string) (*Deploy, error) {
	req.normalize()
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("deploy %s: %w", req.StackID, err)
	}
	stack, err := s.loadStack(ctx, req.StackID, permissions.DeploysWrite)
	if err != nil {
		return nil, fmt.Errorf("deploy %s: %w", req.StackID, err)
	}
	if err := s.requireDeployTo(ctx, stack); err != nil {
		return nil, fmt.Errorf("deploy %s: %w", stack.Name, err)
	}
	if err := s.checkNotTestsRepo(ctx, stack); err != nil {
		return nil, fmt.Errorf("deploy %s: %w", stack.Name, err)
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
		Image: req.Image, Ref: req.Ref, Kind: kind, Status: StatusPending, Strategy: stack.Strategy,
		TriggeredBy: req.TriggeredBy, RuleID: req.RuleID, RuleName: req.RuleName,
		TicketID: req.TicketID, PRNumber: req.PRNumber,
		CreatedAt: now, UpdatedAt: now,
	}
	gatewayContainer, joinNetworks := s.resolveGatewayJoin(ctx, stack)
	evts := []eventbus.OutboxEvent{{ID: ids.New(), Topic: TopicDeployRequested, Payload: req.toEvent(d.ID, stack, gatewayContainer, joinNetworks)}}
	if retryOf != "" {
		evts = append(evts, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeployRetried, Payload: DeployRetriedEvent{
			ID: d.ID, RetryOf: retryOf, StackID: stack.ID, Ref: req.Ref, Image: req.Image,
		}})
	}
	if err := s.repo.Create(ctx, d, evts...); err != nil {
		return nil, fmt.Errorf("create deploy: %w", err)
	}
	return d, nil
}

// Retry starts a failed deploy again on the commit it built, else its ref or image, with the stack's current settings.
func (s *Service) Retry(ctx context.Context, id, triggeredBy string) (*Deploy, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", apperrs.ErrInvalid)
	}
	failed, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("retry deploy %s: %w", id, err)
	}
	if err := s.requireDeploy(ctx, failed.StackID, permissions.DeploysWrite); err != nil {
		return nil, fmt.Errorf("retry deploy %s: %w", id, err)
	}
	if failed.Status != StatusFailed {
		return nil, fmt.Errorf("%w: deploy %s is %s, only a failed deploy can be retried", apperrs.ErrConflict, id, failed.Status)
	}
	if failed.StackID == "" {
		return nil, fmt.Errorf("%w: the stack of deploy %s was deleted", apperrs.ErrConflict, id)
	}
	req := DeployRequest{StackID: failed.StackID, Image: failed.Image, Ref: cmp.Or(failed.Commit, failed.Ref), TriggeredBy: triggeredBy}
	if req.Image == "" && req.Ref == "" {
		return nil, fmt.Errorf("%w: deploy %s did not record what it built; build and deploy a ref of the stack instead", apperrs.ErrConflict, id)
	}
	d, err := s.start(ctx, req, id)
	if err != nil {
		return nil, fmt.Errorf("retry deploy %s: %w", id, err)
	}
	return d, nil
}

// requireDeployTo takes Deploy to on the stack's machine for a deploy a person starts, so taking them off the machine
// stops their next one; the server's own deploys pass. An instance stack took it already in requireStack.
func (s *Service) requireDeployTo(ctx context.Context, stack *Stack) error {
	if s.gate == nil || stack.ProjectID == "" {
		return nil
	}
	return s.gate.RequireMachineOfStack(ctx, stack.Machine, permissions.MachinesDeploy)
}

// requireDeployToOf is requireDeployTo for the stack a deploy ran on; a deleted stack's history has no machine to ask.
func (s *Service) requireDeployToOf(ctx context.Context, stackID string) error {
	stack, err := s.stacks.GetByID(ctx, stackID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stack %s: %w", stackID, err)
	}
	return s.requireDeployTo(ctx, stack)
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
	if err := s.requireCreate(ctx, stack); err != nil {
		return nil, err
	}
	if err := s.requireAim(ctx, stack.ProjectID, stack.Machine); err != nil {
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
		Payload: newStackEvent(stack, services),
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

// requireCreate takes stacks:write where the new stack goes; a branch deployment's is its base stack's (ADR 0148).
func (s *Service) requireCreate(ctx context.Context, stack Stack) error {
	if stack.DerivedFrom == "" || stack.ProjectID == "" {
		return s.requireIn(ctx, stack.ProjectID, permissions.StacksWrite)
	}
	return s.requireStack(ctx, &stack, permissions.StacksWrite)
}

// GetStack returns a stack by id.
func (s *Service) GetStack(ctx context.Context, id string) (*Stack, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	stack, err := s.loadStack(ctx, id, permissions.StacksRead)
	if err != nil {
		return nil, fmt.Errorf("get stack %s: %w", id, err)
	}
	return s.readableBy(ctx, stack)
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
	return s.ListScopedStacks(ctx, projectID, "")
}

// ListScopedStacks narrows to a project when given, else to a workspace's projects, else lists every base stack the
// caller reads, filtered in SQL (ADR 0140).
func (s *Service) ListScopedStacks(ctx context.Context, projectID, workspaceID string) ([]*Stack, error) {
	projectID, workspaceID = strings.TrimSpace(projectID), strings.TrimSpace(workspaceID)
	if projectID != "" {
		workspaceID = ""
	}
	scope, err := s.listScope(ctx, workspaceID, permissions.StacksRead, projectID == "" && workspaceID == "")
	if err != nil {
		return nil, fmt.Errorf("list stacks: %w", err)
	}
	stacks, err := s.stacks.ListReached(ctx, projectID, workspaceID, scope)
	if err != nil {
		return nil, fmt.Errorf("list stacks: %w", err)
	}
	readable, err := s.readableListBy(ctx, stacks, workspaceID, projectID == "" && workspaceID == "")
	if err != nil {
		return nil, fmt.Errorf("list stacks: %w", err)
	}
	return readable, nil
}

func baseStacks(stacks []*Stack) []*Stack {
	out := make([]*Stack, 0, len(stacks))
	for _, stack := range stacks {
		if stack.DerivedFrom == "" {
			out = append(out, stack)
		}
	}
	return out
}

// ListBranchDeployments returns a base stack's live branch deployments, the accessor its API response attaches.
func (s *Service) ListBranchDeployments(ctx context.Context, baseStackID string) ([]*Stack, error) {
	if strings.TrimSpace(baseStackID) == "" {
		return nil, fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	base, err := s.loadStack(ctx, baseStackID, permissions.StacksRead)
	if err != nil {
		return nil, fmt.Errorf("list branch deployments for %s: %w", baseStackID, err)
	}
	stacks, err := s.stacks.ListByDerivedFrom(ctx, baseStackID)
	if err != nil {
		return nil, fmt.Errorf("list branch deployments for %s: %w", baseStackID, err)
	}
	readable, err := s.readableBranchesBy(ctx, base, stacks)
	if err != nil {
		return nil, fmt.Errorf("list branch deployments for %s: %w", baseStackID, err)
	}
	return readable, nil
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
	existing, err := s.loadForUpdate(ctx, stack)
	if err != nil {
		return nil, fmt.Errorf("update stack %s: %w", stack.ID, err)
	}
	if err := s.linkNewRepo(ctx, existing, &stack); err != nil {
		return nil, err
	}
	if err := s.checkProject(ctx, &stack); err != nil {
		return nil, err
	}
	if err := s.checkBranchDeployRules(ctx, &stack); err != nil {
		return nil, err
	}
	if err := s.changeErrorRate(ctx, existing, &stack); err != nil {
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
		Payload: newStackEvent(stack, nil),
	}
	if err := s.stacks.Update(ctx, &stack, evt); err != nil {
		return nil, fmt.Errorf("update stack %s: %w", stack.ID, err)
	}
	return &stack, nil
}

// linkNewRepo links the repository a stack has just been given to its project.
func (s *Service) linkNewRepo(ctx context.Context, existing, stack *Stack) error {
	bs := stack.BuildSource
	if existing.BuildSource != nil || bs == nil || bs.RepoOwner == "" || bs.RepoName == "" || s.projects == nil {
		return nil
	}
	if err := s.projects.LinkRepo(ctx, stack.ProjectID, bs.RepoOwner, bs.RepoName); err != nil {
		return fmt.Errorf("link repository %s/%s: %w", bs.RepoOwner, bs.RepoName, err)
	}
	return nil
}

// DeleteStack releases every hostname the stack holds from the tunnel/DNS first, then tears down the
// definition; deploy history is kept, its services are dropped.
func (s *Service) DeleteStack(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	stack, err := s.loadStack(ctx, id, permissions.StacksDelete)
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
		Payload: StackDeletedEvent{ID: stack.ID, Name: stack.Name, ServiceIDs: serviceIDs, ProjectID: stack.ProjectID},
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
	if _, err := s.loadStack(ctx, stackID, permissions.StacksRead); err != nil {
		return nil, fmt.Errorf("list services for stack %s: %w", stackID, err)
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
	if _, err := s.loadStack(ctx, svc.StackID, permissions.StacksRead); err != nil {
		return nil, fmt.Errorf("get service %s: %w", id, err)
	}
	return svc, nil
}

// checkProject also requires the build source's repo to belong to the project, if one is set.
func (s *Service) checkProject(ctx context.Context, stack *Stack) error {
	if s.projects == nil || stack.ProjectID == "" {
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
	return s.checkNotTestsRepo(ctx, stack)
}

// checkNotTestsRepo keeps a tests repository from ever building a stack or deploying.
func (s *Service) checkNotTestsRepo(ctx context.Context, stack *Stack) error {
	bs := stack.BuildSource
	if s.projects == nil || bs == nil || bs.RepoOwner == "" {
		return nil
	}
	tests, err := s.projects.IsTestsRepo(ctx, bs.RepoOwner, bs.RepoName)
	if err != nil {
		return fmt.Errorf("check repo %s/%s: %w", bs.RepoOwner, bs.RepoName, err)
	}
	if tests {
		return fmt.Errorf("%w: repository %s/%s is a tests repository and is never deployed", apperrs.ErrInvalid, bs.RepoOwner, bs.RepoName)
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
	if err := in.Validate(); err != nil {
		return nil, nil, err
	}
	if opts.LinkRepository && in.BuildSource != nil && in.BuildSource.RepoOwner != "" {
		// Linking changes the project, so the machine is checked before it, not after.
		if err := s.requireAim(ctx, in.ProjectID, in.Machine); err != nil {
			return nil, nil, err
		}
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
	if p.Commit != "" {
		if err := s.repo.SetCommit(ctx, p.ID, p.Commit); err != nil {
			return fmt.Errorf("set deploy %s commit: %w", p.ID, err)
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
	if status == StatusHealthy {
		s.endOutagesAfterDeploy(ctx, p)
	}
	return nil
}

// endOutagesAfterDeploy is best-effort: a healthy deploy has already landed, and the next runner snapshot settles
// whatever this misses.
func (s *Service) endOutagesAfterDeploy(ctx context.Context, p DeployStatusChangedEvent) {
	if s.outages == nil {
		return
	}
	d, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		slog.Warn("outages after deploy: get deploy failed", "deploy_id", p.ID, "error", err)
		return
	}
	if err := s.recoverAfterDeploy(ctx, d.StackID, p.Services); err != nil {
		slog.Warn("outages after deploy failed", "deploy_id", p.ID, "stack_id", d.StackID, "error", err)
	}
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
	cur, err := s.repo.GetByID(ctx, p.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return apperrs.Fatal(fmt.Errorf("append deploy %s log: %w", p.ID, err))
	}
	if err != nil {
		return fmt.Errorf("append deploy %s log: %w", p.ID, err)
	}
	err = s.repo.AppendLogLines(ctx, p.ID, lines, s.updatedEvent(cur, cur.Status))
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
	if err := s.repo.UpdateStatus(ctx, id, to, s.updatedEvent(cur, to)); err != nil {
		return fmt.Errorf("transition deploy %s: %w", id, err)
	}
	return nil
}

// updatedEvent is the deploy.updated outbox row a status or log write commits alongside itself.
func (s *Service) updatedEvent(d *Deploy, status Status) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicDeployUpdated, Payload: DeployUpdatedEvent{ID: d.ID, Status: string(status), StackID: d.StackID}}
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
	if err := s.requireDeploy(ctx, d.StackID, permissions.DeploysWrite); err != nil {
		return fmt.Errorf("cancel deploy %s: %w", id, err)
	}
	if err := s.requireDeployToOf(ctx, d.StackID); err != nil {
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
	if err := s.requireDeploy(ctx, d.StackID, permissions.DeploysRead); err != nil {
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
	return s.readableDeploys(ctx, ds)
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
	return s.readableDeploys(ctx, ds)
}

// ListByStackID returns the deploy history for one stack id, newest first.
func (s *Service) ListByStackID(ctx context.Context, stackID string) ([]*Deploy, error) {
	if strings.TrimSpace(stackID) == "" {
		return nil, fmt.Errorf("%w: stack is required", apperrs.ErrInvalid)
	}
	if err := s.requireDeploy(ctx, stackID, permissions.DeploysRead); err != nil {
		return nil, fmt.Errorf("list deploys for stack %s: %w", stackID, err)
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
	return s.readableDeploys(ctx, ds)
}

// PageDeploys returns one window of the deploys f keeps that the caller may read, newest first, and how many there are;
// the read is filtered in SQL, so the total counts only what the pages hold (ADR 0140).
func (s *Service) PageDeploys(ctx context.Context, f DeployFilter, w paging.Window) ([]*Deploy, int, error) {
	switch f.Status {
	case "", StatusPending, StatusRunning, StatusHealthy, StatusFailed:
	default:
		return nil, 0, fmt.Errorf("%w: unknown deploy status %q", apperrs.ErrInvalid, f.Status)
	}
	scope, err := s.deployScope(ctx, f.StackID)
	if err != nil {
		return nil, 0, err
	}
	ds, total, err := s.repo.Page(ctx, f, scope, w.Clamped())
	if err != nil {
		return nil, 0, fmt.Errorf("list deploys: %w", err)
	}
	return ds, total, nil
}

// deployScope is what a deploy list may show: one readable stack's whole history, or across stacks what the caller
// reads where each stack lives (ADR 0087).
func (s *Service) deployScope(ctx context.Context, stackID string) (DeployScope, error) {
	if stackID != "" {
		if err := s.requireDeploy(ctx, stackID, permissions.DeploysRead); err != nil {
			return DeployScope{}, fmt.Errorf("list deploys for stack %s: %w", stackID, err)
		}
		return DeployScope{All: true}, nil
	}
	scope, err := s.listScope(ctx, "", permissions.DeploysRead, true)
	if err != nil {
		return DeployScope{}, fmt.Errorf("list deploys: %w", err)
	}
	return scope, nil
}

// listScope is what a stack or deploy list in workspaceID, or every one, shows the caller for action: the projects and
// stacks access reaches, and with instance the instance's own stacks when they hold it anywhere (ADR 0079, ADR 0148).
func (s *Service) listScope(ctx context.Context, workspaceID string, action permissions.Action, instance bool) (DeployScope, error) {
	if identity.Internal(ctx) {
		return DeployScope{All: true}, nil
	}
	if s.gate == nil {
		return DeployScope{All: permissions.Ungated(ctx) == nil}, nil
	}
	reach, all, err := s.gate.CallerStacks(ctx, workspaceID, action)
	if err != nil {
		return DeployScope{}, err
	}
	scope := DeployScope{All: all, ProjectIDs: reach.ProjectIDs, Allowed: reach.Allowed, Denied: reach.Denied}
	if !instance {
		return scope, nil
	}
	anywhere := s.gate.RequireAnywhere(ctx, action)
	if anywhere != nil && !permissions.Refused(anywhere) {
		return DeployScope{}, anywhere
	}
	scope.Anywhere = anywhere == nil
	if !scope.Anywhere {
		return scope, nil
	}
	// An instance stack also takes dns:read and its machine's See; a deleted stack's deploys have neither to check.
	scope.Machines = []string{}
	dns := s.gate.RequireAnywhere(ctx, permissions.DNSRead)
	if dns != nil && !permissions.Refused(dns) {
		return DeployScope{}, dns
	}
	if dns != nil {
		return scope, nil
	}
	if scope.Machines, scope.AllMachines, err = s.gate.CallerMachines(ctx); err != nil {
		return DeployScope{}, err
	}
	return scope, nil
}

// applyStackDefaults fills the compose-strategy default compose path so a stack always carries a concrete value.
func applyStackDefaults(stack Stack) Stack {
	stack.EnvMasked = false
	stack.ErrorRate = stack.ErrorRate.withDefaults()
	if stack.Strategy == StrategyCompose && strings.TrimSpace(stack.ComposePath) == "" {
		stack.ComposePath = defaultComposePath
	}
	return stack
}
