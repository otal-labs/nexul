package runner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/coder/websocket"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// discoverTimeout bounds how long the server waits for a discover_result before giving up (spec §8).
const discoverTimeout = 30 * time.Second

// composeProjectLabel/composeServiceLabel are the two compose labels discovery captures (spec §3).
const (
	composeProjectLabel = "com.docker.compose.project"
	composeServiceLabel = "com.docker.compose.service"
)

// builtinNetworks are docker's default networks, never import candidates.
var builtinNetworks = map[string]bool{"bridge": true, "host": true, "none": true}

// gatewayImagePrefixes are the images discovery recognises as gateways (spec §8); adoption is a later ticket.
var gatewayImagePrefixes = []string{"cloudflare/cloudflared", "traefik"}

// ContainerNetwork is one docker network a discovered container is joined to, with its address on it.
type ContainerNetwork struct {
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
}

// DiscoveredContainer is one container found on a machine during discovery (spec §3).
type DiscoveredContainer struct {
	// ID is the container's docker id; not carried on the wire, only used to exclude the runner's own container.
	ID       string             `json:"-"`
	Name     string             `json:"name"`
	Image    string             `json:"image"`
	Status   string             `json:"status"`
	Labels   map[string]string  `json:"labels,omitempty"`
	Networks []ContainerNetwork `json:"networks,omitempty"`
	Ports    []string           `json:"ports,omitempty"`
	// TunnelID is the Cloudflare tunnel a cloudflared container connects, read off its TUNNEL_TOKEN; the token's
	// secret never leaves the machine. Tunnel/TunnelError are filled server-side by DiscoverForImport.
	TunnelID    string      `json:"tunnel_id,omitempty"`
	Tunnel      *TunnelInfo `json:"tunnel,omitempty"`
	TunnelError string      `json:"tunnel_error,omitempty"`
}

// TunnelInfo mirrors dns's tunnel description so runner never imports dns (ADR 0017): what the provider knows about
// a discovered tunnel, so the import door can show which hostnames already reach this machine through it.
type TunnelInfo struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Tracked bool           `json:"tracked"`
	Routes  []TunnelRoute  `json:"routes"`
	Records []TunnelRecord `json:"records"`
}

// TunnelRoute is one ingress rule: a public hostname and the local service it is sent to.
type TunnelRoute struct {
	Hostname string `json:"hostname"`
	Service  string `json:"service"`
}

// TunnelRecord is one DNS record at the provider that points a name at this tunnel.
type TunnelRecord struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// TunnelDescriber is the dns-domain seam DiscoverForImport uses to describe a discovered tunnel (ADR 0017).
type TunnelDescriber interface {
	DescribeTunnel(ctx context.Context, tunnelID string) (*TunnelInfo, error)
}

// NetworkInfo is one non-builtin docker network on the machine (docker network ls).
type NetworkInfo struct {
	Name string `json:"name"`
}

// DiscoverReport is what a runner reports back from a discover job (spec §3).
type DiscoverReport struct {
	Containers []DiscoveredContainer `json:"containers,omitempty"`
	Networks   []NetworkInfo         `json:"networks,omitempty"`
}

// StackGroup is one compose project's containers within a grouped discovery response (spec §8).
type StackGroup struct {
	Project    string                `json:"project"`
	Containers []DiscoveredContainer `json:"containers"`
}

// GroupedDiscovery is the /discover response shape the import wizard renders (spec §8): stacks by compose
// project, standalone containers, recognised gateways, and the machine's non-builtin networks.
type GroupedDiscovery struct {
	Stacks     []StackGroup          `json:"stacks"`
	Standalone []DiscoveredContainer `json:"standalone"`
	Gateways   []DiscoveredContainer `json:"gateways"`
	Networks   []NetworkInfo         `json:"networks"`
}

// GroupDiscovery groups a raw report by compose project, then standalone, then recognised gateways (spec §8).
func GroupDiscovery(report DiscoverReport) GroupedDiscovery {
	groups := map[string]*StackGroup{}
	var order []string
	var standalone, gateways []DiscoveredContainer
	for _, c := range report.Containers {
		if isGatewayImage(c.Image) {
			gateways = append(gateways, c)
			continue
		}
		project := c.Labels[composeProjectLabel]
		if project == "" {
			standalone = append(standalone, c)
			continue
		}
		g, ok := groups[project]
		if !ok {
			g = &StackGroup{Project: project}
			groups[project] = g
			order = append(order, project)
		}
		g.Containers = append(g.Containers, c)
	}
	sort.Strings(order)
	stacks := make([]StackGroup, 0, len(order))
	for _, p := range order {
		stacks = append(stacks, *groups[p])
	}
	return GroupedDiscovery{Stacks: stacks, Standalone: standalone, Gateways: gateways, Networks: report.Networks}
}

