package runner

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// maxLogLen caps the log tail carried in a progress frame so a chatty step can't balloon it unbounded.
const maxLogLen = 4096

// CommandRunner executes one command and streams combined output to logf.
type CommandRunner func(ctx context.Context, dir string, logf func(string), name string, args ...string) error

// ShellCommandRunner streams stdout+stderr through logf; a non-zero exit carries the output tail in its error.
func ShellCommandRunner(ctx context.Context, dir string, logf func(string), name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &lineWriter{linef: logf, buf: &buf}
	cmd.Stderr = &lineWriter{linef: logf, buf: &buf}
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("cancelled: %w", ctx.Err())
		}
		tail := tailString(buf.String(), maxLogLen)
		// A binary that never ran (missing docker, bad path) produces no output; the exec error is all there is.
		if strings.TrimSpace(tail) == "" {
			tail = err.Error()
		}
		return fmt.Errorf("%s: %s", cmdString(name, args), tail)
	}
	return nil
}

// Executor runs an assigned job; every method must honor ctx cancellation (a cancel frame aborts the job).
type Executor interface {
	Build(ctx context.Context, req DeployRequestedEvent, send func(Frame))
	Deploy(ctx context.Context, req DeployRequestedEvent, send func(Frame))
	// Discover scans the host's containers and networks for the import wizard (spec §3, §8).
	Discover(ctx context.Context) (DiscoverReport, error)
	// JoinNetworks handles a standalone join_networks request (network.go).
	JoinNetworks(ctx context.Context, gatewayContainer string, networks []string, send func(Frame))
	// Upgrade starts `nexul upgrade` on the host for req.Version; send's error return reports a transport failure.
	Upgrade(ctx context.Context, req Frame, send func(Frame) error) error
}

// ShellExecutor runs builds and deploys on the host (ADR 0032); gitToken authenticates private clones and never leaves it.
type ShellExecutor struct {
	cmd      CommandRunner
	gitToken string
	log      *slog.Logger
}

// NewShellExecutor wires the host executor; a nil cmd falls back to the real shell runner.
func NewShellExecutor(cmd CommandRunner, gitToken string, log *slog.Logger) *ShellExecutor {
	if cmd == nil {
		cmd = ShellCommandRunner
	}
	return &ShellExecutor{cmd: cmd, gitToken: gitToken, log: log}
}

// Discover scans this host's docker containers and networks (spec §3); selfID (os.Hostname()) excludes the
// runner's own container and its compose project from the report.
func (e *ShellExecutor) Discover(ctx context.Context) (DiscoverReport, error) {
	selfID, _ := os.Hostname()
	return discoverHost(ctx, e.cmd, selfID)
}

// Build executes an assigned build job; a repo-driven request also deploys the result, no registry hop in v1.
func (e *ShellExecutor) Build(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
	defer e.recoverAndReport(send, Frame{Type: FrameDeployResult, ID: req.ID})
	if req.Repo != "" {
		e.repoBuildAndDeploy(ctx, req, send)
		return
	}
	e.stepBuild(ctx, req, send)
}

// repoBuildAndDeploy: a cancelled build reports only build_result; the cancel frame owns the terminal deploy state.
func (e *ShellExecutor) repoBuildAndDeploy(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
	checkout, err := e.checkoutPath(req)
	if err != nil {
		e.failBuild(ctx, req, send, err)
		return
	}

	tag := e.imageTag(req)
	steps, err := e.buildSteps(req, checkout, tag)
	if err != nil {
		e.failBuild(ctx, req, send, err)
		return
	}
	logs := newLogStream(req.ID, send)
	defer logs.close()
	for i, st := range steps {
		if ctx.Err() != nil {
			send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: "cancelled"})
			return
		}
		send(Frame{Type: FrameBuildProgress, ID: req.ID, Step: i + 1, Total: len(steps), Log: st.label})
		var buf bytes.Buffer
		stream := logs.step(st.phase, st.label)
		logStep := func(s string) {
			_, _ = buf.WriteString(s)
			stream(s)
		}
		stepErr := st.run(ctx, logStep)
		logs.flush()
		log := tailString(buf.String(), maxLogLen)
		if stepErr != nil {
			e.log.Warn("build step failed", "id", req.ID, "step", st.label, "error", stepErr)
			send(Frame{Type: FrameBuildProgress, ID: req.ID, Step: i + 1, Total: len(steps), Log: log})
			send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: stepErr.Error()})
			if ctx.Err() != nil {
				return
			}
			send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusFailed, Error: stepErr.Error()})
			return
		}
		send(Frame{Type: FrameBuildProgress, ID: req.ID, Step: i + 1, Total: len(steps), Log: log})
	}
	send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusSuccess, Artifacts: []string{tag}})

	e.deployBuilt(ctx, req, tag, checkout, send, logs)
}

