package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// fakeLogSource answers OpenLogs with canned lines, recording what the use-case asked the runner for.
type fakeLogSource struct {
	lines []ContainerLogLine
	err   error

	machine, container string
	tail               int
	follow             bool
	closed             bool
}

func (f *fakeLogSource) OpenLogs(_ context.Context, machine, container string, tail int, follow bool) (LogFeed, error) {
	f.machine, f.container, f.tail, f.follow = machine, container, tail, follow
	if f.err != nil {
		return nil, f.err
	}
	return &fakeFeed{src: f, lines: f.lines}, nil
}

type fakeFeed struct {
	src   *fakeLogSource
	lines []ContainerLogLine
}

func (f *fakeFeed) Next(context.Context) ([]ContainerLogLine, error) {
	if f.lines == nil {
		return nil, io.EOF
	}
	out := f.lines
	f.lines = nil
	return out, nil
}

func (f *fakeFeed) Close() { f.src.closed = true }

func newLogsService(t *testing.T, src *fakeLogSource, env map[string]string, containers ...Container) *Service {
	t.Helper()
	stacks := newFakeStackRepo()
	stack := validStack()
	stack.Env = env
	requireStack(t, stacks, stack)
	repo := newFakeContainerRepo()
	for _, c := range containers {
		c.StackID = stack.ID
		require.NoError(t, repo.Create(t.Context(), &c))
	}
	s := newTestServiceWith(newFakeRepo(), stacks, repo, newFakeProjects(), newFakeBus())
	s.SetLogSource(src)
	return s
}

func TestServiceLogSnapshot_MasksTheStacksEnvValues(t *testing.T) {
	src := &fakeLogSource{lines: []ContainerLogLine{
		{TS: "t1", Stream: "stdout", Line: "connecting to postgres://app:hunter22@db/app"},
		{TS: "t2", Stream: "stderr", Line: "token sk_live_abcdef rejected, retrying on port 8080"},
		{TS: "t3", Stream: "stdout", Line: "user hunter222 signed in with s3cr3t from abcde"},
	}}
	env := map[string]string{
		"DB_PASSWORD": "hunter22",
		"DB_PASS_OLD": "hunter222",
		"API_KEY":     "sk_live_abcdef",
		"PORT":        "8080",
		"SIX":         "s3cr3t",
		"FIVE":        "abcde",
		"EMPTY":       "",
	}
	s := newLogsService(t, src, env, Container{ID: "c1", Name: "web", ContainerName: "api-web-1"})

	lines, err := s.ServiceLogSnapshot(t.Context(), "svc-1", "web", 50)
	require.NoError(t, err)
	assert.Equal(t, []ContainerLogLine{
		{TS: "t1", Stream: "stdout", Line: "connecting to postgres://app:••••@db/app"},
		{TS: "t2", Stream: "stderr", Line: "token •••• rejected, retrying on port 8080"},
		{TS: "t3", Stream: "stdout", Line: "user •••• signed in with •••• from abcde"},
	}, lines, "values of six or more characters are masked, the longest whole; shorter ones stay")
	assert.Equal(t, "10.0.0.1:22", src.machine)
	assert.Equal(t, "api-web-1", src.container)
	assert.Equal(t, 50, src.tail)
	assert.False(t, src.follow)
	assert.True(t, src.closed, "the snapshot closes its stream")
}

