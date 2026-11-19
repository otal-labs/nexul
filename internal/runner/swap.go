package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// A swapped service's new container gets healthyTimeout to pass its healthcheck, looked at every healthyPoll; cleanup
// after a failed or cancelled swap gets swapCleanupTimeout of its own, since the job's context may already be done.
const (
	healthyTimeout     = 5 * time.Minute
	healthyPoll        = time.Second
	swapCleanupTimeout = 30 * time.Second
	healthLogEntries   = 3
)

// Why a service restarts in place instead of swapping, written to the deploy log.
const (
	noSwapNoRoute       = "no gateway routes to it"
	noSwapHostPort      = "it publishes a host port"
	noSwapNoHealthcheck = "it has no healthcheck"
	noSwapNotRunning    = "nothing is running yet"
	noSwapReplicas      = "it runs more than one container"
	noSwapFixedName     = "it sets container_name"
)

// containerHealth is the part of `docker inspect --format '{{json .State}}'` a swap waits on.
type containerHealth struct {
	Status string `json:"Status"`
	Health *struct {
		Status string `json:"Status"`
		Log    []struct {
			ExitCode int    `json:"ExitCode"`
			Output   string `json:"Output"`
		} `json:"Log"`
	} `json:"Health"`
}

// healthcheckConfig is a Healthcheck as image inspect and compose config print it.
type healthcheckConfig struct {
	Test    []string `json:"Test"`
	Disable bool     `json:"disable"`
}

func (h *healthcheckConfig) present() bool {
	if h == nil || h.Disable {
		return false
	}
	return len(h.Test) > 0 && h.Test[0] != "NONE"
}

// runContainer deploys a run stack: a swap when its container can run twice, else a fresh `docker run` in its place.
func (e *ShellExecutor) runContainer(ctx context.Context, req DeployRequestedEvent, logs *logStream) error {
	reason := e.runSwapBlocker(ctx, req)
	if reason == "" {
		return e.swapRun(ctx, req, logs)
	}
	if reason != noSwapNoRoute {
		logs.step(LogPhaseDeploy, req.StackSlug+" restarts in place: "+reason)
	}
	return e.replaceContainer(ctx, req, logs)
}

// runSwapBlocker says why a run stack's container cannot be swapped, or "" when it can.
func (e *ShellExecutor) runSwapBlocker(ctx context.Context, req DeployRequestedEvent) string {
	if !slices.Contains(req.SwapServices, req.StackSlug) {
		return noSwapNoRoute
	}
	if len(req.Ports) > 0 {
		return noSwapHostPort
	}
	if status, err := e.output(ctx, "docker", "inspect", "--format", "{{.State.Status}}", req.StackSlug); err != nil || status != "running" {
		return noSwapNotRunning
	}
	if !e.imageHasHealthcheck(ctx, req.Image) {
		return noSwapNoHealthcheck
	}
	return ""
}

func (e *ShellExecutor) imageHasHealthcheck(ctx context.Context, image string) bool {
	out, err := e.output(ctx, "docker", "image", "inspect", "--format", "{{json .Config.Healthcheck}}", image)
	if err != nil {
		return false
	}
	var h *healthcheckConfig
	if err := json.Unmarshal([]byte(out), &h); err != nil {
		return false
	}
	return h.present()
}

// swapRun starts the new container beside the old one under a second name, then hands it to swapIn.
func (e *ShellExecutor) swapRun(ctx context.Context, req DeployRequestedEvent, logs *logStream) error {
	old := req.StackSlug
	next := old + "-next"
	// Leftovers of a swap the runner died in the middle of; a missing container is the normal answer.
	_ = e.cmd(ctx, "", discardLog, "docker", "rm", "-f", next, old+"-old")
	e.ensureNetwork(ctx, req.Network)
	label := "start new: docker run --name " + next + " " + req.Image
	if err := e.runStreamed(ctx, "", logs, label, "docker", runArgsNamed(req, next)...); err != nil {
		e.removeContainer(ctx, next)
		return err
	}
	return e.swapIn(ctx, logs, next, old)
}

// swapIn waits for next to turn healthy, gives it old's name, then stops old. Until the rename, old keeps serving, and a
// next that never turns healthy is removed with old untouched.
func (e *ShellExecutor) swapIn(ctx context.Context, logs *logStream, next, old string) error {
	defer logs.flush()
	if err := e.waitHealthy(ctx, logs, next); err != nil {
		e.removeContainer(ctx, next)
		return err
	}
	retired := old + "-old"
	stream := logs.step(LogPhaseDeploy, "swap: "+next+" takes the name "+old)
	if err := e.cmd(ctx, "", stream, "docker", "rename", old, retired); err != nil {
		e.removeContainer(ctx, next)
		return fmt.Errorf("swap %s: %w", old, err)
	}
	if err := e.cmd(ctx, "", stream, "docker", "rename", next, old); err != nil {
		// Give the old container its name back, so it goes on serving under it.
		_ = e.cmd(context.WithoutCancel(ctx), "", stream, "docker", "rename", retired, old)
		e.removeContainer(ctx, next)
		return fmt.Errorf("swap %s: %w", old, err)
	}
	// The new container already serves; an old one that will not stop is reported, never a failed deploy.
	stop := logs.step(LogPhaseDeploy, "stop old: "+retired)
	if err := e.cmd(context.WithoutCancel(ctx), "", stop, "docker", "stop", retired); err != nil {
		stop("stop old failed: " + err.Error() + "\n")
	}
	if err := e.cmd(context.WithoutCancel(ctx), "", stop, "docker", "rm", "-f", retired); err != nil {
		stop("remove old failed: " + err.Error() + "\n")
	}
	return nil
}

