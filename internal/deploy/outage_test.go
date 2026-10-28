package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeOutageRepo is an in-memory deploy.OutageRepo that keeps the outbox rows it was handed.
type fakeOutageRepo struct {
	mu     sync.Mutex
	rows   []*Outage
	outbox []eventbus.OutboxEvent

	openErr, updateErr, closeErr, getErr, listErr, endErr error
}

func (f *fakeOutageRepo) Open(_ context.Context, o *Outage, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.openErr != nil {
		return f.openErr
	}
	for _, r := range f.rows {
		if r.StackID == o.StackID && r.Service == o.Service && r.EndedAt == nil {
			return apperrs.ErrConflict
		}
	}
	cp := *o
	f.rows = append(f.rows, &cp)
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeOutageRepo) Update(_ context.Context, o *Outage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	for _, r := range f.rows {
		if r.ID == o.ID {
			r.Container, r.ExitCode, r.OOM, r.RestartCount = o.Container, o.ExitCode, o.OOM, o.RestartCount
		}
	}
	return nil
}

func (f *fakeOutageRepo) Close(_ context.Context, id string, endedAt time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closeErr != nil {
		return f.closeErr
	}
	for _, r := range f.rows {
		if r.ID == id && r.EndedAt == nil {
			r.EndedAt = &endedAt
			f.outbox = append(f.outbox, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeOutageRepo) GetOpen(_ context.Context, stackID, service string) (*Outage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, r := range f.rows {
		if r.StackID == stackID && r.Service == service && r.EndedAt == nil {
			cp := *r
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeOutageRepo) ListOpenByStack(_ context.Context, stackID string) ([]*Outage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Outage
	for _, r := range f.rows {
		if r.StackID == stackID && r.EndedAt == nil {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (f *fakeOutageRepo) LatestEnd(_ context.Context, stackID, service string) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.endErr != nil {
		return time.Time{}, f.endErr
	}
	var latest time.Time
	for _, r := range f.rows {
		if r.StackID == stackID && r.Service == service && r.EndedAt != nil && r.EndedAt.After(latest) {
			latest = *r.EndedAt
		}
	}
	return latest, nil
}

// events decodes the outage events written so far, by topic.
func (f *fakeOutageRepo) events(t *testing.T, topic string) []OutageEvent {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []OutageEvent
	for _, e := range f.outbox {
		if e.Topic != topic {
			continue
		}
		b, err := json.Marshal(e.Payload)
		require.NoError(t, err)
		var ev OutageEvent
		require.NoError(t, json.Unmarshal(b, &ev))
		out = append(out, ev)
	}
	return out
}

// cascade drops a deleted stack's outages, as the foreign key does.
func (f *fakeOutageRepo) cascade(stackID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = slices.DeleteFunc(f.rows, func(o *Outage) bool { return o.StackID == stackID })
}

func (f *fakeOutageRepo) openCount(stackID string) int {
	open, _ := f.ListOpenByStack(context.Background(), stackID)
	return len(open)
}

const (
	outageMachine = "10.0.0.1:22"
	outageStack   = "svc-1"
)

type outageEnv struct {
	svc     *Service
	repo    *fakeOutageRepo
	deploys *fakeRepo
	stacks  *fakeStackRepo
	rows    *fakeContainerRepo
}

// newOutageEnv is the shared stack (slug api, on outageMachine) with outage storage wired.
func newOutageEnv(t *testing.T) *outageEnv {
	t.Helper()
	deploys, stacks := newFakeRepo(), newFakeStackRepo()
	stack := validStack()
	require.NoError(t, stacks.Create(t.Context(), &stack))
	rows := newFakeContainerRepo()
	svc := newTestServiceWith(deploys, stacks, rows, newFakeProjects(), newFakeBus())
	repo := &fakeOutageRepo{}
	svc.SetOutageRepo(repo)
	return &outageEnv{svc: svc, repo: repo, deploys: deploys, stacks: stacks, rows: rows}
}

func busEvent(t *testing.T, topic string, payload any) eventbus.Event {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return eventbus.Event{ID: "ev", Topic: topic, Payload: b}
}

func dockerEvent(action string, exit int) containerEvent {
	return containerEvent{Machine: outageMachine, Container: "api-web-1", Project: "api", Service: "web", Action: action, ExitCode: exit}
}

func (e *outageEnv) handle(t *testing.T, ev containerEvent) {
	t.Helper()
	require.NoError(t, e.svc.HandleContainerEvent(t.Context(), busEvent(t, TopicRunnerContainerEvent, ev)))
}

func (e *outageEnv) settle(t *testing.T, states ...containerState) error {
	t.Helper()
	return e.svc.HandleContainerStates(t.Context(), busEvent(t, TopicRunnerContainerStates, containerStates{Machine: outageMachine, States: states}))
}

func webState(state string, exit int, finished int64) containerState {
	return containerState{Container: "api-web-1", Project: "api", Service: "web", State: state, ExitCode: exit, FinishedAt: finished}
}

func TestOutageTarget_MatchesTheStackByProjectOrRunContainer(t *testing.T) {
	tests := []struct {
		name                      string
		machine, project, name_   string
		service                   string
		wantStack, wantService    string
		runningDeploy, wantIgnore bool
	}{
		{name: "compose project is the stack slug", machine: outageMachine, project: "api", name_: "api-web-1", service: "web", wantStack: outageStack, wantService: "web"},
		{name: "run container is named after the slug", machine: outageMachine, name_: "api", wantStack: outageStack, wantService: "api"},
		{name: "unknown compose project", machine: outageMachine, project: "unrelated", name_: "unrelated-db-1", service: "db", wantIgnore: true},
		{name: "unknown container", machine: outageMachine, name_: "redis", wantIgnore: true},
		{name: "same slug on another machine", machine: "elsewhere", project: "api", name_: "api-web-1", service: "web", wantIgnore: true},
		{name: "no machine", project: "api", name_: "api-web-1", service: "web", wantIgnore: true},
		{name: "deploy under way", machine: outageMachine, project: "api", name_: "api-web-1", service: "web", runningDeploy: true, wantIgnore: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newOutageEnv(t)
			if tt.runningDeploy {
				seedDeploy(t, env.deploys, &Deploy{ID: "d1", StackID: outageStack, Status: StatusRunning})
			}
			stack, service, err := env.svc.outageTarget(t.Context(), tt.machine, tt.project, tt.name_, tt.service)
			require.NoError(t, err)
			if tt.wantIgnore {
				assert.Nil(t, stack)
				return
			}
			require.NotNil(t, stack)
			assert.Equal(t, tt.wantStack, stack.ID)
			assert.Equal(t, tt.wantService, service)
		})
	}
}

func TestHandleContainerEvent_OnlyACrashOpensAnOutage(t *testing.T) {
	tests := []struct {
		name     string
		ev       containerEvent
		wantOpen int
	}{
		{"die with a non-zero exit", dockerEvent(containerDie, 1), 1},
		{"die with exit 0 is a job that finished", dockerEvent(containerDie, 0), 0},
		{"oom alone leaves a running container up", dockerEvent(containerOOM, 0), 0},
		{"kill alone", dockerEvent(containerKill, 0), 0},
		{"stop alone", dockerEvent(containerStop, 0), 0},
		{"start with nothing open", dockerEvent(containerStart, 0), 0},
		{"an action nobody asked for", dockerEvent("exec_start: sh", 0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newOutageEnv(t)
			env.handle(t, tt.ev)
			assert.Equal(t, tt.wantOpen, env.repo.openCount(outageStack))
			assert.Len(t, env.repo.events(t, TopicContainerStopped), tt.wantOpen)
		})
	}
}

func TestHandleContainerEvent_TheStoppedEventNamesTheOutage(t *testing.T) {
	env := newOutageEnv(t)
	env.handle(t, dockerEvent(containerDie, 137))

	stopped := env.repo.events(t, TopicContainerStopped)
	require.Len(t, stopped, 1)
	got := stopped[0]
	assert.Equal(t, "ws-1", got.WorkspaceID)
	assert.Equal(t, "proj-1", got.ProjectID)
	assert.Equal(t, outageStack, got.StackID)
	assert.Equal(t, "api", got.StackName)
	assert.Equal(t, "web", got.Service)
	assert.Equal(t, "api-web-1", got.Container)
	assert.Equal(t, outageMachine, got.Machine)
	assert.Equal(t, 137, got.ExitCode)
	assert.False(t, got.OOM)
	assert.Equal(t, 0, got.RestartCount)
	assert.Equal(t, env.svc.now().UTC(), got.StartedAt)
	assert.Nil(t, got.EndedAt)
	assert.Equal(t, env.repo.rows[0].ID, got.OutageID)
}

func TestHandleContainerEvent_ADeployUnderWayRaisesNothing(t *testing.T) {
	env := newOutageEnv(t)
	seedDeploy(t, env.deploys, &Deploy{ID: "d1", StackID: outageStack, Status: StatusRunning})
	env.handle(t, dockerEvent(containerDie, 143))
	assert.Zero(t, env.repo.openCount(outageStack))
	assert.Empty(t, env.repo.events(t, TopicContainerStopped))
}

func TestHandleContainerEvent_ARestartLoopIsOneOutage(t *testing.T) {
	env := newOutageEnv(t)
	for range 3 {
		env.handle(t, dockerEvent(containerDie, 1))
		env.handle(t, dockerEvent(containerStart, 0))
	}
	env.handle(t, dockerEvent(containerDie, 2))

	open, err := env.repo.ListOpenByStack(t.Context(), outageStack)
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, 3, open[0].RestartCount, "every crash after the first counts")
	assert.Equal(t, 2, open[0].ExitCode, "the latest exit code wins")
	assert.Len(t, env.repo.events(t, TopicContainerStopped), 1, "one event for the whole loop")
}

func TestHandleContainerEvent_AnOOMKillThenItsDeathIsOneOOMCrash(t *testing.T) {
	env := newOutageEnv(t)
	env.handle(t, dockerEvent(containerOOM, 0))
	env.handle(t, dockerEvent(containerDie, 137))

	open, err := env.repo.ListOpenByStack(t.Context(), outageStack)
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.True(t, open[0].OOM)
	assert.Equal(t, 137, open[0].ExitCode)
	assert.Equal(t, 0, open[0].RestartCount)

	stopped := env.repo.events(t, TopicContainerStopped)
	require.Len(t, stopped, 1)
	assert.Equal(t, 137, stopped[0].ExitCode, "the first notice carries the exit code")
	assert.True(t, stopped[0].OOM)

	env.handle(t, dockerEvent(containerDie, 137))
	open, _ = env.repo.ListOpenByStack(t.Context(), outageStack)
	assert.Equal(t, 1, open[0].RestartCount, "the next death is a second crash")
	assert.True(t, open[0].OOM)
}

func TestHandleContainerEvent_AnOOMMarkLapsesAndIsUsedOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.now = time.Now

		env.handle(t, dockerEvent(containerOOM, 0))
		time.Sleep(oomWindow + time.Second)
		env.handle(t, dockerEvent(containerDie, 1))
		open, err := env.repo.ListOpenByStack(t.Context(), outageStack)
		require.NoError(t, err)
		require.Len(t, open, 1)
		assert.False(t, open[0].OOM, "a die long after the oom is some other crash")

		env.handle(t, dockerEvent(containerOOM, 0))
		time.Sleep(oomWindow)
		env.handle(t, dockerEvent(containerDie, 137))
		open, _ = env.repo.ListOpenByStack(t.Context(), outageStack)
		assert.True(t, open[0].OOM, "the edge of the window still counts")

		env.handle(t, dockerEvent(containerDie, 137))
		open, _ = env.repo.ListOpenByStack(t.Context(), outageStack)
		assert.Equal(t, 2, open[0].RestartCount)
		assert.True(t, open[0].OOM, "an outage that was OOM stays so")
	})
}

func TestHandleContainerEvent_ARequestedStopIsNotACrash(t *testing.T) {
	tests := []struct {
		name     string
		before   []string
		exit     int
		wantOpen int
	}{
		{"die alone is a crash", nil, 1, 1},
		{"kill then die", []string{containerKill}, 137, 0},
		{"stop then die", []string{containerStop}, 143, 0},
		{"kill and stop then die", []string{containerKill, containerStop}, 143, 0},
		{"oom then die with no kill is a crash", []string{containerOOM}, 137, 1},
		{"kill and oom then die was still asked for", []string{containerOOM, containerKill}, 137, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newOutageEnv(t)
			for _, action := range tt.before {
				env.handle(t, dockerEvent(action, 0))
			}
			env.handle(t, dockerEvent(containerDie, tt.exit))
			assert.Equal(t, tt.wantOpen, env.repo.openCount(outageStack))
			assert.Len(t, env.repo.events(t, TopicContainerStopped), tt.wantOpen)
		})
	}
}

func TestHandleContainerEvent_AKillCoversOnlyTheDieItPrecedes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.now = time.Now

		env.handle(t, dockerEvent(containerKill, 0))
		env.handle(t, dockerEvent(containerDie, 137))
		env.handle(t, dockerEvent(containerDie, 137))
		assert.Equal(t, 1, env.repo.openCount(outageStack), "the kill was used up by the first die")
		env.repo.cascade(outageStack)

		env.handle(t, dockerEvent(containerKill, 0))
		time.Sleep(stopWindow + time.Second)
		env.handle(t, dockerEvent(containerDie, 137))
		assert.Equal(t, 1, env.repo.openCount(outageStack), "a die long after the kill is a crash")
		env.repo.cascade(outageStack)

		env.handle(t, dockerEvent(containerStop, 0))
		env.handle(t, dockerEvent(containerStart, 0))
		env.handle(t, dockerEvent(containerDie, 1))
		assert.Equal(t, 1, env.repo.openCount(outageStack), "a restart after the stop clears it, so a crash on boot is seen")
	})
}

