package shelljob

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeStore keeps jobs and the events their changes carried.
type fakeStore struct {
	mu       sync.Mutex
	jobs     map[string]Job
	events   int
	startErr error
}

func (f *fakeStore) InsertShellJob(_ context.Context, job Job, _ eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID] = job
	f.events++
	return nil
}

func (f *fakeStore) StartShellJob(_ context.Context, id string, at time.Time, _ eventbus.OutboxEvent) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return false, f.startErr
	}
	job := f.jobs[id]
	if job.Status != Queued {
		return false, nil
	}
	job.Status, job.StartedAt = Running, &at
	f.jobs[id] = job
	f.events++
	return true, nil
}

func (f *fakeStore) FinishShellJob(_ context.Context, job Job, _ eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID] = job
	f.events++
	return nil
}

func (f *fakeStore) job(id string) Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.jobs[id]
}

// fakeRunners keeps each started job's sink until the test ends it.
type fakeRunners struct {
	mu        sync.Mutex
	sinks     map[string]Sink
	started   chan string
	startErr  error
	cancelErr error
	refuse    map[string]End
}

func (f *fakeRunners) Start(_ context.Context, job Job, _ Run, sink Sink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.startErr != nil {
		return f.startErr
	}
	f.sinks[job.ID] = sink
	f.started <- job.ID
	return nil
}

func (f *fakeRunners) Cancel(_ context.Context, jobID string) error {
	return f.cancelErr
}

func (f *fakeRunners) StillAllowed(_ context.Context, job Job) (End, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	end, refused := f.refuse[job.Command]
	return end, !refused
}

func (f *fakeRunners) sink(id string) Sink {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sinks[id]
}

func newTestQueue(t *testing.T) (*Queue, *fakeStore, *fakeRunners) {
	t.Helper()
	store := &fakeStore{jobs: map[string]Job{}}
	runners := &fakeRunners{sinks: map[string]Sink{}, started: make(chan string, 16), refuse: map[string]End{}}
	q := NewQueue(Config{
		Store: store, Runners: runners, Event: func(Job) eventbus.OutboxEvent { return eventbus.OutboxEvent{} },
		StartError: func(err error) string { return "unreachable: " + err.Error() }, Silent: "no answer",
	})
	t.Cleanup(q.Drain)
	return q, store, runners
}

func add(t *testing.T, q *Queue, id, runner, command string) Job {
	t.Helper()
	job := Job{ID: id, RunnerID: runner, Command: command, TimeoutSeconds: 60, Status: Queued}
	require.NoError(t, q.Add(t.Context(), job))
	return job
}

func waitStarted(t *testing.T, r *fakeRunners) string {
	t.Helper()
	select {
	case id := <-r.started:
		return id
	case <-time.After(2 * time.Second):
		t.Fatal("no job reached its runner")
		return ""
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		in   Input
		ok   bool
	}{
		{"blank", Input{Command: "  "}, false},
		{"unknown shell", Input{Command: "ls", Shell: "zsh"}, false},
		{"over an hour", Input{Command: "ls", Timeout: 2 * time.Hour}, false},
		{"under a second", Input{Command: "ls", Timeout: time.Millisecond}, false},
		{"too long", Input{Command: strings.Repeat("x", maxCommand+1)}, false},
		{"a named shell", Input{Command: "ls", Shell: " pwsh "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.in)
			if !tt.ok {
				require.ErrorIs(t, err, apperrs.ErrInvalid)
				return
			}
			require.NoError(t, err)
		})
	}
	in := Input{Command: "ls", Timeout: 1500 * time.Millisecond}
	require.NoError(t, Validate(&in))
	assert.Equal(t, Input{Command: "ls", Timeout: time.Second, Adapter: AdapterWeb}, in)
	in = Input{Command: "ls"}
	require.NoError(t, Validate(&in))
	assert.Equal(t, DefaultTimeout, in.Timeout)
}

func TestQueue_RunsFourAtATimePerRunner(t *testing.T) {
	q, store, runners := newTestQueue(t)
	var ids []string
	for i := range AtOnce + 1 {
		ids = append(ids, add(t, q, string(rune('a'+i)), "r-1", "sleep").ID)
	}
	add(t, q, "other", "r-2", "sleep")
	for range AtOnce + 1 {
		waitStarted(t, runners)
	}
	assert.Equal(t, Queued, store.job(ids[AtOnce]).Status, "the fifth on one runner waits")

	runners.sink(ids[0]).Done(End{Status: Succeeded})
	assert.Equal(t, ids[AtOnce], waitStarted(t, runners))
	assert.Equal(t, Succeeded, store.job(ids[0]).Status)
	require.NotNil(t, store.job(ids[0]).ExitCode)
}

func TestQueue_KeepsTheLastMiBOfOutput(t *testing.T) {
	q, store, runners := newTestQueue(t)
	job := add(t, q, "j", "r-1", "yes")
	waitStarted(t, runners)
	line := strings.Repeat("y", 1023)
	for range 1100 {
		runners.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: line}})
	}
	runners.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "last"}})
	runners.sink(job.ID).Done(End{Status: Succeeded})
	runners.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "after the end"}})
	runners.sink(job.ID).Done(End{Status: Failed})

	got := store.job(job.ID)
	assert.Equal(t, Succeeded, got.Status, "only the first end counts")
	assert.True(t, got.OutputTruncated)
	assert.LessOrEqual(t, len(got.Output), MaxOutput)
	assert.True(t, strings.HasSuffix(got.Output, "last\n"))
}