func isGatewayImage(image string) bool {
	for _, p := range gatewayImagePrefixes {
		if strings.HasPrefix(image, p) {
			return true
		}
	}
	return false
}

// ---- runner-side: gathering the report ----

// discoverHost walks docker ps -a, inspects each container, and lists non-builtin networks (spec §3). selfID
// is the calling runner's own container id (os.Hostname() inside a container, empty on a bare-metal install);
// it, and any container sharing its compose project (the self-hosted Nexul stack), are excluded.
// ponytail: "Nexul-managed" is approximated as "shares the runner's own compose project" rather than a
// dedicated label — the only self-hosted case today is the shipped docker-compose.yml; add a real label if a
// managed container ever needs excluding without also being the runner's own project-mate.
func discoverHost(ctx context.Context, cmd CommandRunner, selfID string) (DiscoverReport, error) {
	ids, err := listContainerIDs(ctx, cmd)
	if err != nil {
		return DiscoverReport{}, fmt.Errorf("docker ps: %w", err)
	}
	containers := make([]DiscoveredContainer, 0, len(ids))
	for _, id := range ids {
		c, err := inspectContainer(ctx, cmd, id)
		if err != nil {
			continue // a container that vanished between ps and inspect is simply skipped, not a scan failure
		}
		containers = append(containers, c)
	}
	containers = excludeManaged(containers, selfID)
	networks, err := listNetworks(ctx, cmd)
	if err != nil {
		return DiscoverReport{}, fmt.Errorf("docker network ls: %w", err)
	}
	return DiscoverReport{Containers: containers, Networks: networks}, nil
}

func cmdOutput(ctx context.Context, cmd CommandRunner, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	if err := cmd(ctx, "", func(s string) { buf.WriteString(s) }, name, args...); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type psEntry struct {
	ID string `json:"ID"`
}

func listContainerIDs(ctx context.Context, cmd CommandRunner) ([]string, error) {
	out, err := cmdOutput(ctx, cmd, "docker", "ps", "-a", "--format", "json")
	if err != nil {
		return nil, err
	}
	var out2 []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e psEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.ID != "" {
			out2 = append(out2, e.ID)
		}
	}
	return out2, nil
}

type inspectOutput struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
		Env    []string          `json:"Env"`
	} `json:"Config"`
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

func inspectContainer(ctx context.Context, cmd CommandRunner, id string) (DiscoveredContainer, error) {
	out, err := cmdOutput(ctx, cmd, "docker", "inspect", "--format", "{{json .}}", id)
	if err != nil {
		return DiscoveredContainer{}, err
	}
	var raw inspectOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &raw); err != nil {
		return DiscoveredContainer{}, fmt.Errorf("parse inspect output for %s: %w", id, err)
	}
	c := DiscoveredContainer{
		ID:     raw.ID,
		Name:   strings.TrimPrefix(raw.Name, "/"),
		Image:  raw.Config.Image,
		Status: raw.State.Status,
	}
	if isGatewayImage(c.Image) {
		c.TunnelID = tunnelIDFromEnv(raw.Config.Env)
	}
	labels := map[string]string{}
	if v, ok := raw.Config.Labels[composeProjectLabel]; ok {
		labels[composeProjectLabel] = v
	}
	if v, ok := raw.Config.Labels[composeServiceLabel]; ok {
		labels[composeServiceLabel] = v
	}
	if len(labels) > 0 {
		c.Labels = labels
	}
	for name, n := range raw.NetworkSettings.Networks {
		c.Networks = append(c.Networks, ContainerNetwork{Name: name, Address: n.IPAddress})
	}
	sort.Slice(c.Networks, func(i, j int) bool { return c.Networks[i].Name < c.Networks[j].Name })
	for portProto, bindings := range raw.NetworkSettings.Ports {
		for _, b := range bindings {
			if b.HostPort == "" {
				continue
			}
			c.Ports = append(c.Ports, b.HostPort+":"+portProto)
		}
	}
	sort.Strings(c.Ports)
	return c, nil
}

// tunnelIDFromEnv decodes a cloudflared TUNNEL_TOKEN (base64 JSON {a: account, t: tunnel, s: secret}) down to
// its tunnel id; only the id is carried, the secret is dropped here on the machine.
func tunnelIDFromEnv(env []string) string {
	for _, kv := range env {
		token, ok := strings.CutPrefix(kv, "TUNNEL_TOKEN=")
		if !ok || token == "" {
			continue
		}
		raw, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(token, "="))
		if err != nil {
			raw, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(token, "="))
		}
		if err != nil {
			return ""
		}
		var claims struct {
			TunnelID string `json:"t"`
		}
		if json.Unmarshal(raw, &claims) != nil {
			return ""
		}
		return claims.TunnelID
	}
	return ""
}

