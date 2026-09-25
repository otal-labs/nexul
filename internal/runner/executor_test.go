package runner

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cmdCall struct {
	dir  string
	name string
	args []string
}

type fakeCmd struct {
	mu       sync.Mutex
	calls    []cmdCall
	failNext int
	err      error
	panicOn  string
	// composePSOut/composePSErr canned-answer `docker compose ... ps -a --format json`.
	composePSOut string
	composePSErr error
	// inspectByName canned-answers `docker inspect --format '{{json .}}' <name>` per container name; inspectOut is
	// the fallback for tests that don't care which container was inspected (and the older -f address-only calls).
	inspectByName map[string]string
	inspectOut    string
	inspectErr    error
	// runOut/runErr canned-answer `docker run ...` (the upgrade helper's stdout is its container id).
	runOut string
	runErr error
	// stdout is streamed through logf by every non-canned command, so tests can watch output reach the log.
	stdout string
}

func (f *fakeCmd) run(ctx context.Context, dir string, logf func(string), name string, args ...string) error {
	f.mu.Lock()
	f.calls = append(f.calls, cmdCall{dir: dir, name: name, args: args})
	fail := f.failNext > 0
	if fail {
		f.failNext--
	}
	panicOn := f.panicOn == name
	canned, out, err := f.cannedAnswer(name, args)
	f.mu.Unlock()
	if panicOn {
		panic("cmd panicked: " + name)
	}
	if canned {
		if err != nil {
			return err
		}
		logf(out)
		return nil
	}
	if f.stdout != "" {
		logf(f.stdout)
	}
	if fail {
		return f.err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

// cannedAnswer picks the scripted stdout and error for docker inspect, compose ps, and run. Must run under f.mu.
func (f *fakeCmd) cannedAnswer(name string, args []string) (bool, string, error) {
	if name != "docker" || len(args) == 0 {
		return false, "", nil
	}
	if args[0] == "inspect" {
		out, err := f.inspectFor(args)
		return true, out, err
	}
	if args[0] == "compose" && slices.Contains(args, "ps") {
		return true, f.composePSOut, f.composePSErr
	}
	if args[0] == "run" {
		return true, f.runOut, f.runErr
	}
	return false, "", nil
}

// inspectFor resolves the canned inspect answer for the container named in the last arg, falling back to the
// single inspectOut/inspectErr pair older tests set when no per-name entry exists. Must run under f.mu.
func (f *fakeCmd) inspectFor(args []string) (string, error) {
	if len(args) == 0 {
		return f.inspectOut, f.inspectErr
	}
	if out, ok := f.inspectByName[args[len(args)-1]]; ok {
		return out, nil
	}
	return f.inspectOut, f.inspectErr
}

func (f *fakeCmd) names() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.calls))
	for _, c := range f.calls {
		out = append(out, c.name)
	}
	return out
}

func (f *fakeCmd) argsFor(name string) [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out [][]string
	for _, c := range f.calls {
		if c.name == name {
			out = append(out, c.args)
		}
	}
	return out
}

type frameRecorder struct {
	mu     sync.Mutex
	frames []Frame
}

func (r *frameRecorder) send(f Frame) {
	r.mu.Lock()
	r.frames = append(r.frames, f)
	r.mu.Unlock()
}

func (r *frameRecorder) all() []Frame {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Frame, len(r.frames))
	copy(out, r.frames)
	return out
}

// lifecycle is every frame except deploy_log: the progress and result sequence positional assertions index.
func (r *frameRecorder) lifecycle() []Frame {
	var out []Frame
	for _, f := range r.all() {
		if f.Type != FrameDeployLog {
			out = append(out, f)
		}
	}
	return out
}

func (r *frameRecorder) logs() []Frame {
	var out []Frame
	for _, f := range r.all() {
		if f.Type == FrameDeployLog {
			out = append(out, f)
		}
	}
	return out
}

// logText joins every deploy_log batch for one phase.
func (r *frameRecorder) logText(phase string) string {
	var b strings.Builder
	for _, f := range r.logs() {
		if f.Phase == phase {
			b.WriteString(f.Log)
		}
	}
	return b.String()
}

func (r *frameRecorder) last() Frame {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.frames[len(r.frames)-1]
}

func newTestExecutor(cmd CommandRunner) *ShellExecutor {
	return NewShellExecutor(cmd, "", testLogger())
}