// checkoutPath resolves the persistent checkout dir for a repo-driven request: "<stack_root>/stacks/<slug>/repo",
// cloned once and refreshed in place on every later build, never deleted by a deploy (spec §4 step 1).
func (e *ShellExecutor) checkoutPath(req DeployRequestedEvent) (string, error) {
	if req.StackRoot == "" || req.StackSlug == "" {
		return "", fmt.Errorf("stack_root and stack_slug are required for a repo-driven deploy")
	}
	return filepath.Join(req.StackRoot, "stacks", req.StackSlug, "repo"), nil
}

// buildSteps: the checkout, then the image build; compose builds every service from the repo's compose file
// as its own step so the build output streams like the run strategy's, with `up` left to the deploy phase.
func (e *ShellExecutor) buildSteps(req DeployRequestedEvent, checkout, tag string) ([]buildStep, error) {
	if err := os.MkdirAll(filepath.Dir(checkout), 0o755); err != nil {
		return nil, fmt.Errorf("create stack root: %w", err)
	}
	steps := []buildStep{e.checkoutStep(req, checkout)}
	switch req.Strategy {
	case "compose":
		args := composeArgs(req, "build")
		steps = append(steps, buildStep{
			phase: LogPhaseBuild,
			label: cmdString("docker", args),
			run: func(ctx context.Context, logf func(string)) error {
				// The .env must exist before build, not just up: compose interpolates build args from it too.
				if err := writeComposeEnv(checkout, req); err != nil {
					return err
				}
				return e.cmd(ctx, checkout, logf, "docker", args...)
			},
		})
	case "run", "":
		steps = append(steps, buildStep{
			phase: LogPhaseBuild,
			label: "docker build " + tag,
			run: func(ctx context.Context, logf func(string)) error {
				args := []string{"build"}
				if req.Dockerfile != "" {
					args = append(args, "-f", req.Dockerfile)
				}
				args = append(args, "-t", tag, ".")
				return e.cmd(ctx, checkout, logf, "docker", args...)
			},
		})
	default:
		return nil, fmt.Errorf("unsupported build strategy %q", req.Strategy)
	}
	return steps, nil
}

// checkoutStep clones once, then fetches and force-checks out the ref in place on every later build — the
// persistent checkout is never deleted by a deploy (spec §4 step 1).
func (e *ShellExecutor) checkoutStep(req DeployRequestedEvent, checkout string) buildStep {
	if _, err := os.Stat(filepath.Join(checkout, ".git")); err == nil {
		return e.fetchStep(req, checkout)
	}
	return e.cloneStep(req, checkout)
}

// authArgs prepends the git-credential flag a clone/fetch needs; the server's per-job connector token wins,
// the runner's own env token is the fallback.
func (e *ShellExecutor) authArgs(req DeployRequestedEvent, args []string) []string {
	token := req.GitToken
	if token == "" {
		token = e.gitToken
	}
	if token == "" {
		return args
	}
	basic := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	return append([]string{"-c", "http.extraheader=Authorization: Basic " + basic}, args...)
}

// cloneStep injects the git token via http.extraheader, never the URL, so it can't leak into process listings.
func (e *ShellExecutor) cloneStep(req DeployRequestedEvent, checkout string) buildStep {
	return buildStep{
		phase: LogPhaseCheckout,
		label: "clone " + req.Repo + "@" + req.Ref,
		run: func(ctx context.Context, logf func(string)) error {
			url := "https://github.com/" + req.Repo + ".git"
			args := e.authArgs(req, []string{"clone", "--depth", "1", "--branch", req.Ref, url, checkout})
			return e.cmd(ctx, "", logf, "git", args...)
		},
	}
}

// fetchStep refreshes an existing persistent checkout in place: fetch the ref, then force-checkout it (spec §4).
func (e *ShellExecutor) fetchStep(req DeployRequestedEvent, checkout string) buildStep {
	return buildStep{
		phase: LogPhaseCheckout,
		label: "fetch " + req.Repo + "@" + req.Ref,
		run: func(ctx context.Context, logf func(string)) error {
			args := e.authArgs(req, []string{"fetch", "--depth", "1", "origin", req.Ref})
			if err := e.cmd(ctx, checkout, logf, "git", args...); err != nil {
				return err
			}
			return e.cmd(ctx, checkout, logf, "git", "checkout", "--force", "FETCH_HEAD")
		},
	}
}