func TestHandleContainerEvent_AKillOnAnotherContainerChangesNothing(t *testing.T) {
	env := newOutageEnv(t)
	other := dockerEvent(containerKill, 0)
	other.Container, other.Service = "api-worker-1", "worker"
	env.handle(t, other)
	env.handle(t, dockerEvent(containerDie, 1))
	assert.Equal(t, 1, env.repo.openCount(outageStack))
}

func TestOutageTarget_NamesTheServiceAsTheStacksRowsDo(t *testing.T) {
	tests := []struct {
		name          string
		rows          []Container
		container     string
		label         string
		want          string
		wantStackSlug string
	}{
		{name: "deployed compose stack names rows after the compose service", rows: []Container{{Name: "web", ContainerName: "api-web-1"}}, container: "api-web-1", label: "web", want: "web"},
		{name: "imported compose stack names rows after the container", rows: []Container{{Name: "api-worker-1", ContainerName: "api-worker-1"}, {Name: "api-web-1", ContainerName: "api-web-1"}}, container: "api-worker-1", label: "worker", want: "api-worker-1"},
		{name: "run stack row is the container", rows: []Container{{Name: "api", ContainerName: "api"}}, container: "api", want: "api"},
		{name: "a container no row holds yet keeps its compose service", rows: []Container{{Name: "api-web-1", ContainerName: "api-web-1"}}, container: "api-worker-1", label: "worker", want: "worker"},
		{name: "a stack with no rows yet", container: "api-web-1", label: "web", want: "web"},
		{name: "no rows and no label is the stack itself", container: "api", want: "api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newOutageEnv(t)
			for i := range tt.rows {
				tt.rows[i].ID, tt.rows[i].StackID = fmt.Sprintf("row-%d", i), outageStack
				require.NoError(t, env.rows.Upsert(t.Context(), &tt.rows[i]))
			}
			project := ""
			if tt.label != "" {
				project = "api"
			}
			stack, service, err := env.svc.outageTarget(t.Context(), outageMachine, project, tt.container, tt.label)
			require.NoError(t, err)
			require.NotNil(t, stack)
			assert.Equal(t, tt.want, service)
		})
	}
}