func TestServiceLogs_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		service string
		tail    int
		src     *fakeLogSource
		want    error
	}{
		{"a service the stack does not have", "worker", 10, &fakeLogSource{}, apperrs.ErrNotFound},
		{"a service that never started a container", "migrate", 10, &fakeLogSource{}, apperrs.ErrConflict},
		{"a negative tail", "web", -1, &fakeLogSource{}, apperrs.ErrInvalid},
		{"no runner on the machine", "web", 10, &fakeLogSource{err: apperrs.ErrRetryable}, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newLogsService(t, tt.src, nil,
				Container{ID: "c1", Name: "web", ContainerName: "api-web-1"},
				Container{ID: "c2", Name: "migrate", Status: ServiceStatusPending})
			_, err := s.ServiceLogs(t.Context(), "svc-1", tt.service, tt.tail, true)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

// denyLogsGate holds stacks:read for everyone and refuses stacks:logs.
type denyLogsGate struct{ allowGate }

func (denyLogsGate) RequireProject(_ context.Context, _ string, action permissions.Action) error {
	if action == permissions.StacksLogs {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
	}
	return nil
}

func TestStackGet_Logs(t *testing.T) {
	web := Container{ID: "c1", Name: "web", ContainerName: "api-web-1"}

	t.Run("returns the service's lines with the stack's env values masked", func(t *testing.T) {
		src := &fakeLogSource{lines: []ContainerLogLine{
			{TS: "t1", Stream: "stdout", Line: "listening on 8080"},
			{TS: "t2", Stream: "stderr", Line: "token sk_live_abcdef rejected"},
		}}
		s := newLogsService(t, src, map[string]string{"API_KEY": "sk_live_abcdef"}, web)

		got, err := call(t, s, "stack_get", `{"id":"svc-1","logs":{"service":"web"}}`)
		require.NoError(t, err)
		assert.JSONEq(t, `{"service":"web","lines":[
			{"ts":"t1","stream":"stdout","line":"listening on 8080"},
			{"ts":"t2","stream":"stderr","line":"token •••• rejected"}]}`, asJSON(t, got.(stackDetail).Logs))
		assert.Equal(t, "api-web-1", src.container)
	})

	t.Run("lines defaults to 200 and is capped at 1000", func(t *testing.T) {
		for _, tt := range []struct {
			args string
			tail int
		}{
			{`{"service":"web"}`, 200},
			{`{"service":"web","lines":25}`, 25},
			{`{"service":"web","lines":50000}`, 1000},
		} {
			src := &fakeLogSource{}
			s := newLogsService(t, src, nil, web)
			_, err := call(t, s, "stack_get", `{"id":"svc-1","logs":`+tt.args+`}`)
			require.NoError(t, err)
			assert.Equal(t, tt.tail, src.tail, tt.args)
		}
	})

	t.Run("a line over 2000 characters is cut and ends in an ellipsis", func(t *testing.T) {
		long := strings.Repeat("é", 2500)
		src := &fakeLogSource{lines: []ContainerLogLine{{Stream: "stdout", Line: long}, {Stream: "stdout", Line: "short"}}}
		s := newLogsService(t, src, nil, web)

		got, err := call(t, s, "stack_get", `{"id":"svc-1","logs":{"service":"web"}}`)
		require.NoError(t, err)
		lines := got.(stackDetail).Logs.Lines
		assert.Equal(t, strings.Repeat("é", 2000)+"…", lines[0].Line)
		assert.Equal(t, "short", lines[1].Line)
	})

	t.Run("without stacks:logs the call fails and the runner is never asked", func(t *testing.T) {
		src := &fakeLogSource{}
		s := newLogsService(t, src, nil, web)
		s.SetGate(denyLogsGate{})

		_, err := call(t, s, "stack_get", `{"id":"svc-1","logs":{"service":"web"}}`)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Empty(t, src.container)

		got, err := call(t, s, "stack_get", `{"id":"svc-1"}`)
		require.NoError(t, err, "the rest of stack_get stays under stacks:read")
		assert.Nil(t, got.(stackDetail).Logs)
	})

	t.Run("an offline runner names the machine", func(t *testing.T) {
		src := &fakeLogSource{err: fmt.Errorf("%w: no runner is connected on machine %q", apperrs.ErrRetryable, "10.0.0.1:22")}
		s := newLogsService(t, src, nil, web)

		_, err := call(t, s, "stack_get", `{"id":"svc-1","logs":{"service":"web"}}`)
		require.ErrorIs(t, err, apperrs.ErrConflict, "a sentinel the adapter passes through, not the hidden retryable")
		assert.Contains(t, err.Error(), "10.0.0.1:22")
	})
}