// deployBuilt starts the locally-built image on this runner; no registry hop.
func (e *ShellExecutor) deployBuilt(ctx context.Context, req DeployRequestedEvent, tag, checkout string, send func(Frame), logs *logStream) {
	req.Image = tag
	send(Frame{Type: FrameDeployProgress, ID: req.ID, Phase: DeployPhaseStarting})
	if err := e.startBuilt(ctx, req, checkout, logs); err != nil {
		e.log.Warn("deploy start failed", "id", req.ID, "error", err)
		if ctx.Err() != nil {
			// A cancel frame already reports terminal state; reporting again would double-transition it.
			return
		}
		send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusFailed, Error: err.Error()})
		return
	}
	send(Frame{Type: FrameDeployProgress, ID: req.ID, Phase: DeployPhaseHealthy})
	if req.GatewayContainer != "" {
		e.joinGatewayNetworks(ctx, req, logs)
	}
	services := e.observe(ctx, req)
	send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusHealthy, Address: primaryAddress(services, req.Network), Services: services})
}

// startBuilt runs the freshly checked-out stack: compose as a named project from the checkout directory (so
// bind mounts resolve against it), or a run-strategy container named after the slug (spec §4 step 2).
func (e *ShellExecutor) startBuilt(ctx context.Context, req DeployRequestedEvent, checkout string, logs *logStream) error {
	switch req.Strategy {
	case "compose":
		args := composeArgs(req, "up", "-d", "--remove-orphans")
		return e.runStreamed(ctx, checkout, logs, cmdString("docker", args), "docker", args...)
	case "run", "":
		return e.runContainer(ctx, req, logs)
	default:
		return fmt.Errorf("unsupported deploy strategy %q", req.Strategy)
	}
}

// composeArgs renders `compose -p <slug> [-f <path>] <verb...>`, the project flags every compose call shares.
func composeArgs(req DeployRequestedEvent, verb ...string) []string {
	args := []string{"compose", "-p", req.StackSlug}
	if req.ComposePath != "" {
		args = append(args, "-f", req.ComposePath)
	}
	return append(args, verb...)
}

// runContainer replaces the slug's previous container with a fresh `docker run`; only the run itself is
// streamed, since the rm and network probes are expected to fail on a first deploy and say nothing useful.
// The label names the container and image but never the args: env values ride `-e` and are secrets.
func (e *ShellExecutor) runContainer(ctx context.Context, req DeployRequestedEvent, logs *logStream) error {
	_ = e.cmd(ctx, "", discardLog, "docker", "rm", "-f", req.StackSlug)
	e.ensureNetwork(ctx, req.Network)
	label := "docker run --name " + req.StackSlug + " " + req.Image
	return e.runStreamed(ctx, "", logs, label, "docker", runArgs(req)...)
}

// runStreamed runs one deploy-phase command with its output streamed under label, flushing when it ends.
func (e *ShellExecutor) runStreamed(ctx context.Context, dir string, logs *logStream, label, name string, args ...string) error {
	defer logs.flush()
	return e.cmd(ctx, dir, logs.step(LogPhaseDeploy, label), name, args...)
}

// ensureNetwork creates the run stack's docker network when it does not exist yet: a fresh run stack names its
// own `<slug>_default`, and `docker run --network` refuses a name it cannot find. "already exists" is the
// normal answer on every later deploy, so the error is ignored; a real failure surfaces at `docker run`.
func (e *ShellExecutor) ensureNetwork(ctx context.Context, network string) {
	if network == "" {
		return
	}
	_ = e.cmd(ctx, "", discardLog, "docker", "network", "create", network)
}