func TestHandleContainerEvent_AnImportedStacksOutageNamesARowTheLogsCanRead(t *testing.T) {
	env := newOutageEnv(t)
	for _, name := range []string{"api-web-1", "api-worker-1"} {
		require.NoError(t, env.rows.Upsert(t.Context(), &Container{ID: name, StackID: outageStack, Name: name, ContainerName: name}))
	}
	ev := dockerEvent(containerDie, 1)
	ev.Container, ev.Service = "api-worker-1", "worker"
	env.handle(t, ev)

	open, err := env.repo.ListOpenByStack(t.Context(), outageStack)
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, "api-worker-1", open[0].Service)
	stack, err := env.stacks.GetByID(t.Context(), outageStack)
	require.NoError(t, err)
	container, err := env.svc.containerOf(t.Context(), stack, open[0].Service)
	require.NoError(t, err)
	assert.Equal(t, "api-worker-1", container)
	assert.Equal(t, "api-worker-1", env.repo.events(t, TopicContainerStopped)[0].Service)
}

func TestHandleContainerStates_ARunningContainerNeverOpensAnOutage(t *testing.T) {
	env := newOutageEnv(t)
	require.NoError(t, env.settle(t, webState(snapshotRunning, 0, 0)))
	require.NoError(t, env.settle(t, webState(snapshotRunning, 137, 1_700_000_000)))
	assert.Zero(t, env.repo.openCount(outageStack))
}