type networkEntry struct {
	Name string `json:"Name"`
}

func listNetworks(ctx context.Context, cmd CommandRunner) ([]NetworkInfo, error) {
	out, err := cmdOutput(ctx, cmd, "docker", "network", "ls", "--format", "json")
	if err != nil {
		return nil, err
	}
	var out2 []NetworkInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e networkEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if builtinNetworks[e.Name] {
			continue
		}
		out2 = append(out2, NetworkInfo(e))
	}
	return out2, nil
}

// excludeManaged drops the runner's own container and, if it is compose-managed, every sibling in its compose
// project (the self-hosted Nexul stack) — spec §3/§8's "excluded by label and by name".
func excludeManaged(containers []DiscoveredContainer, selfID string) []DiscoveredContainer {
	if selfID == "" {
		return containers
	}
	var selfProject string
	found := false
	for _, c := range containers {
		if matchesSelf(c.ID, selfID) {
			selfProject = c.Labels[composeProjectLabel]
			found = true
			break
		}
	}
	if !found {
		return containers
	}
	out := make([]DiscoveredContainer, 0, len(containers))
	for _, c := range containers {
		if matchesSelf(c.ID, selfID) {
			continue
		}
		if selfProject != "" && c.Labels[composeProjectLabel] == selfProject {
			continue
		}
		out = append(out, c)
	}
	return out
}

// matchesSelf compares a container's full docker id against selfID (os.Hostname(), usually the short id).
func matchesSelf(containerID, selfID string) bool {
	if containerID == "" || selfID == "" {
		return false
	}
	return containerID == selfID || strings.HasPrefix(containerID, selfID) || strings.HasPrefix(selfID, containerID)
}

// ---- server-side: dispatching a discover job and waiting for its result ----

// Discover asks one idle, connected runner on machine to scan its host (spec §8), blocking up to timeout for
// the discover_result frame. Unlike a deploy, the runner's job slot is not occupied while it runs: a scan is a
// quick, side-effect-free read that can safely interleave with a concurrently dispatched deploy.
func (h *Handler) Discover(ctx context.Context, machine string, timeout time.Duration) (DiscoverReport, error) {
	c := h.idleConnected(DeployRequestedEvent{Target: machine})
	if c == nil {
		return DiscoverReport{}, fmt.Errorf("%w: no idle runner connected on machine %q", apperrs.ErrConflict, machine)
	}
	id := ids.New()
	ch := make(chan Frame, 1)
	h.discoverMu.Lock()
	h.discoverWaiters[id] = ch
	h.discoverMu.Unlock()
	defer func() {
		h.discoverMu.Lock()
		delete(h.discoverWaiters, id)
		h.discoverMu.Unlock()
	}()
	if err := h.sendFrame(ctx, c, Frame{Type: FrameDiscover, ID: id}); err != nil {
		return DiscoverReport{}, fmt.Errorf("send discover frame: %w", err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case f := <-ch:
		return DiscoverReport{Containers: f.Containers, Networks: f.Networks}, nil
	case <-waitCtx.Done():
		return DiscoverReport{}, fmt.Errorf("%w: discover timed out waiting for runner on machine %q", apperrs.ErrRetryable, machine)
	}
}

// deliverDiscoverResult routes an inbound discover_result frame to the goroutine waiting on it, if any.
func (h *Handler) deliverDiscoverResult(f Frame) {
	h.discoverMu.Lock()
	ch, ok := h.discoverWaiters[f.ID]
	h.discoverMu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- f:
	default:
	}
}

// ---- client-side: answering a discover frame ----

// handleDiscover runs the assigned discover job and reports its result; it never occupies the client's job
// slot (see Handler.Discover), so it can run alongside a concurrently assigned build or deploy.
func (c *Client) handleDiscover(ctx context.Context, conn *websocket.Conn, f Frame) {
	report, err := c.cfg.Executor.Discover(ctx)
	if err != nil {
		c.log.Warn("discover failed", "runner_id", c.cfg.RunnerID, "id", f.ID, "error", err)
		c.sendFrame(ctx, conn, Frame{Type: FrameDiscoverResult, ID: f.ID, Error: err.Error()})
		return
	}
	c.sendFrame(ctx, conn, Frame{Type: FrameDiscoverResult, ID: f.ID, Containers: report.Containers, Networks: report.Networks})
}
