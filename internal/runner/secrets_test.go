package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

func secretExecutor(cmd *fakeCmd, resolvers ...secretref.Resolver) *ShellExecutor {
	return NewShellExecutor(cmd.run, ExecutorConfig{
		Registry: func() secretref.Registry { return secretref.NewRegistry(resolvers...) },
	}, testLogger())
}

// everything a frame carries, so a leak anywhere in the stream is one assertion.
func frameText(rec *frameRecorder) string {
	var all string
	for _, f := range rec.all() {
		all += f.Log + "\n" + f.Error + "\n" + string(f.Phase) + "\n"
	}
	return all
}

// A reference that cannot resolve must stop the deploy before the first command, name the variable and a plain
// reason, and never the reference or a value, whichever way the deploy comes in.
func TestShellExecutor_UnresolvedReference_StopsBeforeAnyCommand(t *testing.T) {
	failing := &secretreftest.Fake{
		Values: map[string]string{"op://Acme/web/token": "tok-live-9"},
		Errs: map[string]error{
			"op://Acme/db/password": secretref.ErrFieldMissing,
			"op://Acme/db/host":     secretref.ErrNoAccess,
		},
	}
	env := map[string]string{
		"PLAIN": "hello", "DB_PASSWORD": "op://Acme/db/password", "DB_HOST": "op://Acme/db/host", "TOKEN": "op://Acme/web/token",
	}
	deploy := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, StackSlug: "api", Image: "img:1", Strategy: "run", Env: env}
	build := buildRequest(t, "b1")
	build.Env = env
	compose := buildRequest(t, "b2")
	compose.Strategy, compose.ComposePath, compose.Env = "compose", "docker-compose.yml", env

	tests := []struct {
		name     string
		resolver []secretref.Resolver
		run      func(e *ShellExecutor, rec *frameRecorder)
		want     string
	}{
		{"deploy", []secretref.Resolver{failing}, func(e *ShellExecutor, rec *frameRecorder) { e.Deploy(context.Background(), deploy, rec.send) },
			"DB_HOST: no access; DB_PASSWORD: field not found"},
		{"run build", []secretref.Resolver{failing}, func(e *ShellExecutor, rec *frameRecorder) { e.Build(context.Background(), build, rec.send) },
			"DB_HOST: no access; DB_PASSWORD: field not found"},
		{"compose build", []secretref.Resolver{failing}, func(e *ShellExecutor, rec *frameRecorder) { e.Build(context.Background(), compose, rec.send) },
			"DB_HOST: no access; DB_PASSWORD: field not found"},
		{"no token on the runner", nil, func(e *ShellExecutor, rec *frameRecorder) { e.Deploy(context.Background(), deploy, rec.send) },
			"DB_HOST: no 1Password token on this runner; DB_PASSWORD: no 1Password token on this runner; TOKEN: no 1Password token on this runner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &fakeCmd{}
			rec := &frameRecorder{}
			tt.run(secretExecutor(cmd, tt.resolver...), rec)

			assert.Empty(t, cmd.names(), "no pull, clone, build, compose or run before the references resolve")
			last := rec.last()
			assert.Equal(t, FrameDeployResult, last.Type)
			assert.Equal(t, DeployStatusFailed, last.Status)
			assert.Contains(t, last.Error, tt.want)
			assert.NotContains(t, last.Error, "PLAIN")
			text := frameText(rec)
			for _, leak := range []string{"op://", "Acme", "tok-live-9", "hello"} {
				assert.NotContains(t, text, leak)
			}
		})
	}
}

