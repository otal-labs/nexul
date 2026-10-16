package pairing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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

func (f *fakeContainers) OpenComputerLogs(_ context.Context, computerID, container string, _ int, _, mask bool) (containerlog.Feed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, "logs "+computerID+" "+container)
	if mask {
		f.requests[len(f.requests)-1] += " masked"
	}
	if f.err != nil {
		return nil, f.err
	}
	return &fakeLines{lines: f.lines}, nil
}

func (f *fakeContainers) OpenRunnerLogs(_ context.Context, computerID string, tail int, _ bool) (containerlog.Feed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, fmt.Sprintf("runner %s %d", computerID, tail))
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

// fakeGate is the computer rules of the access domain: held[userID] is what the owner's rule gives that person.
type fakeGate struct {
	owner string
	held  map[string]permissions.Set
	err   error
}

func (g fakeGate) ComputerPermissions(_ context.Context, userID, _ string) (string, permissions.Set, error) {
	if g.err != nil {
		return "", nil, g.err
	}
	if userID == g.owner {
		return g.owner, permissions.SetOf(permissions.ComputerActions...), nil
	}
	held, ok := g.held[userID]
	if !ok {
		return "", nil, apperrs.ErrNotFound
	}
	return g.owner, held, nil
}

// newSharedFixture is newContainersFixture with alice's laptop shared by rule: bob holds Read logs, carol See alone.
func newSharedFixture(t *testing.T, docker *DockerFacts, gate fakeGate) (*Service, *fakeContainers) {
	t.Helper()
	svc, containers := newContainersFixture(t, docker)
	svc.computerAccess = gate
	return svc, containers
}

func aliceShares() fakeGate {
	return fakeGate{owner: "u-alice", held: map[string]permissions.Set{
		"u-bob":   permissions.SetOf(permissions.ComputersSee, permissions.ComputersLogs),
		"u-carol": permissions.SetOf(permissions.ComputersSee),
	}}
}

// TestComputerLogs_OwnerOrAShareAllowingReadLogs: the owner reads unmasked, a person Read logs reaches reads it masked, and
// everyone else, a See-only sharee and a stranger included, gets not found before anything reaches the computer.
func TestComputerLogs_OwnerOrAShareAllowingReadLogs(t *testing.T) {
	t.Parallel()
	available := &DockerFacts{Available: true}
	tests := []struct {
		name   string
		user   string
		docker *DockerFacts
		gate   fakeGate
		want   error
		asked  []string
	}{
		{"the owner is unmasked", "u-alice", available, aliceShares(), nil, []string{"logs c-laptop shop-web-1", "runner c-laptop 5"}},
		{"a sharee with Read logs is masked", "u-bob", available, aliceShares(), nil, []string{"logs c-laptop shop-web-1 masked", "runner c-laptop 5"}},
		{"a sharee with See alone", "u-carol", available, aliceShares(), apperrs.ErrNotFound, nil},
		{"a stranger", "u-dave", available, aliceShares(), apperrs.ErrNotFound, nil},
		{"a sharee when the share cannot be read", "u-bob", available, fakeGate{err: apperrs.ErrNotFound}, apperrs.ErrNotFound, nil},
		{"a sharee with Read logs on a computer without Docker is told so, the runner log still opens", "u-bob", nil, aliceShares(), apperrs.ErrConflict, []string{"runner c-laptop 5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, containers := newSharedFixture(t, tt.docker, tt.gate)
			_, containerErr := svc.ContainerLogSnapshot(t.Context(), tt.user, "c-laptop", "shop-web-1", 5)
			_, runnerErr := svc.RunnerLogSnapshot(t.Context(), tt.user, "c-laptop", 5)
			if tt.want == nil {
				require.NoError(t, containerErr)
				require.NoError(t, runnerErr)
				assert.Equal(t, tt.asked, containers.asked())
				return
			}
			require.ErrorIs(t, containerErr, tt.want)
			if tt.want == apperrs.ErrConflict {
				require.NoError(t, runnerErr)
			} else {
				require.ErrorIs(t, runnerErr, tt.want)
			}
			assert.Equal(t, tt.asked, containers.asked())
		})
	}
}