// writeComposeEnv puts the stack's env values in a .env file beside the compose file, which is where compose
// reads `${VAR}` interpolation from and what an `env_file: .env` entry points at. With no env set, a committed
// .env stays as the repo shipped it and a missing one is created empty, so `env_file: .env` never fails the
// start; when values are set, Nexul's win.
func writeComposeEnv(checkout string, req DeployRequestedEvent) error {
	dir := filepath.Join(checkout, filepath.Dir(req.ComposePath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create compose dir: %w", err)
	}
	if len(req.Env) == 0 {
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return nil
		}
	}
	var b strings.Builder
	for _, k := range slices.Sorted(maps.Keys(req.Env)) {
		b.WriteString(k + "=" + req.Env[k] + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("write compose env: %w", err)
	}
	return nil
}

// imageTag stays on the runner (no registry address needed); the deploy id makes each build distinct for rollback.
func (e *ShellExecutor) imageTag(req DeployRequestedEvent) string {
	name := sanitizeImageName(req.StackSlug)
	if name == "" {
		name = "service"
	}
	return name + ":" + req.ID
}

// sanitizeImageName maps characters Docker image names reject onto dashes.
func sanitizeImageName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

func (e *ShellExecutor) failBuild(ctx context.Context, req DeployRequestedEvent, send func(Frame), err error) {
	send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: err.Error()})
	if ctx.Err() == nil {
		send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusFailed, Error: err.Error()})
	}
}

// buildStep is one ordered command in a build phase, labelled for progress frames and the deploy log.
type buildStep struct {
	phase string
	label string
	run   func(ctx context.Context, logf func(string)) error
}

// stepBuild is the legacy steps-driven build path: each step is a shell command in an ephemeral temp dir.
func (e *ShellExecutor) stepBuild(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
	defer e.recoverAndReport(send, Frame{Type: FrameBuildResult, ID: req.ID})
	if len(req.Steps) == 0 {
		send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusSuccess})
		return
	}
	workdir, err := os.MkdirTemp("", "nexul-build-*")
	if err != nil {
		send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: err.Error()})
		return
	}
	defer func() {
		if rerr := os.RemoveAll(workdir); rerr != nil {
			e.log.Debug("remove build workdir", "workdir", workdir, "error", rerr)
		}
	}()

	logs := newLogStream(req.ID, send)
	defer logs.close()
	for i, step := range req.Steps {
		if ctx.Err() != nil {
			send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: "cancelled"})
			return
		}
		send(Frame{Type: FrameBuildProgress, ID: req.ID, Step: i + 1, Total: len(req.Steps)})
		var buf bytes.Buffer
		stream := logs.step(LogPhaseBuild, step)
		logStep := func(s string) {
			_, _ = buf.WriteString(s)
			stream(s)
		}
		stepErr := e.cmd(ctx, workdir, logStep, "sh", "-c", step)
		logs.flush()
		progress := Frame{Type: FrameBuildProgress, ID: req.ID, Step: i + 1, Total: len(req.Steps), Log: tailString(buf.String(), maxLogLen)}
		send(progress)
		if stepErr != nil {
			e.log.Warn("build step failed", "id", req.ID, "step", i+1, "error", stepErr)
			send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusFailed, Error: stepErr.Error()})
			return
		}
	}
	send(Frame{Type: FrameBuildResult, ID: req.ID, Status: BuildStatusSuccess})
}

// Deploy pulls the image, starts the service, then reports the phases and a terminal deploy_result.
func (e *ShellExecutor) Deploy(ctx context.Context, req DeployRequestedEvent, send func(Frame)) {
	defer e.recoverAndReport(send, Frame{Type: FrameDeployResult, ID: req.ID})
	logs := newLogStream(req.ID, send)
	defer logs.close()
	send(Frame{Type: FrameDeployProgress, ID: req.ID, Phase: DeployPhasePulling})
	if err := e.runStreamed(ctx, "", logs, "docker pull "+req.Image, "docker", "pull", req.Image); err != nil {
		e.log.Warn("deploy pull failed", "id", req.ID, "error", err)
		send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusFailed, Error: err.Error()})
		return
	}
	send(Frame{Type: FrameDeployProgress, ID: req.ID, Phase: DeployPhaseStarting})
	if err := e.start(ctx, req, logs); err != nil {
		e.log.Warn("deploy start failed", "id", req.ID, "error", err)
		send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusFailed, Error: err.Error()})
		return
	}
	send(Frame{Type: FrameDeployProgress, ID: req.ID, Phase: DeployPhaseHealthy})
	if req.GatewayContainer != "" {
		e.joinGatewayNetworks(ctx, req, logs)
	}
	services := e.observe(ctx, req)
	send(Frame{Type: FrameDeployResult, ID: req.ID, Status: DeployStatusHealthy, Address: primaryAddress(services, req.Network), Services: services})
}

// primaryAddress keeps deploy_result.address populated from the observation report, for consumers that haven't
// moved onto Services yet; an empty network takes the first reported network, matching the old runAddress.
func primaryAddress(services []ObservedService, network string) string {
	if len(services) == 0 {
		return ""
	}
	nets := services[0].Networks
	if network != "" {
		for _, n := range nets {
			if n.Name == network {
				return n.Address
			}
		}
		return ""
	}
	if len(nets) == 0 {
		return ""
	}
	return nets[0].Address
}

