package pairing

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeContainers is a computer's runner answering for its Docker, recording every request that reached it.
type fakeContainers struct {
	mu       sync.Mutex
	list     []Container
	lines    []containerlog.Line
	err      error
	requests []string
}

func (f *fakeContainers) ComputerContainers(_ context.Context, computerID string) ([]Container, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, "list "+computerID)
	return f.list, f.err
}

func (f *fakeContainers) OpenComputerLogs(_ context.Context, computerID, container string, tail int, follow bool) (containerlog.Feed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, "logs "+computerID+" "+container)
	if f.err != nil {
		return nil, f.err
	}
	return &fakeLines{lines: f.lines}, nil
}

func (f *fakeContainers) asked() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

type fakeLines struct{ lines []containerlog.Line }

func (f *fakeLines) Next(context.Context) ([]containerlog.Line, error) {
	if f.lines == nil {
		return nil, io.EOF
	}
	out := f.lines
	f.lines = nil
	return out, nil
}

func (f *fakeLines) Close() {}

// newContainersFixture is alice's laptop with docker as its runner last reported it; nil means a runner too old to say.
func newContainersFixture(t *testing.T, docker *DockerFacts) (*Service, *fakeContainers) {
	t.Helper()
	repo := newFakeRepo()
	repo.computers["c-laptop"] = Computer{ID: "c-laptop", UserID: "u-alice", Name: "Laptop", Facts: &Facts{Docker: docker}}
	containers := &fakeContainers{
		list:  []Container{{Name: "shop-web-1", Image: "shop:1", Status: "running"}},
		lines: []containerlog.Line{{TS: "2026-10-11T10:00:00Z", Stream: "stdout", Line: "listening on :8080"}},
	}
	return NewService(Config{Repo: repo, Harnesses: registry(&fakeExchanger{}), EncryptionKey: testEncKey, Tokens: newFakeTokens(), Containers: containers}), containers
}

// TestContainerLogs_RefusedBeforeReachingTheComputer: another person's computer is not found, a computer without
// Docker is a conflict, and a name docker would read as a flag is invalid; none of them reaches the runner.
func TestContainerLogs_RefusedBeforeReachingTheComputer(t *testing.T) {
	t.Parallel()
	available := &DockerFacts{Available: true, Version: "27.3.1"}
	tests := []struct {
		name      string
		docker    *DockerFacts
		user      string
		container string
		tail      int
		want      error
	}{
		{"another person's computer", available, "u-bob", "shop-web-1", 10, apperrs.ErrNotFound},
		{"a runner too old to report docker", nil, "u-alice", "shop-web-1", 10, apperrs.ErrConflict},
		{"docker not reachable", &DockerFacts{}, "u-alice", "shop-web-1", 10, apperrs.ErrConflict},
		{"a name that reads as a flag", available, "u-alice", "--since=1h", 10, apperrs.ErrInvalid},
		{"no name", available, "u-alice", "", 10, apperrs.ErrInvalid},
		{"a negative tail", available, "u-alice", "shop-web-1", -1, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, containers := newContainersFixture(t, tt.docker)
			_, err := svc.ContainerLogs(t.Context(), tt.user, "c-laptop", tt.container, tt.tail, true)
			require.ErrorIs(t, err, tt.want)
			_, err = svc.ContainerLogSnapshot(t.Context(), tt.user, "c-laptop", tt.container, tt.tail)
			require.ErrorIs(t, err, tt.want)
			assert.Empty(t, containers.asked())
		})
	}
}

func TestListContainers(t *testing.T) {
	t.Parallel()
	svc, containers := newContainersFixture(t, &DockerFacts{Available: true})

	got, err := svc.ListContainers(t.Context(), "u-alice", "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, []Container{{Name: "shop-web-1", Image: "shop:1", Status: "running"}}, got)

	_, err = svc.ListContainers(t.Context(), "u-bob", "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, []string{"list c-laptop"}, containers.asked(), "bob's call never reached the computer")

	containers.err = apperrs.Retryable(io.ErrUnexpectedEOF)
	_, err = svc.ListContainers(t.Context(), "u-alice", "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestContainerLogSnapshot_ReadsUntilTheStreamEnds(t *testing.T) {
	t.Parallel()
	svc, containers := newContainersFixture(t, &DockerFacts{Available: true})
	got, err := svc.ContainerLogSnapshot(t.Context(), "u-alice", "c-laptop", "shop-web-1", 5)
	require.NoError(t, err)
	assert.Equal(t, containers.lines, got, "the person's own output, unmasked")
	assert.Equal(t, []string{"logs c-laptop shop-web-1"}, containers.asked())
}

func TestComputerList_ContainersAndTheirLogs(t *testing.T) {
	t.Parallel()
	svc, _ := newContainersFixture(t, &DockerFacts{Available: true, Version: "27.3.1"})
	ctx := actorCtx(t, "u-alice")

	t.Run("one computer with docker lists its containers", func(t *testing.T) {
		out, err := callTool(t, ctx, svc, "computer_list", `{"id":"c-laptop"}`)
		require.NoError(t, err)
		raw, err := json.Marshal(out)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"containers":[{"name":"shop-web-1","image":"shop:1","status":"running"}]`)
		assert.Contains(t, string(raw), `"docker":{"available":true,"version":"27.3.1"}`)
	})

	t.Run("container_logs reads the last lines", func(t *testing.T) {
		out, err := callTool(t, ctx, svc, "computer_list", `{"id":"c-laptop","container_logs":{"container":"shop-web-1","tail":5}}`)
		require.NoError(t, err)
		raw, err := json.Marshal(out)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"container_logs":{"container":"shop-web-1","lines":[{"ts":"2026-10-11T10:00:00Z","stream":"stdout","line":"listening on :8080"}]}`)
	})

	t.Run("container_logs needs the computer's id", func(t *testing.T) {
		_, err := callTool(t, ctx, svc, "computer_list", `{"container_logs":{"container":"shop-web-1"}}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("an offline runner is a conflict the agent can act on", func(t *testing.T) {
		svc, containers := newContainersFixture(t, &DockerFacts{Available: true})
		containers.err = apperrs.Retryable(io.ErrUnexpectedEOF)
		_, err := callTool(t, ctx, svc, "computer_list", `{"id":"c-laptop","container_logs":{"container":"shop-web-1"}}`)
		require.ErrorIs(t, err, apperrs.ErrConflict)
		out, err := callTool(t, ctx, svc, "computer_list", `{"id":"c-laptop"}`)
		require.NoError(t, err, "the list survives; its containers become a line")
		raw, _ := json.Marshal(out)
		assert.Contains(t, string(raw), `"containers_error":"Laptop didn't list its containers`)
	})
}
