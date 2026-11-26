package pairing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// fakeShells stands in for the computers' runners: it keeps each run's sink until the test ends it.
type fakeShells struct {
	mu        sync.Mutex
	runs      []shelljob.Run
	sinks     map[string]shelljob.Sink
	cancelled []string
	runErr    error
	started   chan string
}

func newFakeShells() *fakeShells {
	return &fakeShells{sinks: map[string]shelljob.Sink{}, started: make(chan string, 16)}
}

func (f *fakeShells) RunShell(_ context.Context, _ string, run shelljob.Run, sink shelljob.Sink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.runErr != nil {
		return f.runErr
	}
	f.runs = append(f.runs, run)
	f.sinks[run.ID] = sink
	f.started <- run.ID
	return nil
}

func (f *fakeShells) OpenTerminal(context.Context, string, string, shelljob.TerminalSize) (shelljob.TerminalStream, error) {
	return nil, errors.New("the computer's Nexul app is not connected")
}

func (f *fakeShells) CancelShell(_ context.Context, jobID string) error {
	f.mu.Lock()
	sink := f.sinks[jobID]
	f.cancelled = append(f.cancelled, jobID)
	f.mu.Unlock()
	if sink != nil {
		sink.Done(shelljob.End{Status: shelljob.Cancelled})
	}
	return nil
}

func (f *fakeShells) sink(id string) shelljob.Sink {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sinks[id]
}

func (f *fakeShells) runCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.runs)
}

func newShellService(t *testing.T) (*Service, *fakeRepo, *fakeShells) {
	t.Helper()
	repo, shells := newFakeRepo(), newFakeShells()
	repo.computers["c-laptop"] = Computer{ID: "c-laptop", UserID: "alice", Name: "laptop", ShellEnabled: true}
	s := NewService(Config{Repo: repo, Shells: shells})
	t.Cleanup(s.shellQueue.Drain)
	return s, repo, shells
}

func waitStarted(t *testing.T, shells *fakeShells) string {
	t.Helper()
	select {
	case id := <-shells.started:
		return id
	case <-time.After(2 * time.Second):
		t.Fatal("no command reached the runner")
		return ""
	}
}

func TestRunCommand_RecordsWhatRanAndHowItEnded(t *testing.T) {
	s, repo, shells := newShellService(t)

	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "go test ./...", Cwd: "~/code/app"})
	require.NoError(t, err)
	assert.Equal(t, shelljob.Queued, job.Status)
	assert.Equal(t, 60, job.TimeoutSeconds)
	assert.Equal(t, shelljob.AdapterWeb, job.Adapter)
	id := waitStarted(t, shells)
	assert.Equal(t, job.ID, id)
	assert.Equal(t, shelljob.Running, repo.shellJob(id).Status)

	shells.sink(id).Output([]containerlog.Line{{Stream: "stdout", Line: "ok  app 0.4s"}, {Stream: "stderr", Line: "warn"}})
	shells.sink(id).Done(shelljob.End{Status: shelljob.Succeeded})

	got, err := s.WaitShellJob(t.Context(), "alice", id)
	require.NoError(t, err)
	assert.Equal(t, shelljob.Succeeded, got.Status)
	require.NotNil(t, got.ExitCode)
	assert.Equal(t, 0, *got.ExitCode)
	assert.Equal(t, "ok  app 0.4s\nwarn\n", got.Output)
	assert.NotNil(t, got.EndedAt)
}

func TestRunCommand_RefusedByTheComputerIsRecordedAsRefused(t *testing.T) {
	s, repo, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "ls"})
	require.NoError(t, err)
	waitStarted(t, shells)

	shells.sink(job.ID).Done(shelljob.End{Status: shelljob.Refused, Error: "commands are turned off on this computer"})

	got := repo.shellJob(job.ID)
	assert.Equal(t, shelljob.Refused, got.Status)
	assert.Nil(t, got.ExitCode)
	assert.Contains(t, got.Error, "turned off")
}