// output runs a command through the same CommandRunner seam as everything else, collecting stdout instead of streaming it.
func (e *ShellExecutor) output(ctx context.Context, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	if err := e.cmd(ctx, "", func(s string) { buf.WriteString(s) }, name, args...); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// runArgs builds `docker run` for the run strategy, naming the container after the stack's slug (spec §4 step 2).
// The container survives a host reboot on its own (unless-stopped: a deliberate stop stays stopped); compose
// stacks carry their own restart policy.
func runArgs(req DeployRequestedEvent) []string {
	args := []string{"run", "-d", "--restart", "unless-stopped", "--name", req.StackSlug}
	if req.Network != "" {
		args = append(args, "--network", req.Network)
	}
	for _, p := range req.Ports {
		args = append(args, "-p", p)
	}
	for _, m := range req.Mounts {
		args = append(args, "-v", m)
	}
	for _, k := range slices.Sorted(maps.Keys(req.Env)) {
		args = append(args, "-e", k+"="+req.Env[k])
	}
	return append(append(args, req.Image), req.Command...)
}

// start runs a pre-built image directly (no repo, no checkout). Compose always needs its repo's compose file, so
// a plain (non-repo-driven) deploy only ever supports the run strategy; compose goes through Build.
func (e *ShellExecutor) start(ctx context.Context, req DeployRequestedEvent, logs *logStream) error {
	switch req.Strategy {
	case "run", "":
		return e.runContainer(ctx, req, logs)
	default:
		return fmt.Errorf("unsupported deploy strategy %q", req.Strategy)
	}
}

// upgradeUnit is the transient systemd unit `nexul upgrade` runs in; `journalctl -u nexul-upgrade` holds its output.
const upgradeUnit = "nexul-upgrade"

// lookPathFn finds the nexul command on the host; a package var so tests need no real binary on the PATH.
var lookPathFn = exec.LookPath

// Upgrade starts `nexul upgrade` for req.Version in its own transient systemd unit, so the upgrade outlives this
// runner's connection while the server restarts, then reports started (ADR 0069).
func (e *ShellExecutor) Upgrade(ctx context.Context, req Frame, send func(Frame) error) error {
	nexul, err := lookPathFn("nexul")
	if err != nil {
		return send(upgradeFailed(req, "the nexul command is not on this host; upgrading from the UI needs an install made with nexul install"))
	}
	if _, err := e.output(ctx, "systemd-run", "--unit", upgradeUnit, "--collect", "--quiet", nexul, "upgrade", "--version", req.Version); err != nil {
		return send(upgradeFailed(req, err.Error()))
	}
	if err := send(Frame{Type: FrameUpgradeProgress, ID: req.ID, Log: "started " + upgradeUnit + ".service"}); err != nil {
		return err
	}
	return send(Frame{Type: FrameUpgradeResult, ID: req.ID, Status: UpgradeStatusStarted, Log: upgradeUnit})
}

// upgradeFailed builds the terminal frame for an update that could not be started.
func upgradeFailed(req Frame, msg string) Frame {
	return Frame{Type: FrameUpgradeResult, ID: req.ID, Status: UpgradeStatusFailed, Error: msg}
}

// recoverAndReport converts an executor panic into a failed result frame so a buggy step cannot hang the job silently.
func (e *ShellExecutor) recoverAndReport(send func(Frame), result Frame) {
	if r := recover(); r != nil {
		e.log.Error("executor panic", "id", result.ID, "panic", r)
		result.Status = BuildStatusFailed
		send(result)
	}
}

// lineWriter streams output one line at a time through linef while buffering the full text for error tails.
type lineWriter struct {
	linef func(string)
	buf   *bytes.Buffer
}

func (w *lineWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if err != nil {
		return n, err
	}
	start := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '\n' {
			w.linef(string(p[start : i+1]))
			start = i + 1
		}
	}
	if start < len(p) {
		w.linef(string(p[start:]))
	}
	return n, nil
}

func discardLog(string) {}

func tailString(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return "…" + s[len(s)-max:]
}

// cmdString renders a command for errors and logs; any argument carrying credentials is masked so a failed clone never echoes its token.
func cmdString(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, name)
	for _, a := range args {
		if strings.Contains(a, "Authorization:") || strings.Contains(a, "x-access-token") {
			a = "http.extraheader=Authorization: <redacted>"
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}
