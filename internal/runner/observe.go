package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// containerRef names one container a stack started, before it's inspected: the compose service label (or the
// slug for a run stack) and the container name docker actually gave it.
type containerRef struct {
	name          string
	containerName string
}

// composePSEntry is one line of `docker compose ps -a --format json` output (one JSON object per container).
type composePSEntry struct {
	Name    string `json:"Name"`
	Service string `json:"Service"`
}

// inspectResult is the subset of `docker inspect --format '{{json .}}'` the observation report reads.
type inspectResult struct {
	Config struct {
		Image string `json:"Image"`
	} `json:"Config"`
	State struct {
		Status string `json:"Status"`
		Health *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
		Ports map[string][]struct {
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

// observe builds the deploy_result services report (spec §4 step 3): every container the stack's strategy
// started, each inspected for its live status, networks and published ports. An inspect failure is logged and
// that container is skipped — the deploy already succeeded, a thin report beats failing a healthy deploy.
func (e *ShellExecutor) observe(ctx context.Context, req DeployRequestedEvent) []ObservedService {
	refs := e.containerRefs(ctx, req)
	services := make([]ObservedService, 0, len(refs))
	for _, ref := range refs {
		svc, err := e.inspectService(ctx, ref)
		if err != nil {
			e.log.Warn("observation inspect failed", "container", ref.containerName, "error", err)
			continue
		}
		services = append(services, svc)
	}
	return services
}

// containerRefs lists the containers a strategy started: one compose ps row per service, or the run stack's
// single container named after its slug. A compose ps failure is logged and yields an empty (not fatal) report.
func (e *ShellExecutor) containerRefs(ctx context.Context, req DeployRequestedEvent) []containerRef {
	if req.Strategy != "compose" {
		return []containerRef{{name: req.StackSlug, containerName: req.StackSlug}}
	}
	out, err := e.output(ctx, "docker", "compose", "-p", req.StackSlug, "ps", "-a", "--format", "json")
	if err != nil {
		e.log.Warn("compose ps failed", "stack", req.StackSlug, "error", err)
		return nil
	}
	var refs []containerRef
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry composePSEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			e.log.Warn("compose ps line unparsable", "line", line, "error", err)
			continue
		}
		refs = append(refs, containerRef{name: entry.Service, containerName: entry.Name})
	}
	return refs
}

// inspectService runs `docker inspect` on one container and builds its report entry.
func (e *ShellExecutor) inspectService(ctx context.Context, ref containerRef) (ObservedService, error) {
	out, err := e.output(ctx, "docker", "inspect", "--format", "{{json .}}", ref.containerName)
	if err != nil {
		return ObservedService{}, err
	}
	var res inspectResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return ObservedService{}, fmt.Errorf("parse inspect output: %w", err)
	}
	return ObservedService{
		Name:          ref.name,
		ContainerName: ref.containerName,
		Image:         res.Config.Image,
		Status:        containerStatus(res),
		Networks:      containerNetworks(res),
		Ports:         containerPorts(res),
	}, nil
}

// containerStatus maps docker's raw state into the report vocabulary (running|healthy|exited); a health check
// promotes "running" to "healthy", anything else docker reports (paused, restarting, dead...) falls back to "exited".
func containerStatus(res inspectResult) string {
	if res.State.Health != nil && res.State.Health.Status == "healthy" {
		return "healthy"
	}
	if res.State.Status == "running" {
		return "running"
	}
	return "exited"
}

// containerNetworks lists a container's joined networks, sorted by name for a deterministic report.
func containerNetworks(res inspectResult) []ObservedNetwork {
	names := make([]string, 0, len(res.NetworkSettings.Networks))
	for name := range res.NetworkSettings.Networks {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ObservedNetwork, 0, len(names))
	for _, name := range names {
		out = append(out, ObservedNetwork{Name: name, Address: res.NetworkSettings.Networks[name].IPAddress})
	}
	return out
}

// containerPorts lists published bindings as "host:container/proto", sorted by container port for determinism.
func containerPorts(res inspectResult) []string {
	keys := make([]string, 0, len(res.NetworkSettings.Ports))
	for key := range res.NetworkSettings.Ports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out []string
	seen := map[string]bool{}
	for _, key := range keys {
		proto := "tcp"
		containerPort := key
		if idx := strings.IndexByte(key, '/'); idx >= 0 {
			containerPort, proto = key[:idx], key[idx+1:]
		}
		for _, binding := range res.NetworkSettings.Ports[key] {
			if binding.HostPort == "" {
				continue
			}
			// Docker reports one binding per host address (0.0.0.0 and ::), which is the same port twice here.
			entry := fmt.Sprintf("%s:%s/%s", binding.HostPort, containerPort, proto)
			if seen[entry] {
				continue
			}
			seen[entry] = true
			out = append(out, entry)
		}
	}
	return out
}