func TestQueue_OpenReplaysThenFollowsAndOverlayShowsTheOutputSoFar(t *testing.T) {
	q, store, runners := newTestQueue(t)
	job := add(t, q, "j", "r-1", "build")
	waitStarted(t, runners)
	runners.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "one"}})

	shown := store.job(job.ID)
	q.Overlay(&shown)
	assert.Equal(t, "one\n", shown.Output)

	feed := q.Open(job)
	defer feed.Close()
	lines, err := feed.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "one", lines[0].Line)
	runners.sink(job.ID).Output([]containerlog.Line{{Stream: "stderr", Line: "two"}})
	lines, err = feed.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "two", lines[0].Line)
	runners.sink(job.ID).Done(End{Status: Succeeded})
	_, err = feed.Next(t.Context())
	require.ErrorIs(t, err, io.EOF)
	feed.Close()

	ended := q.Open(store.job(job.ID))
	lines, err = ended.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, []string{lines[0].Line, lines[1].Line})
	_, err = ended.Next(t.Context())
	require.ErrorIs(t, err, io.EOF)
}

func TestQueue_ASlowViewerSkipsLinesRatherThanHoldTheJob(t *testing.T) {
	q, _, runners := newTestQueue(t)
	job := add(t, q, "j", "r-1", "yes")
	waitStarted(t, runners)
	feed := q.Open(job)
	defer feed.Close()
	many := make([]containerlog.Line, feedBuffer+5)
	runners.sink(job.ID).Output(many)
	lines, err := feed.Next(t.Context())
	require.NoError(t, err)
	assert.Len(t, lines, feedBuffer+1)
	assert.Equal(t, "… 5 lines skipped", lines[feedBuffer].Line)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = feed.Next(ctx)
	require.ErrorIs(t, err, context.Canceled)
	runners.sink(job.ID).Done(End{Status: Succeeded})
}

func TestQueue_Cancel(t *testing.T) {
	q, store, runners := newTestQueue(t)
	for _, id := range []string{"run-a", "run-b", "run-c", "run-d"} {
		add(t, q, id, "r-1", "sleep")
		waitStarted(t, runners)
	}
	queued := add(t, q, "queued", "r-1", "sleep")
	q.Cancel(t.Context(), queued.ID, "")
	assert.Equal(t, Cancelled, store.job(queued.ID).Status, "a queued job ends here")

	runners.cancelErr = errors.New("gone")
	q.Cancel(t.Context(), "run-a", "")
	assert.Equal(t, "the stop could not reach the runner", store.job("run-a").Error)

	runners.cancelErr = nil
	q.CancelWhere(t.Context(), func(j Job) bool { return j.ID == "run-b" }, "access revoked")
	assert.Equal(t, Job{}.Error, store.job("run-c").Error)
	assert.Equal(t, "access revoked", store.job("run-b").Error)

	q.Cancel(t.Context(), "unknown", "")
	q.Wait(t.Context(), "unknown")
	for _, id := range []string{"run-c", "run-d"} {
		runners.sink(id).Done(End{Status: Succeeded})
		q.Wait(t.Context(), id)
	}
}

func TestQueue_AJobThatCannotStartEndsWithTheReason(t *testing.T) {
	t.Run("refused as it reaches the front", func(t *testing.T) {
		q, store, runners := newTestQueue(t)
		runners.refuse["rm"] = End{Status: Cancelled, Error: "access revoked"}
		job := add(t, q, "j", "r-1", "rm")
		q.Drain()
		assert.Equal(t, Cancelled, store.job(job.ID).Status)
		assert.Equal(t, "access revoked", store.job(job.ID).Error)
	})
	t.Run("its start not recorded", func(t *testing.T) {
		q, store, _ := newTestQueue(t)
		store.startErr = errors.New("disk full")
		job := add(t, q, "j", "r-1", "ls")
		q.Drain()
		assert.Equal(t, "could not record the command's start", store.job(job.ID).Error)
	})
	t.Run("its runner unreachable", func(t *testing.T) {
		q, store, runners := newTestQueue(t)
		runners.startErr = errors.New("offline")
		job := add(t, q, "j", "r-1", "ls")
		q.Drain()
		assert.Equal(t, Failed, store.job(job.ID).Status)
		assert.Equal(t, "unreachable: offline", store.job(job.ID).Error)
	})
}

func TestQueue_ARunnerThatNeverAnswersFailsTheJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q, store, runners := newTestQueue(t)
		job := add(t, q, "j", "r-1", "ls")
		synctest.Wait()
		<-runners.started
		time.Sleep(time.Minute + AnswerGrace + time.Second)
		synctest.Wait()
		assert.Equal(t, Failed, store.job(job.ID).Status)
		assert.Equal(t, "no answer", store.job(job.ID).Error)
	})
}

func TestJob_TailForAgent(t *testing.T) {
	short := Job{Output: "ok\n"}
	short.TailForAgent()
	assert.Equal(t, Job{Output: "ok\n"}, short)
	long := Job{Output: strings.Repeat("a", AgentTail) + "end"}
	long.TailForAgent()
	assert.Len(t, long.Output, AgentTail)
	assert.True(t, long.OutputTruncated)
	assert.True(t, strings.HasSuffix(long.Output, "end"))
}