func TestShellJobs_OnlyTheOwnerStartsOrReadsThem(t *testing.T) {
	s, _, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "whoami"})
	require.NoError(t, err)
	waitStarted(t, shells)

	_, err = s.RunCommand(t.Context(), "mallory", "c-laptop", shelljob.Input{Command: "whoami"})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.GetShellJob(t.Context(), "mallory", job.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, _, err = s.ListShellJobs(t.Context(), "mallory", "c-laptop", paging.Window{})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.CancelShellJob(t.Context(), "mallory", job.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.OpenShellOutput(t.Context(), "mallory", job.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, 1, shells.runCount())
}

func TestRunCommand_RunsFourAtATimePerComputer(t *testing.T) {
	s, repo, shells := newShellService(t)
	var ids []string
	for range 5 {
		job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "sleep 1"})
		require.NoError(t, err)
		ids = append(ids, job.ID)
	}
	for range 4 {
		waitStarted(t, shells)
	}
	assert.Equal(t, 4, shells.runCount())
	assert.Equal(t, shelljob.Queued, repo.shellJob(ids[4]).Status)

	shells.sink(ids[0]).Done(shelljob.End{Status: shelljob.Succeeded})

	assert.Equal(t, ids[4], waitStarted(t, shells))
}

func TestRunCommand_TheSwitchRefusesNewOnesAndStopsTheRest(t *testing.T) {
	s, repo, shells := newShellService(t)
	var ids []string
	for range 5 {
		job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "sleep 9"})
		require.NoError(t, err)
		ids = append(ids, job.ID)
	}
	for range 4 {
		waitStarted(t, shells)
	}

	c, err := s.UpdateComputer(t.Context(), "alice", "c-laptop", ComputerPatch{ShellEnabled: new(false)})
	require.NoError(t, err)
	assert.False(t, c.ShellEnabled)

	for _, id := range ids {
		assert.Equal(t, shelljob.Cancelled, repo.shellJob(id).Status, id)
	}
	assert.Len(t, shells.cancelled, 4)
	_, err = s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "ls"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestRunCommand_AComputerThatCannotRunItFailsTheJob(t *testing.T) {
	s, repo, shells := newShellService(t)
	shells.runErr = apperrs.ErrRetryable
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "ls"})
	require.NoError(t, err)

	got, err := s.WaitShellJob(t.Context(), "alice", job.ID)
	require.NoError(t, err)
	assert.Equal(t, shelljob.Failed, got.Status)
	assert.Equal(t, "the computer's Nexul app is not connected", repo.shellJob(job.ID).Error)
}

func TestRunCommand_ValidatesTheCommand(t *testing.T) {
	s, _, _ := newShellService(t)
	for name, in := range map[string]shelljob.Input{
		"blank":         {Command: "  "},
		"unknown shell": {Command: "ls", Shell: "zsh"},
		"over an hour":  {Command: "ls", Timeout: 2 * time.Hour},
		"too long":      {Command: strings.Repeat("x", 64<<10+1)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.RunCommand(t.Context(), "alice", "c-laptop", in)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestShellJob_KeepsTheLastMiBOfOutput(t *testing.T) {
	s, repo, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "yes"})
	require.NoError(t, err)
	waitStarted(t, shells)
	line := strings.Repeat("y", 1023)
	for range 1100 {
		shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: line}})
	}
	shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "last"}})
	shells.sink(job.ID).Done(shelljob.End{Status: shelljob.Succeeded})

	got := repo.shellJob(job.ID)
	assert.True(t, got.OutputTruncated)
	assert.LessOrEqual(t, len(got.Output), shelljob.MaxOutput)
	assert.True(t, strings.HasSuffix(got.Output, "last\n"))
}

func TestOpenShellOutput_ReplaysThenFollowsARunningJob(t *testing.T) {
	s, _, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "build"})
	require.NoError(t, err)
	waitStarted(t, shells)
	shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "one"}})

	feed, err := s.OpenShellOutput(t.Context(), "alice", job.ID)
	require.NoError(t, err)
	defer feed.Close()
	first, err := feed.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "one", first[0].Line)

	shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "two"}})
	shells.sink(job.ID).Done(shelljob.End{Status: shelljob.Succeeded})
	second, err := feed.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "two", second[0].Line)
	_, err = feed.Next(t.Context())
	require.ErrorIs(t, err, io.EOF)
}

