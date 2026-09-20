package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// ImportNetwork mirrors the runner domain's discovered-network shape so deploy never imports runner (ADR 0017).
type ImportNetwork struct {
	Name    string
	Address string
}

// DiscoveredContainer mirrors the runner domain's observed-container shape so deploy never imports runner (ADR 0017).
type DiscoveredContainer struct {
	Name     string
	Image    string
	Status   string
	Networks []ImportNetwork
	Ports    []string
	// TunnelID is the Cloudflare tunnel a discovered cloudflared container connects (runner read it off the token).
	TunnelID string
}

// AdoptGatewayInput is what the dns domain needs to adopt a running tunnel as a gateway: the tunnel, the
// adopted cloudflared container row backing it, the networks it sits on, and the containers on the machine
// its routes may target (container name → services.id), so each routed hostname becomes an exposure.
type AdoptGatewayInput struct {
	TunnelID    string
	Machine     string
	ServiceID   string
	ServiceName string
	Networks    []string
	Targets     map[string]string
}

// AdoptedGateway reports what adopting one tunnel produced.
type AdoptedGateway struct {
	GatewayID string
	Exposed   []string
	Unmatched []string
}

// GatewayAdopter is the dns seam Import uses for a ticked tunnel gateway; deploy never imports dns (ADR 0017).
type GatewayAdopter interface {
	AdoptTunnelGateway(ctx context.Context, in AdoptGatewayInput) (*AdoptedGateway, error)
}

// GatewayAdoption is one ticked gateway's outcome in the import result: adopted with its hostnames, or the
// reason it could not be (no Cloudflare connector, no tunnel token on the container), never a failed import.
type GatewayAdoption struct {
	Name      string   `json:"name"`
	GatewayID string   `json:"gateway_id,omitempty"`
	Exposed   []string `json:"exposed"`
	Unmatched []string `json:"unmatched"`
	Error     string   `json:"error,omitempty"`
}

// MachineLookup resolves a machine id to its name, since Stack.Machine is the name, not the id (ADR 0017).
type MachineLookup interface {
	MachineName(ctx context.Context, machineID string) (string, error)
}

// MachineDiscoverer fetches fresh observed facts for a machine's containers, the runner-domain seam Import
// uses instead of trusting client-supplied facts (ADR 0017).
type MachineDiscoverer interface {
	DiscoverContainers(ctx context.Context, machineID string) ([]DiscoveredContainer, error)
}

// ImportStackGroup is one compose project's ticked container names (spec §8).
type ImportStackGroup struct {
	Project    string   `json:"project"`
	Containers []string `json:"containers"`
}

// ImportRequest is the ticked selection from the machine-discovery wizard (spec §8). ProjectID is required:
// the spec doesn't say which workspace project an adopted stack belongs to, so Import asks for it explicitly,
// the same way a normal CreateStack does.
type ImportRequest struct {
	ProjectID  string             `json:"project_id"`
	Stacks     []ImportStackGroup `json:"stacks"`
	Standalone []string           `json:"standalone"`
	Gateways   []string           `json:"gateways"`
}

// ImportResult reports what Import adopted: the stacks, and each ticked gateway's outcome.
type ImportResult struct {
	Stacks   []*Stack          `json:"stacks"`
	Gateways []GatewayAdoption `json:"gateways"`
}

// Import adopts machine-discovery selections as unmanaged stacks (spec §8, issue 08): one stack per compose
// project (slug = project name) and one stack-of-one per standalone container, each filled from freshly
// observed facts. Re-import matches by container name within the machine's stack and updates observed facts;
// it never deletes.
func (s *Service) Import(ctx context.Context, machineID string, req ImportRequest) (*ImportResult, error) {
	if s.machines == nil || s.discoverer == nil {
		return nil, fmt.Errorf("%w: machine import is not configured", apperrs.ErrConflict)
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return nil, fmt.Errorf("%w: project is required", apperrs.ErrInvalid)
	}
	machineName, err := s.machines.MachineName(ctx, machineID)
	if err != nil {
		return nil, fmt.Errorf("resolve machine %s: %w", machineID, err)
	}
	observed, err := s.discoverer.DiscoverContainers(ctx, machineID)
	if err != nil {
		return nil, fmt.Errorf("discover machine %s: %w", machineID, err)
	}
	byName := make(map[string]DiscoveredContainer, len(observed))
	for _, c := range observed {
		byName[c.Name] = c
	}

	stacks := make([]*Stack, 0, len(req.Stacks)+len(req.Standalone))
	for _, group := range req.Stacks {
		stack, err := s.importStack(ctx, req.ProjectID, machineName, group.Project, group.Containers, byName)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, stack)
	}
	for _, name := range req.Standalone {
		stack, err := s.importStack(ctx, req.ProjectID, machineName, name, []string{name}, byName)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, stack)
	}
	gateways := make([]GatewayAdoption, 0, len(req.Gateways))
	for _, name := range req.Gateways {
		stack, err := s.importStack(ctx, req.ProjectID, machineName, name, []string{name}, byName)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, stack)
		gateways = append(gateways, s.adoptGateway(ctx, req.ProjectID, machineName, stack, byName[name]))
	}
	return &ImportResult{Stacks: stacks, Gateways: gateways}, nil
}

