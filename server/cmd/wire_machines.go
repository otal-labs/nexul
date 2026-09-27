package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/runner"
)

// deployGatewayAdopterAdapter adapts dns's AdoptTunnelGateway to deploy's GatewayAdopter seam (ADR 0017).
type deployGatewayAdopterAdapter struct {
	dns *dns.Service
}

func (a deployGatewayAdopterAdapter) AdoptTunnelGateway(ctx context.Context, in deploy.AdoptGatewayInput) (*deploy.AdoptedGateway, error) {
	adopted, err := a.dns.AdoptTunnelGateway(ctx, dns.AdoptTunnelGatewayInput{
		TunnelID: in.TunnelID, Machine: in.Machine, ServiceID: in.ServiceID, ServiceName: in.ServiceName,
		Networks: in.Networks, Targets: in.Targets,
	})
	if err != nil {
		return nil, err
	}
	return &deploy.AdoptedGateway{GatewayID: adopted.Gateway.ID, Exposed: adopted.Exposed, Unmatched: adopted.Unmatched}, nil
}

// runnerTunnelDescriberAdapter adapts dns's DescribeTunnel to runner's TunnelDescriber seam (ADR 0017: runner never
// imports dns), so the import door can show what a discovered cloudflared container already serves.
type runnerTunnelDescriberAdapter struct {
	dns *dns.Service
}

func (a runnerTunnelDescriberAdapter) DescribeTunnel(ctx context.Context, tunnelID string) (*runner.TunnelInfo, error) {
	info, err := a.dns.DescribeTunnel(ctx, tunnelID)
	if err != nil {
		return nil, err
	}
	out := &runner.TunnelInfo{ID: info.ID, Name: info.Name, Status: info.Status, Tracked: info.Tracked, Routes: []runner.TunnelRoute{}, Records: []runner.TunnelRecord{}}
	for _, r := range info.Routes {
		out.Routes = append(out.Routes, runner.TunnelRoute{Hostname: r.Hostname, Service: r.Service})
	}
	for _, r := range info.Records {
		out.Records = append(out.Records, runner.TunnelRecord{Name: r.Name, Content: r.Content})
	}
	return out, nil
}

// deployMachineLookupAdapter adapts runner's machines to deploy's MachineLookup seam (ADR 0017: deploy never imports runner).
type deployMachineLookupAdapter struct {
	runner *runner.Service
}

func (a deployMachineLookupAdapter) MachineName(ctx context.Context, machineID string) (string, error) {
	m, err := a.runner.GetMachine(ctx, machineID)
	if err != nil {
		return "", err
	}
	return m.Name, nil
}

// deployMachineDiscovererAdapter adapts runner's discovery to deploy's MachineDiscoverer seam (ADR 0017); Import
// uses this instead of trusting client-supplied facts about a machine's containers.
type deployMachineDiscovererAdapter struct {
	runner *runner.Service
}

func (a deployMachineDiscovererAdapter) DiscoverContainers(ctx context.Context, machineID string) ([]deploy.DiscoveredContainer, error) {
	report, err := a.runner.Discover(ctx, machineID)
	if err != nil {
		return nil, err
	}
	out := make([]deploy.DiscoveredContainer, 0, len(report.Containers))
	for _, c := range report.Containers {
		dc := deploy.DiscoveredContainer{Name: c.Name, Image: c.Image, Status: c.Status, Ports: c.Ports, TunnelID: c.TunnelID}
		for _, n := range c.Networks {
			dc.Networks = append(dc.Networks, deploy.ImportNetwork{Name: n.Name, Address: n.Address})
		}
		out = append(out, dc)
	}
	return out, nil
}

// runnerHostKind adapts the runner use-cases to the shared host_create and host_delete MCP tools.
type runnerHostKind struct {
	svc *runner.Service
}

func (k runnerHostKind) Enroll(ctx context.Context, name, machine string) (any, error) {
	return k.svc.CreateEnrollment(ctx, name, machine)
}

func (k runnerHostKind) Remove(ctx context.Context, id string) error {
	return k.svc.RemoveRunner(ctx, id)
}

// automationsHostKind adapts the automations host use-cases to the shared host_create and host_delete MCP tools.
type automationsHostKind struct {
	svc *automations.HostsService
}

func (k automationsHostKind) Enroll(ctx context.Context, name, machine string) (any, error) {
	return k.svc.CreateEnrollment(ctx, name, machine)
}

func (k automationsHostKind) Remove(ctx context.Context, id string) error {
	return k.svc.Remove(ctx, id)
}

// runnerAutomationsHostsAdapter lets machine_list file automations hosts under their machines (ADR 0017: runner
// never imports automations).
type runnerAutomationsHostsAdapter struct {
	svc *automations.HostsService
}

func (a runnerAutomationsHostsAdapter) ListAutomationsHosts(ctx context.Context) ([]runner.AutomationsHost, error) {
	hosts, err := a.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]runner.AutomationsHost, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, runner.AutomationsHost{ID: h.ID, Name: h.Name, Machine: h.Machine, Version: h.Version, Connected: h.Connected, LastSeen: h.LastSeen})
	}
	return out, nil
}