// The dispatch-time connector token outranks the runner's env token, so an env-less runner still clones private repos.
func TestShellExecutor_Clone_PrefersJobToken(t *testing.T) {
	cmd := &fakeCmd{}
	e := newTokenExecutor(cmd.run, "env-token")
	step := e.cloneStep(DeployRequestedEvent{Repo: "org/app", Ref: "main", GitToken: "job-token"}, "/tmp/checkout")
	require.NoError(t, step.run(context.Background(), func(string) {}))
	args := cmd.argsFor("git")[0]
	assert.Equal(t, "http.extraheader=Authorization: Basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:job-token")), args[1])
}

func TestCmdString_RedactsCloneCredentials(t *testing.T) {
	got := cmdString("git", []string{"-c", "http.extraheader=Authorization: Basic eC1hY2Nlc3M6c2VjcmV0", "clone", "https://github.com/org/app.git"})
	assert.NotContains(t, got, "eC1hY2Nlc3M6c2VjcmV0")
	assert.Contains(t, got, "Authorization: <redacted>")
	assert.Contains(t, got, "clone https://github.com/org/app.git")
}

func newTokenExecutor(cmd CommandRunner, token string) *ShellExecutor {
	return NewShellExecutor(cmd, token, testLogger())
}

// buildRequest is a repo-driven build request: the assign_build fields
// a build-and-deploy job carries.
func buildRequest(t *testing.T, id string) DeployRequestedEvent {
	t.Helper()
	return DeployRequestedEvent{
		ID: id, Kind: RequestBuild, Repo: "org/app", Ref: "main",
		Service: "Api", Strategy: "run", Dockerfile: "Dockerfile", Network: "app-net",
		StackSlug: "api", StackRoot: t.TempDir(),
		Env: map[string]string{"B": "2", "A": "1"},
	}
}

func TestShellExecutor_Build_RepoDriven_RunStrategy(t *testing.T) {
	cmd := &fakeCmd{inspectByName: map[string]string{
		"api": `{"Config":{"Image":"api:b1"},"State":{"Status":"running"},"NetworkSettings":{"Networks":{"app-net":{"IPAddress":"172.18.0.4"}},"Ports":{"80/tcp":[{"HostPort":"8080"}]}}}`,
	}, stdout: "output line\n"}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}
	e.Build(context.Background(), buildRequest(t, "b1"), rec.send)

	frames := rec.lifecycle()
	require.Len(t, frames, 8)
	assert.Equal(t, FrameBuildProgress, frames[0].Type)
	assert.Equal(t, 1, frames[0].Step)
	assert.Equal(t, 2, frames[0].Total)
	assert.Contains(t, frames[0].Log, "clone org/app@main")
	assert.Equal(t, FrameBuildProgress, frames[2].Type)
	assert.Contains(t, frames[2].Log, "docker build")
	assert.Equal(t, FrameBuildResult, frames[4].Type)
	assert.Equal(t, BuildStatusSuccess, frames[4].Status)
	assert.Equal(t, []string{"api:b1"}, frames[4].Artifacts)
	assert.Equal(t, FrameDeployProgress, frames[5].Type)
	assert.Equal(t, DeployPhaseStarting, frames[5].Phase)
	assert.Equal(t, FrameDeployProgress, frames[6].Type)
	assert.Equal(t, DeployPhaseHealthy, frames[6].Phase)
	assert.Equal(t, FrameDeployResult, frames[7].Type)
	assert.Equal(t, DeployStatusHealthy, frames[7].Status)
	assert.Equal(t, "172.18.0.4", frames[7].Address, "the primary address is derived from the observation report")
	require.Len(t, frames[7].Services, 1)
	assert.Equal(t, ObservedService{
		Name: "api", ContainerName: "api", Image: "api:b1", Status: "running",
		Networks: []ObservedNetwork{{Name: "app-net", Address: "172.18.0.4"}},
		Ports:    []string{"8080:80/tcp"},
	}, frames[7].Services[0])

	gitArgs := cmd.argsFor("git")[0]
	assert.Equal(t, "clone", gitArgs[0])
	assert.Equal(t, []string{"--depth", "1", "--branch", "main", "https://github.com/org/app.git"}, gitArgs[1:6])
	assert.NotEmpty(t, gitArgs[6], "clone lands in the checkout dir")
	assert.Equal(t, []string{"build", "-f", "Dockerfile", "-t", "api:b1", "."}, cmd.argsFor("docker")[0])
	assert.Equal(t, []string{"rm", "-f", "api"}, cmd.argsFor("docker")[1])
	assert.Equal(t, []string{"network", "create", "app-net"}, cmd.argsFor("docker")[2], "a run stack's network is created before docker run needs it")
	assert.Equal(t, []string{"run", "-d", "--restart", "unless-stopped", "--name", "api", "--network", "app-net", "-e", "A=1", "-e", "B=2", "api:b1"}, cmd.argsFor("docker")[3])
	assert.Equal(t, []string{"inspect", "--format", "{{json .}}", "api"}, cmd.argsFor("docker")[4])

	// Every step announces itself, then streams its output under its phase; the rm/network probes stay silent.
	assert.Equal(t, "clone org/app@main\noutput line\n", rec.logText(LogPhaseCheckout))
	assert.Equal(t, "docker build api:b1\noutput line\n", rec.logText(LogPhaseBuild))
	assert.Equal(t, "docker run --name api api:b1\n", rec.logText(LogPhaseDeploy))
	for _, f := range rec.logs() {
		assert.Equal(t, "b1", f.ID)
		assert.Positive(t, f.TS)
		assert.NoError(t, f.Validate())
	}
	all := rec.all()
	assert.Equal(t, FrameDeployLog, all[1].Type, "a step's output batch lands before its closing progress frame")
	assert.Equal(t, FrameDeployResult, all[len(all)-1].Type, "nothing is sent after the terminal result")
}