// adoptGateway turns a ticked cloudflared container into a dns gateway with one exposure per routed hostname.
// The container itself was already adopted as a stack-of-one so the canvas has a node for the gateway to sit
// on; anything that stops the gateway part lands on the row's Error instead of failing the import.
func (s *Service) adoptGateway(ctx context.Context, projectID, machineName string, stack *Stack, c DiscoveredContainer) GatewayAdoption {
	out := GatewayAdoption{Name: c.Name, Exposed: []string{}, Unmatched: []string{}}
	if c.TunnelID == "" {
		out.Error = "no tunnel token on the container, so there is no tunnel to adopt"
		return out
	}
	if s.adopter == nil {
		out.Error = "gateway adoption is not configured"
		return out
	}
	rows, err := s.services.ListByStack(ctx, stack.ID)
	if err != nil || len(rows) == 0 {
		out.Error = fmt.Sprintf("adopted container row for %s is missing", c.Name)
		return out
	}
	targets, err := s.containerTargets(ctx, projectID, machineName)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	networks := make([]string, 0, len(c.Networks))
	for _, n := range c.Networks {
		networks = append(networks, n.Name)
	}
	adopted, err := s.adopter.AdoptTunnelGateway(ctx, AdoptGatewayInput{
		TunnelID: c.TunnelID, Machine: machineName, ServiceID: rows[0].ID, ServiceName: c.Name,
		Networks: networks, Targets: targets,
	})
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.GatewayID = adopted.GatewayID
	if adopted.Exposed != nil {
		out.Exposed = adopted.Exposed
	}
	if adopted.Unmatched != nil {
		out.Unmatched = adopted.Unmatched
	}
	return out
}

// containerTargets maps every container Nexul tracks on the machine within the project, by its runtime
// container name (what a tunnel ingress route names), to its services.id.
func (s *Service) containerTargets(ctx context.Context, projectID, machineName string) (map[string]string, error) {
	stacks, err := s.stacks.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list stacks for project %s: %w", projectID, err)
	}
	targets := map[string]string{}
	for _, st := range stacks {
		if st.Machine != machineName {
			continue
		}
		rows, err := s.services.ListByStack(ctx, st.ID)
		if err != nil {
			return nil, fmt.Errorf("list containers for stack %s: %w", st.Slug, err)
		}
		for _, r := range rows {
			if r.ContainerName != "" {
				targets[r.ContainerName] = r.ID
			}
			targets[r.Name] = r.ID
		}
	}
	return targets, nil
}

// importStack finds-or-creates the unmanaged stack for slugSource on machineName, upserts one container per
// name with its freshly observed facts, then publishes TopicStackCreated carrying the newly adopted rows: the
// topology subscriber anchors one node per container from that payload, so an event without them draws nothing.
func (s *Service) importStack(ctx context.Context, projectID, machineName, slugSource string, names []string, byName map[string]DiscoveredContainer) (*Stack, error) {
	slug := Slug(slugSource, dnsLabelMaxLen)
	stack, err := s.stacks.GetBySlugAndMachine(ctx, slug, machineName)
	if err != nil {
		if !errors.Is(err, apperrs.ErrNotFound) {
			return nil, fmt.Errorf("get stack %s on %s: %w", slug, machineName, err)
		}
		stack = nil
	}
	now := s.now().UTC()
	if stack == nil {
		stack = &Stack{
			ID: ids.New(), ProjectID: projectID, Name: slugSource, Slug: slug, Machine: machineName,
			Strategy: StrategyCompose, ComposePath: defaultComposePath, Managed: false,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.stacks.Create(ctx, stack); err != nil {
			return nil, fmt.Errorf("create unmanaged stack %s: %w", slug, err)
		}
	}
	known, err := s.services.ListByStack(ctx, stack.ID)
	if err != nil {
		return nil, fmt.Errorf("list containers for stack %s: %w", slug, err)
	}
	tracked := make(map[string]bool, len(known))
	for _, c := range known {
		tracked[c.Name] = true
	}
	for _, name := range names {
		if err := s.importContainer(ctx, stack.ID, slug, name, byName[name]); err != nil {
			return nil, err
		}
	}
	// Read the rows back rather than reusing the ids built above: Upsert keeps an existing row's id on re-import.
	adopted, err := s.services.ListByStack(ctx, stack.ID)
	if err != nil {
		return nil, fmt.Errorf("list containers for stack %s: %w", slug, err)
	}
	var newly []Container
	for _, c := range adopted {
		if !tracked[c.Name] {
			newly = append(newly, *c)
		}
	}
	if len(newly) == 0 {
		return stack, nil
	}
	stack.UpdatedAt = now
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicStackCreated, Payload: StackEvent{Stack: *stack, Services: newly}}
	if err := s.stacks.Update(ctx, stack, evt); err != nil {
		return nil, fmt.Errorf("publish adopted containers for stack %s: %w", slug, err)
	}
	return stack, nil
}

// importContainer upserts one observed container by (stack, name); an unmatched name (discovery ran again and
// no longer sees it) still gets a row so the wizard's ticked selection is honored, just with facts left blank.
func (s *Service) importContainer(ctx context.Context, stackID, slug, name string, c DiscoveredContainer) error {
	svc := &Container{
		ID: ids.New(), StackID: stackID, Name: name, ContainerName: name,
		Image: c.Image, Status: mapObservedStatus(c.Status), Ports: c.Ports, ObservedAt: s.now().UTC(),
	}
	for _, n := range c.Networks {
		svc.Networks = append(svc.Networks, Network(n))
	}
	if err := s.services.Upsert(ctx, svc); err != nil {
		return fmt.Errorf("upsert container %s for stack %s: %w", name, slug, err)
	}
	return nil
}

// mapObservedStatus folds docker's container states onto the domain's ServiceStatus enum (spec §8: "status
// from the report").
func mapObservedStatus(dockerStatus string) ServiceStatus {
	switch dockerStatus {
	case "running":
		return ServiceStatusRunning
	case "exited", "dead":
		return ServiceStatusExited
	default:
		return ServiceStatusStopped
	}
}