// A mixed env keeps its literals and gets values for its references, through the same .env and -e paths literals use,
// one batch per manager, with a log line that counts and never names.
func TestShellExecutor_MixedEnv_SubstitutesReferencesAndLeavesLiterals(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{
		"op://Acme/db/password": "s3cret-pw", "op://Acme/web/token": "tok-live-9",
	}}
	env := map[string]string{
		"PLAIN": "hello", "URL": "postgres://example.com/db",
		"DB_PASSWORD": "op://Acme/db/password", "TOKEN": "op://Acme/web/token", "SAME": "op://Acme/db/password",
	}

	t.Run("run stack", func(t *testing.T) {
		cmd := &fakeCmd{}
		rec := &frameRecorder{}
		fake := &secretreftest.Fake{Values: fake.Values}
		req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, StackSlug: "api", Image: "img:1", Strategy: "run", Env: env}
		secretExecutor(cmd, fake).Deploy(context.Background(), req, rec.send)

		assert.Equal(t, DeployStatusHealthy, rec.last().Status)
		run := cmd.argsFor("docker")[2]
		assert.Equal(t, []string{
			"-e", "DB_PASSWORD=s3cret-pw", "-e", "PLAIN=hello", "-e", "SAME=s3cret-pw", "-e", "TOKEN=tok-live-9", "-e", "URL=postgres://example.com/db", "img:1",
		}, run[len(run)-11:])
		require.Len(t, fake.Calls(), 1, "one batch for the manager")
		assert.Len(t, fake.Calls()[0], 2, "a reference used twice is asked for once")
		assert.Equal(t, "op://Acme/db/password", env["DB_PASSWORD"], "the request's own env keeps its references")
		assert.Regexp(t, regexp.MustCompile(`^Resolved 3 secret references in \d+\.\ds\n`), rec.logText(LogPhaseDeploy))
		for _, leak := range []string{"op://", "s3cret-pw", "tok-live-9"} {
			assert.NotContains(t, frameText(rec), leak)
		}
	})

	t.Run("compose stack", func(t *testing.T) {
		cmd := &fakeCmd{}
		rec := &frameRecorder{}
		req := buildRequest(t, "b1")
		req.Strategy, req.ComposePath, req.Env = "compose", "docker-compose.yml", env
		secretExecutor(cmd, &secretreftest.Fake{Values: fake.Values}).Build(context.Background(), req, rec.send)

		assert.Equal(t, DeployStatusHealthy, rec.last().Status)
		raw, err := os.ReadFile(filepath.Join(req.StackRoot, "stacks", "api", "repo", ".env"))
		require.NoError(t, err)
		assert.Equal(t, "DB_PASSWORD=s3cret-pw\nPLAIN=hello\nSAME=s3cret-pw\nTOKEN=tok-live-9\nURL=postgres://example.com/db\n", string(raw))
		info, err := os.Stat(filepath.Join(req.StackRoot, "stacks", "api", "repo", ".env"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		assert.Contains(t, rec.logText(LogPhaseDeploy), "Resolved 3 secret references in ")
		for _, leak := range []string{"op://", "s3cret-pw", "tok-live-9"} {
			assert.NotContains(t, frameText(rec), leak)
		}
	})
}

// A command that echoes a resolved value, in its output or its error, must not carry it into the deploy log or the
// terminal error.
func TestShellExecutor_ResolvedValue_IsMaskedInCommandOutputAndErrors(t *testing.T) {
	cmd := &fakeCmd{failNext: 1, err: errors.New("docker run -e DB_PASSWORD=s3cret-pw: exit 125"), stdout: "connecting with s3cret-pw\n"}
	rec := &frameRecorder{}
	req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, StackSlug: "api", Image: "img:1", Strategy: "run",
		Env: map[string]string{"DB_PASSWORD": "op://Acme/db/password"}}
	e := secretExecutor(cmd, &secretreftest.Fake{Values: map[string]string{"op://Acme/db/password": "s3cret-pw"}})

	e.Deploy(context.Background(), req, rec.send)

	assert.Equal(t, DeployStatusFailed, rec.last().Status)
	assert.NotContains(t, frameText(rec), "s3cret-pw")
	assert.Contains(t, rec.logText(LogPhaseDeploy), "connecting with ***")
}

// The mask is for the deploy log and errors only: a resolved value equal to a port must not rewrite the inspect output
// the observation report is parsed from.
func TestShellExecutor_ResolvedValue_NeverRewritesWhatObserveParses(t *testing.T) {
	cmd := &fakeCmd{inspectOut: `{"Config":{"Image":"img:1"},"State":{"Status":"running"},"NetworkSettings":{"Networks":{"app-net":{"IPAddress":"172.18.0.4"}},"Ports":{"80/tcp":[{"HostPort":"8080"}]}}}`}
	rec := &frameRecorder{}
	req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, StackSlug: "api", Image: "img:1", Strategy: "run",
		Env: map[string]string{"PORT": "op://Acme/web/port"}}
	e := secretExecutor(cmd, &secretreftest.Fake{Values: map[string]string{"op://Acme/web/port": "8080"}})

	e.Deploy(context.Background(), req, rec.send)

	require.Equal(t, DeployStatusHealthy, rec.last().Status)
	require.Len(t, rec.last().Services, 1)
	assert.Equal(t, []string{"8080:80/tcp"}, rec.last().Services[0].Ports)
	assert.Equal(t, "172.18.0.4", rec.last().Address)
}

// A value that starts another must not leave the rest of the longer one showing, and each line of a multi-line value
// is hidden on its own, because a command's output reaches the log line by line.
func TestShellExecutor_ResolvedValues_AreMaskedWhole_PrefixesAndEachLine(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		stdout string
		want   string
		never  string
	}{
		{"a value that starts another", map[string]string{"op://Acme/x/a": "abc", "op://Acme/x/b": "abcdef"},
			"saw abcdef and abc\n", "saw *** and ***", "def"},
		{"each line of a multi-line value", map[string]string{"op://Acme/x/a": "-----BEGIN KEY-----\nMIIEvQIBADAN\n-----END KEY-----"},
			"key line: MIIEvQIBADAN\n", "key line: ***", "MIIEvQIBADAN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &fakeCmd{failNext: 1, err: errors.New("exit 125"), stdout: tt.stdout}
			rec := &frameRecorder{}
			env := map[string]string{}
			for ref := range tt.values {
				env["V"+ref[len(ref)-1:]] = ref
			}
			req := DeployRequestedEvent{ID: "d1", Kind: RequestDeploy, StackSlug: "api", Image: "img:1", Strategy: "run", Env: env}

			secretExecutor(cmd, &secretreftest.Fake{Values: tt.values}).Deploy(context.Background(), req, rec.send)

			assert.Contains(t, rec.logText(LogPhaseDeploy), tt.want)
			assert.NotContains(t, frameText(rec), tt.never)
		})
	}
}