func TestOutageTarget_AFailedRowReadIsReturned(t *testing.T) {
	env := newOutageEnv(t)
	env.svc.services = failingRows{env.rows}
	_, _, err := env.svc.outageTarget(t.Context(), outageMachine, "api", "api-web-1", "web")
	assert.ErrorContains(t, err, "rows unreadable")
}

// failingRows is a container repo whose listing fails.
type failingRows struct{ *fakeContainerRepo }

func (failingRows) ListByStack(context.Context, string) ([]*Container, error) {
	return nil, errors.New("rows unreadable")
}

func TestHandleContainerEvent_AServiceHasItsOwnOutage(t *testing.T) {
	env := newOutageEnv(t)
	env.handle(t, dockerEvent(containerDie, 1))
	other := dockerEvent(containerDie, 1)
	other.Service, other.Container = "worker", "api-worker-1"
	env.handle(t, other)
	assert.Equal(t, 2, env.repo.openCount(outageStack))
}

func TestHandleContainerEvent_ARecoveryNeedsSixtySecondsRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.now = time.Now
		env.handle(t, dockerEvent(containerDie, 1))
		env.handle(t, dockerEvent(containerStart, 0))

		time.Sleep(outageSteady - time.Second)
		synctest.Wait()
		assert.Equal(t, 1, env.repo.openCount(outageStack), "59 seconds is not steady yet")
		assert.Empty(t, env.repo.events(t, TopicContainerRecovered))

		time.Sleep(time.Second)
		synctest.Wait()
		assert.Zero(t, env.repo.openCount(outageStack))
		recovered := env.repo.events(t, TopicContainerRecovered)
		require.Len(t, recovered, 1)
		require.NotNil(t, recovered[0].EndedAt)
		assert.Equal(t, env.repo.rows[0].ID, recovered[0].OutageID)
		assert.Equal(t, outageSteady, recovered[0].EndedAt.Sub(recovered[0].StartedAt))
	})
}