func TestShellExecutor_Build_RepoDriven_ComposeStrategy(t *testing.T) {
	cmd := &fakeCmd{
		composePSOut: `{"Name":"api-web-1","Service":"web"}` + "\n" + `{"Name":"api-worker-1","Service":"worker"}` + "\n",
		inspectByName: map[string]string{
			"api-web-1":    `{"Config":{"Image":"nginx:1"},"State":{"Status":"running","Health":{"Status":"healthy"}},"NetworkSettings":{"Networks":{"api_default":{"IPAddress":"172.18.0.2"}},"Ports":{"80/tcp":[{"HostPort":"8080"}]}}}`,
			"api-worker-1": `{"Config":{"Image":"worker:1"},"State":{"Status":"running"},"NetworkSettings":{"Networks":{"api_default":{"IPAddress":"172.18.0.3"}}}}`,
		},
	}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}
	req := buildRequest(t, "b1")
	req.Strategy = "compose"
	req.ComposePath = "compose/docker-compose.yml"
	req.Network = "" // compose takes its networks from the compose file, not the Network field.
	e.Build(context.Background(), req, rec.send)

	frames := rec.lifecycle()
	require.GreaterOrEqual(t, len(frames), 7)
	assert.Equal(t, FrameBuildProgress, frames[2].Type)
	assert.Equal(t, 2, frames[2].Step)
	assert.Equal(t, "docker compose -p api -f compose/docker-compose.yml build", frames[2].Log)
	assert.Equal(t, FrameBuildResult, frames[4].Type)
	assert.Equal(t, BuildStatusSuccess, frames[4].Status)
	assert.Equal(t, DeployStatusHealthy, rec.last().Status)
	assert.Equal(t, FrameDeployResult, rec.last().Type)

	// One clone step, then compose builds as a named project from the checkout (the compose file is the source
	// of truth) as its own build step, `up` starts it in the deploy phase, then the report lists it by
	// `docker compose ps` and inspects each container.
	assert.Len(t, cmd.argsFor("git"), 1)
	assert.Equal(t, []string{"compose", "-p", "api", "-f", "compose/docker-compose.yml", "build"}, cmd.argsFor("docker")[0])
	assert.Equal(t, []string{"compose", "-p", "api", "-f", "compose/docker-compose.yml", "up", "-d", "--remove-orphans"}, cmd.argsFor("docker")[1])
	assert.Equal(t, []string{"compose", "-p", "api", "ps", "-a", "--format", "json"}, cmd.argsFor("docker")[2])
	assert.Equal(t, "docker compose -p api -f compose/docker-compose.yml build\n", rec.logText(LogPhaseBuild))
	assert.Equal(t, "docker compose -p api -f compose/docker-compose.yml up -d --remove-orphans\n", rec.logText(LogPhaseDeploy))

	require.Len(t, rec.last().Services, 2)
	assert.Equal(t, ObservedService{
		Name: "web", ContainerName: "api-web-1", Image: "nginx:1", Status: "healthy",
		Networks: []ObservedNetwork{{Name: "api_default", Address: "172.18.0.2"}},
		Ports:    []string{"8080:80/tcp"},
	}, rec.last().Services[0])
	assert.Equal(t, ObservedService{
		Name: "worker", ContainerName: "api-worker-1", Image: "worker:1", Status: "running",
		Networks: []ObservedNetwork{{Name: "api_default", Address: "172.18.0.3"}},
	}, rec.last().Services[1])
	assert.Equal(t, "172.18.0.2", rec.last().Address, "the primary address falls back to the first reported container")

	// The stack's env lands in a .env beside the compose file before the build: interpolation and `env_file:
	// .env` both read it.
	envFile, err := os.ReadFile(filepath.Join(req.StackRoot, "stacks", "api", "repo", "compose", ".env"))
	require.NoError(t, err)
	assert.Equal(t, "A=1\nB=2\n", string(envFile))
}