// waitHealthy polls container until its healthcheck passes; unhealthy, stopped or past healthyTimeout fails with the
// healthcheck's last output.
func (e *ShellExecutor) waitHealthy(ctx context.Context, logs *logStream, container string) error {
	defer logs.flush()
	stream := logs.step(LogPhaseDeploy, "wait healthy: "+container)
	timeout := time.NewTimer(e.healthyTimeout)
	defer timeout.Stop()
	tick := time.NewTicker(e.healthyPoll)
	defer tick.Stop()
	for {
		st, err := e.containerHealth(ctx, container)
		if err != nil {
			return fmt.Errorf("wait for %s to turn healthy: %w", container, err)
		}
		if done, err := healthVerdict(container, st); done {
			if err != nil {
				stream(err.Error() + "\n")
			}
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("cancelled: %w", ctx.Err())
		case <-timeout.C:
			err := fmt.Errorf("new container %s was not healthy after %s; the old one goes on serving%s", container, e.healthyTimeout, healthLog(st))
			stream(err.Error() + "\n")
			return err
		case <-tick.C:
		}
	}
}

// healthVerdict says whether a wait is over: healthy ends it with no error, unhealthy or stopped with one.
func healthVerdict(container string, st containerHealth) (bool, error) {
	if st.Status != "running" {
		return true, fmt.Errorf("new container %s stopped (%s) before it turned healthy; the old one goes on serving%s", container, st.Status, healthLog(st))
	}
	if st.Health == nil {
		return true, fmt.Errorf("new container %s has no healthcheck to wait on; the old one goes on serving", container)
	}
	if st.Health.Status == "healthy" {
		return true, nil
	}
	if st.Health.Status == "unhealthy" {
		return true, fmt.Errorf("new container %s turned unhealthy; the old one goes on serving%s", container, healthLog(st))
	}
	return false, nil
}

// healthLog renders the last few healthcheck results for an error, newest last.
func healthLog(st containerHealth) string {
	if st.Health == nil || len(st.Health.Log) == 0 {
		return ""
	}
	entries := st.Health.Log[max(0, len(st.Health.Log)-healthLogEntries):]
	var b strings.Builder
	b.WriteString("\nhealth log:")
	for _, l := range entries {
		fmt.Fprintf(&b, "\n  exit %d: %s", l.ExitCode, strings.TrimSpace(l.Output))
	}
	return b.String()
}

func (e *ShellExecutor) containerHealth(ctx context.Context, container string) (containerHealth, error) {
	var st containerHealth
	out, err := e.output(ctx, "docker", "inspect", "--format", "{{json .State}}", container)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		return st, fmt.Errorf("parse %s state: %w", container, err)
	}
	return st, nil
}

// removeContainer drops a new container that will not take over, even when the job was cancelled.
func (e *ShellExecutor) removeContainer(ctx context.Context, container string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), swapCleanupTimeout)
	defer cancel()
	if err := e.cmd(ctx, "", discardLog, "docker", "rm", "-f", container); err != nil {
		e.log.Warn("remove new container failed", "container", container, "error", err)
	}
}

// composeService is the part of `docker compose config --format json` a swap decision reads.
type composeService struct {
	Image         string             `json:"image"`
	ContainerName string             `json:"container_name"`
	Healthcheck   *healthcheckConfig `json:"healthcheck"`
	Ports         []composePort      `json:"ports"`
}

// composeUp brings a compose stack up: every service that cannot swap first, as today, then each one that can, one
// at a time, then a last `up` that leaves the swapped containers alone and removes orphans.
func (e *ShellExecutor) composeUp(ctx context.Context, req DeployRequestedEvent, checkout string, logs *logStream) error {
	swaps, rest := e.composeSwaps(ctx, req, checkout, logs)
	if len(swaps) == 0 {
		args := composeArgs(req, "up", "-d", "--remove-orphans")
		return e.runStreamed(ctx, checkout, logs, cmdString("docker", args), "docker", args...)
	}
	if len(rest) > 0 {
		args := composeArgs(req, append([]string{"up", "-d", "--no-deps"}, rest...)...)
		if err := e.runStreamed(ctx, checkout, logs, cmdString("docker", args), "docker", args...); err != nil {
			return err
		}
	}
	for _, s := range swaps {
		if err := e.swapCompose(ctx, req, checkout, logs, s); err != nil {
			return err
		}
	}
	args := composeArgs(req, "up", "-d", "--remove-orphans")
	return e.runStreamed(ctx, checkout, logs, cmdString("docker", args), "docker", args...)
}