func TestHandleContainerEvent_ADeathInsideTheWindowKeepsTheOutageOpen(t *testing.T) {
	tests := []struct {
		name string
		evs  []containerEvent
	}{
		{"another crash", []containerEvent{dockerEvent(containerDie, 1)}},
		{"a clean exit still means it is not running", []containerEvent{dockerEvent(containerDie, 0)}},
		{"a requested stop still means it is not running", []containerEvent{dockerEvent(containerKill, 0), dockerEvent(containerDie, 143)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				env := newOutageEnv(t)
				env.svc.now = time.Now
				env.handle(t, dockerEvent(containerDie, 1))
				env.handle(t, dockerEvent(containerStart, 0))
				time.Sleep(30 * time.Second)
				for _, ev := range tt.evs {
					env.handle(t, ev)
				}

				time.Sleep(10 * outageSteady)
				synctest.Wait()
				assert.Equal(t, 1, env.repo.openCount(outageStack), "the first start's timer was cancelled")
				assert.Empty(t, env.repo.events(t, TopicContainerRecovered))
			})
		})
	}
}

func TestHandleContainerEvent_AnOOMKillOfAChildDoesNotInterruptTheWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.now = time.Now
		env.handle(t, dockerEvent(containerDie, 1))
		env.handle(t, dockerEvent(containerStart, 0))
		time.Sleep(30 * time.Second)
		env.handle(t, dockerEvent(containerOOM, 0))

		time.Sleep(outageSteady)
		synctest.Wait()
		assert.Zero(t, env.repo.openCount(outageStack), "the container kept running, so it recovers")
	})
}

func TestHandleContainerEvent_ARestartInTheWindowStartsTheWaitOver(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.now = time.Now
		env.handle(t, dockerEvent(containerDie, 1))
		env.handle(t, dockerEvent(containerStart, 0))
		time.Sleep(40 * time.Second)
		env.handle(t, dockerEvent(containerDie, 1))
		env.handle(t, dockerEvent(containerStart, 0))

		time.Sleep(40 * time.Second)
		synctest.Wait()
		assert.Equal(t, 1, env.repo.openCount(outageStack), "80 seconds in, but only 40 since the last start")

		time.Sleep(20 * time.Second)
		synctest.Wait()
		assert.Zero(t, env.repo.openCount(outageStack))
		assert.Len(t, env.repo.events(t, TopicContainerRecovered), 1)
	})
}

func TestHandleContainerEvent_ACrashAfterRecoveryIsANewOutage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.now = time.Now
		env.handle(t, dockerEvent(containerDie, 1))
		env.handle(t, dockerEvent(containerStart, 0))
		time.Sleep(outageSteady)
		synctest.Wait()
		env.handle(t, dockerEvent(containerDie, 1))

		assert.Equal(t, 1, env.repo.openCount(outageStack))
		assert.Len(t, env.repo.rows, 2)
		assert.Len(t, env.repo.events(t, TopicContainerStopped), 2)
	})
}

func TestHandleContainerEvent_ADeletedStackLosesItsPendingRecovery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.now = time.Now
		env.handle(t, dockerEvent(containerDie, 1))
		env.handle(t, dockerEvent(containerStart, 0))
		env.repo.cascade(outageStack)

		time.Sleep(outageSteady)
		synctest.Wait()
		assert.Empty(t, env.repo.events(t, TopicContainerRecovered))
	})
}