func TestShellExecutor_Build_RepoDriven_ComposeBuildFailure_StopsBeforeUp(t *testing.T) {
	cmd := &fakeCmd{}
	var mu sync.Mutex
	var docker [][]string
	cmdFn := func(ctx context.Context, dir string, logf func(string), name string, args ...string) error {
		if name == "docker" {
			mu.Lock()
			docker = append(docker, args)
			mu.Unlock()
			logf("failed to solve: Dockerfile missing\n")
			return assert.AnError
		}
		return cmd.run(ctx, dir, logf, name, args...)
	}
	e := newTestExecutor(cmdFn)
	rec := &frameRecorder{}
	req := buildRequest(t, "b1")
	req.Strategy = "compose"
	e.Build(context.Background(), req, rec.send)

	assert.Equal(t, DeployStatusFailed, rec.last().Status)
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, docker, 1, "up never runs after a failed compose build")
	assert.Equal(t, "build", docker[0][len(docker[0])-1])
	assert.Equal(t, "docker compose -p api build\nfailed to solve: Dockerfile missing\n", rec.logText(LogPhaseBuild))
	assert.Empty(t, rec.logText(LogPhaseDeploy))
}

func TestWriteComposeEnv_NoEnv(t *testing.T) {
	t.Run("creates an empty .env when the repo has none", func(t *testing.T) {
		checkout := t.TempDir()
		require.NoError(t, writeComposeEnv(checkout, DeployRequestedEvent{ComposePath: "docker-compose.yml"}))
		b, err := os.ReadFile(filepath.Join(checkout, ".env"))
		require.NoError(t, err)
		assert.Empty(t, b)
	})

	t.Run("leaves a committed .env alone", func(t *testing.T) {
		checkout := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(checkout, ".env"), []byte("SHIPPED=1\n"), 0o600))
		require.NoError(t, writeComposeEnv(checkout, DeployRequestedEvent{ComposePath: "docker-compose.yml"}))
		b, err := os.ReadFile(filepath.Join(checkout, ".env"))
		require.NoError(t, err)
		assert.Equal(t, "SHIPPED=1\n", string(b))
	})
}

func TestShellExecutor_Build_RepoDriven_CloneAuth(t *testing.T) {
	cmd := &fakeCmd{}
	e := newTokenExecutor(cmd.run, "ghp_secret")
	rec := &frameRecorder{}
	e.Build(context.Background(), buildRequest(t, "b1"), rec.send)

	require.Equal(t, DeployStatusHealthy, rec.last().Status)
	args := cmd.argsFor("git")[0]
	require.Len(t, args, 9, "-c http.extraheader + clone flags + url + checkout")
	assert.Equal(t, "-c", args[0])
	assert.Equal(t, "http.extraheader=Authorization: Basic eC1hY2Nlc3MtdG9rZW46Z2hwX3NlY3JldA==", args[1])
	assert.Equal(t, "https://github.com/org/app.git", args[7], "the token rides the header, never the URL")
	for _, f := range rec.logs() {
		assert.NotContains(t, f.Log, "ghp_secret")
		assert.NotContains(t, f.Log, "Authorization")
	}
	assert.Equal(t, "clone org/app@main\n", rec.logText(LogPhaseCheckout), "the label names the repo and ref, not the git args")
}