// composeSwap is one compose service a deploy swaps, and the container now serving it.
type composeSwap struct {
	service string
	old     string
}

// composeSwaps splits the stack's services into the ones a deploy swaps and the rest, logging why each routed one
// that cannot swap restarts. A config that cannot be read swaps nothing.
func (e *ShellExecutor) composeSwaps(ctx context.Context, req DeployRequestedEvent, checkout string, logs *logStream) ([]composeSwap, []string) {
	if len(req.SwapServices) == 0 {
		return nil, nil
	}
	var cfg struct {
		Services map[string]composeService `json:"services"`
	}
	var out strings.Builder
	args := composeArgs(req, "config", "--format", "json")
	if err := e.rawCmd(ctx, checkout, func(s string) { out.WriteString(s) }, "docker", args...); err != nil {
		e.log.Warn("compose config failed", "stack", req.StackSlug, "error", err)
		return nil, nil
	}
	if err := json.Unmarshal([]byte(out.String()), &cfg); err != nil {
		e.log.Warn("compose config unparsable", "stack", req.StackSlug, "error", err)
		return nil, nil
	}
	var swaps []composeSwap
	var rest []string
	for _, name := range slices.Sorted(maps.Keys(cfg.Services)) {
		if !slices.Contains(req.SwapServices, name) {
			rest = append(rest, name)
			continue
		}
		old, reason := e.composeSwapBlocker(ctx, req, name, cfg.Services[name])
		if reason != "" {
			logs.step(LogPhaseDeploy, name+" restarts in place: "+reason)
			rest = append(rest, name)
			continue
		}
		swaps = append(swaps, composeSwap{service: name, old: old})
	}
	return swaps, rest
}

// composeSwapBlocker says why a compose service cannot be swapped, or returns the one container serving it now.
func (e *ShellExecutor) composeSwapBlocker(ctx context.Context, req DeployRequestedEvent, name string, svc composeService) (string, string) {
	if svc.ContainerName != "" {
		return "", noSwapFixedName
	}
	if slices.ContainsFunc(svc.Ports, func(p composePort) bool { return p.Published != "" }) {
		return "", noSwapHostPort
	}
	running := e.serviceContainers(ctx, req.StackSlug, name)
	if len(running) == 0 {
		return "", noSwapNotRunning
	}
	if len(running) > 1 {
		return "", noSwapReplicas
	}
	image := svc.Image
	if image == "" {
		image = req.StackSlug + "-" + name
	}
	if !e.composeHasHealthcheck(ctx, svc, image) {
		return "", noSwapNoHealthcheck
	}
	return running[0], ""
}

// serviceContainers names the running containers of one compose service.
func (e *ShellExecutor) serviceContainers(ctx context.Context, project, service string) []string {
	out, err := e.output(ctx, "docker", "ps",
		"--filter", "label=com.docker.compose.project="+project,
		"--filter", "label=com.docker.compose.service="+service,
		"--format", "{{.Names}}")
	if err != nil || out == "" {
		return nil
	}
	return strings.Fields(out)
}

// swapCompose scales the service to two, the second built from the new config, and swaps that one in.
func (e *ShellExecutor) swapCompose(ctx context.Context, req DeployRequestedEvent, checkout string, logs *logStream, s composeSwap) error {
	args := composeArgs(req, "up", "-d", "--no-deps", "--no-recreate", "--scale", s.service+"=2", s.service)
	if err := e.runStreamed(ctx, checkout, logs, "start new: "+cmdString("docker", args), "docker", args...); err != nil {
		e.removeNewReplicas(ctx, req.StackSlug, s)
		return err
	}
	var next string
	for _, c := range e.serviceContainers(ctx, req.StackSlug, s.service) {
		if c != s.old {
			next = c
		}
	}
	if next == "" {
		return fmt.Errorf("start new %s: compose started no second container", s.service)
	}
	return e.swapIn(ctx, logs, next, s.old)
}

// removeNewReplicas drops whatever a failed scale-up started beside the old container.
func (e *ShellExecutor) removeNewReplicas(ctx context.Context, project string, s composeSwap) {
	for _, c := range e.serviceContainers(context.WithoutCancel(ctx), project, s.service) {
		if c != s.old {
			e.removeContainer(ctx, c)
		}
	}
}

// composeHasHealthcheck reads the service's own healthcheck, which also turns the image's off, else the image's.
func (e *ShellExecutor) composeHasHealthcheck(ctx context.Context, svc composeService, image string) bool {
	if svc.Healthcheck != nil {
		return svc.Healthcheck.present()
	}
	return e.imageHasHealthcheck(ctx, image)
}

// composePort is one ports entry of a compose service; Published is empty for a port only its networks reach.
type composePort struct {
	Published string `json:"published"`
}