func TestHandleContainerEvent_ErrorPaths(t *testing.T) {
	t.Run("an undecodable event is fatal", func(t *testing.T) {
		env := newOutageEnv(t)
		err := env.svc.HandleContainerEvent(t.Context(), eventbus.Event{Topic: TopicRunnerContainerEvent, Payload: []byte(`{`)})
		assert.ErrorIs(t, err, apperrs.ErrFatal)
	})
	t.Run("without outage storage events are ignored", func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.outages = nil
		env.handle(t, dockerEvent(containerDie, 1))
	})
	t.Run("a stack lookup failure is returned for the bus to retry", func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.stacks = failingSlug{env.stacks}
		err := env.svc.HandleContainerEvent(t.Context(), busEvent(t, TopicRunnerContainerEvent, dockerEvent(containerDie, 1)))
		assert.ErrorContains(t, err, "slug lookup failed")
	})
	t.Run("a project whose workspace cannot be found raises nothing and is retried", func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.projects.(*fakeProjects).workspaceErr = errors.New("project gone")
		err := env.svc.HandleContainerEvent(t.Context(), busEvent(t, TopicRunnerContainerEvent, dockerEvent(containerDie, 1)))
		assert.ErrorContains(t, err, "project gone")
		assert.Empty(t, env.repo.rows)
	})
	t.Run("a failed open is returned", func(t *testing.T) {
		env := newOutageEnv(t)
		env.repo.openErr = errors.New("disk full")
		err := env.svc.HandleContainerEvent(t.Context(), busEvent(t, TopicRunnerContainerEvent, dockerEvent(containerDie, 1)))
		assert.ErrorContains(t, err, "disk full")
	})
	t.Run("a failed read of the open outage is returned", func(t *testing.T) {
		env := newOutageEnv(t)
		env.repo.getErr = errors.New("read failed")
		for _, action := range []string{containerDie, containerStart} {
			err := env.svc.HandleContainerEvent(t.Context(), busEvent(t, TopicRunnerContainerEvent, dockerEvent(action, 1)))
			assert.ErrorContains(t, err, "read failed", action)
		}
	})
	t.Run("a failed update is returned", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		env.repo.updateErr = errors.New("locked")
		err := env.svc.HandleContainerEvent(t.Context(), busEvent(t, TopicRunnerContainerEvent, dockerEvent(containerDie, 1)))
		assert.ErrorContains(t, err, "locked")
	})
	t.Run("a failed deploy check is returned", func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.repo = failingActive{env.deploys}
		err := env.svc.HandleContainerEvent(t.Context(), busEvent(t, TopicRunnerContainerEvent, dockerEvent(containerDie, 1)))
		assert.ErrorContains(t, err, "active failed")
	})
}

// failingSlug is a stack repo whose slug lookup fails.
type failingSlug struct{ *fakeStackRepo }

func (failingSlug) GetBySlugAndMachine(context.Context, string, string) (*Stack, error) {
	return nil, errors.New("slug lookup failed")
}

// failingActive is a deploy repo whose active-deploy check fails.
type failingActive struct{ *fakeRepo }

func (failingActive) HasActive(context.Context, string) (bool, error) {
	return false, errors.New("active failed")
}