func TestComputerLogs_WithoutAGateOnlyTheOwnerReads(t *testing.T) {
	t.Parallel()
	svc, containers := newContainersFixture(t, &DockerFacts{Available: true})
	_, err := svc.RunnerLogSnapshot(t.Context(), "u-bob", "c-laptop", 5)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = svc.ListContainers(t.Context(), "u-bob", "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Empty(t, containers.asked())
}

func TestComputerLogs_ErrorsBeforeReachingTheComputer(t *testing.T) {
	t.Parallel()
	svc, containers := newSharedFixture(t, &DockerFacts{Available: true}, aliceShares())

	_, err := svc.RunnerLogs(t.Context(), "u-alice", "c-laptop", -1, false)
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	svc.computerAccess = fakeGate{err: io.ErrUnexpectedEOF}
	_, err = svc.RunnerLogs(t.Context(), "u-bob", "c-laptop", 5, false)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.NotErrorIs(t, err, apperrs.ErrNotFound, "a failed read of the share is not a refusal")

	svc.containers = nil
	_, err = svc.RunnerLogs(t.Context(), "u-alice", "c-laptop", 5, false)
	require.ErrorIs(t, err, apperrs.ErrFatal)
	assert.Empty(t, containers.asked())
}

func TestListContainers_ForASharee(t *testing.T) {
	t.Parallel()
	svc, containers := newSharedFixture(t, &DockerFacts{Available: true}, aliceShares())
	got, err := svc.ListContainers(t.Context(), "u-bob", "c-laptop")
	require.NoError(t, err)
	assert.Len(t, got, 1)
	_, err = svc.ListContainers(t.Context(), "u-carol", "c-laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, []string{"list c-laptop"}, containers.asked())
}

func TestComputerList_RunnerLogsAndAShare(t *testing.T) {
	t.Parallel()
	svc, containers := newSharedFixture(t, &DockerFacts{Available: true}, aliceShares())
	laptop := Computer{ID: "c-laptop", Name: "Laptop", Shared: &SharedComputer{OwnerID: "u-alice", OwnerName: "Alice", Permissions: permissions.SetOf(permissions.ComputersSee, permissions.ComputersLogs)}}
	svc.repo.(*fakeRepo).sharedWith = map[string][]Computer{"u-bob": {laptop}, "u-carol": {{ID: "c-laptop", Name: "Laptop", Shared: &SharedComputer{OwnerID: "u-alice", OwnerName: "Alice", Permissions: permissions.SetOf(permissions.ComputersSee)}}}}
	svc.repo.(*fakeRepo).computers["c-laptop"] = Computer{ID: "c-laptop", UserID: "u-alice", Name: "Laptop", Facts: &Facts{Docker: &DockerFacts{Available: true}}}

	t.Run("runner_logs reads the last lines for the owner", func(t *testing.T) {
		out, err := callTool(t, actorCtx(t, "u-alice"), svc, "computer_list", `{"id":"c-laptop","runner_logs":{"lines":7}}`)
		require.NoError(t, err)
		raw, _ := json.Marshal(out)
		assert.Contains(t, string(raw), `"runner_logs":{"lines":[{"ts":"2026-10-11T10:00:00Z","stream":"stdout","line":"listening on :8080"}]}`)
		assert.Contains(t, containers.asked(), "runner c-laptop 7")
	})

	t.Run("a sharee with Read logs lists the containers and reads both logs", func(t *testing.T) {
		out, err := callTool(t, actorCtx(t, "u-bob"), svc, "computer_list", `{"id":"c-laptop","runner_logs":{},"container_logs":{"container":"shop-web-1"}}`)
		require.NoError(t, err)
		raw, _ := json.Marshal(out)
		assert.Contains(t, string(raw), `"containers":[{"name":"shop-web-1"`)
		assert.Contains(t, string(raw), `"runner_logs"`)
		assert.Contains(t, containers.asked(), "runner c-laptop 200")
		assert.Contains(t, containers.asked(), "logs c-laptop shop-web-1 masked")
	})

	t.Run("a sharee with See alone is refused both reads and sees no containers", func(t *testing.T) {
		before := len(containers.asked())
		_, err := callTool(t, actorCtx(t, "u-carol"), svc, "computer_list", `{"id":"c-laptop","runner_logs":{}}`)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		_, err = callTool(t, actorCtx(t, "u-carol"), svc, "computer_list", `{"id":"c-laptop","container_logs":{"container":"shop-web-1"}}`)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		out, err := callTool(t, actorCtx(t, "u-carol"), svc, "computer_list", `{"id":"c-laptop"}`)
		require.NoError(t, err)
		raw, _ := json.Marshal(out)
		assert.NotContains(t, string(raw), "shop-web-1")
		assert.Len(t, containers.asked(), before)
	})

	t.Run("runner_logs needs the computer's id", func(t *testing.T) {
		_, err := callTool(t, actorCtx(t, "u-alice"), svc, "computer_list", `{"runner_logs":{}}`)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("an offline runner is a conflict the agent can act on", func(t *testing.T) {
		containers.err = apperrs.Retryable(io.ErrUnexpectedEOF)
		_, err := callTool(t, actorCtx(t, "u-alice"), svc, "computer_list", `{"id":"c-laptop","runner_logs":{}}`)
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})
}

func TestComputerList_NoDockerIsNoContainersLineForOwnerAndSharee(t *testing.T) {
	t.Parallel()
	svc, containers := newSharedFixture(t, nil, aliceShares())
	laptop := Computer{ID: "c-laptop", Name: "Laptop", Shared: &SharedComputer{OwnerID: "u-alice", OwnerName: "Alice", Permissions: permissions.SetOf(permissions.ComputersSee, permissions.ComputersLogs)}}
	svc.repo.(*fakeRepo).sharedWith = map[string][]Computer{"u-bob": {laptop}}
	for _, user := range []string{"u-alice", "u-bob"} {
		out, err := callTool(t, actorCtx(t, user), svc, "computer_list", `{"id":"c-laptop"}`)
		require.NoError(t, err)
		raw, _ := json.Marshal(out)
		assert.NotContains(t, string(raw), "containers", user)
	}
	assert.Empty(t, containers.asked(), "no Docker, nothing reached the computer")
}