func TestShellExecutor_Build_RepoDriven_ErrorPaths(t *testing.T) {
	t.Run("clone failure reports failed build and deploy", func(t *testing.T) {
		cmd := &fakeCmd{failNext: 1, err: assert.AnError}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Build(context.Background(), buildRequest(t, "b1"), rec.send)

		frames := rec.all()
		assert.Equal(t, FrameBuildResult, frames[len(frames)-2].Type)
		assert.Equal(t, BuildStatusFailed, frames[len(frames)-2].Status)
		assert.Equal(t, FrameDeployResult, rec.last().Type)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
		assert.NotEmpty(t, rec.last().Error)
		assert.Equal(t, 1, len(cmd.names()), "no build step runs after a failed clone")
	})

	t.Run("docker build failure stops before deploy", func(t *testing.T) {
		// Fail only the docker command: clone succeeds, build fails, so the
		// run step must never execute.
		var calls []string
		var mu sync.Mutex
		cmdFn := func(_ context.Context, _ string, _ func(string), name string, _ ...string) error {
			mu.Lock()
			calls = append(calls, name)
			mu.Unlock()
			if name == "docker" {
				return assert.AnError
			}
			return nil
		}
		e := newTestExecutor(cmdFn)
		rec := &frameRecorder{}
		e.Build(context.Background(), buildRequest(t, "b1"), rec.send)

		frames := rec.all()
		assert.Equal(t, FrameBuildResult, frames[len(frames)-2].Type)
		assert.Equal(t, BuildStatusFailed, frames[len(frames)-2].Status)
		assert.Equal(t, FrameDeployResult, rec.last().Type)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
		mu.Lock()
		assert.Equal(t, []string{"git", "docker"}, calls, "docker run must not start after a failed build")
		mu.Unlock()
	})

	t.Run("deploy start failure reports failed result", func(t *testing.T) {
		cmd := &fakeCmd{failNext: 3, err: assert.AnError}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Build(context.Background(), buildRequest(t, "b1"), rec.send)
		assert.Equal(t, FrameDeployResult, rec.last().Type)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
	})

	t.Run("unsupported strategy reports failed results", func(t *testing.T) {
		cmd := &fakeCmd{}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := buildRequest(t, "b1")
		req.Strategy = "kube"
		e.Build(context.Background(), req, rec.send)
		assert.Equal(t, BuildStatusFailed, rec.all()[len(rec.all())-2].Status)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
	})

	t.Run("cancelled build reports only a failed build_result", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		e := newTestExecutor((&fakeCmd{}).run)
		rec := &frameRecorder{}
		cancel()
		e.Build(ctx, buildRequest(t, "b1"), rec.send)
		assert.Equal(t, FrameBuildResult, rec.last().Type)
		assert.Equal(t, BuildStatusFailed, rec.last().Status)
		assert.Equal(t, "cancelled", rec.last().Error)
	})
}

func TestImageTag(t *testing.T) {
	assert.Equal(t, "api:b1", newTestExecutor(nil).imageTag(DeployRequestedEvent{ID: "b1", StackSlug: "api"}))
	assert.Equal(t, "my-api:b1", newTestExecutor(nil).imageTag(DeployRequestedEvent{ID: "b1", StackSlug: "my-api"}))
	assert.Equal(t, "service:b1", newTestExecutor(nil).imageTag(DeployRequestedEvent{ID: "b1"}))
}

func TestShellExecutor_Build_ErrorPaths(t *testing.T) {
	t.Run("zero steps reports immediate success", func(t *testing.T) {
		e := newTestExecutor((&fakeCmd{}).run)
		rec := &frameRecorder{}
		e.Build(context.Background(), DeployRequestedEvent{ID: "b1", Kind: RequestBuild}, rec.send)
		assert.Equal(t, BuildStatusSuccess, rec.last().Status)
		assert.Equal(t, FrameBuildResult, rec.last().Type)
	})

	t.Run("failed step reports failed result", func(t *testing.T) {
		cmd := &fakeCmd{failNext: 1, err: context.DeadlineExceeded}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Steps: []string{"false", "true"}}
		e.Build(context.Background(), req, rec.send)
		frames := rec.all()
		assert.Equal(t, FrameBuildResult, frames[len(frames)-1].Type)
		assert.Equal(t, BuildStatusFailed, frames[len(frames)-1].Status)
		assert.NotEmpty(t, frames[len(frames)-1].Error)
		assert.Equal(t, 1, len(cmd.names()), "second step must not run after a failure")
	})

	t.Run("cancelled context aborts the job", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		e := newTestExecutor((&fakeCmd{}).run)
		rec := &frameRecorder{}
		cancel()
		e.Build(ctx, DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Steps: []string{"a"}}, rec.send)
		assert.Equal(t, BuildStatusFailed, rec.last().Status)
		assert.Equal(t, "cancelled", rec.last().Error)
	})

	t.Run("panic in a step becomes a failed result", func(t *testing.T) {
		cmd := &fakeCmd{panicOn: "sh"}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Build(context.Background(), DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Steps: []string{"boom"}}, rec.send)
		assert.Equal(t, FrameBuildResult, rec.last().Type)
		assert.Equal(t, BuildStatusFailed, rec.last().Status)
	})
}