func TestHandleContainerStates(t *testing.T) {
	finished := time.Date(2026, 9, 1, 11, 30, 0, 0, time.UTC).Unix()
	t.Run("a container found exited non-zero opens an outage from when it exited", func(t *testing.T) {
		env := newOutageEnv(t)
		require.NoError(t, env.settle(t, webState("exited", 137, finished)))

		open, err := env.repo.ListOpenByStack(t.Context(), outageStack)
		require.NoError(t, err)
		require.Len(t, open, 1)
		assert.Equal(t, time.Unix(finished, 0).UTC(), open[0].StartedAt)
		assert.Equal(t, 137, open[0].ExitCode)
		assert.Len(t, env.repo.events(t, TopicContainerStopped), 1)
	})
	t.Run("an OOM-killed container opens one whatever its exit code", func(t *testing.T) {
		env := newOutageEnv(t)
		st := webState("exited", 0, finished)
		st.OOM = true
		require.NoError(t, env.settle(t, st))
		open, _ := env.repo.ListOpenByStack(t.Context(), outageStack)
		require.Len(t, open, 1)
		assert.True(t, open[0].OOM)
	})
	t.Run("a container that finished cleanly raises nothing", func(t *testing.T) {
		env := newOutageEnv(t)
		require.NoError(t, env.settle(t, webState("exited", 0, finished)))
		assert.Zero(t, env.repo.openCount(outageStack))
	})
	t.Run("a running container closes the open outage", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		require.NoError(t, env.settle(t, webState("running", 0, 0)))
		assert.Zero(t, env.repo.openCount(outageStack))
		assert.Len(t, env.repo.events(t, TopicContainerRecovered), 1)
	})
	t.Run("a running container with nothing open changes nothing", func(t *testing.T) {
		env := newOutageEnv(t)
		require.NoError(t, env.settle(t, webState("running", 0, 0)))
		assert.Empty(t, env.repo.rows)
	})
	t.Run("an exited container with an outage already open changes nothing", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		require.NoError(t, env.settle(t, webState("exited", 1, finished)))
		assert.Len(t, env.repo.rows, 1)
		assert.Len(t, env.repo.events(t, TopicContainerStopped), 1)
	})
	t.Run("the same snapshot twice, as after a server restart, raises one outage", func(t *testing.T) {
		env := newOutageEnv(t)
		require.NoError(t, env.settle(t, webState("exited", 1, finished)))
		require.NoError(t, env.settle(t, webState("exited", 1, finished)))
		assert.Len(t, env.repo.events(t, TopicContainerStopped), 1)
	})
	t.Run("an exit the service has been up since is not raised again", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		require.NoError(t, env.settle(t, webState("running", 0, 0)))
		require.NoError(t, env.settle(t, webState("exited", 1, 1)))
		assert.Len(t, env.repo.events(t, TopicContainerStopped), 1)
	})
	t.Run("an exit after the last outage ended is a new one", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		require.NoError(t, env.settle(t, webState("running", 0, 0)))
		later := env.svc.now().Add(time.Hour).Unix()
		require.NoError(t, env.settle(t, webState("exited", 1, later)))
		assert.Len(t, env.repo.events(t, TopicContainerStopped), 2)
	})
	t.Run("a container of no stack, and one of a stack mid-deploy, are skipped", func(t *testing.T) {
		env := newOutageEnv(t)
		stranger := containerState{Container: "redis", State: "exited", ExitCode: 1, FinishedAt: finished}
		require.NoError(t, env.settle(t, stranger))
		seedDeploy(t, env.deploys, &Deploy{ID: "d1", StackID: outageStack, Status: StatusPending})
		require.NoError(t, env.settle(t, webState("exited", 1, finished)))
		assert.Empty(t, env.repo.rows)
	})
	t.Run("an exit with no time is dated now", func(t *testing.T) {
		env := newOutageEnv(t)
		require.NoError(t, env.settle(t, webState("exited", 1, 0)))
		assert.Equal(t, env.svc.now().UTC(), env.repo.rows[0].StartedAt)
	})
}

func TestHandleContainerStates_ErrorPaths(t *testing.T) {
	t.Run("an undecodable snapshot is fatal", func(t *testing.T) {
		env := newOutageEnv(t)
		err := env.svc.HandleContainerStates(t.Context(), eventbus.Event{Topic: TopicRunnerContainerStates, Payload: []byte(`[`)})
		assert.ErrorIs(t, err, apperrs.ErrFatal)
	})
	t.Run("without outage storage a snapshot is ignored", func(t *testing.T) {
		env := newOutageEnv(t)
		env.svc.outages = nil
		require.NoError(t, env.settle(t, webState("exited", 1, 1)))
	})
	t.Run("storage failures are returned for the bus to retry", func(t *testing.T) {
		tests := []struct {
			name string
			set  func(*fakeOutageRepo)
			want string
		}{
			{"read of the open outage", func(r *fakeOutageRepo) { r.getErr = errors.New("get failed") }, "get failed"},
			{"read of the last end", func(r *fakeOutageRepo) { r.endErr = errors.New("end failed") }, "end failed"},
			{"open", func(r *fakeOutageRepo) { r.openErr = errors.New("open failed") }, "open failed"},
		}
		for _, tt := range tests {
			env := newOutageEnv(t)
			tt.set(env.repo)
			assert.ErrorContains(t, env.settle(t, webState("exited", 1, 5)), tt.want, tt.name)
		}
	})
	t.Run("a failed close is returned", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		env.repo.closeErr = errors.New("close failed")
		assert.ErrorContains(t, env.settle(t, webState("running", 0, 0)), "close failed")
	})
}