func TestCancelShellJob_StopsAQueuedOneWithoutTheRunner(t *testing.T) {
	s, repo, shells := newShellService(t)
	var last *shelljob.Job
	for range 5 {
		job, err := s.RunCommand(t.Context(), "alice", "c-laptop", shelljob.Input{Command: "sleep 9"})
		require.NoError(t, err)
		last = job
	}
	for range 4 {
		waitStarted(t, shells)
	}

	got, err := s.CancelShellJob(t.Context(), "alice", last.ID)
	require.NoError(t, err)

	assert.Equal(t, shelljob.Cancelled, got.Status)
	assert.Equal(t, shelljob.Cancelled, repo.shellJob(last.ID).Status)
	assert.Empty(t, shells.cancelled)
}

func TestRunShellUpkeep_EndsUnfinishedJobsAndPurgesOldOutput(t *testing.T) {
	s, repo, _ := newShellService(t)
	old := time.Now().Add(-31 * 24 * time.Hour)
	repo.jobs["j-old"] = shelljob.Job{ID: "j-old", Status: shelljob.Succeeded, Command: "ls", Output: "secret\n", EndedAt: &old}
	repo.jobs["j-left"] = shelljob.Job{ID: "j-left", Status: shelljob.Running, Command: "make"}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	s.RunShellUpkeep(ctx, slog.New(slog.DiscardHandler))

	assert.Equal(t, shelljob.Failed, repo.shellJob("j-left").Status)
	purged := repo.shellJob("j-old")
	assert.Empty(t, purged.Output)
	assert.True(t, purged.OutputPurged)
	assert.Equal(t, "ls", purged.Command)
}

func TestCommandRoutes_AnswerNotFoundToAnyoneElse(t *testing.T) {
	s, _, shells := newShellService(t)
	h := NewHandler(s).Routes()
	res := doRequest(h, http.MethodPost, "/api/pairing/computers/c-laptop/commands", "alice", map[string]any{"command": "uname -a"})
	require.Equal(t, http.StatusAccepted, res.Code, res.Body.String())
	var job shelljob.Job
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &job))
	waitStarted(t, shells)

	for _, req := range []struct{ method, path string }{
		{http.MethodPost, "/api/pairing/computers/c-laptop/commands"},
		{http.MethodGet, "/api/pairing/computers/c-laptop/commands"},
		{http.MethodGet, "/api/pairing/commands/" + job.ID},
		{http.MethodPost, "/api/pairing/commands/" + job.ID + "/cancel"},
	} {
		res := doRequest(h, req.method, req.path, "mallory", map[string]any{"command": "id"})
		assert.Equal(t, http.StatusNotFound, res.Code, req.method+" "+req.path)
	}
	res = doRequest(h, http.MethodGet, "/api/pairing/computers/c-laptop/commands", "alice", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Contains(t, res.Body.String(), `"total":1`)
}

func TestOpenTerminal_RefusesWhatCommandsRefuse(t *testing.T) {
	size := shelljob.TerminalSize{Cols: 80, Rows: 24}
	t.Run("someone else's computer is not found", func(t *testing.T) {
		s, repo, _ := newShellService(t)
		_, err := s.OpenTerminal(t.Context(), "mallory", "c-laptop", size)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Empty(t, repo.jobs)
	})
	t.Run("the owner's switch is off", func(t *testing.T) {
		s, repo, _ := newShellService(t)
		repo.computers["c-laptop"] = Computer{ID: "c-laptop", UserID: "alice", Name: "laptop"}
		_, err := s.OpenTerminal(t.Context(), "alice", "c-laptop", size)
		require.ErrorIs(t, err, apperrs.ErrConflict)
		assert.Empty(t, repo.jobs)
	})
	t.Run("the computer is offline", func(t *testing.T) {
		s, repo, _ := newShellService(t)
		_, err := s.OpenTerminal(t.Context(), "alice", "c-laptop", size)
		require.ErrorIs(t, err, apperrs.ErrConflict)
		require.Len(t, repo.jobs, 1)
		for _, job := range repo.jobs {
			assert.Equal(t, shelljob.Refused, job.Status)
			assert.Equal(t, shelljob.AdapterTerminal, job.Adapter)
			assert.Equal(t, "the computer's Nexul app is not connected", job.Error)
		}
	})
}