func TestShellExecutor_Build_HappyPath(t *testing.T) {
	cmd := &fakeCmd{}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}
	req := DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Steps: []string{"go build", "go test"}}
	e.Build(context.Background(), req, rec.send)

	frames := rec.lifecycle()
	require.Len(t, frames, 5)
	assert.Equal(t, FrameBuildProgress, frames[0].Type)
	assert.Equal(t, 1, frames[0].Step)
	assert.Equal(t, 2, frames[0].Total)
	assert.Equal(t, FrameBuildProgress, frames[1].Type)
	assert.Equal(t, 1, frames[1].Step)
	assert.Equal(t, FrameBuildProgress, frames[2].Type)
	assert.Equal(t, 2, frames[2].Step)
	assert.Equal(t, FrameBuildResult, frames[4].Type)
	assert.Equal(t, BuildStatusSuccess, frames[4].Status)
	assert.Equal(t, "go build\ngo test\n", rec.logText(LogPhaseBuild), "each step announces its command")
}

func TestShellExecutor_Build_StepsRunInTempDir(t *testing.T) {
	cmd := &fakeCmd{}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}
	e.Build(context.Background(), DeployRequestedEvent{ID: "b1", Kind: RequestBuild, Steps: []string{"a"}}, rec.send)

	cmd.mu.Lock()
	dir := cmd.calls[0].dir
	cmd.mu.Unlock()
	assert.NotEmpty(t, dir, "steps must run in a working directory")
	assert.Equal(t, "sh", cmd.names()[0])
	assert.Equal(t, []string{"-c", "a"}, cmd.argsFor("sh")[0])
}

// cloudflared only connects with `tunnel run`, so a service command must follow the image verbatim.
func TestShellExecutor_Deploy_RunStrategyAppendsCommand(t *testing.T) {
	cmd := &fakeCmd{}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}
	req := DeployRequestedEvent{
		ID: "d1", Kind: RequestDeploy, Service: "cloudflared-instance", StackSlug: "cloudflared-instance",
		Image:    "cloudflare/cloudflared:latest",
		Strategy: "run", Network: "nexul_default", Env: map[string]string{"TUNNEL_TOKEN": "tok"},
		Command: []string{"tunnel", "--no-autoupdate", "run"},
	}
	e.Deploy(context.Background(), req, rec.send)
	assert.Equal(t, DeployStatusHealthy, rec.last().Status)
	assert.Equal(t, []string{
		"run", "-d", "--restart", "unless-stopped", "--name", "cloudflared-instance", "--network", "nexul_default",
		"-e", "TUNNEL_TOKEN=tok", "cloudflare/cloudflared:latest", "tunnel", "--no-autoupdate", "run",
	}, cmd.argsFor("docker")[3])
}

func TestShellExecutor_Deploy_ErrorPaths(t *testing.T) {
	t.Run("pull failure reports failed result", func(t *testing.T) {
		cmd := &fakeCmd{failNext: 1, err: assert.AnError}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Deploy(context.Background(), DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Image: "img"}, rec.send)
		assert.Equal(t, FrameDeployResult, rec.last().Type)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
		assert.NotEmpty(t, rec.last().Error)
	})

	t.Run("start failure reports failed result", func(t *testing.T) {
		cmd := &fakeCmd{failNext: 2, err: assert.AnError}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Deploy(context.Background(), DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Image: "img", Service: "s"}, rec.send)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
	})

	t.Run("unsupported strategy reports failed result", func(t *testing.T) {
		cmd := &fakeCmd{}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Deploy(context.Background(), DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Image: "img", Strategy: "kube"}, rec.send)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
		assert.Contains(t, rec.last().Error, "unsupported")
	})

	t.Run("panic during deploy becomes a failed result", func(t *testing.T) {
		cmd := &fakeCmd{panicOn: "docker"}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Deploy(context.Background(), DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Image: "img"}, rec.send)
		assert.Equal(t, FrameDeployResult, rec.last().Type)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
	})
}