func TestRecoverAfterDeploy(t *testing.T) {
	healthy := func(t *testing.T, env *outageEnv, services ...ObservedService) {
		t.Helper()
		seedDeploy(t, env.deploys, &Deploy{ID: "d1", StackID: outageStack, Status: StatusRunning, CreatedAt: at(0), UpdatedAt: at(0)})
		ev := busEvent(t, TopicDeployStatusChanged, DeployStatusChangedEvent{ID: "d1", Status: "healthy", Services: services})
		require.NoError(t, env.svc.HandleStatusChanged(t.Context(), ev))
	}
	t.Run("a healthy deploy starts the wait for a service it brought up", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			env := newOutageEnv(t)
			env.svc.now = time.Now
			env.handle(t, dockerEvent(containerDie, 1))
			healthy(t, env, ObservedService{Name: "web", Status: "healthy"}, ObservedService{Name: "worker", Status: "running"})

			time.Sleep(outageSteady - time.Second)
			synctest.Wait()
			assert.Equal(t, 1, env.repo.openCount(outageStack))
			time.Sleep(time.Second)
			synctest.Wait()
			assert.Zero(t, env.repo.openCount(outageStack))
			assert.Len(t, env.repo.events(t, TopicContainerRecovered), 1)
		})
	})
	t.Run("a service the deploy did not bring up stays open", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			env := newOutageEnv(t)
			env.svc.now = time.Now
			env.handle(t, dockerEvent(containerDie, 1))
			healthy(t, env, ObservedService{Name: "web", Status: "exited"})

			time.Sleep(10 * outageSteady)
			synctest.Wait()
			assert.Equal(t, 1, env.repo.openCount(outageStack))
		})
	})
	t.Run("a failed outage read does not fail the deploy's own status", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		env.repo.listErr = errors.New("list failed")
		healthy(t, env, ObservedService{Name: "web", Status: "healthy"})
		env.repo.listErr = nil
		assert.Equal(t, 1, env.repo.openCount(outageStack))
	})
	t.Run("a failed deploy starts no wait", func(t *testing.T) {
		env := newOutageEnv(t)
		env.handle(t, dockerEvent(containerDie, 1))
		seedDeploy(t, env.deploys, &Deploy{ID: "d1", StackID: outageStack, Status: StatusRunning, CreatedAt: at(0), UpdatedAt: at(0)})
		ev := busEvent(t, TopicDeployStatusChanged, DeployStatusChangedEvent{ID: "d1", Status: "failed", Services: []ObservedService{{Name: "web", Status: "running"}}})
		require.NoError(t, env.svc.HandleStatusChanged(t.Context(), ev))
		assert.Empty(t, env.svc.recoveries)
	})
}

func TestOpenOutages(t *testing.T) {
	env := newOutageEnv(t)
	got, err := env.svc.OpenOutages(t.Context(), outageStack)
	require.NoError(t, err)
	assert.Empty(t, got)

	env.handle(t, dockerEvent(containerDie, 1))
	got, err = env.svc.OpenOutages(t.Context(), outageStack)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "web", got[0].Service)

	env.repo.listErr = errors.New("list failed")
	_, err = env.svc.OpenOutages(t.Context(), outageStack)
	assert.ErrorContains(t, err, "list failed")

	env.svc.outages = nil
	got, err = env.svc.OpenOutages(t.Context(), outageStack)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetStackRoute_CarriesTheOpenOutages(t *testing.T) {
	env := newOutageEnv(t)
	h := NewHandler(env.svc).Routes()
	get := func() stackResponse {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/"+outageStack, nil))
		require.Equal(t, http.StatusOK, rec.Code)
		var resp stackResponse
		decodeDeploy(t, rec, &resp)
		return resp
	}

	assert.Equal(t, `[]`, rawOutages(t, h), "a stack with nothing down says so with an empty list")
	env.handle(t, dockerEvent(containerOOM, 0))
	env.handle(t, dockerEvent(containerDie, 137))

	resp := get()
	require.Len(t, resp.Outages, 1)
	assert.Equal(t, "web", resp.Outages[0].Service)
	assert.Equal(t, 137, resp.Outages[0].ExitCode)
	assert.True(t, resp.Outages[0].OOM)

	env.repo.listErr = errors.New("list failed")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/"+outageStack, nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func rawOutages(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/"+outageStack, nil))
	var raw map[string]json.RawMessage
	decodeDeploy(t, rec, &raw)
	return string(raw["outages"])
}

func TestStackGet_ListsTheServicesThatAreDown(t *testing.T) {
	env := newOutageEnv(t)
	got, err := call(t, env.svc, "stack_get", `{"id":"`+outageStack+`"}`)
	require.NoError(t, err)
	assert.Empty(t, got.(stackDetail).OpenOutages)
	assert.Contains(t, asJSON(t, got), `"open_outages":[]`)

	env.handle(t, dockerEvent(containerDie, 137))
	env.handle(t, dockerEvent(containerDie, 1))
	got, err = call(t, env.svc, "stack_get", `{"id":"`+outageStack+`"}`)
	require.NoError(t, err)
	open := got.(stackDetail).OpenOutages
	require.Len(t, open, 1)
	assert.Equal(t, outageResult{Service: "web", Since: env.svc.now().UTC(), ExitCode: 1, RestartCount: 1}, open[0])

	env.repo.listErr = errors.New("list failed")
	_, err = call(t, env.svc, "stack_get", `{"id":"`+outageStack+`"}`)
	assert.ErrorContains(t, err, "list failed")
}
