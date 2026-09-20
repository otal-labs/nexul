package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/runner"
)

// dnsSettingsAdapter adapts the auth settings store to dns's SettingsReader seam (ADR 0017).
type dnsSettingsAdapter struct {
	repo *storage.SettingsRepo
}

func (a dnsSettingsAdapter) GetInstanceURL(ctx context.Context) (string, error) {
	st, err := a.repo.Get(ctx)
	if err != nil {
		return "", err
	}
	return st.InstanceURL, nil
}

// dnsCloudflareTokenAdapter adapts connectors' AccessToken to dns's TokenProvider seam (dns never imports connectors).
type dnsCloudflareTokenAdapter struct {
	connectors gitTokenResolver
}

func (a dnsCloudflareTokenAdapter) Token(ctx context.Context) (string, error) {
	return a.connectors.AccessToken(ctx, "cloudflare")
}

// dnsProvisioner adapts deploy to dns's ServiceProvisioner seam: provisioning an agent creates and deploys a stack.
type dnsProvisioner struct {
	deploy *deploy.Service
}

func (a dnsProvisioner) Provision(ctx context.Context, in dns.AgentSpec) (*dns.AgentProvisioned, error) {
	// A retried entry-path step reuses the stack it already created and redeploys it; an already-running deploy is fine.
	if existing, err := a.deploy.GetServiceByName(ctx, in.Name); err == nil && existing.Machine == in.Target {
		if in.Image != "" {
			if _, err := a.deploy.Deploy(ctx, deploy.DeployRequest{StackID: existing.ID, Image: in.Image}); err != nil && !errors.Is(err, apperrs.ErrConflict) {
				return nil, fmt.Errorf("redeploy entry-path stack: %w", err)
			}
		}
		return &dns.AgentProvisioned{ServiceID: existing.ID}, nil
	}
	stack, err := a.deploy.CreateStack(ctx, deploy.Stack{
		ProjectID:     in.ProjectID,
		Name:          in.Name,
		Machine:       in.Target,
		Strategy:      deploy.Strategy(in.Strategy),
		ComposePath:   in.ComposeDir,
		DockerNetwork: in.DockerNetwork,
		Ports:         in.Ports,
		Mounts:        in.Mounts,
		Command:       in.Command,
		Env:           in.Env,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("create entry-path stack: %w", err)
	}
	if in.Image != "" {
		if _, err := a.deploy.Deploy(ctx, deploy.DeployRequest{StackID: stack.ID, Image: in.Image}); err != nil {
			return nil, fmt.Errorf("deploy entry-path stack: %w", err)
		}
	}
	return &dns.AgentProvisioned{ServiceID: stack.ID}, nil
}

// Deprovision drops only the stack; deleting an already-absent one is a no-op success.
func (a dnsProvisioner) Deprovision(ctx context.Context, stackID string) error {
	if err := a.deploy.DeleteStack(ctx, stackID); err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("deprovision entry-path stack %s: %w", stackID, err)
	}
	return nil
}

// dnsContainerLookupAdapter adapts deploy to dns's ContainerLookup seam (ADR 0017): dns never imports deploy directly.
type dnsContainerLookupAdapter struct {
	deploy *deploy.Service
}

// ContainerByID resolves an exposure's service_id directly to the container it targets.
func (a dnsContainerLookupAdapter) ContainerByID(ctx context.Context, containerID string) (*dns.ExposureTarget, error) {
	container, err := a.deploy.GetService(ctx, containerID)
	if err != nil {
		return nil, err
	}
	stack, err := a.deploy.GetStack(ctx, container.StackID)
	if err != nil {
		return nil, err
	}
	return exposureTargetFor(stack, container), nil
}

// ContainerByStackName resolves the legacy "service" name — a stack's name — to its single container.
func (a dnsContainerLookupAdapter) ContainerByStackName(ctx context.Context, stackName string) (*dns.ExposureTarget, error) {
	stack, err := a.deploy.GetServiceByName(ctx, stackName)
	if err != nil {
		return nil, err
	}
	return a.singleContainer(ctx, stack)
}

// ContainerByStackID resolves a stack id to its single container, for a gateway's own backing stack.
func (a dnsContainerLookupAdapter) ContainerByStackID(ctx context.Context, stackID string) (*dns.ExposureTarget, error) {
	stack, err := a.deploy.GetStack(ctx, stackID)
	if err != nil {
		return nil, err
	}
	return a.singleContainer(ctx, stack)
}

// singleContainer resolves a stack's one container: the run strategy every gateway and the legacy Service
// name fallback both assume, since neither names a specific compose service.
func (a dnsContainerLookupAdapter) singleContainer(ctx context.Context, stack *deploy.Stack) (*dns.ExposureTarget, error) {
	containers, err := a.deploy.ListServices(ctx, stack.ID)
	if err != nil {
		return nil, err
	}
	if len(containers) != 1 {
		return nil, fmt.Errorf("%w: stack %q has %d containers, not exactly one — target it by service_id instead",
			apperrs.ErrInvalid, stack.Name, len(containers))
	}
	return exposureTargetFor(stack, containers[0]), nil
}

// exposureTargetFor adapts a deploy stack+container to dns's ExposureTarget DTO (ADR 0017). Networks falls back to
// the stack's own default network when the container hasn't reported in yet, so a not-yet-deployed stack can
// still be exposed/provisioned for.
func exposureTargetFor(stack *deploy.Stack, container *deploy.Container) *dns.ExposureTarget {
	name := container.ContainerName
	if name == "" {
		name = stack.Slug
	}
	networks := containerNetworkNames(container)
	if len(networks) == 0 {
		if def := stack.DefaultNetwork(); def != "" {
			networks = []string{def}
		}
	}
	running := container.Status == deploy.ServiceStatusRunning || container.Status == deploy.ServiceStatusHealthy
	return &dns.ExposureTarget{
		ContainerID: container.ID, Name: name, StackID: stack.ID, ProjectID: stack.ProjectID,
		Machine: stack.Machine, Networks: networks, Running: running,
	}
}

func containerNetworkNames(container *deploy.Container) []string {
	out := make([]string, 0, len(container.Networks))
	for _, n := range container.Networks {
		out = append(out, n.Name)
	}
	return out
}

// dnsRunnerJoinAdapter adapts the runner WS handler to dns's RunnerJoiner seam (ADR 0017).
type dnsRunnerJoinAdapter struct {
	handler *runner.Handler
}

func (a dnsRunnerJoinAdapter) JoinNetworks(ctx context.Context, machine, gatewayContainer string, networks []string) error {
	return a.handler.JoinNetworks(ctx, machine, gatewayContainer, networks)
}