func TestShellExecutor_Deploy_HappyPaths(t *testing.T) {
	t.Run("run strategy pulls and starts with env", func(t *testing.T) {
		cmd := &fakeCmd{stdout: "Status: Downloaded newer image\n"}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", StackSlug: "api", Image: "img:1", Env: map[string]string{"B": "2", "A": "1"}, Strategy: "run"}
		e.Deploy(context.Background(), req, rec.send)

		frames := rec.lifecycle()
		require.Len(t, frames, 4)
		assert.Equal(t, FrameDeployProgress, frames[0].Type)
		assert.Equal(t, DeployPhasePulling, frames[0].Phase)
		assert.Equal(t, FrameDeployProgress, frames[1].Type)
		assert.Equal(t, DeployPhaseStarting, frames[1].Phase)
		assert.Equal(t, FrameDeployProgress, frames[2].Type)
		assert.Equal(t, DeployPhaseHealthy, frames[2].Phase)
		assert.Equal(t, FrameDeployResult, frames[3].Type)
		assert.Equal(t, DeployStatusHealthy, frames[3].Status)

		assert.Equal(t, []string{"pull", "img:1"}, cmd.argsFor("docker")[0])
		assert.Equal(t, []string{"rm", "-f", "api"}, cmd.argsFor("docker")[1])
		assert.Equal(t, []string{"run", "-d", "--restart", "unless-stopped", "--name", "api", "-e", "A=1", "-e", "B=2", "img:1"}, cmd.argsFor("docker")[2])

		// The pull streams under the deploy phase; the run label names the container and image, never the env.
		assert.Equal(t, "docker pull img:1\nStatus: Downloaded newer image\ndocker run --name api img:1\n", rec.logText(LogPhaseDeploy))
		assert.NotContains(t, rec.logText(LogPhaseDeploy), "A=1")
		assert.Equal(t, FrameDeployResult, rec.all()[len(rec.all())-1].Type)
	})

	t.Run("run strategy publishes ports", func(t *testing.T) {
		cmd := &fakeCmd{}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "proxy", StackSlug: "proxy", Image: "traefik:v3", Ports: []string{"80:80", "443:443"}, Strategy: "run"}
		e.Deploy(context.Background(), req, rec.send)
		assert.Equal(t, DeployStatusHealthy, rec.last().Status)
		assert.Equal(t, []string{"rm", "-f", "proxy"}, cmd.argsFor("docker")[1])
		assert.Equal(t, []string{"run", "-d", "--restart", "unless-stopped", "--name", "proxy", "-p", "80:80", "-p", "443:443", "traefik:v3"}, cmd.argsFor("docker")[2])
	})

	// Compose always needs its repo's compose file, so a plain pre-built-image deploy only ever supports the
	// run strategy; a compose stack always goes through Build, which owns the persistent checkout.
	t.Run("compose strategy without a repo is rejected", func(t *testing.T) {
		cmd := &fakeCmd{}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Deploy(context.Background(), DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Image: "img", Service: "s", StackSlug: "s", Strategy: "compose"}, rec.send)
		assert.Equal(t, DeployStatusFailed, rec.last().Status)
		assert.Contains(t, rec.last().Error, "unsupported")
	})

	t.Run("default strategy is docker run", func(t *testing.T) {
		cmd := &fakeCmd{}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		e.Deploy(context.Background(), DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Image: "img", Service: "s", StackSlug: "s"}, rec.send)
		assert.Equal(t, DeployStatusHealthy, rec.last().Status)
	})
}

func TestShellExecutor_Deploy_ReportsServices(t *testing.T) {
	t.Run("run strategy inspects the container and builds a services report", func(t *testing.T) {
		cmd := &fakeCmd{inspectByName: map[string]string{
			"api": `{"Config":{"Image":"img:1"},"State":{"Status":"running"},"NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.18.0.4"}},"Ports":{"80/tcp":[{"HostPort":"8080"}]}}}`,
		}}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", StackSlug: "api", Image: "img:1", Strategy: "run"}
		e.Deploy(context.Background(), req, rec.send)

		assert.Equal(t, DeployStatusHealthy, rec.last().Status)
		assert.Equal(t, "172.18.0.4", rec.last().Address)
		require.Len(t, rec.last().Services, 1)
		assert.Equal(t, ObservedService{
			Name: "api", ContainerName: "api", Image: "img:1", Status: "running",
			Networks: []ObservedNetwork{{Name: "bridge", Address: "172.18.0.4"}},
			Ports:    []string{"8080:80/tcp"},
		}, rec.last().Services[0])
		inspectArgs := cmd.argsFor("docker")[len(cmd.argsFor("docker"))-1]
		assert.Equal(t, []string{"inspect", "--format", "{{json .}}", "api"}, inspectArgs)
	})

	t.Run("a named network selects that network's address as primary", func(t *testing.T) {
		cmd := &fakeCmd{inspectByName: map[string]string{
			"api": `{"State":{"Status":"running"},"NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.18.0.2"},"app-net":{"IPAddress":"172.20.0.5"}}}}`,
		}}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", StackSlug: "api", Image: "img:1", Strategy: "run", Network: "app-net"}
		e.Deploy(context.Background(), req, rec.send)

		assert.Equal(t, "172.20.0.5", rec.last().Address)
	})

	t.Run("a healthy check promotes the reported status", func(t *testing.T) {
		cmd := &fakeCmd{inspectByName: map[string]string{
			"api": `{"State":{"Status":"running","Health":{"Status":"healthy"}},"NetworkSettings":{"Networks":{}}}`,
		}}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", StackSlug: "api", Image: "img:1", Strategy: "run"}
		e.Deploy(context.Background(), req, rec.send)

		require.Len(t, rec.last().Services, 1)
		assert.Equal(t, "healthy", rec.last().Services[0].Status)
	})

	t.Run("an inspect failure still reports a healthy deploy with an empty report", func(t *testing.T) {
		cmd := &fakeCmd{inspectErr: assert.AnError}
		e := newTestExecutor(cmd.run)
		rec := &frameRecorder{}
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, Service: "api", StackSlug: "api", Image: "img:1", Strategy: "run"}
		e.Deploy(context.Background(), req, rec.send)

		assert.Equal(t, DeployStatusHealthy, rec.last().Status)
		assert.Empty(t, rec.last().Address)
		assert.Empty(t, rec.last().Services)
	})
}

// withNexulOnPath makes lookPathFn resolve the nexul command to path, or fail when path is empty.
func withNexulOnPath(t *testing.T, path string) {
	t.Helper()
	orig := lookPathFn
	lookPathFn = func(string) (string, error) {
		if path == "" {
			return "", errors.New("not found")
		}
		return path, nil
	}
	t.Cleanup(func() { lookPathFn = orig })
}

func TestShellExecutor_Upgrade(t *testing.T) {
	t.Run("starts nexul upgrade in its own systemd unit and reports started", func(t *testing.T) {
		withNexulOnPath(t, "/usr/local/bin/nexul")
		cmd := &fakeCmd{}
		e := newTestExecutor(cmd.run)
		var frames []Frame
		send := func(fr Frame) error { frames = append(frames, fr); return nil }

		err := e.Upgrade(t.Context(), Frame{Type: FrameAssignUpgrade, ID: "up-1", Version: "v0.2.2"}, send)
		require.NoError(t, err)

		assert.Equal(t, [][]string{{"--unit", "nexul-upgrade", "--collect", "--quiet", "/usr/local/bin/nexul", "upgrade", "--version", "v0.2.2"}}, cmd.argsFor("systemd-run"))
		require.Len(t, frames, 2)
		assert.Equal(t, FrameUpgradeProgress, frames[0].Type)
		assert.Equal(t, "started nexul-upgrade.service", frames[0].Log)
		assert.Equal(t, FrameUpgradeResult, frames[1].Type)
		assert.Equal(t, UpgradeStatusStarted, frames[1].Status)
	})

	t.Run("a host without the nexul command fails without running anything", func(t *testing.T) {
		withNexulOnPath(t, "")
		cmd := &fakeCmd{}
		e := newTestExecutor(cmd.run)
		var frames []Frame
		send := func(fr Frame) error { frames = append(frames, fr); return nil }

		require.NoError(t, e.Upgrade(t.Context(), Frame{Type: FrameAssignUpgrade, ID: "up-2", Version: "v1"}, send))

		require.Len(t, frames, 1)
		assert.Equal(t, UpgradeStatusFailed, frames[0].Status)
		assert.Contains(t, frames[0].Error, "nexul install")
		assert.Empty(t, cmd.names())
	})

	t.Run("a systemd-run failure reports upgrade_result failed with the error text", func(t *testing.T) {
		withNexulOnPath(t, "/usr/local/bin/nexul")
		cmd := &fakeCmd{failNext: 1, err: errors.New("Unit nexul-upgrade.service already exists")}
		e := newTestExecutor(cmd.run)
		var frames []Frame
		send := func(fr Frame) error { frames = append(frames, fr); return nil }

		require.NoError(t, e.Upgrade(t.Context(), Frame{Type: FrameAssignUpgrade, ID: "up-3", Version: "v1"}, send))

		require.Len(t, frames, 1)
		assert.Equal(t, UpgradeStatusFailed, frames[0].Status)
		assert.Equal(t, "Unit nexul-upgrade.service already exists", frames[0].Error)
	})

	t.Run("a send failure stops before the result frame", func(t *testing.T) {
		withNexulOnPath(t, "/usr/local/bin/nexul")
		e := newTestExecutor((&fakeCmd{}).run)
		sendErr := errors.New("write failed")
		sent := 0
		send := func(fr Frame) error { sent++; return sendErr }

		err := e.Upgrade(t.Context(), Frame{Type: FrameAssignUpgrade, ID: "up-4", Version: "v1"}, send)

		assert.Equal(t, sendErr, err)
		assert.Equal(t, 1, sent)
	})
}

func TestTailString(t *testing.T) {
	assert.Equal(t, "abc", tailString("  abc\n", 10))
	big := "0123456789abcdef"
	got := tailString(big, 8)
	assert.Equal(t, 11, len(got), "3-byte ellipsis plus the 8-byte tail")
	assert.Contains(t, got, "89abcdef")
	assert.Equal(t, "", tailString("   \n", 4))
}
